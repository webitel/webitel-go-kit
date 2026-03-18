package profiler

import (
	"context"
	"errors"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"
)

type Profiler struct {
	server *http.Server
	logger Logger
}

func New(config Config, logger Logger) *Profiler {
	if config.Addr == "" {
		return nil
	}

	runtime.SetMutexProfileFraction(1)
	runtime.SetBlockProfileRate(1)

	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
	mux.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	mux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	mux.HandleFunc("/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
	mux.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)

	srv := &http.Server{
		Addr:    config.Addr,
		Handler: mux,
	}

	return &Profiler{
		server: srv,
		logger: logger,
	}
}

func (p *Profiler) Start() error {
	if p == nil {
		return nil
	}

	go func() {
		p.logger.Info("pprof server starting", "addr", p.server.Addr)

		if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.logger.Error("pprof server failed", "err", err)
		}
	}()

	return nil
}

func (p *Profiler) Stop(ctx context.Context) error {
	if p == nil {
		return nil
	}

	p.logger.Info("pprof server stopping")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.server.Shutdown(ctx)
}
