package handler

import (
	"net/http"
	"strings"
	"time"

	applog "zero-web-kit/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var logWSUpgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   4096,
	EnableCompression: false, // 避免经代理时出现 RSV1 帧错误
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// LogChannel WebSocket 实时日志，对齐 WVP /channel/log。
// 鉴权：query access-token，或兼容 Sec-WebSocket-Protocol。
func LogChannel(c *gin.Context) {
	proto := c.GetHeader("Sec-WebSocket-Protocol")
	header := http.Header{}
	if proto != "" {
		// 仅回显简单子协议；JWT 含非法 token 字符时不要当 protocol
		first := strings.TrimSpace(strings.Split(proto, ",")[0])
		if first != "" && !strings.Contains(first, ".") {
			header.Set("Sec-WebSocket-Protocol", first)
		}
	}
	conn, err := logWSUpgrader.Upgrade(c.Writer, c.Request, header)
	if err != nil {
		applog.Warn("log websocket upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	ch := applog.DefaultHub.Subscribe()
	defer applog.DefaultHub.Unsubscribe(ch)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case line, ok := <-ch:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}
