package xlog

import (
	"context"
	"log/slog"
	"os"
	"runtime"
)

var logger *slog.Logger

func init() {
	var level = slog.LevelDebug

	switch os.Getenv("LOG_LEVEL") {
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	case "debug":
		level = slog.LevelDebug
	}

	var opts = &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	logger = slog.New(handler)
}

func _log(level slog.Level, msg string, args ...any) {
	_, f, l, _ := runtime.Caller(2)
	group := slog.Group(
		"source",
		slog.Attr{
			Key:   "file",
			Value: slog.AnyValue(f),
		},
		slog.Attr{
			Key:   "L",
			Value: slog.AnyValue(l),
		},
	)
	args = append(args, group)
	logger.Log(context.Background(), level, msg, args...)
}

func Info(msg string, args ...any) {
	_log(slog.LevelInfo, msg, args...)
}

func Debug(msg string, args ...any) {
	_log(slog.LevelDebug, msg, args...)
}

func Error(msg string, args ...any) {
	_log(slog.LevelError, msg, args...)
}

func Warn(msg string, args ...any) {
	_log(slog.LevelWarn, msg, args...)
}
