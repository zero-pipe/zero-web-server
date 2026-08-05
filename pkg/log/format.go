package log

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// 行业常见控制台日志格式（对齐运维可读约定）：
//
//	2026-07-14 17:09:21.165 INFO  message key=value
//
// 级别固定 5 字符宽：DEBUG / INFO  / WARN  / ERROR
const timeLayout = "2006-01-02 15:04:05.000"

// FormatRecordText 将 slog.Record 格式化为统一文本行（文件 / 实时日志共用）。
func FormatRecordText(r slog.Record) string {
	var b strings.Builder
	b.Grow(128)
	b.WriteString(r.Time.Local().Format(timeLayout))
	b.WriteByte(' ')
	b.WriteString(formatLevel(r.Level))
	b.WriteByte(' ')
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "" || a.Equal(slog.Attr{}) {
			return true
		}
		a.Value = a.Value.Resolve()
		if a.Value.Kind() == slog.KindAny {
			if err, ok := a.Value.Any().(error); ok && err != nil {
				b.WriteByte(' ')
				b.WriteString(a.Key)
				b.WriteByte('=')
				b.WriteString(quoteIfNeeded(err.Error()))
				return true
			}
		}
		b.WriteByte(' ')
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(formatValue(a.Value))
		return true
	})
	return b.String()
}

func formatLevel(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO "
	case level < slog.LevelError:
		return "WARN "
	default:
		return "ERROR"
	}
}

func formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return quoteIfNeeded(v.String())
	case slog.KindTime:
		return v.Time().Local().Format(time.RFC3339)
	case slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool, slog.KindDuration:
		return v.String()
	default:
		return quoteIfNeeded(fmt.Sprint(v.Any()))
	}
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	for _, r := range s {
		if r <= ' ' || r == '=' || r == '"' || r == '\\' {
			return strconv.Quote(s)
		}
	}
	return s
}

// jsonReplaceAttr 对齐常见采集规范：ts / level(小写) / msg。
func jsonReplaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		t := a.Value.Time()
		return slog.String("ts", t.Local().Format(time.RFC3339Nano))
	case slog.LevelKey:
		level := a.Value.Any().(slog.Level)
		name := "info"
		switch {
		case level < slog.LevelInfo:
			name = "debug"
		case level < slog.LevelWarn:
			name = "info"
		case level < slog.LevelError:
			name = "warn"
		default:
			name = "error"
		}
		return slog.String("level", name)
	case slog.MessageKey:
		return slog.Attr{Key: "msg", Value: a.Value}
	case slog.SourceKey:
		return a
	default:
		return a
	}
}
