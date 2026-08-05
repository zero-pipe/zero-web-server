package handler

import (
	"net/http"
	"strings"
	"time"

	applog "zero-web-server/pkg/log"

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

const realtimeLogTailLines = 50

// LogChannel WebSocket 实时日志（/channel/log）。
// 连接后先推送最近约 200 行，再以 tail -f 方式跟随日志文件；同时订阅内存 Hub 兜底。
func LogChannel(c *gin.Context) {
	proto := c.GetHeader("Sec-WebSocket-Protocol")
	header := http.Header{}
	if proto != "" {
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

	writeLine := func(line string) error {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteMessage(websocket.TextMessage, []byte(line))
	}

	// 有落盘文件时以文件跟随为准（与历史日志一致）；无文件时走内存 Hub
	ch := applog.DefaultHub.Subscribe()
	defer applog.DefaultHub.Unsubscribe(ch)
	useFile := applog.FilePath() != ""

	recent := applog.TailRecent(realtimeLogTailLines)
	follower := applog.NewFileFollower()
	if len(recent) == 0 {
		_ = writeLine("--- 实时日志已连接，等待新日志 ---")
	} else {
		for _, line := range recent {
			if err := writeLine(line); err != nil {
				return
			}
		}
	}
	follower.SeekEnd()

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

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()
	pollTicker := time.NewTicker(400 * time.Millisecond)
	defer pollTicker.Stop()

	for {
		select {
		case <-done:
			return
		case line, ok := <-ch:
			if !ok {
				return
			}
			// 已跟文件时忽略 Hub，避免同一条日志两种格式各推一次
			if useFile {
				continue
			}
			if err := writeLine(line); err != nil {
				return
			}
		case <-pollTicker.C:
			if !useFile {
				continue
			}
			for _, line := range follower.Poll() {
				if err := writeLine(line); err != nil {
					return
				}
			}
		case <-pingTicker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}
