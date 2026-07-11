package utils

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Is999/go-utils/errors"
)

// Zip 使用zip打包并压缩
//
//	zipFile 打包压缩后文件
//	files 待打包压缩文件【夹】
func Zip(zipFile string, files []string) error {
	if err := validateArchiveOutput(zipFile, ".zip", "zip", files); err != nil {
		return errors.Tag(err)
	}

	// 使用原子写入避免归档生成失败时留下半截 zip 文件。
	return writeFileAtomic(zipFile, 0644, func(file *os.File) (err error) {
		// 创建 zip.Writer，最终必须显式关闭以刷新中央目录。
		zipWriter := zip.NewWriter(file)
		defer func() {
			if closeErr := zipWriter.Close(); err == nil && closeErr != nil {
				err = errors.Tag(closeErr)
			}
		}()

		// 遍历文件和目录列表，将它们添加到 zip 文件。
		for _, filePath := range files {
			if err = AddFileToZip(zipWriter, filePath, ""); err != nil {
				return errors.Tag(err)
			}
		}
		return nil
	})
}

// AddFileToZip 添加文件【夹】到zip
//
//	fileToCompress 需要压缩的文件
//	baseDir 打包文件根目录
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

	if fileInfo.IsDir() {
		// 压缩目录
		archiveBaseDir := fileInfo.Name()
		if baseDir != "" {
			archiveBaseDir = filepath.Join(baseDir, archiveBaseDir)
		}
		return addDirectoryToZip(zipWriter, fileToCompress, fileInfo, archiveBaseDir)
	}
	// 压缩文件
	return addSingleFileToZip(zipWriter, fileToCompress, fileInfo, baseDir)
}

// addSingleFileToZip 添加单个文件到zip
func addSingleFileToZip(zipWriter *zip.Writer, fileToCompress string, fileInfo os.FileInfo, baseDir string) error {
	// 创建 zip 文件中的文件头
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return errors.Tag(err)
	}

	// 修改 header 中的 Name 字段，确保文件名正确
	header.Name = filepath.ToSlash(filepath.Join(baseDir, header.Name))

	// 压缩文件
	header.Method = zip.Deflate

	// 创建一个新的ZIP文件条目
	zipFile, err := zipWriter.CreateHeader(header)
	if err != nil {
		return errors.Tag(err)
	}
	if !fileInfo.IsDir() {
		// 打开要压缩的文件
		file, err := os.Open(fileToCompress)
		if err != nil {
			return errors.Tag(err)
		}
		defer file.Close()

		// 将文件数据拷贝到ZIP文件条目
		_, err = io.Copy(zipFile, file)
		if err != nil {
			return errors.Tag(err)
		}
	}

	return nil
}

// addDirectoryToZip 添加目录到zip
func addDirectoryToZip(zipWriter *zip.Writer, directoryToCompress string, fileInfo os.FileInfo, baseDir string) error {
	// 压缩目录
	err := addSingleFileToZip(zipWriter, directoryToCompress, fileInfo, strings.TrimSuffix(baseDir, fileInfo.Name()))
	if err != nil {
		return errors.Tag(err)
	}

	// 读取目录
	files, err := os.ReadDir(directoryToCompress)
	if err != nil {
		return errors.Tag(err)
	}

	for _, file := range files {
		if err = rejectArchiveSymlink(filepath.Join(directoryToCompress, file.Name()), file.Type(), "zip"); err != nil {
			return errors.Tag(err)
		}

		// 获取文件信息
		info, err := file.Info()
		if err != nil {
			return errors.Tag(err)
		}

		// 获取完整路径
		filePath := filepath.Join(directoryToCompress, file.Name())
		if file.IsDir() {
			// 递归地压缩子目录
			if err = addDirectoryToZip(zipWriter, filePath, info, filepath.Join(baseDir, file.Name())); err != nil {
				return errors.Tag(err)
			}
			continue
		}
		// 压缩单个文件
		if err = addSingleFileToZip(zipWriter, filePath, info, baseDir); err != nil {
			return errors.Tag(err)
		}
	}

	return nil
}

// UnZip 解压zip文件
//
//	zipFile 代解压的文件
//	destDir 解压文件目录
func UnZip(zipFile, destDir string) error {
	if !strings.HasSuffix(zipFile, ".zip") {
		return errors.New("文件名错误：非.zip文件")
	}

	// 打开ZIP文件进行读取
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return errors.Tag(err)
	}
	defer r.Close()

	destRoot, err := prepareArchiveDestRoot(destDir)
	if err != nil {
		return errors.Tag(err)
	}

	// 遍历ZIP文件中的文件和目录
	counter := archiveCounter{}
	for _, file := range r.File {
		if err = extractZipEntry(destRoot, file, &counter); err != nil {
			return errors.Tag(err)
		}
	}

	return nil
}

// extractZipEntry 解包单个 zip 条目。
func extractZipEntry(destRoot string, f *zip.File, counter *archiveCounter) error {
	destPath, err := safeArchivePath("zip", destRoot, f.Name)
	if err != nil {
		return errors.Tag(err)
	}

	mode := f.Mode()
	if mode&os.ModeSymlink != 0 || !mode.IsDir() && !mode.IsRegular() {
		return errors.Errorf("不支持的 zip 条目类型: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
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
		if copyErr != nil {
			return errors.Tag(copyErr)
		}
		return nil
	})
	closeErr := rc.Close()
	if writeErr != nil {
		return errors.Tag(writeErr)
	}
	if closeErr != nil {
		return errors.Tag(closeErr)
	}
	return nil
}
