package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Is999/go-utils/errors"
)

// 归档安全限制常量用于控制解压资源消耗。
const (
	// archiveMaxEntries 限制单个压缩包内的最大条目数，避免异常归档占用过多 inode 与 CPU。
	archiveMaxEntries = 10000
	// archiveMaxSingleFileSize 限制单个条目的最大解压大小，避免单文件异常膨胀。
	archiveMaxSingleFileSize int64 = 256 << 20
	// archiveMaxTotalSize 限制整个压缩包的总解压大小，降低 zip bomb/tar bomb 风险。
	archiveMaxTotalSize int64 = 1 << 30
)

// archiveCounter 统计解压过程中的条目数与累计展开大小。
type archiveCounter struct {
	entryCount int   // 已处理条目数
	totalSize  int64 // 已累计展开大小
}

// add 在写入磁盘前校验条目数量与大小是否超过安全阈值。
func (c *archiveCounter) add(entryName string, size int64) error {
	c.entryCount++
	if c.entryCount > archiveMaxEntries {
		return errors.Errorf("压缩包条目过多: name=%s, count=%d, limit=%d", entryName, c.entryCount, archiveMaxEntries)
	}

	if size < 0 {
		return errors.Errorf("压缩包条目大小非法: name=%s, size=%d", entryName, size)
	}
	if size > archiveMaxSingleFileSize {
		return errors.Errorf("压缩包单文件过大: name=%s, size=%d, limit=%d", entryName, size, archiveMaxSingleFileSize)
	}

	c.totalSize += size
	if c.totalSize > archiveMaxTotalSize {
		return errors.Errorf("压缩包总展开大小超限: name=%s, total=%d, limit=%d", entryName, c.totalSize, archiveMaxTotalSize)
	}
	return nil
}

// validateArchiveOutput 校验归档输出路径和输入列表。
func validateArchiveOutput(outputFile, suffix, format string, files []string) error {
	outputFile = strings.TrimSpace(outputFile)
	if outputFile == "" {
		return errors.Errorf("%s 输出文件名不能为空", format)
	}
	if !strings.HasSuffix(outputFile, suffix) {
		return errors.Errorf("文件名错误：非%s文件", suffix)
	}

	outputAbs, err := filepath.Abs(outputFile)
	if err != nil {
		return errors.Tag(err)
	}

	// 输出文件不能位于待打包目录内，否则临时文件或最终归档可能被打包进自身。
	for _, filePath := range files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			return errors.Errorf("%s 待打包路径不能为空", format)
		}

		info, err := os.Lstat(filePath)
		if err != nil {
			return errors.Tag(err)
		}
		inputAbs, err := filepath.Abs(filePath)
		if err != nil {
			return errors.Tag(err)
		}

		if info.IsDir() {
			inside, err := isPathInside(inputAbs, outputAbs)
			if err != nil {
				return errors.Tag(err)
			}
			if inside {
				return errors.Errorf("%s 输出文件不能位于待打包目录内: output=%s input=%s", format, outputFile, filePath)
			}
			continue
		}

		if filepath.Clean(inputAbs) == filepath.Clean(outputAbs) {
			return errors.Errorf("%s 输出文件不能与待打包文件相同: %s", format, outputFile)
		}
	}
	return nil
}

// prepareArchiveDestRoot 规范化并创建解包根目录。
// 返回的路径为绝对路径，后续条目必须限制在该目录内。
func prepareArchiveDestRoot(destDir string) (string, error) {
	destRoot, err := filepath.Abs(destDir)
	if err != nil {
		return "", errors.Tag(err)
	}
	if err = os.MkdirAll(destRoot, 0755); err != nil {
		return "", errors.Tag(err)
	}
	if err = assertNoSymlinkPath(destRoot, destRoot); err != nil {
		return "", errors.Tag(err)
	}
	return destRoot, nil
}

// safeArchivePath 计算安全的解包目标路径。
// 仅允许写入目标目录内，拒绝空路径、绝对路径、目录穿越和 Windows 风格分隔符绕过。
func safeArchivePath(format, destRoot, entryName string) (string, error) {
	if entryName == "" {
		return "", errors.Errorf("%s 条目名称不能为空", format)
	}
	if strings.Contains(entryName, "\x00") {
		return "", errors.Errorf("%s 条目名称不能包含空字符", format)
	}

	normalizedName := strings.ReplaceAll(entryName, "\\", "/")
	if strings.HasPrefix(normalizedName, "/") || filepath.IsAbs(normalizedName) {
		return "", errors.Errorf("%s 条目不允许使用绝对路径: %s", format, entryName)
	}

	cleanName := filepath.Clean(filepath.FromSlash(normalizedName))
	if cleanName == "." {
		return destRoot, nil
	}

	destPath := filepath.Join(destRoot, cleanName)
	relPath, err := filepath.Rel(destRoot, destPath)
	if err != nil {
		return "", errors.Tag(err)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", errors.Errorf("%s 条目路径越界: %s", format, entryName)
	}
	return destPath, nil
}

// createArchiveDir 安全创建解包目录并恢复权限。
func createArchiveDir(destRoot, destPath string, perm os.FileMode) error {
	if err := assertNoSymlinkPath(destRoot, destPath); err != nil {
		return errors.Tag(err)
	}
	if err := os.MkdirAll(destPath, perm); err != nil {
		return errors.Tag(err)
	}
	if err := assertNoSymlinkPath(destRoot, destPath); err != nil {
		return errors.Tag(err)
	}
	if err := os.Chmod(destPath, perm); err != nil {
		return errors.Tag(err)
	}
	return nil
}

// writeArchiveFile 安全写入解包文件并恢复权限。
func writeArchiveFile(destRoot, destPath string, perm os.FileMode, write func(file *os.File) error) error {
	if err := assertNoSymlinkPath(destRoot, destPath); err != nil {
		return errors.Tag(err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return errors.Tag(err)
	}
	if err := writeFileAtomic(destPath, perm, write); err != nil {
		return errors.Tag(err)
	}
	if err := os.Chmod(destPath, perm); err != nil {
		return errors.Tag(err)
	}
	return nil
}

// archiveDirPerm 计算解包目录权限。
// 目录至少保留拥有者的读写执行权限，避免创建出不可进入的目录。
func archiveDirPerm(mode os.FileMode) os.FileMode {
	perm := mode & os.ModePerm
	if perm == 0 {
		return 0755
	}
	if perm&0700 != 0700 {
		perm |= 0700
	}
	return perm
}

// archiveFilePerm 计算解包文件权限。
func archiveFilePerm(mode os.FileMode) os.FileMode {
	perm := mode & os.ModePerm
	if perm == 0 {
		return 0644
	}
	return perm
}

// isPathInside 判断 target 是否位于 root 路径内部或与 root 相同。
func isPathInside(root, target string) (bool, error) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false, errors.Tag(err)
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

// rejectArchiveSymlink 统一拒绝打包符号链接，避免归档结果依赖宿主机路径状态。
func rejectArchiveSymlink(filePath string, mode os.FileMode, format string) error {
	if mode&os.ModeSymlink != 0 {
		return errors.Errorf("%s 打包不支持符号链接: %s", format, filePath)
	}
	return nil
}

// assertNoSymlinkPath 校验 root 到 target 的路径链路上不存在符号链接。
//
// tar/zip 解包时，即使归档条目本身不是符号链接，磁盘上的既有目录或文件仍可能是符号链接；
// 这会导致 OpenFile/MkdirAll 跟随符号链接，把内容写到解包目录之外。
func assertNoSymlinkPath(root, target string) error {
	root = strings.TrimSpace(root)
	target = strings.TrimSpace(target)
	if root == "" || target == "" {
		return errors.New("路径不能为空")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return errors.Tag(err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return errors.Tag(err)
	}
	if info, err := os.Lstat(rootAbs); err == nil {
		if info.Mode()&os.ModeSymlink != 0 && !isAllowedSystemSymlinkRoot(rootAbs) {
			return errors.Errorf("不允许符号链接路径段: %s", rootAbs)
		}
	} else if !os.IsNotExist(err) {
		return errors.Tag(err)
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return errors.Tag(err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.Errorf("路径越界: root=%s target=%s", rootAbs, targetAbs)
	}
	if rel == "." {
		return nil
	}
	parts := strings.Split(rel, string(filepath.Separator))
	cur := rootAbs
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return errors.Tag(err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.Errorf("不允许符号链接路径段: %s", cur)
		}
	}
	return nil
}

// isAllowedSystemSymlinkRoot 放行操作系统级临时目录别名。
//
// macOS 上 /tmp 通常是指向 /private/tmp 的系统符号链接；历史调用方和测试都可能直接使用 /tmp。
// 这里仍会拒绝业务目录中的符号链接，只避免把系统临时目录入口误判为攻击路径。
func isAllowedSystemSymlinkRoot(path string) bool {
	cleanPath := filepath.Clean(path)
	return cleanPath == filepath.Clean(os.TempDir()) || cleanPath == string(filepath.Separator)+"tmp"
}
