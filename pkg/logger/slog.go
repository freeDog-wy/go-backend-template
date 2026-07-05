package logger

import "log/slog"

// SlogLogger 基于标准库 log/slog 的 Logger 实现。
type SlogLogger struct {
	inner *slog.Logger
}

// NewSlog 将 *slog.Logger 包装为 Logger 接口。
func NewSlog(l *slog.Logger) *SlogLogger {
	return &SlogLogger{inner: l}
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.inner.Info(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.inner.Error(msg, args...)
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.inner.Debug(msg, args...)
}

func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{inner: l.inner.With(args...)}
}
