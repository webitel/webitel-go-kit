package wlog

import (
	"fmt"
	"github.com/webitel/wlog"
)

type AdapterLogger struct {
	base *wlog.Logger
}

func NewAdapterLogger(base *wlog.Logger) *AdapterLogger {
	return &AdapterLogger{base: base}
}

func (l *AdapterLogger) Info(msg string, args ...any) {
	l.base.Info(fmt.Sprintf(msg, args...))
}

func (l *AdapterLogger) Warn(msg string, args ...any) {
	l.base.Warn(fmt.Sprintf(msg, args...))
}

func (l *AdapterLogger) Error(msg string, err error, args ...any) {
	formattedMsg := fmt.Sprintf(msg, args...)
	l.base.Error(fmt.Sprintf("%s: %v", formattedMsg, err))
}
