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

// Tar 使用tar打包
//
//	tarFile 打包后文件
//	files 待打包文件【夹】
func Tar(tarFile string, files []string) error {
	if err := validateArchiveOutput(tarFile, ".tar", "tar", files); err != nil {
		return errors.Tag(err)
	}

	// 使用原子写入避免归档生成失败时留下半截 tar 文件。
	return writeFileAtomic(tarFile, 0644, func(file *os.File) (err error) {
		// 创建 tar 写入器，最终必须显式关闭以刷新尾部块。
		tarWriter := tar.NewWriter(file)
		defer func() {
			if closeErr := tarWriter.Close(); err == nil && closeErr != nil {
				err = errors.Tag(closeErr)
			}
		}()

		// 遍历文件和目录列表，将它们添加到 tar 归档文件中。
		for _, filePath := range files {
			if err = AddFileToTar(tarWriter, filePath, ""); err != nil {
				return errors.Tag(err)
			}
		}
		return nil
	})
}

// TarGz 使用tar打包gzip压缩
//
//	tarGzFile 打包压缩后文件
//	files 待打包压缩文件【夹】
func TarGz(tarGzFile string, files []string) error {
	if err := validateArchiveOutput(tarGzFile, ".tar.gz", "tar.gz", files); err != nil {
		return errors.Tag(err)
	}

	// 使用原子写入避免归档生成失败时留下半截 tar.gz 文件。
	return writeFileAtomic(tarGzFile, 0644, func(file *os.File) (err error) {
		// 创建 gzip 写入器，tar 数据会先写入 gzip 流。
		gzipWriter := gzip.NewWriter(file)
		defer func() {
			if closeErr := gzipWriter.Close(); err == nil && closeErr != nil {
				err = errors.Tag(closeErr)
			}
		}()

		// 创建 tar 写入器，关闭顺序必须先 tar 后 gzip。
		tarWriter := tar.NewWriter(gzipWriter)
		defer func() {
			if closeErr := tarWriter.Close(); err == nil && closeErr != nil {
				err = errors.Tag(closeErr)
			}
		}()

		// 遍历文件和目录列表，将它们添加到 tar 归档文件中。
		for _, filePath := range files {
			if err = AddFileToTar(tarWriter, filePath, ""); err != nil {
				return errors.Tag(err)
			}
		}
		return nil
	})
}

// AddFileToTar 添加文件【夹】到tar
//
//	fileToCompress 需要压缩的文件
//	baseDir 打包文件根目录
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

	if fileInfo.IsDir() {
		// 压缩目录
		archiveBaseDir := fileInfo.Name()
		if baseDir != "" {
			archiveBaseDir = filepath.Join(baseDir, archiveBaseDir)
		}
		return addDirectoryToTar(tarWriter, fileToCompress, fileInfo, archiveBaseDir)
	}
	// 压缩文件
	return addSingleFileToTar(tarWriter, fileToCompress, fileInfo, baseDir)
}

// addSingleFileToTar 添加单个文件到tar
func addSingleFileToTar(tarWriter *tar.Writer, fileToCompress string, fileInfo os.FileInfo, baseDir string) error {
	// 创建一个新的tar文件头
	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil {
		return errors.Tag(err)
	}

	// 修改 header 中的 Name 字段，确保文件名正确
	header.Name = filepath.ToSlash(filepath.Join(baseDir, header.Name))

	// 将tar文件头写入tar归档文件
	err = tarWriter.WriteHeader(header)
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

		// 将文件数据拷贝到tar归档文件
		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return errors.Tag(err)
		}
	}

	return nil
}

// addDirectoryToTar 添加目录到tar
func addDirectoryToTar(tarWriter *tar.Writer, directoryToCompress string, fileInfo os.FileInfo, baseDir string) error {
	// 压缩目录
	err := addSingleFileToTar(tarWriter, directoryToCompress, fileInfo, strings.TrimSuffix(baseDir, fileInfo.Name()))
	if err != nil {
		return errors.Tag(err)
	}

	// 读取目录
	files, err := os.ReadDir(directoryToCompress)
	if err != nil {
		return errors.Tag(err)
	}

	for _, file := range files {
		if err = rejectArchiveSymlink(filepath.Join(directoryToCompress, file.Name()), file.Type(), "tar"); err != nil {
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
			if err = addDirectoryToTar(tarWriter, filePath, info, filepath.Join(baseDir, file.Name())); err != nil {
				return errors.Tag(err)
			}
			continue
		}
		// 压缩单个文件
		if err = addSingleFileToTar(tarWriter, filePath, info, baseDir); err != nil {
			return errors.Tag(err)
		}
	}

	return nil
}

// UnTar 解压.tar或.tar.gz文件
//
//	tarFile 代解压的文件
//	destDir 解压文件目录
func UnTar(tarFile, destDir string) error {
	if !(strings.HasSuffix(tarFile, ".tar") || strings.HasSuffix(tarFile, ".tar.gz")) {
		return errors.New("文件类型错误：非.tar、.tar.gz文件")
	}

	// 打开tar归档文件
	file, err := os.Open(tarFile)
	if err != nil {
		return errors.Tag(err)
	}
	defer file.Close()

	var reader io.Reader

	// 判断解压文件是否是.gz
	if strings.HasSuffix(tarFile, ".tar.gz") {
		// 创建 gzip.Reader 用于读取压缩数据
		gzReader, gzErr := gzip.NewReader(file)
		if gzErr != nil {
			return errors.Tag(gzErr)
		}
		defer gzReader.Close()
		reader = gzReader
	} else {
		reader = file
	}

	// 创建一个tar读取器
	tarReader := tar.NewReader(reader)

	destRoot, err := prepareArchiveDestRoot(destDir)
	if err != nil {
		return errors.Tag(err)
	}

	// 遍历tar归档文件中的每个文件条目
	counter := archiveCounter{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// 读取完所有文件条目
			break
		}
		if err != nil {
			return errors.Tag(err)
		}

		if err = extractTarEntry(destRoot, tarReader, header, &counter); err != nil {
			return errors.Tag(err)
		}
	}

	return nil
}

// extractTarEntry 解包单个 tar 条目。
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
			_, copyErr := io.Copy(file, reader)
			if copyErr != nil {
				return errors.Tag(copyErr)
			}
			return nil
		})
	case tar.TypeXGlobalHeader, tar.TypeXHeader:
		return nil
	default:
		return errors.Errorf("不支持的 tar 条目类型: %d, name=%s", header.Typeflag, header.Name)
	}
}
