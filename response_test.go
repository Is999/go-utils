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

func TestResponse(t *testing.T) {
	// 退出
	exit := make(chan os.Signal)

	// 请求该路由退出
	// http://localhost:54333/response/exit
	http.HandleFunc("/response/exit", func(w http.ResponseWriter, r *http.Request) {
		// 退出信号
		exit <- syscall.Signal(1)
	})

	// 响应html、xml、text、file、image
	// http://localhost:54333/response/html
	// http://localhost:54333/response/xml
	// http://localhost:54333/response/text
	// http://localhost:54333/response/show?file=go.mod
	// http://localhost:54333/response/show?file=golang_icon.png
	// http://localhost:54333/response/download?file=go.mod
	// http://localhost:54333/response/download?file=golang_icon.png
	ExampleView()

	// 响应json
	// http://localhost:54333/response/json
	ExampleJsonResp()

	// 重定向
	// http://localhost:54333/response/redirect
	ExampleRedirect()

	//使用默认路由创建 http server
	srv := http.Server{
		Addr:    ":54333",
		Handler: http.DefaultServeMux,
	}

	//监听 Ctrl+C 信号
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		timer := time.NewTimer(3 * time.Minute)
		for {
			select {
			case <-exit:
				fmt.Println("Exit...")
				srv.Shutdown(context.Background())
			case <-timer.C:
				fmt.Println("Delayed 5s Exit...")
				//使用context控制srv.Shutdown的超时时间
				//ctx, _ := context.WithTimeout(context.Background(), time.Second)
				srv.Shutdown(context.Background())
			default:
				time.Sleep(time.Second)
				fmt.Println("default 1s...")
			}
		}
	}()

	// 启动HTTP服务器，监听在指定端口
	err := srv.ListenAndServe()
	if err != nil {
		fmt.Println("HTTP server failed to start:", err)
	}
}
