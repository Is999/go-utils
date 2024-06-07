package utils_test

import (
	"bufio"
	"fmt"
	"github.com/Is999/go-utils"
	"io"
	"os"
	"sync"
	"testing"
)

func TestFindFiles(t *testing.T) {
	type args struct {
		path  string
		depth bool
		match []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{`./`, false, []string{}}, wantErr: false},                                                 // 匹配所有文件
		{name: "002", args: args{`./`, false, []string{`*`, "test.go"}}, wantErr: false},                                   // 匹配所有文件
		{name: "003", args: args{`./../`, true, []string{`p`, `co`, `ip`}}, wantErr: false},                                // 匹配前缀为str开头的文件
		{name: "004", args: args{`./`, false, []string{`s`, `test.go`}}, wantErr: false},                                   // 匹配后缀为test.go结尾的文件
		{name: "005", args: args{`./`, false, []string{`r`, `^([[a-Z]{2,4})_test.go`}}, wantErr: true},                     // 错误表达式, 返回错误信息
		{name: "006", args: args{`./`, false, []string{`r`, `^([[A-z]{2,4})_test.go`, `^([[A-z]{2}).go`}}, wantErr: false}, // 正则表达式匹配文件
		{name: "007", args: args{`./`, false, []string{`file.go`}}, wantErr: false},                                        // 精准匹配文件名为file.go的文件
		{name: "008", args: args{`./`, false, []string{`e`, `file.go`, `ip.go`}}, wantErr: false},                          // 精准匹配文件名为file.go的文件
		{name: "009", args: args{`./`, false, []string{`e`, `file`}}, wantErr: false},                                      // 精准匹配文件名为file的文件(文件不存在, 返回空)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//t.Logf("FindFiles 匹配规则 path = %v, match = %v", tt.args.path, tt.args.match)
			//got, WrapError := utils.FindFiles(tt.args.path, tt.args.depth, tt.args.match...)
			_, err := utils.FindFiles(tt.args.path, tt.args.depth, tt.args.match...)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//for i, info := range got {
			//	t.Logf("FindFiles() i = %v, path = %v, info.name = %+v", i, info.Path, info.FileInfo.Name())
			//}
		})
	}
}

func TestIsDir(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{"./../utils"}, want: true},
		{name: "002", args: args{"./slices.go"}, want: false}, // 文件-非目录
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsDir(tt.args.path); got != tt.want {
				t.Errorf("IsDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFile(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{"./../utils"}, want: false}, // 目录-非文件
		{name: "002", args: args{"./array.go"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsFile(tt.args.path); got != tt.want {
				t.Errorf("IsFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExist(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "001", args: args{"./../utils"}, want: true},      // 目录的大小
		{name: "002", args: args{"./slices.go"}, want: true},     // 文件大小
		{name: "003", args: args{"./not_exist.go"}, want: false}, // 不存在的文件
		{name: "004", args: args{"./file.go"}, want: true},       // 文件大小
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsExist(tt.args.path); got != tt.want {
				t.Errorf("IsExist() = %v, want %v", got, tt.want)
			}
			if size, err := utils.Size(tt.args.path); (err != nil) == tt.want {
				t.Errorf("Size() path = %v, size = %v, WrapError %v", tt.args.path, utils.SizeFormat(size, 4), err)
			} else {
				//t.Logf("Size() path = %v, size = %d, Humane = %v", tt.args.path, size, SizeFormat(size, 4))
			}
		})
	}
}

func TestCopy(t *testing.T) {
	type args struct {
		source string
		dest   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{source: "./json.go", dest: "/tmp/json.txt"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.Copy(tt.args.source, tt.args.dest); (err != nil) != tt.wantErr {
				t.Errorf("Copy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScan(t *testing.T) {
	type args struct {
		name string
		size []int
	}

	type test struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []test{
		{name: "001", args: args{name: "./file.go", size: []int{int(utils.MB)}}},
	}

	var f = func(t *testing.T, tt test) {
		// 打开文件
		open, err := os.OpenFile(tt.args.name, os.O_RDONLY, 0)
		if err != nil {
			t.Errorf("Open() WrapError %v", err)
			return
		}
		// 关闭文件
		defer func() {
			if err := open.Close(); err != nil {
				t.Errorf("Close() WrapError %v", err)
			}
		}()

		// 处理扫描的行数据
		stat, _ := open.Stat()
		var content = make([]byte, 0, stat.Size())
		var handle = func(num int, line []byte, err error) error {
			if err != nil {
				if err == io.EOF {
					return utils.DONE
				}
				t.Errorf("handle() WrapError %v", err)
				return err
			}

			// 读取前20行数据
			//if num > 20 {
			//	return DONE
			//}

			content = append(content, line...)
			//content = append(content, '\n')

			//t.Logf("第%d行 %v\n", num, string(line))
			return nil
		}

		if err := utils.Scan(open, handle, tt.args.size...); (err != nil) != tt.wantErr {
			t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
		}
		//t.Logf("content size = %v, fileSize = %v", len(content), stat.Size())
		//t.Logf("content\n%v", string(content))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f(t, tt)
		})
	}
}

// go test -bench=Scan$ -run ^$  -count 5 -benchmem
func BenchmarkScan(t *testing.B) {
	type args struct {
		name string
		size []int
	}

	type test struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []test{
		{name: "001", args: args{name: "./file.go", size: []int{int(utils.MB)}}},
	}

	var f = func(t *testing.B, tt test) {
		// 打开文件
		open, err := os.OpenFile(tt.args.name, os.O_RDONLY, 0)
		if err != nil {
			t.Errorf("Open() WrapError %v", err)
			return
		}
		// 关闭文件
		defer func() {
			if err := open.Close(); err != nil {
				t.Errorf("Close() WrapError %v", err)
			}
		}()

		// 处理扫描的行数据
		//stat, _ := open.Stat()
		//var content = make([]byte, 0, stat.Size())
		var handle = func(num int, line []byte, err error) error {
			if err != nil {
				if err == io.EOF {
					return utils.DONE
				}
				t.Errorf("handle() WrapError %v", err)
				return err
			}

			// 读取前20行数据
			//if num > 20 {
			//	return DONE
			//}

			//content = append(content, line...)
			//content = append(content, '\n')

			//t.Logf("第%d行 %v\n", num, string(line))
			return nil
		}

		if err := utils.Scan(open, handle, tt.args.size...); (err != nil) != tt.wantErr {
			t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
		}
		//t.Logf("content size = %v, fileSize = %v", len(content), stat.Size())
		//t.Logf("content\n%v", string(content))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				f(t, tt)
			}
		})
	}
}

func TestLine(t *testing.T) {
	type args struct {
		name string
	}

	type test struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []test{
		{name: "001", args: args{name: "./file.go"}},
	}

	var f = func(t *testing.T, tt test) {
		// 打开文件
		open, err := os.OpenFile(tt.args.name, os.O_RDONLY, 0)
		if err != nil {
			t.Errorf("Open() WrapError %v", err)
			return
		}
		// 关闭文件
		defer func() {
			if err := open.Close(); err != nil {
				t.Errorf("Close() WrapError %v", err)
			}
		}()

		// 处理读取的数据
		stat, _ := open.Stat()
		var content = make([]byte, 0, stat.Size())
		var handle = func(num int, line []byte, lineDone bool) error {
			if err != nil {
				t.Errorf("handle() WrapError %v", err)
				return err
			}

			content = append(content, line...)
			if lineDone {
				content = append(content, '\n') // 每行数据末尾添加换行符
			}
			return nil
		}

		if err := utils.Line(open, handle); (err != nil) != tt.wantErr {
			t.Errorf("Line() error = %v, wantErr %v", err, tt.wantErr)
		}
		//t.Logf("content size = %v, fileSize = %v", len(content), stat.Size())
		//t.Logf("content\n%v", string(content))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f(t, tt)
		})
	}
}

// go test -bench=Line$ -run ^$  -count 5 -benchmem
func BenchmarkLine(t *testing.B) {
	type args struct {
		name string
	}

	type test struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []test{
		{name: "001", args: args{name: "./file.go"}},
	}

	var f = func(t *testing.B, tt test) {
		// 打开文件
		open, err := os.OpenFile(tt.args.name, os.O_RDONLY, 0)
		if err != nil {
			t.Errorf("Open() WrapError %v", err)
			return
		}
		// 关闭文件
		defer func() {
			if err := open.Close(); err != nil {
				t.Errorf("Close() WrapError %v", err)
			}
			return
		}()

		// 处理读取的数据
		//stat, _ := open.Stat()
		//var content = make([]byte, 0, stat.Size())
		var handle = func(num int, line []byte, lineDone bool) error {
			if err != nil {
				t.Errorf("handle() WrapError %v", err)
				return err
			}

			//content = append(content, line...)
			//if lineDone {
			//	content = append(content, '\n')
			//}
			return nil
		}

		if err := utils.Line(open, handle); (err != nil) != tt.wantErr {
			t.Errorf("Line() error = %v, wantErr %v", err, tt.wantErr)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				f(t, tt)
			}
		})
	}
}

func TestRead(t *testing.T) {
	type args struct {
		name string
	}

	type test struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []test{
		{name: "001", args: args{name: "./file.go"}},
	}

	var f = func(t *testing.T, tt test) {
		// 打开文件
		open, err := os.OpenFile(tt.args.name, os.O_RDONLY, 0)
		if err != nil {
			t.Errorf("Open() WrapError %v", err)
			return
		}
		// 关闭文件
		defer func() {
			if err := open.Close(); err != nil {
				t.Errorf("Close() WrapError %v", err)
			}
		}()

		// 处理读取的数据
		stat, _ := open.Stat()
		var content = make([]byte, 0, stat.Size())
		var handle = func(size int, block []byte) error {
			// t.Logf("handle() size = %v", num)
			content = append(content, block...)
			return nil
		}

		if err := utils.Read(open, handle); (err != nil) != tt.wantErr {
			t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
		}
		// t.Logf("content \n%v", string(content))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f(t, tt)
		})
	}
}

// go test -bench=Read$ -run ^$  -count 5 -benchmem
func BenchmarkRead(t *testing.B) {
	type args struct {
		name string
	}

	type test struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []test{
		{name: "001", args: args{name: "./file.go"}},
	}

	var f = func(t *testing.B, tt test) {
		// 打开文件
		open, err := os.OpenFile(tt.args.name, os.O_RDONLY, 0)
		if err != nil {
			t.Errorf("Open() WrapError %v", err)
			return
		}
		// 关闭文件
		defer func() {
			if err := open.Close(); err != nil {
				t.Errorf("Close() WrapError %v", err)
			}
		}()

		// 处理读取的数据
		//stat, _ := open.Stat()
		//var content = make([]byte, 0, stat.Size())
		var handle = func(size int, block []byte) error {
			// t.Logf("handle() size = %v", num)
			// content = append(content, block...)
			return nil
		}

		if err := utils.Read(open, handle); (err != nil) != tt.wantErr {
			t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				f(t, tt)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	type args struct {
		fileName string
		perm     os.FileMode
		isAppend bool
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{fileName: "/tmp/test.log", perm: 0744, isAppend: false}},           // 不追加
		{name: "002", args: args{fileName: "/tmp/test2.log", perm: 0744, isAppend: true}},           // 追加
		{name: "003", args: args{fileName: "/tmp/test/test/test.log", perm: 0711, isAppend: false}}, // 创建多级目录
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := utils.NewWrite(tt.args.fileName, tt.args.isAppend, tt.args.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWrite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// 关闭文件
			defer func() {
				if err := w.Close(); err != nil {
					t.Errorf("Close() WrapError %v", err)
				}
			}()

			group := &sync.WaitGroup{}
			// 写入数据
			for i := 1; i <= 30; i++ {
				group.Add(1)
				go func(g *sync.WaitGroup, i int, t *testing.T) {
					defer g.Done()
					if i%3 == 0 {
						for j := 0; j < 10; j++ {
							_, err := w.Write([]byte(fmt.Sprintf("Write %d-%d Name %v; 太液仙舟迥，西园引上才。未晓征车度，鸡鸣关早开。\n", i, j, tt.name)))
							if err != nil {
								t.Errorf("Write() error = %v", err)
								return
							}
							// t.Logf("Write content size = %v", size)
						}
					} else if i%3 == 1 {
						for j := 0; j < 10; j++ {
							_, err := w.WriteString(fmt.Sprintf("WriteString %d-%d Name %v; 隔户杨柳弱袅袅，恰似十五女儿腰。谁谓朝来不作意，狂风挽断最长条。\n", i, j, tt.name))
							if err != nil {
								t.Errorf("WriteString() error = %v", err)
								return
							}
							//t.Logf("WriteString content size = %v", size)
						}

					} else {
						_, err := w.WriteBuf(func(write *bufio.Writer) (int, error) {
							for j := 0; j < 10000; j++ {
								_, err := write.WriteString(fmt.Sprintf("WriteBuf %d-%d Name %v; 红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n", i, j, tt.name))
								if err != nil {
									return 0, err
								}
							}
							return 0, nil
						})
						if err != nil {
							t.Errorf("WriteBuf() error = %v", err)
							return
						}
						//t.Logf("WriteBuf content size = %v", size)
					}
				}(group, i, t)
			}
			group.Wait()

			/*open, WrapError := os.Open(tt.args.fileName)
			if WrapError != nil {
				t.Errorf("Open() error = %v", WrapError)
			}

			if WrapError := Line(open, func(num int, line []byte, lineDone bool) error {
				t.Logf("第%d行 %v\n", num, string(line))
				return nil
			}); WrapError != nil {
				t.Errorf("Line() error = %v", WrapError)
			}*/

		})
	}
}

// go test -bench=Write$ -run ^$  -count 5 -benchmem
func BenchmarkWrite(t *testing.B) {
	type args struct {
		fileName string
		perm     os.FileMode
		isAppend bool
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "001", args: args{fileName: "/tmp/test3.log", perm: 0744, isAppend: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.B) {
			w, err := utils.NewWrite(tt.args.fileName, tt.args.isAppend, tt.args.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWrite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer func() {
				if err := w.Close(); err != nil {
					t.Errorf("Close() WrapError %v", err)
				}
			}()

			t.ResetTimer()
			// 写入10行数据
			for i := 0; i <= t.N; i++ {
				// 写入带缓存(测试结果速度最快)
				_, err := w.WriteBuf(func(write *bufio.Writer) (int, error) {
					for j := 0; j < 10; j++ {
						_, err := write.WriteString("红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n")
						if err != nil {
							return 0, err
						}
					}
					return 0, nil
				})
				if err != nil {
					t.Errorf("WriteBuf() error = %v", err)
					return
				}

				// 加锁
				/*for j := 0; j < 10; j++ {
					_, WrapError := w.WriteString("红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n")
					if WrapError != nil {
						t.Errorf("WriteString() error = %v", WrapError)
						return
					}
					//t.Logf("WriteString content size = %v", size)
				}*/

				// 不加锁
				/*for j := 0; j < 10; j++ {
					_, WrapError := w.File.WriteString("红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n")
					if WrapError != nil {
						t.Errorf("File.WriteString() error = %v", WrapError)
						return
					}
					//t.Logf("WriteString content size = %v", size)
				}*/

				// 加锁
				/*for j := 0; j < 10; j++ {
					_, WrapError := w.Write([]byte("红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n"))
					if WrapError != nil {
						t.Errorf("Write() error = %v", WrapError)
						return
					}
					// t.Logf("Write content size = %v", size)
				}*/

				// 不加锁
				/*for j := 0; j < 10; j++ {
					_, WrapError := w.File.Write([]byte("红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n"))
					if WrapError != nil {
						t.Errorf("Write() error = %v", WrapError)
						return
					}
					// t.Logf("Write content size = %v", size)
				}*/

			}
		})
	}
}
