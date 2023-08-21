package utils

import (
	"archive/zip"
	"github.com/Is999/go-utils/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Zip 使用zip打包并压缩
//
//	zipFile 打包压缩后文件
//	files 待打包压缩文件【夹】
func Zip(zipFile string, files []string) error {
	if !strings.HasSuffix(zipFile, ".zip") {
		return errors.New("文件名错误：非.zip文件")
	}

	// 创建 zip 文件
	file, err := os.Create(zipFile)
	if err != nil {
		return errors.Wrap(err)
	}
	defer file.Close()

	// 创建 zip.Writer
	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for _, filePath := range files {
		// 将文件添加到 zip 文件
		err = AddFileToZip(zipWriter, filePath, "")
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

// AddFileToZip 添加文件【夹】到zip
//
//	fileToCompress 需要压缩的文件
//	baseDir 打包文件根目录
func AddFileToZip(zipWriter *zip.Writer, fileToCompress string, baseDir string) error {
	fileInfo, err := os.Stat(fileToCompress)
	if err != nil {
		return errors.Wrap(err)
	}

	if fileInfo.IsDir() {
		// 压缩目录
		return addDirectoryToZip(zipWriter, fileToCompress, fileInfo, fileInfo.Name())
	} else {
		// 压缩文件
		return addSingleFileToZip(zipWriter, fileToCompress, fileInfo, baseDir)
	}
}

// addSingleFileToZip 添加单个文件到zip
func addSingleFileToZip(zipWriter *zip.Writer, fileToCompress string, fileInfo os.FileInfo, baseDir string) error {
	// 创建 zip 文件中的文件头
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return errors.Wrap(err)
	}

	// 修改 header 中的 Name 字段，确保文件名正确
	header.Name = filepath.ToSlash(filepath.Join(baseDir, header.Name))

	// 压缩文件
	header.Method = zip.Deflate

	// 创建一个新的ZIP文件条目
	zipFile, err := zipWriter.CreateHeader(header)
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

		// 将文件数据拷贝到ZIP文件条目
		_, err = io.Copy(zipFile, file)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

// addDirectoryToZip 添加目录到zip
func addDirectoryToZip(zipWriter *zip.Writer, directoryToCompress string, fileInfo os.FileInfo, baseDir string) error {
	// 压缩目录
	err := addSingleFileToZip(zipWriter, directoryToCompress, fileInfo, strings.TrimRight(baseDir, fileInfo.Name()))
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
			err = addDirectoryToZip(zipWriter, filePath, info, filepath.Join(baseDir, file.Name()))
			if err != nil {
				return errors.Wrap(err)
			}
		} else {
			// 压缩单个文件
			err = addSingleFileToZip(zipWriter, filePath, info, baseDir)
			if err != nil {
				return errors.Wrap(err)
			}
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
		return errors.Wrap(err)
	}
	defer r.Close()

	// 创建目标目录
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return errors.Wrap(err)
	}

	// 遍历ZIP文件中的文件和目录
	for _, file := range r.File {
		err = func(f *zip.File) error {
			// 构建解压后的文件路径
			destPath := filepath.Join(destDir, f.Name)

			// 如果文件是一个目录，则创建对应的目录
			if f.FileInfo().IsDir() {
				err := os.MkdirAll(destPath, 0744)
				if err != nil {
					return errors.Wrap(err)
				}
				return nil
			}

			// 判断目录是否存在, 不存在则创建
			if !IsExist(filepath.Dir(destPath)) {
				err := os.MkdirAll(filepath.Dir(destPath), 0744)
				if err != nil {
					return errors.Wrap(err)
				}
			}

			// 创建解压后的文件
			file, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0744)
			if err != nil {
				return errors.Wrap(err)
			}
			defer file.Close()

			// 读取ZIP文件中的数据并写入解压后的文件
			rc, err := f.Open()
			if err != nil {
				return errors.Wrap(err)
			}
			defer rc.Close()

			_, err = io.Copy(file, rc)
			if err != nil {
				return errors.Wrap(err)
			}
			return nil
		}(file)

		if err != nil {
			return err
		}
	}

	return nil
}
