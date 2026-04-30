package utils_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"
)

var serveMux = http.NewServeMux()

func httpServer(addr string, header http.Handler, exit chan os.Signal) {
	//使用默认路由创建 http server
	srv := http.Server{
		Addr:    addr,
		Handler: header,
	}

	//监听 Ctrl+C 信号
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		select {
		case <-exit:
		case <-timer.C:
		}
		_ = srv.Shutdown(context.Background())
	}()

	// 启动 HTTP 服务器。部分测试复用固定端口，race 模式下前一个 server
	// 刚 Shutdown 时端口可能短暂未释放，这里做有限重试，避免测试偶发失败。
	for i := 0; i < 40; i++ {
		err := srv.ListenAndServe()
		if err == nil || err == http.ErrServerClosed {
			return
		}
		if !strings.Contains(err.Error(), "address already in use") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

}

func waitHTTPServer(t *testing.T, addr string) {
	t.Helper()
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("HTTP server %s did not start", addr)
}

func TestResponse(t *testing.T) {
	// 退出
	exit := make(chan os.Signal)

	// 请求该路由退出
	// http://localhost:54333/response/exit
	serveMux.HandleFunc("/response/exit", func(w http.ResponseWriter, r *http.Request) {
		// 退出信号
		exit <- syscall.Signal(1)
	})

	// 响应html、xml、text、file、image
	// http://localhost:54333/response/html
	// http://localhost:54333/response/xml
	// http://localhost:54333/response/text
	// http://localhost:54333/response/show?file=go.mod
	// http://localhost:54333/response/show?file=resource/golang_icon.png
	// http://localhost:54333/response/download?file=go.mod
	// http://localhost:54333/response/download?file=resource/golang_icon.png
	ExampleView()

	// 响应json
	// http://localhost:54333/response/json
	ExampleJson()

	// 重定向
	// http://localhost:54333/response/redirect
	ExampleRedirect()

	httpServer(":54333", serveMux, exit)
}
