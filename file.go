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

// DONE 由读取回调返回时正常结束 Scan、Line 或 Read，支持 errors.Is 匹配。
var DONE = errors.New("DONE")

// IsDir 跟随符号链接判断目录，Stat 失败时返回 false。
func IsDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil {
		return false
	}
	return f.IsDir()
}

// IsFile 跟随符号链接判断非目录项，包含特殊文件；Stat 失败时返回 false。
func IsFile(filepath string) bool {
	f, err := os.Stat(filepath)
	if err != nil {
		return false
	}
	return !f.IsDir()
}

// IsExist 仅在 Stat 确认路径不存在时返回 false，权限等其他错误仍返回 true。
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// Size 跟随符号链接返回 Stat 报告的字节数，目录大小由文件系统决定。
func Size(filepath string) (int64, error) {
	f, err := os.Stat(filepath)
	if err != nil {
		return 0, errors.Tag(err)
	}
	return f.Size(), nil
}

// Copy 写完同目录临时文件后替换 dst，并沿用源权限；目标父目录须已存在。
// 源路径可跟随符号链接，目标符号链接及同一文件的硬链接会被拒绝。
func Copy(src, dst string) error {
	f1, err := os.Open(src)
	if err != nil {
		return errors.Tag(err)
	}
	defer f1.Close()

	stat, err := f1.Stat()
	if err != nil {
		return errors.Tag(err)
	}

	// 先拒绝目标符号链接和同一文件，保留源数据。
	dstInfo, err := os.Lstat(dst)
	switch {
	case err == nil:
		if dstInfo.Mode()&os.ModeSymlink != 0 {
			return errors.Errorf("Copy() 不允许目标文件为符号链接: dst=%s", dst)
		}
		if os.SameFile(stat, dstInfo) {
			return errors.Errorf("Copy 不允许源文件和目标文件相同: src=%s dst=%s", src, dst)
		}
	case os.IsNotExist(err):
		// 目标文件不存在时允许继续创建。
	default:
		return errors.Tag(err)
	}

	return errors.Tag(writeFileAtomic(dst, stat.Mode(), func(file *os.File) error {
		_, copyErr := io.Copy(file, f1)
		return errors.Tag(copyErr)
	}))
}

// writeFileAtomic 通过同目录临时文件原子替换目标，回调只负责写入。
func writeFileAtomic(fileName string, perm os.FileMode, write func(file *os.File) error) error {
	dir := filepath.Dir(fileName)
	var err error

	// 检查直接父目录和目标路径，后续 Lstat 保留原始路径的文件系统解析结果。
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
	// 失败时清理临时文件，清理错误不覆盖首个失败原因。
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
	// 刷盘和关闭均成功后才替换目标，避免发布未完成的文件。
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

// FileInfo 记录遍历时的元数据及绝对路径，不跟踪文件后续变化。
type FileInfo struct {
	fs.FileInfo        // 原始文件信息
	Path        string // 文件绝对路径
}

// FindFiles 匹配文件名并返回绝对路径，遍历子项时忽略目录且不跟随目录符号链接。
// depth 控制是否递归；遍历中途失败时同时返回已收集的结果。
//
// match 为空或首项为 "*" 时匹配全部，单个参数按完整文件名匹配；
// 多参数以 e/p/s/r 指定完整名、前缀、后缀或正则，后续规则满足任一项即匹配。
func FindFiles(path string, depth bool, match ...string) (files []FileInfo, err error) {
	matcher, err := newFindMatcher(match)
	if err != nil {
		return files, errors.Tag(err)
	}

	fc := func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Tag(err)
		}

		// 先按名称筛选，未命中的条目无需读取元数据。
		if d.IsDir() || !matcher(d.Name()) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return errors.Tag(err)
		}

		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return errors.Tag(err)
		}
		files = append(files, FileInfo{info, absPath})
		return nil
	}

	if depth {
		if err = filepath.WalkDir(path, fc); err != nil {
			return files, errors.Tag(err)
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return files, errors.Tag(err)
		}

		for _, entry := range entries {
			if err := fc(filepath.Join(path, entry.Name()), entry, nil); err != nil {
				return files, err
			}
		}
	}
	return files, nil
}

// newFindMatcher 在遍历前编译规则；正则错误不会等到遇见文件时才返回。
func newFindMatcher(match []string) (func(string) bool, error) {
	switch {
	// 通配规则忽略后续参数，与未提供规则时使用同一匹配逻辑。
	case len(match) == 0 || match[0] == "*":
		return func(string) bool { return true }, nil
	case len(match) == 1:
		rule := match[0]
		return func(name string) bool { return name == rule }, nil
	}

	rules := match[1:]
	switch match[0] {
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

// Scan 去掉 LF/CRLF 后逐行同步调用 handle，行号从 1 开始，切片仅在回调内有效。
// size[0] 仅在大于默认 64 KiB 时生效，最多 4 GiB；上限还需容纳行尾分隔符。
// 回调的 err 参数始终为 nil，读取错误由 Scan 返回；DONE 正常结束，r 由调用方关闭。
func Scan(r io.Reader, handle ReadScan, size ...int) error {
	scan := bufio.NewScanner(r)

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

// Line 去掉 LF/CRLF 后按物理行同步调用 handle，长行的多个分块使用同一行号。
// 切片仅在回调内有效；有效尾块先交给回调，回调错误或 DONE 优先于读取错误，r 由调用方关闭。
// 末行恰好占满缓冲区时，可用空末块通知 lineDone=true。
func Line(r io.Reader, handle ReadLine) error {
	reader := bufio.NewReaderSize(r, bufio.MaxScanTokenSize)
	n := 1           // 行号
	pending := false // 前一块尚未结束本行，后续空 EOF 或读取错误也需补结束通知。
	for {
		line, err := reader.ReadSlice('\n')
		isPrefix := err == bufio.ErrBufferFull
		if isPrefix {
			// CRLF 跨块时把 CR 留给下一次读取；ReadSlice 已保证此时可回退一个字节。
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
				_ = reader.UnreadByte()
			}
		} else if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
		}

		if len(line) > 0 || err == nil || isPrefix || pending {
			if handleErr := handle(n, line, !isPrefix); handleErr != nil {
				if errors.Is(handleErr, DONE) {
					return nil
				}
				return errors.Tag(handleErr)
			}
		}
		pending = isPrefix
		if err != nil && !isPrefix {
			if err == io.EOF {
				return nil
			}
			return errors.Tag(err)
		}
		if !isPrefix {
			n++
		}
	}
}

// Read 分块读取数据，直到 Reader 返回错误或回调返回 DONE。
// Reader 返回 (0, nil) 只表示本次没有数据，不作为读取结束；回调中的切片仅在本次调用内有效。
// 有效数据先交给回调，回调错误或 DONE 优先于读取错误；EOF 正常结束，不关闭 r。
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
	}
}

// WriteOption 写入文件配置项
type WriteOption func(*writeOptions)

// writeOptions 保存文件写入行为配置。
type writeOptions struct {
	isAppend bool        // 是否以追加模式写入。
	perm     os.FileMode // 文件权限，默认 0644。
}

// WithWriteAppend 控制追加模式；默认关闭，NewWrite 会截断已有文件。
func WithWriteAppend(isAppend bool) WriteOption {
	return func(o *writeOptions) {
		o.isAppend = isAppend
	}
}

// WithWritePerm 设置新建文件的权限，默认 0644；已有文件的权限不变。
func WithWritePerm(perm os.FileMode) WriteOption {
	return func(o *writeOptions) {
		o.perm = perm
	}
}

// WriteFileAtomic 先将 data 写入同目录临时文件并刷盘、关闭，再以 Rename 替换目标。
// 父目录须已存在；perm 应用于替换后的文件，data 仅在本次调用期间使用。
func WriteFileAtomic(fileName string, data []byte, perm os.FileMode) error {
	return writeFileAtomic(fileName, perm, func(file *os.File) error {
		_, err := file.Write(data)
		return errors.Tag(err)
	})
}

// WriteStringAtomic 按 WriteFileAtomic 的替换与权限规则写入字符串。
func WriteStringAtomic(fileName, data string, perm os.FileMode) error {
	return writeFileAtomic(fileName, perm, func(file *os.File) error {
		_, err := file.WriteString(data)
		return errors.Tag(err)
	})
}

// NewWrite 创建可并发写入的句柄，默认截断已有文件，调用方负责 Close。
// 缺失父目录会自动创建；新建文件默认 0644，目录默认 0744，均受系统 umask 影响。
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

	// 检查直接父目录和目标路径，避免跟随现有符号链接写入。
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
		// 沿用目录权限规则：文件权限数值达到 0700 时也用于新建目录。
		var premDir os.FileMode = 0744
		if permFile >= os.FileMode(0700) {
			premDir = permFile
		}

		err := os.MkdirAll(path, premDir)
		if err != nil {
			return nil, errors.Tag(err)
		}
	}

	flag := os.O_CREATE | os.O_WRONLY
	if cfg.isAppend {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(fileName, flag, permFile)
	if err != nil {
		return nil, errors.Tag(err)
	}

	return &WriteFile{File: file}, nil
}

// WriteFile 串行执行写入和关闭，首次使用后不得复制；直接访问 File 须自行同步。
type WriteFile struct {
	Lock sync.RWMutex // 保护 File 状态，并覆盖每次写入及回调的完整过程。
	File *os.File     // 当前文件句柄；Close 后置为 nil。
}

// WriteString 在持锁期间写入字符串，返回实际字节数及写入错误。
func (f *WriteFile) WriteString(data string) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	if f.File == nil {
		return 0, errors.New("文件已关闭")
	}

	n, err := f.File.WriteString(data)
	return n, errors.Tag(err)
}

// Write 在持锁期间写入 data，返回实际字节数及写入错误。
func (f *WriteFile) Write(data []byte) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	if f.File == nil {
		return 0, errors.New("文件已关闭")
	}

	n, err := f.File.Write(data)
	return n, errors.Tag(err)
}

// WriteBuf 持锁执行 handler，成功后 Flush；回调不得重入当前 WriteFile 或保留 writer。
// 返回字节数沿用 handler 的结果；回调失败不 Flush，已写入文件的内容不会回滚。
func (f *WriteFile) WriteBuf(handler func(write *bufio.Writer) (int, error)) (int, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	if handler == nil {
		return 0, errors.New("handler 不能为空")
	}

	if f.File == nil {
		return 0, errors.New("文件已关闭")
	}
	w := bufio.NewWriter(f.File)
	size, err := handler(w)
	if err != nil {
		return size, errors.Tag(err)
	}
	return size, errors.Tag(w.Flush())
}

// Close 等待当前写入结束后关闭句柄；nil 接收者和重复关闭均返回 nil。
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

// FormatFileSize 按 1024 进制缩放字节数，decimals 指定小数位数；小于 1 KiB 时直接输出整数 B。
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

// FileType 优先按扩展名获取文件类型；未知扩展名读取文件头最多 512 字节，不改变文件偏移。
func FileType(f *os.File) (string, error) {
	ctype := mime.TypeByExtension(filepath.Ext(f.Name()))
	if ctype != "" {
		return ctype, nil
	}

	var buf [512]byte
	n, err := f.ReadAt(buf[:], 0)
	if err != nil && err != io.EOF {
		return "", errors.Tag(err)
	}
	return http.DetectContentType(buf[:n]), nil
}
