package wlog

import (
	"fmt"
	"github.com/webitel/wlog"
)

type Logger struct {
	base *wlog.Logger
}

func NewWlogLogger(base *wlog.Logger) *Logger {
	return &Logger{base: base}
}

func (l *Logger) Info(msg string, args ...any) {
	l.base.Info(fmt.Sprintf(msg, args...))
}

func (l *Logger) Warn(msg string, args ...any) {
	l.base.Warn(fmt.Sprintf(msg, args...))
}

func (l *Logger) Error(msg string, err error, args ...any) {
	formattedMsg := fmt.Sprintf(msg, args...)
	l.base.Error(fmt.Sprintf("%s: %v", formattedMsg, err))
}
