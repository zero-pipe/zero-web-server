package log

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// textHandler 输出行业标准可读文本行，供 stdout / 文件使用。
type textHandler struct {
	w     io.Writer
	opts  *slog.HandlerOptions
	mu    *sync.Mutex
	attrs []slog.Attr
	group string
}

func newTextHandler(w io.Writer, opts *slog.HandlerOptions) *textHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &textHandler{w: w, opts: opts, mu: &sync.Mutex{}}
}

func (h *textHandler) Enabled(_ context.Context, level slog.Level) bool {
	min := slog.LevelInfo
	if h.opts.Level != nil {
		min = h.opts.Level.Level()
	}
	return level >= min
}

func (h *textHandler) Handle(_ context.Context, r slog.Record) error {
	if len(h.attrs) > 0 {
		r.AddAttrs(h.attrs...)
	}
	line := FormatRecordText(r) + "\n"
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, line)
	return err
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *textHandler) WithGroup(name string) slog.Handler {
	next := *h
	if h.group != "" {
		next.group = h.group + "." + name
	} else {
		next.group = name
	}
	// 简化：group 前缀叠到后续 attr key（Handle 时 attrs 已带入）
	_ = next.group
	return &next
}
