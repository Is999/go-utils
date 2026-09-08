package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Is999/go-utils/errors"
)

// 解包限制按文件和目录条目计数，大小单位为字节。
const (
	// archiveMaxEntries 限制单次解包处理的文件和目录条目数，重复条目仍计数。
	archiveMaxEntries = 10000
	// archiveMaxSingleFileSize 限制归档中声明的单文件展开大小。
	archiveMaxSingleFileSize int64 = 256 << 20
	// archiveMaxTotalSize 限制累计文件内容；tar.gz 还计入 tar 结束后的展开数据。
	archiveMaxTotalSize int64 = 1 << 30
)

// archiveCounter 在单次解包中累计条目及其声明大小，不跨 goroutine 共享。
type archiveCounter struct {
	entryCount int   // 已计数的文件和目录条目。
	totalSize  int64 // 文件条目声明大小之和，单位字节。
}

// add 在写盘前计数并校验限制；调用方遇错即终止解包，计数无需回滚。
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
	// 后缀校验沿用去除首尾空白的规则，磁盘访问则保留文件名中的空格。
	trimmedOutput := strings.TrimSpace(outputFile)
	if trimmedOutput == "" {
		return errors.Errorf("%s 输出文件名不能为空", format)
	}
	if !strings.HasSuffix(trimmedOutput, suffix) {
		return errors.Errorf("文件名错误：非%s文件", suffix)
	}

	outputAbs, err := filepath.Abs(outputFile)
	if err != nil {
		return errors.Tag(err)
	}

	// 输出文件不能位于待打包目录内，否则临时文件或最终归档可能被打包进自身。
	for _, filePath := range files {
		if strings.TrimSpace(filePath) == "" {
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
			// 用相对路径判断目录归属，避免字符串前缀误匹配同名前缀目录。
			rel, err := filepath.Rel(inputAbs, outputAbs)
			if err != nil {
				return errors.Tag(err)
			}
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
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

// prepareArchiveDestRoot 返回已创建的绝对根目录，供后续条目复用。
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

// safeArchivePath 限制条目路径文本在 destRoot 内，符号链接由写入入口检查。
func safeArchivePath(format, destRoot, entryName string) (string, error) {
	if entryName == "" {
		return "", errors.Errorf("%s 条目名称不能为空", format)
	}
	if strings.Contains(entryName, "\x00") {
		return "", errors.Errorf("%s 条目名称不能包含空字符", format)
	}

	// 反斜杠也按分隔符校验，保持跨平台归档的目录边界。
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

// createArchiveDir 创建解包目录并设置权限，已有目录也会应用 perm。
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
	return errors.Tag(os.Chmod(destPath, perm))
}

// writeArchiveFile 逐个替换解包文件，最终权限在临时文件替换目标前设置。
func writeArchiveFile(destRoot, destPath string, perm os.FileMode, write func(file *os.File) error) error {
	if err := assertNoSymlinkPath(destRoot, destPath); err != nil {
		return errors.Tag(err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return errors.Tag(err)
	}
	return errors.Tag(writeFileAtomic(destPath, perm, write))
}

// archiveDirPerm 为目录补齐拥有者的读写执行权限；未提供权限时使用 0755。
func archiveDirPerm(mode os.FileMode) os.FileMode {
	perm := mode & os.ModePerm
	if perm == 0 {
		return 0755
	}
	return perm | 0700
}

// archiveFilePerm 仅保留普通权限位，归档未提供权限时使用 0644。
func archiveFilePerm(mode os.FileMode) os.FileMode {
	perm := mode & os.ModePerm
	if perm == 0 {
		return 0644
	}
	return perm
}

// rejectArchiveSymlink 拒绝打包符号链接，不跟随链接收集目标内容。
func rejectArchiveSymlink(filePath string, mode os.FileMode, format string) error {
	if mode&os.ModeSymlink != 0 {
		return errors.Errorf("%s 打包不支持符号链接: %s", format, filePath)
	}
	return nil
}

// assertNoSymlinkPath 检查规范化后的 root 至 target 路径段，不检查 root 的祖先目录或锁定路径。
func assertNoSymlinkPath(root, target string) error {
	// 只在空值判断时裁剪，路径中的空格须与后续文件操作保持一致。
	if strings.TrimSpace(root) == "" || strings.TrimSpace(target) == "" {
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
	// 缺失路径留给后续创建，其他读取错误仍须返回。
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
	cur := rootAbs
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
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

// isAllowedSystemSymlinkRoot 兼容 os.TempDir() 和 /tmp 根别名，不放行其下的符号链接。
func isAllowedSystemSymlinkRoot(path string) bool {
	cleanPath := filepath.Clean(path)
	return cleanPath == filepath.Clean(os.TempDir()) || cleanPath == string(filepath.Separator)+"tmp"
}
