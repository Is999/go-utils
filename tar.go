package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Is999/go-utils/errors"
)

// Tar 将文件和目录打包为 .tar，保留各输入的基名；空列表生成空归档。
// 归档写完并关闭后替换目标，输出父目录须已存在，且不能位于输入目录中。
func Tar(tarFile string, files []string) error {
	if err := validateArchiveOutput(tarFile, ".tar", "tar", files); err != nil {
		return errors.Tag(err)
	}

	// 使用原子写入避免归档生成失败时留下半截 tar 文件。
	return writeFileAtomic(tarFile, 0644, func(file *os.File) (err error) {
		tarWriter := tar.NewWriter(file)
		// Close 补齐 tar 结束块，但不覆盖之前的写入错误。
		defer func() {
			if closeErr := tarWriter.Close(); err == nil && closeErr != nil {
				err = errors.Tag(closeErr)
			}
		}()

		for _, filePath := range files {
			if err = AddFileToTar(tarWriter, filePath, ""); err != nil {
				return errors.Tag(err)
			}
		}
		return nil
	})
}

// TarGz 将文件和目录打包为 .tar.gz，输入及目标路径规则与 Tar 相同。
func TarGz(tarGzFile string, files []string) error {
	if err := validateArchiveOutput(tarGzFile, ".tar.gz", "tar.gz", files); err != nil {
		return errors.Tag(err)
	}

	// 使用原子写入避免归档生成失败时留下半截 tar.gz 文件。
	return writeFileAtomic(tarGzFile, 0644, func(file *os.File) (err error) {
		gzipWriter := gzip.NewWriter(file)
		defer func() {
			if closeErr := gzipWriter.Close(); err == nil && closeErr != nil {
				err = errors.Tag(closeErr)
			}
		}()

		tarWriter := tar.NewWriter(gzipWriter)
		// 先结束 tar 再关闭 gzip，保证结束块也写入压缩流。
		defer func() {
			if closeErr := tarWriter.Close(); err == nil && closeErr != nil {
				err = errors.Tag(closeErr)
			}
		}()

		for _, filePath := range files {
			if err = AddFileToTar(tarWriter, filePath, ""); err != nil {
				return errors.Tag(err)
			}
		}
		return nil
	})
}

// AddFileToTar 递归添加普通文件或目录，条目名为 baseDir 加输入基名，拒绝符号链接等特殊文件。
// baseDir 是归档内前缀；调用方负责关闭 tarWriter，失败时已写入的条目不会撤销。
func AddFileToTar(tarWriter *tar.Writer, fileToCompress string, baseDir string) error {
	if tarWriter == nil {
		return errors.New("tar.Writer 不能为空")
	}
	fileInfo, err := os.Lstat(fileToCompress)
	if err != nil {
		return errors.Tag(err)
	}
	if err = rejectArchiveSymlink(fileToCompress, fileInfo.Mode(), "tar"); err != nil {
		return errors.Tag(err)
	}

	return addFileToTar(tarWriter, fileToCompress, fileInfo, baseDir)
}

// addFileToTar 将普通文件或目录写入归档，保留目录层次。
func addFileToTar(tarWriter *tar.Writer, filePath string, fileInfo os.FileInfo, baseDir string) error {
	// FIFO 等特殊文件不能按普通正文读取，否则可能一直等待外部读写端。
	if !fileInfo.IsDir() && !fileInfo.Mode().IsRegular() {
		return errors.Errorf("不支持的 tar 条目类型: %s", filePath)
	}
	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil {
		return errors.Tag(err)
	}

	archiveDir := baseDir
	if fileInfo.IsDir() {
		// 沿用先拼目录全名再取父前缀的规则，保留输入为 . 或 .. 时的条目名。
		archiveDir = filepath.Join(baseDir, fileInfo.Name())
		baseDir = strings.TrimSuffix(archiveDir, fileInfo.Name())
	}
	// 归档内统一使用正斜杠，与宿主机路径分隔符无关。
	header.Name = filepath.ToSlash(filepath.Join(baseDir, header.Name))

	// 先写目录自身的条目，空目录才不会在归档中丢失。
	if err = tarWriter.WriteHeader(header); err != nil {
		return errors.Tag(err)
	}
	if !fileInfo.IsDir() {
		file, err := os.Open(filePath)
		if err != nil {
			return errors.Tag(err)
		}
		// 每个文件复制后即关闭，递归遍历不积压文件句柄。
		defer file.Close()
		_, err = io.Copy(tarWriter, file)
		return errors.Tag(err)
	}

	files, err := os.ReadDir(filePath)
	if err != nil {
		return errors.Tag(err)
	}
	for _, file := range files {
		childPath := filepath.Join(filePath, file.Name())
		if err = rejectArchiveSymlink(childPath, file.Type(), "tar"); err != nil {
			return errors.Tag(err)
		}
		info, err := file.Info()
		if err != nil {
			return errors.Tag(err)
		}
		if err = addFileToTar(tarWriter, childPath, info, archiveDir); err != nil {
			return errors.Tag(err)
		}
	}
	return nil
}

// UnTar 根据 .tar/.tar.gz 后缀解包，自动创建 destDir，逐个替换普通文件并恢复权限。
// 任一条目或 gzip 尾部校验失败即返回错误，已完成的文件和目录仍会保留。
func UnTar(tarFile, destDir string) error {
	// 与打包端使用相同的后缀规则，磁盘读取仍保留完整路径。
	trimmedName := strings.TrimSpace(tarFile)
	if !(strings.HasSuffix(trimmedName, ".tar") || strings.HasSuffix(trimmedName, ".tar.gz")) {
		return errors.New("文件类型错误：非.tar、.tar.gz文件")
	}

	file, err := os.Open(tarFile)
	if err != nil {
		return errors.Tag(err)
	}
	defer file.Close()

	var reader io.Reader

	if strings.HasSuffix(trimmedName, ".tar.gz") {
		gzReader, gzErr := gzip.NewReader(file)
		if gzErr != nil {
			return errors.Tag(gzErr)
		}
		defer gzReader.Close()
		reader = gzReader
	} else {
		reader = file
	}

	tarReader := tar.NewReader(reader)

	destRoot, err := prepareArchiveDestRoot(destDir)
	if err != nil {
		return errors.Tag(err)
	}

	counter := archiveCounter{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Tag(err)
		}

		if err = extractTarEntry(destRoot, tarReader, header, &counter); err != nil {
			return errors.Tag(err)
		}
	}

	// tar 结束块不保证 gzip 已读到 EOF；继续读取才能验证尾部校验和及完整性。
	if gzReader, ok := reader.(*gzip.Reader); ok {
		// 尾部填充和后续 gzip 成员也占用剩余展开预算，多读一字节用于判定超限。
		remaining := archiveMaxTotalSize - counter.totalSize
		n, err := io.Copy(io.Discard, io.LimitReader(gzReader, remaining+1))
		if err != nil {
			return errors.Tag(err)
		}
		if n > remaining {
			return errors.Errorf("压缩包总展开大小超限: total=%d, limit=%d", counter.totalSize+n, archiveMaxTotalSize)
		}
	}

	return nil
}

// extractTarEntry 仅写入普通文件和目录，跳过扩展头，其他条目类型报错。
func extractTarEntry(destRoot string, reader io.Reader, header *tar.Header, counter *archiveCounter) error {
	destPath, err := safeArchivePath("tar", destRoot, header.Name)
	if err != nil {
		return errors.Tag(err)
	}

	switch header.Typeflag {
	case tar.TypeDir:
		if err = counter.add(header.Name, 0); err != nil {
			return errors.Tag(err)
		}
		return createArchiveDir(destRoot, destPath, archiveDirPerm(os.FileMode(header.Mode)))
	case tar.TypeReg:
		if err = counter.add(header.Name, header.Size); err != nil {
			return errors.Tag(err)
		}
		return writeArchiveFile(destRoot, destPath, archiveFilePerm(os.FileMode(header.Mode)), func(file *os.File) error {
			// tar.Reader 只暴露当前条目的内容，io.Copy 会在本条目结束处停止。
			_, copyErr := io.Copy(file, reader)
			return errors.Tag(copyErr)
		})
	case tar.TypeXGlobalHeader, tar.TypeXHeader:
		return nil
	default:
		return errors.Errorf("不支持的 tar 条目类型: %d, name=%s", header.Typeflag, header.Name)
	}
}
