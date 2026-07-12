package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"zero-web-kit/internal/infrastructure/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	mu     sync.Mutex
	logger *slog.Logger
)

// Init configures process-wide structured logging (stdout and/or rotating file).
func Init(cfg config.LogConfig) error {
	mu.Lock()
	defer mu.Unlock()

	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}
	var writers []io.Writer

	switch strings.ToLower(cfg.Output) {
	case "file":
		w, err := openLogFile(cfg.File)
		if err != nil {
			return err
		}
		writers = append(writers, w)
	case "both":
		writers = append(writers, os.Stdout)
		w, err := openLogFile(cfg.File)
		if err != nil {
			return err
		}
		writers = append(writers, w)
	default:
		writers = append(writers, os.Stdout)
	}

	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "json") {
		handler = slog.NewJSONHandler(io.MultiWriter(writers...), opts)
	} else {
		handler = slog.NewTextHandler(io.MultiWriter(writers...), opts)
	}
	handler = NewBroadcastHandler(handler, DefaultHub)
	logger = slog.New(handler)
	slog.SetDefault(logger)
	return nil
}

// LogDir returns the directory containing the rotating log file.
func LogDir(cfg config.LogFileConfig) string {
	path := cfg.Path
	if path == "" {
		path = "logs/zero-web-kit.log"
	}
	return filepath.Dir(path)
}

func openLogFile(cfg config.LogFileConfig) (io.Writer, error) {
	path := cfg.Path
	if path == "" {
		path = "logs/zero-web-kit.log"
	}
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
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
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
