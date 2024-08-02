package wlog_provider

import (
	"context"
	"github.com/webitel/webitel-go-kit/logs/model"
	"github.com/webitel/wlog"
)

type WlogProvider struct {
	base   *wlog.Logger
	config *config
}

func New(logger *wlog.Logger, opts ...Option) (*WlogProvider, error) {
	var (
		conf config
	)
	if logger == nil {
		logger = wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: true, ConsoleLevel: wlog.LevelDebug})
	}
	for _, opt := range opts {
		opt.apply(&conf)
	}
	return &WlogProvider{base: logger}, nil
}

func (o *WlogProvider) Info(ctx context.Context, s *model.Record) error {
	o.base.Info(s.Message)
	return nil
}

func (o *WlogProvider) Debug(ctx context.Context, s *model.Record) error {
	o.base.Debug(s.Message)
	return nil
}

func (o *WlogProvider) Warn(ctx context.Context, s *model.Record) error {
	o.base.Warn(s.Message)
	return nil
}

func (o *WlogProvider) Critical(ctx context.Context, s *model.Record) error {
	o.base.Critical(s.Error.Message)
	return nil
}

func (o *WlogProvider) Error(ctx context.Context, s *model.Record) error {
	o.base.Error(s.Error.Message)
	return nil
}

func (o *WlogProvider) SetAsGlobal() {
	wlog.InitGlobalLogger(o.base)
}
