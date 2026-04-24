package errors

import "log/slog"

func (e *Error) LogValue() slog.Value {
	if e == nil {
		return slog.GroupValue(
			slog.Int("code", 0),
			slog.String("status", "OK"),
		)
	}
	return slog.GroupValue(
		slog.Int("code", int(e.Code)),
		slog.String("status", e.Status),
		slog.String("message", e.Message),
	)
}
