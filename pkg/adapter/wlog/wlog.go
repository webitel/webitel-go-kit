package wlogadapter

import (
	"github.com/webitel/wlog"
)

type Adapter struct {
	log *wlog.Logger
}

func New(log *wlog.Logger) *Adapter {
	return &Adapter{log: log}
}

// Info logs informational messages with structured args.
func (a *Adapter) Info(msg string, args ...any) {
	a.log.Info(msg, toWlogFields(args)...)
}

// Error logs error messages with the error as an attribute.
func (a *Adapter) Error(msg string, args ...any) {
	a.log.Error(msg, toWlogFields(args)...)
}

// Debug logs debug messages with the structured args.
func (a *Adapter) Debug(msg string, args ...any) {
	a.log.Debug(msg, toWlogFields(args)...)
}

// Warn logs warning messages with structured args.
func (a *Adapter) Warn(msg string, args ...any) {
	a.log.Warn(msg, toWlogFields(args)...)
}

// toWlogFields converts key-value pairs to wlog.Field.
func toWlogFields(args ...any) []wlog.Field {
	fields := make([]wlog.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		value := args[i+1].(string)
		fields = append(fields, wlog.String(key, value))
	}
	return fields
}
