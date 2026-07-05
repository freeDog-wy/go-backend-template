// Package logger 定义独立的日志接口，与具体实现解耦。
// 调用方（service / repository / handler）依赖本接口，
// 初始化由 internal/infra/logging 完成。
package logger

// Logger 结构化日志接口——Info / Error / Debug + 派生带固定字段的子 Logger。
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	With(args ...any) Logger
}

// Noop 返回一个静默的 Logger（各方法均为空操作）。
// 用于测试或不需要日志的场景。
func Noop() Logger { return &noopLogger{} }

type noopLogger struct{}

func (n *noopLogger) Info(string, ...any)  {}
func (n *noopLogger) Error(string, ...any) {}
func (n *noopLogger) Debug(string, ...any) {}
func (n *noopLogger) With(...any) Logger   { return n }
