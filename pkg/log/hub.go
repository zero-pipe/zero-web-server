package log

import (
	"sync"
)

// Hub broadcasts log lines to WebSocket subscribers (实时日志).
type Hub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[chan string]struct{})}
}

var DefaultHub = NewHub()

// Subscribe returns a buffered channel of log lines. Caller must Unsubscribe.
func (h *Hub) Subscribe() chan string {
	ch := make(chan string, 256)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
	h.mu.Unlock()
}

func (h *Hub) Broadcast(line string) {
	if line == "" {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- line:
		default:
			// 客户端慢则丢弃，避免阻塞写日志
		}
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
