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
			err = addDirectoryToZip(zipWriter, filePath, info, filepath.Join(baseDir, file.Name()))
			if err != nil {
				return errors.Tag(err)
			}
		} else {
			// 压缩单个文件
			err = addSingleFileToZip(zipWriter, filePath, info, baseDir)
			if err != nil {
				return errors.Tag(err)
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
		return errors.Tag(err)
	}
	defer r.Close()

	// 规范化目标目录，后续所有解压路径都必须限制在该目录下。
	destRoot, err := filepath.Abs(destDir)
	if err != nil {
		return errors.Tag(err)
	}

	// 创建目标目录
	err = os.MkdirAll(destRoot, 0755)
	if err != nil {
		return errors.Tag(err)
	}
	if err = assertNoSymlinkPath(destRoot, destRoot); err != nil {
		return errors.Tag(err)
	}

	// 遍历ZIP文件中的文件和目录
	counter := archiveCounter{}
	for _, file := range r.File {
		err = func(f *zip.File) error {
			// 解析并校验解压路径，防止 ../、绝对路径和跨平台分隔符绕过。
			destPath, err := safeUnzipPath(destRoot, f.Name)
			if err != nil {
				return errors.Tag(err)
			}

			mode := f.Mode()
			if mode&os.ModeSymlink != 0 || !mode.IsDir() && !mode.IsRegular() {
				return errors.Errorf("不支持的 zip 条目类型: %s", f.Name)
			}

			// 如果文件是一个目录，则创建对应的目录
			if f.FileInfo().IsDir() {
				if err = counter.add(f.Name, 0); err != nil {
					return errors.Tag(err)
				}
				// 解包落盘前，拒绝目标路径链路中的符号链接，避免跟随写到解包目录之外。
				if err = assertNoSymlinkPath(destRoot, destPath); err != nil {
					return errors.Tag(err)
				}
				err := os.MkdirAll(destPath, unzipDirPerm(mode))
				if err != nil {
					return errors.Tag(err)
				}
				// 显式恢复目录权限，避免受 umask 或既有目录影响导致权限偏差。
				if err := os.Chmod(destPath, unzipDirPerm(mode)); err != nil {
					return errors.Tag(err)
				}
				return nil
			}

			if err = counter.add(f.Name, int64(f.UncompressedSize64)); err != nil {
				return errors.Tag(err)
			}

			// 解包落盘前，拒绝目标路径链路中的符号链接，避免跟随写到解包目录之外。
			if err = assertNoSymlinkPath(destRoot, destPath); err != nil {
				return errors.Tag(err)
			}

			// 判断目录是否存在, 不存在则创建
			if !IsExist(filepath.Dir(destPath)) {
				err := os.MkdirAll(filepath.Dir(destPath), 0755)
				if err != nil {
					return errors.Tag(err)
				}
			}

			// 创建解压后的文件
			// 读取ZIP文件中的数据并写入解压后的文件
			rc, err := f.Open()
			if err != nil {
				return errors.Tag(err)
			}

			// 使用同目录临时文件 + Rename 原子落盘，避免直接截断已有目标文件。
			writeErr := writeFileAtomic(destPath, unzipFilePerm(mode), func(file *os.File) error {
				_, copyErr := io.Copy(file, rc)
				if copyErr != nil {
					return errors.Tag(copyErr)
				}
				return nil
			})
			closeReadErr := rc.Close()
			if writeErr != nil {
				return errors.Tag(writeErr)
			}
			if closeReadErr != nil {
				return errors.Tag(closeReadErr)
			}
			// 显式恢复文件权限，避免受 umask 或既有文件影响导致权限偏差。
			if err := os.Chmod(destPath, unzipFilePerm(mode)); err != nil {
				return errors.Tag(err)
			}
			return nil
		}(file)

		if err != nil {
			return err
		}
	}

	return nil
}

// safeUnzipPath 计算安全的解压目标路径。
// 仅允许写入目标目录内，拒绝空路径、绝对路径、目录穿越和 Windows 风格分隔符绕过。
func safeUnzipPath(destRoot, entryName string) (string, error) {
	if entryName == "" {
		return "", errors.New("zip 条目名称不能为空")
	}
	if strings.Contains(entryName, "\x00") {
		return "", errors.New("zip 条目名称不能包含空字符")
	}

	normalizedName := strings.ReplaceAll(entryName, "\\", "/")
	if strings.HasPrefix(normalizedName, "/") || filepath.IsAbs(normalizedName) {
		return "", errors.Errorf("zip 条目不允许使用绝对路径: %s", entryName)
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
		return "", errors.Errorf("zip 条目路径越界: %s", entryName)
	}
	return destPath, nil
}

// unzipDirPerm 计算解压目录权限。
func unzipDirPerm(mode os.FileMode) os.FileMode {
	perm := mode.Perm()
	if perm == 0 {
		return 0755
	}
	if perm&0700 != 0700 {
		perm |= 0700
	}
	return perm
}

// unzipFilePerm 计算解压文件权限。
func unzipFilePerm(mode os.FileMode) os.FileMode {
	perm := mode.Perm()
	if perm == 0 {
		return 0644
	}
	return perm
}
