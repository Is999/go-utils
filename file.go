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
	return err == nil || !os.IsNotExist(err)
}

// Size 取得文件大小
func Size(filepath string) (int64, error) {
	f, err := os.Stat(filepath)
	if err != nil {
		return 0, errors.Tag(err)
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
		return errors.Tag(err)
	}
	defer f1.Close()

	// 获取文件权限
	stat, err := f1.Stat()
	if err != nil {
		return errors.Tag(err)
	}

	// 目标文件已存在时，先拒绝符号链接，并判断是否与源文件指向同一 inode，避免覆盖写把源文件截断。
	dstInfo, err := os.Lstat(dst)
	switch {
	case err == nil:
		if dstInfo.Mode()&os.ModeSymlink != 0 {
			return errors.Errorf("Copy() 不允许目标文件为符号链接: dst=%s", dst)
		}
		dstStat, statErr := os.Stat(dst)
		if statErr != nil {
			return errors.Tag(statErr)
		}
		if os.SameFile(stat, dstStat) {
			return errors.Errorf("Copy 不允许源文件和目标文件相同: src=%s dst=%s", src, dst)
		}
	case os.IsNotExist(err):
		// 目标文件不存在时允许继续创建。
	default:
		return errors.Tag(err)
	}

	// 使用同目录临时文件 + Rename 原子替换，避免直接截断目标文件。
	if err = writeFileAtomic(dst, stat.Mode(), func(file *os.File) error {
		_, copyErr := io.Copy(file, f1)
		if copyErr != nil {
			return errors.Tag(copyErr)
		}
		return nil
	}); err != nil {
		return errors.Tag(err)
	}
	return nil
}

// writeFileAtomic 使用同目录临时文件完成原子写入。
// 该方法会拒绝通过符号链接目录或符号链接目标写入，降低覆盖写越界和半写文件风险。
func writeFileAtomic(fileName string, perm os.FileMode, write func(file *os.File) error) error {
	dir := filepath.Dir(fileName)
	var err error

	// 拒绝目标路径链路中的符号链接，避免把内容写入符号链接指向的其它位置。
	if err = assertNoSymlinkPath(dir, fileName); err != nil {
		return errors.Wrapf(err, "writeFileAtomic() 校验路径失败: path=%s", fileName)
	}
	if info, err := os.Lstat(fileName); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.Errorf("writeFileAtomic() 不允许目标文件为符号链接: path=%s", fileName)
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "writeFileAtomic() 校验目标文件失败: path=%s", fileName)
	}

	tmpFile, err := os.CreateTemp(dir, "."+filepath.Base(fileName)+".tmp-*")
	if err != nil {
		return errors.Wrapf(err, "writeFileAtomic() 创建临时文件失败: path=%s", fileName)
	}
	tmpName := tmpFile.Name()
	needCleanup := true
	defer func() {
		if needCleanup {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	// 临时文件先写入最终权限，Rename 后即可保持目标权限一致。
	if err = tmpFile.Chmod(perm); err != nil {
		return errors.Wrapf(err, "writeFileAtomic() 设置临时文件权限失败: path=%s", fileName)
	}
	if err = write(tmpFile); err != nil {
		return errors.Wrapf(err, "writeFileAtomic() 写入临时文件失败: path=%s", fileName)
	}
	if err = tmpFile.Sync(); err != nil {
		return errors.Wrapf(err, "writeFileAtomic() 刷盘失败: path=%s", fileName)
	}
	if err = tmpFile.Close(); err != nil {
		return errors.Wrapf(err, "writeFileAtomic() 关闭临时文件失败: path=%s", fileName)
	}
	if err = os.Rename(tmpName, fileName); err != nil {
		return errors.Wrapf(err, "writeFileAtomic() 原子替换失败: path=%s", fileName)
	}
	needCleanup = false
	return nil
}

// FileInfo 文件信息
type FileInfo struct {
	fs.FileInfo        // 原始文件信息
	Path        string // 文件绝对路径
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
	matcher, err := newFindMatcher(match)
	if err != nil {
		return files, errors.Tag(err)
	}

	// 处理文件匹配
	fc := func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Tag(err)
		}

		if d.IsDir() {
			return nil
		}

		if matcher(d.Name()) {
			info, err := d.Info()
			if err != nil {
				return errors.Tag(err)
			}

			// 获取绝对路径
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return errors.Tag(err)
			}
			files = append(files, FileInfo{info, absPath})
		}
		return nil
	}

	// 深度模式或当前模式
	if depth {
		// 深度模式
		if err = filepath.WalkDir(path, fc); err != nil {
			return files, errors.Tag(err)
		}
	} else {
		// 当前模式读取当前目录
		entries, err := os.ReadDir(path)
		if err != nil {
			return files, errors.Tag(err)
		}

		// 遍历当前目录所有目录和文件
		for _, entry := range entries {
			if err := fc(filepath.Join(path, entry.Name()), entry, nil); err != nil {
				return files, err
			}
		}
	}
	return files, nil
}

// newFindMatcher 将 FindFiles 的匹配参数编译为文件名匹配函数。
func newFindMatcher(match []string) (func(string) bool, error) {
	switch {
	case len(match) == 0:
		return func(string) bool { return true }, nil
	case len(match) == 1:
		if match[0] == "*" {
			return func(string) bool { return true }, nil
		}
		rule := match[0]
		return func(name string) bool { return name == rule }, nil
	}

	rules := match[1:]
	switch match[0] {
	case "*":
		return func(string) bool { return true }, nil
	case "p":
		return func(name string) bool {
			for _, rule := range rules {
				if strings.HasPrefix(name, rule) {
					return true
				}
			}
			return false
		}, nil
	case "s":
		return func(name string) bool {
			for _, rule := range rules {
				if strings.HasSuffix(name, rule) {
					return true
				}
			}
			return false
		}, nil
	case "e":
		return func(name string) bool {
			for _, rule := range rules {
				if name == rule {
					return true
				}
			}
			return false
		}, nil
	case "r":
		compiles := make([]*regexp.Regexp, 0, len(rules))
		for _, expr := range rules {
			compile, err := regexp.Compile(expr)
			if err != nil {
				return nil, errors.Wrapf(err, "格式错误的表达式[%s]", expr)
			}
			compiles = append(compiles, compile)
		}
		return func(name string) bool {
			for _, compile := range compiles {
				if compile.MatchString(name) {
					return true
				}
			}
			return false
		}, nil
	default:
		return nil, errors.Errorf("match第一个参数[%s]错误的规则", match[0])
	}
}

// Scan 使用scan扫描文件每一行数据
//
//	size 设置Scanner.maxTokenSize 的大小(默认值: 64*1024): 单行内容大于该值则无法读取
func Scan(r io.Reader, handle ReadScan, size ...int) error {
	scan := bufio.NewScanner(r)

	// 设置buf和maxTokenSize
	if len(size) > 0 && size[0] > bufio.MaxScanTokenSize {
		maxTokenSize := int(min(int64(size[0]), int64(GB*4)))
		scan.Buffer(make([]byte, bufio.MaxScanTokenSize), maxTokenSize)
	}

	var n = 0 // 行号
	for scan.Scan() {
		n++
		if err := handle(n, scan.Bytes(), nil); err != nil {
			if errors.Is(err, DONE) {
				return nil
			}
			return errors.Tag(err)
		}
	}
	return errors.Tag(scan.Err())
}

// Line 读取一行数据: 读取大文件大行数据性能略优于Scan
func Line(r io.Reader, handle ReadLine) error {
	reader := bufio.NewReaderSize(r, bufio.MaxScanTokenSize)
	n := 1 // 行号
	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Tag(err)
		}

		if err := handle(n, line, !isPrefix); err != nil {
			if errors.Is(err, DONE) {
				return nil
			}
			return errors.Tag(err)
		}
		if !isPrefix {
			n++
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
					return nil
				}
				return errors.Tag(err)
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Tag(err)
		}
		if n == 0 {
			return nil
		}
	}
}

// WriteOption 写入文件配置项
type WriteOption func(*writeOptions)

// writeOptions 保存文件写入行为配置。
type writeOptions struct {
	isAppend bool        // 是否以追加模式写入。
	perm     os.FileMode // 文件权限，默认 0644。
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

// WriteFileAtomic 原子写入完整文件内容。
// 适用于配置文件、密钥文件、状态文件等需要“覆盖即完整替换”的场景。
// 内部使用同目录临时文件 + Sync + Close + Rename，避免直接 O_TRUNC 截断目标文件。
func WriteFileAtomic(fileName string, data []byte, perm os.FileMode) error {
	return writeFileAtomic(fileName, perm, func(file *os.File) error {
		_, err := file.Write(data)
		return errors.Tag(err)
	})
}

// WriteStringAtomic 原子写入完整字符串内容。
// 适用于希望以字符串形式原子覆盖目标文件的场景。
func WriteStringAtomic(fileName, data string, perm os.FileMode) error {
	return writeFileAtomic(fileName, perm, func(file *os.File) error {
		_, err := file.WriteString(data)
		return errors.Tag(err)
	})
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

	// 写入前拒绝目标文件为符号链接，避免通过通用写入口写穿到其它路径。
	// 同时拒绝目标路径链路中的符号链接目录，避免写入穿透到预期目录之外。
	if err := assertNoSymlinkPath(path, fileName); err != nil {
		return nil, errors.Tag(err)
	}
	if info, err := os.Lstat(fileName); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.Errorf("NewWrite() 不允许目标文件为符号链接: %s", fileName)
		}
	} else if !os.IsNotExist(err) {
		return nil, errors.Tag(err)
	}

	if !IsExist(path) {
		// 本用户组必须拥有读写执行(7)权限
		var premDir os.FileMode = 0744
		if permFile >= os.FileMode(0700) {
			premDir = permFile
		}

		// 创建目录
		err := os.MkdirAll(path, premDir)
		if err != nil {
			return nil, errors.Tag(err)
		}
	}

	// 打开文件标识
	flag := os.O_CREATE | os.O_WRONLY
	if cfg.isAppend {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	// 打开文件没有则创建
	file, err := os.OpenFile(fileName, flag, permFile)
	if err != nil {
		return nil, errors.Tag(err)
	}

	return &WriteFile{File: file}, nil
}

// WriteFile 文件读写操作
type WriteFile struct {
	Lock sync.RWMutex // 文件句柄读写锁
	File *os.File     // 当前文件句柄
}

// currentFileLocked 获取当前文件句柄。
// 调用方必须先持有写锁或读锁。
func (f *WriteFile) currentFileLocked() (*os.File, error) {
	if f == nil || f.File == nil {
		return nil, errors.New("文件已关闭")
	}
	return f.File, nil
}

// WriteString 写入数据
func (f *WriteFile) WriteString(data string) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	file, err := f.currentFileLocked()
	if err != nil {
		return 0, errors.Tag(err)
	}

	// 写入数据
	n, err := file.WriteString(data)
	return n, errors.Tag(err)
}

// Write 写入数据
func (f *WriteFile) Write(data []byte) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	file, err := f.currentFileLocked()
	if err != nil {
		return 0, errors.Tag(err)
	}

	// 写入数据
	n, err := file.Write(data)
	return n, errors.Tag(err)
}

// WriteBuf 使用 bufio.Writer 写入数据
func (f *WriteFile) WriteBuf(handler func(write *bufio.Writer) (int, error)) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	if handler == nil {
		return 0, errors.New("handler 不能为空")
	}

	file, err := f.currentFileLocked()
	if err != nil {
		return 0, errors.Tag(err)
	}
	w := bufio.NewWriter(file)
	size, err := handler(w)
	if err != nil {
		return size, errors.Tag(err)
	}
	if err = w.Flush(); err != nil {
		return size, errors.Tag(err)
	}
	return size, nil
}

// Close 关闭文件
func (f *WriteFile) Close() error {
	if f == nil {
		return nil
	}
	f.Lock.Lock()
	defer f.Lock.Unlock()

	file := f.File
	if file == nil {
		return nil
	}
	f.File = nil
	return errors.Tag(file.Close())
}

// FormatFileSize 将字节数格式化为易读的文件大小。
//
//	size 文件实际大小(Byte)
//	decimals 保留几位小数
func FormatFileSize(size int64, decimals uint) string {
	for _, unit := range [...]struct {
		size   int64  // 单位字节数
		suffix string // 单位后缀
	}{
		{EB, "E"},
		{PB, "P"},
		{TB, "T"},
		{GB, "G"},
		{MB, "M"},
		{KB, "K"},
	} {
		if size >= unit.size {
			return FormatNumber(float64(size)/float64(unit.size), decimals, ".", ",") + unit.suffix
		}
	}
	return strconv.FormatInt(size, 10) + "B"
}

// FileType 文件类型
func FileType(f *os.File) (string, error) {
	ctype := mime.TypeByExtension(filepath.Ext(f.Name()))
	if ctype == "" {
		// 记录当前文件偏移，检测完成后恢复，避免影响调用方后续读取逻辑。
		currentOffset, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return "", errors.Tag(err)
		}

		var buf [512]byte
		n, err := io.ReadFull(f, buf[:])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return "", errors.Tag(err)
		}

		ctype = http.DetectContentType(buf[:n])

		// 恢复文件指针到调用前位置，保持函数无副作用。
		_, err = f.Seek(currentOffset, io.SeekStart)
		if err != nil {
			return "", errors.Tag(err)
		}
	}
	return ctype, nil
}
