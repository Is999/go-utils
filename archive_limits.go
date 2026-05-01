package utils

import (
	"os"

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
