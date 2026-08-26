// Command web 是中心 Web 服务：节点注册 / 管理 / 发起测速，并托管 static/ 下的前端页面。
//
//	go run ./cmd/web
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"librespeed-service/internal/config"
	"librespeed-service/internal/store"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatalf("[web] 配置加载失败: %v", err)
	}

	st, err := store.Open(settings.DatabasePath())
	if err != nil {
		log.Fatalf("[web] 打开数据库失败: %v", err)
	}
	defer st.Close()

	handler := NewServer(st, settings, "static")

	addr := fmt.Sprintf("%s:%d", settings.Host, settings.Port)
	log.Printf("[web] 监听 %s", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[web] 服务退出: %v", err)
	}
}
