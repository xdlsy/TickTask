package logger

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func init() {
	// Initialize with default logger
	Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Init 初始化日志
func Init(mode string) {
	var level slog.Level
	switch mode {
	case "debug":
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	Logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
}
