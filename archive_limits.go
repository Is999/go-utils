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
	entryCount int
	totalSize  int64
}

// add 在写入磁盘前校验条目数量与大小是否超过安全阈值。
//
// 参数说明：
//   - entryName：当前条目名称
//   - size：当前条目的解压后大小
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
//
// 参数说明：
//   - outputFile：最终归档文件路径
//   - suffix：归档文件名后缀，例如 .zip、.tar、.tar.gz
//   - format：归档格式名称，用于错误信息
//   - files：待打包文件或目录列表
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

// isPathInside 判断 target 是否位于 root 路径内部或与 root 相同。
func isPathInside(root, target string) (bool, error) {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false, errors.Tag(err)
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

// rejectArchiveSymlink 统一拒绝打包符号链接，避免归档结果依赖宿主机路径状态。
//
// 参数说明：
//   - filePath：待归档路径
//   - mode：文件模式
//   - format：归档格式名称
func rejectArchiveSymlink(filePath string, mode os.FileMode, format string) error {
	if mode&os.ModeSymlink != 0 {
		return errors.Errorf("%s 打包不支持符号链接: %s", format, filePath)
	}
	return nil
}

// assertNoSymlinkPath 校验 root 到 target 的路径链路上不存在符号链接。
//
// 用途：
//   - tar/zip 解包时，即使归档条目本身不是符号链接，磁盘上的既有目录/文件仍可能是符号链接；
//     这会导致 OpenFile/MkdirAll 跟随符号链接，把内容写到解包目录之外。
//
// 参数说明：
//   - root：解包根目录（绝对/相对均可）
//   - target：即将写入/创建的目标路径（绝对/相对均可）
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
