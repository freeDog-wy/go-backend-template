// Package logging 负责日志基础设施的初始化。
// 根据运行模式创建 slog.Handler 并封装为 pkg/logger.Logger。
package logging

import (
	"log/slog"
	"os"

	"github.com/freeDog-wy/go-backend-template/pkg/logger"
)

// Init 根据运行模式初始化全局日志实例。
//   - "development" → 控制台 TextHandler + Debug 级别
//   - "production"  → stdout JSONHandler + Info 级别
func Init(mode string) logger.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		AddSource: false,
	}

	switch mode {
	case "development":
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stderr, opts)
	default:
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return logger.NewSlog(slog.New(handler))
}
