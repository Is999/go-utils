package utils

import (
	"archive/tar"
	"compress/gzip"
	"github.com/Is999/go-utils/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Tar 使用tar打包
//
//	tarFile 打包后文件
//	files 待打包文件【夹】
func Tar(tarFile string, files []string) error {
	if !strings.HasSuffix(tarFile, ".tar") {
		return errors.New("文件名错误：非.tar文件")
	}

	// 创建压缩文件
	file, err := os.Create(tarFile)
	if err != nil {
		return errors.Wrap(err)
	}
	defer file.Close()

	// 创建一个tar写入器
	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()

	// 遍历文件和目录列表，将它们添加到tar归档文件中
	for _, filePath := range files {
		// 将文件添加到 zip 文件
		err = AddFileToTar(tarWriter, filePath, "")
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

// TarGz 使用tar打包gzip压缩
//
//	tarGzFile 打包压缩后文件
//	files 待打包压缩文件【夹】
func TarGz(tarGzFile string, files []string) error {
	if !strings.HasSuffix(tarGzFile, ".tar.gz") {
		return errors.New("文件名错误：非.tar.gz文件")
	}

	// 创建压缩文件
	file, err := os.Create(tarGzFile)
	if err != nil {
		return errors.Wrap(err)
	}
	defer file.Close()

	// 创建 gzip
	gw := gzip.NewWriter(file)
	defer gw.Close()

	// 创建一个tar写入器
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	// 遍历文件和目录列表，将它们添加到tar归档文件中
	for _, filePath := range files {
		// 将文件添加到 zip 文件
		err = AddFileToTar(tarWriter, filePath, "")
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

// AddFileToTar 添加文件【夹】到tar
//
//	fileToCompress 需要压缩的文件
//	baseDir 打包文件根目录
func AddFileToTar(tarWriter *tar.Writer, fileToCompress string, baseDir string) error {
	fileInfo, err := os.Stat(fileToCompress)
	if err != nil {
		return errors.Wrap(err)
	}

	if fileInfo.IsDir() {
		// 压缩目录
		return addDirectoryToTar(tarWriter, fileToCompress, fileInfo, fileInfo.Name())
	} else {
		// 压缩文件
		return addSingleFileToTar(tarWriter, fileToCompress, fileInfo, baseDir)
	}
}

// addSingleFileToTar 添加单个文件到tar
func addSingleFileToTar(tarWriter *tar.Writer, fileToCompress string, fileInfo os.FileInfo, baseDir string) error {
	// 创建一个新的tar文件头
	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil {
		return errors.Wrap(err)
	}

	// 修改 header 中的 Name 字段，确保文件名正确
	header.Name = filepath.ToSlash(filepath.Join(baseDir, header.Name))

	// 将tar文件头写入tar归档文件
	err = tarWriter.WriteHeader(header)
	if err != nil {
		return errors.Wrap(err)
	}

	if !fileInfo.IsDir() {
		// 打开要压缩的文件
		file, err := os.Open(fileToCompress)
		if err != nil {
			return errors.Wrap(err)
		}
		defer file.Close()

		// 将文件数据拷贝到tar归档文件
		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

// addDirectoryToTar 添加目录到tar
func addDirectoryToTar(tarWriter *tar.Writer, directoryToCompress string, fileInfo os.FileInfo, baseDir string) error {
	// 压缩目录
	err := addSingleFileToTar(tarWriter, directoryToCompress, fileInfo, strings.TrimRight(baseDir, fileInfo.Name()))
	if err != nil {
		return errors.Wrap(err)
	}

	// 读取目录
	files, err := os.ReadDir(directoryToCompress)
	if err != nil {
		return errors.Wrap(err)
	}

	for _, file := range files {
		// 获取文件信息
		info, err := file.Info()
		if err != nil {
			return errors.Wrap(err)
		}

		// 获取完整路径
		filePath := filepath.Join(directoryToCompress, file.Name())
		if file.IsDir() {
			// 递归地压缩子目录
			err = addDirectoryToTar(tarWriter, filePath, info, filepath.Join(baseDir, file.Name()))
			if err != nil {
				return errors.Wrap(err)
			}
		} else {
			// 压缩单个文件
			err = addSingleFileToTar(tarWriter, filePath, info, baseDir)
			if err != nil {
				return errors.Wrap(err)
			}
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
		return errors.Wrap(err)
	}
	defer file.Close()

	var reader io.Reader

	// 判断解压文件是否是.gz
	if strings.HasSuffix(tarFile, ".tar.gz") {
		// 创建 gzip.Reader 用于读取压缩数据
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return errors.Wrap(err)
		}
		defer gzReader.Close()
		reader = gzReader
	} else {
		reader = file
	}

	// 创建一个tar读取器
	tarReader := tar.NewReader(reader)

	// 创建目标目录
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return errors.Wrap(err)
	}

	// 遍历tar归档文件中的每个文件条目
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// 读取完所有文件条目
			break
		}
		if err != nil {
			return errors.Wrap(err)
		}

		// 创建解压后的文件路径
		destPath := filepath.Join(destDir, header.Name)

		// 判断文件条目是一个目录还是一个普通文件
		switch header.Typeflag {
		case tar.TypeDir:
			// 如果是目录，创建目录
			err := os.MkdirAll(destPath, 0744)
			if err != nil {
				return errors.Wrap(err)
			}
		case tar.TypeReg:
			// 判断目录是否存在, 不存在则创建
			if !IsExist(filepath.Dir(destPath)) {
				err := os.MkdirAll(filepath.Dir(destPath), 0744)
				if err != nil {
					return errors.Wrap(err)
				}
			}

			// 如果是文件，创建文件并将tar数据写入文件
			file, err := os.Create(destPath)
			if err != nil {
				return errors.Wrap(err)
			}

			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return errors.Wrap(err)
			}
		}
	}

	return nil
}
