package utils

import (
	"bufio"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Is999/go-utils/errors"
)

// DONE 完成终止
var DONE = errors.New("DONE")

// IsDir 判断给定路径是否是一个目录
func IsDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil {
		return false
	}
	return f.IsDir()
}

// IsFile 判断给定的文件路径名是否是一个文件
func IsFile(filepath string) bool {
	f, err := os.Stat(filepath)
	if err != nil {
		return false
	}
	return !f.IsDir()
}

// IsExist 判断一个文件（夹）是否存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// Size 取得文件大小
func Size(filepath string) (int64, error) {
	f, err := os.Stat(filepath)
	if err != nil {
		return 0, errors.Wrap(err)
	}
	return f.Size(), nil
}

// Copy 拷贝文件
//
//	src 拷贝的源文件
//	dst 拷贝后的文件
func Copy(src, dst string) error {
	// 打开source文件
	f1, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err)
	}
	defer f1.Close()

	// 获取文件权限
	stat, err := f1.Stat()
	if err != nil {
		return errors.Wrap(err)
	}

	// 创建或打开拷贝文件
	f2, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, stat.Mode())
	if err != nil {
		return errors.Wrap(err)
	}
	defer f2.Close()

	// 拷贝文件
	_, err = io.Copy(f2, f1)
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}

// FileInfo 文件信息
type FileInfo struct {
	fs.FileInfo
	Path string // 文件相对路径
}

// FindFiles 获取目录下所有匹配文件
//
//	path 目录
//	depth 深度查找: true 采用filepath.WalkDir遍历; false 只在当前目录查找
//	match 匹配规则:
//	 - `无参` : 匹配所有文件名 FindFiles(path, depth)
//	 - `*`   : 匹配所有文件名 FindFiles(path, depth, `*`)
//	 - `文件完整名`      : 精准匹配文件名 FindFiles(path, depth, fullFileName)
//	 - `e`, `文件完整名` : 精准匹配文件名 FindFiles(path, depth, `e`, fullFileName)
//	 - `p`, `文件前缀名` : 匹配前缀文件名 FindFiles(path, depth, `p`, fileNamePrefix)
//	 - `s`, `文件后缀名` : 匹配后缀文件名 FindFiles(path, depth, `s`, fileNameSuffix)
//	 - `r`, `正则表达式` : 正则匹配文件名 FindFiles(path, depth, `r`, fileNameReg)
func FindFiles(path string, depth bool, match ...string) (files []FileInfo, err error) {
	var (
		mode     = "*"            // 匹配模式
		regs     []string         // 匹配规则
		compiles []*regexp.Regexp // 匹配模式为r时，正则表达式
	)

	// match参数处理
	if len(match) == 1 {
		if match[0] != "*" {
			mode = "e" // 精准匹配
			regs = append(regs, match[0])
		}
	} else if len(match) >= 2 {
		if IsHas[string](match[0], []string{"*", "p", "s", "r", "e"}) {
			mode = match[0]
			if mode != "*" {
				regs = make([]string, 0, len(match)-1)
				regs = append(regs, match[1:]...)

				// 正则匹配, 验证正则表达式是否正确
				if mode == "r" {
					compiles = make([]*regexp.Regexp, 0, len(regs))
					for i := 0; i < len(regs); i++ {
						compile, err := regexp.Compile(regs[i])
						if err != nil {
							return nil, errors.Errorf("格式错误的表达式[%s]: %s", regs[i], err.Error())
						}
						compiles = append(compiles, compile)
					}
				}
			}
		} else {
			return files, errors.Errorf("match第一个参数[%s]错误的规则", match[0])
		}
	}

	// 处理文件匹配
	var fc fs.WalkDirFunc = func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Wrap(err)
		}

		if d.IsDir() {
			return nil
		}

		ok := false
		if mode == "*" {
			ok = true
		} else {
			for i := 0; i < len(regs); i++ {
				switch mode {
				case "p": // 匹配前缀
					if strings.HasPrefix(d.Name(), regs[i]) {
						ok = true
					}
				case "s": // 匹配后缀
					if strings.HasSuffix(d.Name(), regs[i]) {
						ok = true
					}
				case "r": // 正则表达式匹配
					if compiles[i].MatchString(d.Name()) {
						ok = true
					}
				case "e": // 精确匹配
					if regs[i] == d.Name() {
						ok = true
					}
				}

				// 当前文件匹配规则，则终止匹配
				if ok {
					break
				}
			}
		}

		if ok {
			info, err := d.Info()
			if err != nil {
				return errors.Wrap(err)
			}

			// 获取绝对路径
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return errors.Wrap(err)
			}
			files = append(files, FileInfo{info, absPath})
		}
		return nil
	}

	// 深度模式或当前模式
	if depth {
		// 深度模式
		err = errors.Wrap(filepath.WalkDir(path, fc))
	} else {
		// 当前模式读取当前目录
		entries, err := os.ReadDir(path)
		if err != nil {
			return files, errors.Wrap(err)
		}

		// 处理目录路径末尾路径分割符
		if !strings.HasSuffix(path, string(filepath.Separator)) {
			path += string(filepath.Separator)
		}

		// 遍历当前目录所有目录和文件
		for _, v := range entries {
			err = fc(path+v.Name(), v, nil)
			if err != nil {
				break
			}
		}

	}
	return files, err
}

// Scan 使用scan扫描文件每一行数据
//
//	size 设置Scanner.maxTokenSize 的大小(默认值: 64*1024): 单行内容大于该值则无法读取
func Scan(r io.Reader, handle ReadScan, size ...int) error {
	scan := bufio.NewScanner(r)

	// 设置buf和maxTokenSize
	if len(size) > 0 && size[0] > bufio.MaxScanTokenSize {
		maxTokenSize := size[0]
		if int64(maxTokenSize) > (GB * 4) {
			maxTokenSize = int(GB * 4)
		}
		scan.Buffer(make([]byte, bufio.MaxScanTokenSize), maxTokenSize)
	}

	var n = 0 // 行号
	for scan.Scan() {
		n++
		if err := handle(n, scan.Bytes(), scan.Err()); err != nil {
			if errors.Is(err, DONE) {
				return nil
			}
			return errors.Wrap(err)
		}
	}
	return errors.Wrap(scan.Err())
}

// Line 读取一行数据: 读取大文件大行数据性能略优于Scan
func Line(r io.Reader, handle ReadLine) error {
	reader := bufio.NewReaderSize(r, bufio.MaxScanTokenSize)
	var n = 0 // 行号
	for {
		n++
		line, isPrefix, err := reader.ReadLine()

		// 大行数据未读取完不加行号
		if isPrefix {
			n--
		}

		// 处理数据
		if err == nil {
			if err := handle(n, line, !isPrefix); err != nil {
				if errors.Is(err, DONE) {
					return nil
				}
				return errors.Wrap(err)
			}
		} else {
			if err == io.EOF {
				err = nil
			}
			return errors.Wrap(err)
		}
	}
}

// Read 使用分块读取文件数据, 读取大文件或无换行的文件
func Read(r io.Reader, handle ReadBlock) error {
	block := make([]byte, bufio.MaxScanTokenSize)
	for {
		n, err := r.Read(block)
		if n > 0 {
			if err := handle(n, block[:n]); err != nil {
				if errors.Is(err, DONE) {
					err = nil
				}
				return errors.Wrap(err)
			}
		}

		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return errors.Wrap(err)
		}
		if n == 0 {
			return nil
		}
	}
}

// WriteOption 写入文件配置项
type WriteOption func(*writeOptions)

type writeOptions struct {
	isAppend bool
	perm     os.FileMode
}

// WithWriteAppend 设置是否追加写入
func WithWriteAppend(isAppend bool) WriteOption {
	return func(o *writeOptions) {
		o.isAppend = isAppend
	}
}

// WithWritePerm 设置文件权限
func WithWritePerm(perm os.FileMode) WriteOption {
	return func(o *writeOptions) {
		o.perm = perm
	}
}

// NewWrite 返回一个WriteFile实例
//
//	fileName 文件路径: 不存在则创建
//	perm 文件权限: 默认权限 文件夹0744, 文件0644
func NewWrite(fileName string, opts ...WriteOption) (*WriteFile, error) {
	cfg := writeOptions{
		perm: 0644,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	permFile := cfg.perm
	path := filepath.Dir(fileName)
	if !IsExist(path) {
		// 本用户组必须拥有读写执行(7)权限
		var premDir os.FileMode = 0744
		if permFile >= os.FileMode(0700) {
			premDir = permFile
		}

		// 创建目录
		err := os.MkdirAll(path, premDir)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	// 打开文件标识
	flag := os.O_CREATE | os.O_WRONLY
	if cfg.isAppend {
		flag = flag | os.O_APPEND
	} else {
		flag = flag | os.O_TRUNC
	}

	// 打开文件没有则创建
	file, err := os.OpenFile(fileName, flag, permFile)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &WriteFile{File: file}, nil
}

// WriteFile 文件读写操作
type WriteFile struct {
	Lock sync.RWMutex
	File *os.File
}

// WriteString 写入数据
func (f *WriteFile) WriteString(data string) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	//写入数据
	return f.File.WriteString(data)
}

// Write 写入数据
func (f *WriteFile) Write(data []byte) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	//写入数据
	return f.File.Write(data)
}

// WriteBuf 使用 bufio.Writer 写入数据
func (f *WriteFile) WriteBuf(handler func(write *bufio.Writer) (int, error)) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	w := bufio.NewWriter(f.File)

	defer w.Flush()

	return handler(w)
}

// Close 关闭文件
func (f *WriteFile) Close() error {
	if f.File != nil {
		return errors.Wrap(f.File.Close())
	}
	return nil
}

// SizeFormat 文件大小格式化已可读式显示文件大小
//
//	size 文件实际大小(Byte)
//	decimals 保留几位小数
func SizeFormat(size int64, decimals uint) string {
	/*var base float64 = 1024
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	e := math.Floor(math.Log(float64(size)) / math.Log(base))
	suffix := sizes[int(e)]
	val := float64(size) / math.Pow(base, math.Floor(e))
	f := "%.0f"
	if val < 10 {
		f = "%.1f"
	}
	return fmt.Sprintf(f+"%s", val, suffix)*/

	switch {
	case size >= EB:
		return NumberFormat(float64(size)/float64(EB), decimals, ".", ",") + "E"
	case size >= PB:
		return NumberFormat(float64(size)/float64(PB), decimals, ".", ",") + "P"
	case size >= TB:
		return NumberFormat(float64(size)/float64(TB), decimals, ".", ",") + "T"
	case size >= GB:
		return NumberFormat(float64(size)/float64(GB), decimals, ".", ",") + "G"
	case size >= MB:
		return NumberFormat(float64(size)/float64(MB), decimals, ".", ",") + "M"
	case size >= KB:
		return NumberFormat(float64(size)/float64(KB), decimals, ".", ",") + "K"
	default:
		return strconv.FormatInt(size*Byte, 10) + "B"
	}
}

// FileType 文件类型
func FileType(f *os.File) (string, error) {
	ctype := mime.TypeByExtension(filepath.Ext(f.Name()))
	if ctype == "" {
		var buf [512]byte
		n, _ := io.ReadFull(f, buf[:])

		ctype = http.DetectContentType(buf[:n])

		// 重置文件指针到原点
		_, err := f.Seek(0, io.SeekStart)
		if err != nil {
			return "", errors.Wrap(err)
		}
	}
	return ctype, nil
}
