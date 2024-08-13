package slog

import (
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

type Option = otelslog.Option

func New(name string, opts ...Option) *slog.Logger {
	return otelslog.NewLogger(name, opts...)
}
