package utils_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
		for {
			select {
			case <-exit:
				fmt.Println(addr + " Exit...")
				srv.Shutdown(context.Background())
			case <-timer.C:
				fmt.Println(addr + " Delayed 10s Exit...")
				srv.Shutdown(context.Background())
			default:
				time.Sleep(time.Second)
				fmt.Println(addr + " Sleep 1s...")
			}
		}
	}()

	// 启动HTTP服务器，监听在指定端口
	err := srv.ListenAndServe()
	if err != nil {
		fmt.Println("HTTP server failed to start:", err)
	}

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
