package utils

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Is999/go-utils/errors"
)

// Zip 将文件和目录打包为 .zip，保留各输入的基名；空列表生成空归档。
// 归档写完并关闭后替换目标，输出父目录须已存在，且不能位于输入目录中。
func Zip(zipFile string, files []string) error {
	if err := validateArchiveOutput(zipFile, ".zip", "zip", files); err != nil {
		return errors.Tag(err)
	}

	// 使用原子写入避免归档生成失败时留下半截 zip 文件。
	return writeFileAtomic(zipFile, 0644, func(file *os.File) (err error) {
		zipWriter := zip.NewWriter(file)
		// Close 补齐 ZIP 中央目录，但不覆盖之前的写入错误。
		defer func() {
			if closeErr := zipWriter.Close(); err == nil && closeErr != nil {
				err = errors.Tag(closeErr)
			}
		}()

		for _, filePath := range files {
			if err = AddFileToZip(zipWriter, filePath, ""); err != nil {
				return errors.Tag(err)
			}
		}
		return nil
	})
}

// AddFileToZip 递归添加普通文件或目录，条目名为 baseDir 加输入基名，拒绝符号链接等特殊文件。
// baseDir 是归档内前缀；调用方负责关闭 zipWriter，失败时已写入的条目不会撤销。
func AddFileToZip(zipWriter *zip.Writer, fileToCompress string, baseDir string) error {
	if zipWriter == nil {
		return errors.New("zip.Writer 不能为空")
	}
	fileInfo, err := os.Lstat(fileToCompress)
	if err != nil {
		return errors.Tag(err)
	}
	if err = rejectArchiveSymlink(fileToCompress, fileInfo.Mode(), "zip"); err != nil {
		return errors.Tag(err)
	}

	return addFileToZip(zipWriter, fileToCompress, fileInfo, baseDir)
}

// addFileToZip 将普通文件或目录写入归档，保留目录层次。
func addFileToZip(zipWriter *zip.Writer, filePath string, fileInfo os.FileInfo, baseDir string) error {
	// FIFO 等特殊文件不能按普通正文读取，否则可能一直等待外部读写端。
	if !fileInfo.IsDir() && !fileInfo.Mode().IsRegular() {
		return errors.Errorf("不支持的 zip 条目类型: %s", filePath)
	}
	header, err := zip.FileInfoHeader(fileInfo)
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

	// ZIP 以末尾 / 标记目录，标准库据此写入无压缩数据的目录条目。
	if fileInfo.IsDir() && !strings.HasSuffix(header.Name, "/") {
		header.Name += "/"
	}

	header.Method = zip.Deflate
	// 先写目录自身的条目，空目录才不会在归档中丢失。
	zipFile, err := zipWriter.CreateHeader(header)
	if err != nil {
		return errors.Tag(err)
	}
	if !fileInfo.IsDir() {
		file, err := os.Open(filePath)
		if err != nil {
			return errors.Tag(err)
		}
		// 每个文件复制后即关闭，递归遍历不积压文件句柄。
		defer file.Close()
		_, err = io.Copy(zipFile, file)
		return errors.Tag(err)
	}

	files, err := os.ReadDir(filePath)
	if err != nil {
		return errors.Tag(err)
	}
	for _, file := range files {
		childPath := filepath.Join(filePath, file.Name())
		if err = rejectArchiveSymlink(childPath, file.Type(), "zip"); err != nil {
			return errors.Tag(err)
		}
		info, err := file.Info()
		if err != nil {
			return errors.Tag(err)
		}
		if err = addFileToZip(zipWriter, childPath, info, archiveDir); err != nil {
			return errors.Tag(err)
		}
	}
	return nil
}

// UnZip 解包 .zip 文件，自动创建 destDir，逐个替换普通文件并恢复权限。
// 任一条目失败即返回错误，已完成的文件和目录仍会保留。
func UnZip(zipFile, destDir string) error {
	// 与打包端使用相同的后缀规则，磁盘读取仍保留完整路径。
	if !strings.HasSuffix(strings.TrimSpace(zipFile), ".zip") {
		return errors.New("文件名错误：非.zip文件")
	}

	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return errors.Tag(err)
	}
	defer r.Close()

	destRoot, err := prepareArchiveDestRoot(destDir)
	if err != nil {
		return errors.Tag(err)
	}

	counter := archiveCounter{}
	for _, file := range r.File {
		if err = extractZipEntry(destRoot, file, &counter); err != nil {
			return errors.Tag(err)
		}
	}

	return nil
}

// extractZipEntry 仅接受目录和普通文件。
func extractZipEntry(destRoot string, f *zip.File, counter *archiveCounter) error {
	destPath, err := safeArchivePath("zip", destRoot, f.Name)
	if err != nil {
		return errors.Tag(err)
	}

	mode := f.Mode()
	if mode&os.ModeSymlink != 0 || !mode.IsDir() && !mode.IsRegular() {
		return errors.Errorf("不支持的 zip 条目类型: %s", f.Name)
	}

	if mode.IsDir() {
		if err = counter.add(f.Name, 0); err != nil {
			return errors.Tag(err)
		}
		return createArchiveDir(destRoot, destPath, archiveDirPerm(mode))
	}

	if err = counter.add(f.Name, int64(f.UncompressedSize64)); err != nil {
		return errors.Tag(err)
	}
	rc, err := f.Open()
	if err != nil {
		return errors.Tag(err)
	}
	writeErr := writeArchiveFile(destRoot, destPath, archiveFilePerm(mode), func(file *os.File) error {
		_, copyErr := io.Copy(file, rc)
		return errors.Tag(copyErr)
	})
	// 写入失败也先关闭读取器，保留写入错误作为主因。
	closeErr := rc.Close()
	if writeErr != nil {
		return errors.Tag(writeErr)
	}
	return errors.Tag(closeErr)
}
