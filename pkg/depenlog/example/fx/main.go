package main

import (
	"context"

	gokitlog "github.com/webitel/webitel-go-kit/pkg/depenlog"
	"github.com/webitel/webitel-go-kit/pkg/logger"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type Server struct {
	log logger.Logger
}

func NewServer(l logger.Logger) *Server {
	return &Server{log: gokitlog.WithComponent(l, "server")}
}

func main() {
	l, err := gokitlog.New(gokitlog.Config{Level: "debug", JSON: true, Console: true})
	if err != nil {
		panic(err)
	}

	app := fx.New(
		// Make the kit logger available to the graph (as the interface).
		fx.Provide(func() logger.Logger { return l }),
		fx.Provide(NewServer),
		// Route fx's own provide/invoke/hook logs through the unified logger.
		fx.WithLogger(func() fxevent.Logger { return gokitlog.FxLogger(l) }),
		fx.Invoke(func(s *Server, lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error { s.log.Info("started"); return nil },
				OnStop:  func(context.Context) error { s.log.Info("stopping"); return nil },
			})
		}),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		l.Error("fx start failed", "err", err)
		return
	}
	if err := app.Stop(ctx); err != nil {
		l.Error("fx stop failed", "err", err)
	}
}
