package profiler

import (
	"context"
	"errors"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/logging"
)

const Component logging.Component = "github.com/webitel/webitel-go-kit/infra/profiler"

type Profiler struct {
	server *http.Server
	logger *logging.Logger
}

func New(logger *logging.Logger, config Config) *Profiler {
	if config.Addr == "" {
		return nil
	}

	runtime.SetMutexProfileFraction(config.MutexProfileFraction)
	runtime.SetBlockProfileRate(config.BlockProfileRate)

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
		logger: logger.WithComponent(Component),
	}
}

func (p *Profiler) Start() error {
	if p == nil {
		return nil
	}

	go func() {
		ctx := context.Background()
		p.logger.InfoCtx(ctx, "pprof server starting", logging.Str("addr", p.server.Addr))

		if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.logger.ErrorCtx(ctx, "pprof server failed", logging.Err(err))
		}
	}()

	return nil
}

func (p *Profiler) Stop(ctx context.Context) error {
	if p == nil {
		return nil
	}

	p.logger.InfoCtx(ctx, "pprof server stopping")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.server.Shutdown(ctx)
}
