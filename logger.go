package plugmgr

import (
	"log/slog"
)

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// logger 默认日志记录器
type logger struct {
	logger *slog.Logger
}

func (l *logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}
