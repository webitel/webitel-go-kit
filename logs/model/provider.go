package model

import "context"

type LogProvider interface {
	Info(ctx context.Context, s *Record) error
	Debug(ctx context.Context, s *Record) error
	Warn(ctx context.Context, s *Record) error
	Error(ctx context.Context, s *Record) error
	Critical(ctx context.Context, s *Record) error
	SetAsGlobal()
}
