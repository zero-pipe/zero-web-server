package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"zero-web-server/internal/infrastructure/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	mu      sync.Mutex
	logger  *slog.Logger
	logPath string // 当前写入的日志文件路径（空表示未落盘）
)

// Init configures process-wide structured logging (stdout and/or rotating file).
//
// 约定（对齐常见运维 / 采集实践）：
//   - level:  debug | info | warn | error
//   - format: text（人读）| json（采集）
//   - text:   2026-07-14 15:04:05.000 INFO  message key=value
//   - json:   {"ts":"...","level":"info","msg":"...","key":"..."}
func Init(cfg config.LogConfig) error {
	mu.Lock()
	defer mu.Unlock()

	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level <= slog.LevelDebug,
	}
	var writers []io.Writer

	switch strings.ToLower(cfg.Output) {
	case "file":
		w, err := openLogFile(cfg.File)
		if err != nil {
			return err
		}
		writers = append(writers, w)
		logPath = resolveLogPath(cfg.File)
	case "both":
		writers = append(writers, os.Stdout)
		w, err := openLogFile(cfg.File)
		if err != nil {
			return err
		}
		writers = append(writers, w)
		logPath = resolveLogPath(cfg.File)
	default:
		writers = append(writers, os.Stdout)
		logPath = ""
	}

	mw := io.MultiWriter(writers...)
	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "json") {
		jsonOpts := *opts
		jsonOpts.ReplaceAttr = jsonReplaceAttr
		handler = slog.NewJSONHandler(mw, &jsonOpts)
	} else {
		handler = newTextHandler(mw, opts)
	}
	handler = NewBroadcastHandler(handler, DefaultHub)
	logger = slog.New(handler)
	slog.SetDefault(logger)
	return nil
}

// LogDir returns the directory containing the rotating log file.
func LogDir(cfg config.LogFileConfig) string {
	return filepath.Dir(resolveLogPath(cfg))
}

// FilePath returns the active log file path set by Init (may be empty if stdout-only).
func FilePath() string {
	mu.Lock()
	defer mu.Unlock()
	return logPath
}

func resolveLogPath(cfg config.LogFileConfig) string {
	path := cfg.Path
	if path == "" {
		path = "logs/zero-web-server.log"
	}
	return path
}

func openLogFile(cfg config.LogFileConfig) (io.Writer, error) {
	path := resolveLogPath(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	maxSize := cfg.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 100
	}
	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 20
	}
	maxAge := cfg.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 7
	}
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   cfg.Compress,
	}, nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug", "trace":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error", "fatal", "panic":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

func L() *slog.Logger {
	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		logger = slog.Default()
	}
	return logger
}

func Debug(msg string, args ...any) { L().Debug(msg, args...) }
func Info(msg string, args ...any)  { L().Info(msg, args...) }
func Warn(msg string, args ...any)  { L().Warn(msg, args...) }
func Error(msg string, args ...any) { L().Error(msg, args...) }

// Debugf logs at debug level (verbose tracing; hidden when log.level=info).
func Debugf(format string, args ...any) {
	L().Debug(fmt.Sprintf(format, args...))
}

// Infof logs at info level.
func Infof(format string, args ...any) {
	L().Info(fmt.Sprintf(format, args...))
}

// Warnf logs at warn level.
func Warnf(format string, args ...any) {
	L().Warn(fmt.Sprintf(format, args...))
}

// Errorf logs at error level.
func Errorf(format string, args ...any) {
	L().Error(fmt.Sprintf(format, args...))
}

// Printf logs at info level (legacy bridge for log.Printf call sites).
func Printf(format string, args ...any) {
	L().Info(fmt.Sprintf(format, args...))
}

// Fatalf logs at error level then exits (startup failures).
func Fatalf(format string, args ...any) {
	L().Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
