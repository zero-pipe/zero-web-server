package log

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestFormatRecordText(t *testing.T) {
	r := slog.NewRecord(time.Date(2026, 7, 14, 17, 9, 21, 165000000, time.Local), slog.LevelInfo, "zero-web-server starting", 0)
	r.AddAttrs(slog.String("version", "1.0.0"), slog.String("http", ":18080"))
	got := FormatRecordText(r)
	if !strings.Contains(got, " INFO ") {
		t.Fatalf("level: %q", got)
	}
	if !strings.Contains(got, "zero-web-server starting") {
		t.Fatalf("msg: %q", got)
	}
	if !strings.Contains(got, "version=1.0.0") || !strings.Contains(got, "http=:18080") {
		t.Fatalf("attrs: %q", got)
	}
	if !strings.HasPrefix(got, "2026-07-14 17:09:21.165") {
		t.Fatalf("time prefix: %q", got)
	}
}

func TestFormatLevelWidth(t *testing.T) {
	cases := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO "},
		{slog.LevelWarn, "WARN "},
		{slog.LevelError, "ERROR"},
	}
	for _, c := range cases {
		if got := formatLevel(c.level); got != c.want {
			t.Fatalf("level %v: got %q want %q", c.level, got, c.want)
		}
	}
}
