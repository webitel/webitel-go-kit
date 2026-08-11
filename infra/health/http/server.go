package healthhttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/webitel/webitel-go-kit/infra/health"
)

// Server serves the probes on its own listener. A nil *Server is a no-op.
type Server struct {
	h    *handler
	srv  *http.Server
	addr string
	log  *slog.Logger

	// mu guards what a second Start or a concurrent Addr would race.
	mu      sync.Mutex
	ln      net.Listener
	started bool
}

// NewServer builds a probe server, or nil when addr is empty.
func NewServer(r *health.Registry, addr string, opts ...Option) *Server {
	if addr == "" {
		return nil
	}

	o := newOptions(opts)
	h := newHandler(r, o, false, "") // verbose is granted in serve, once the address is known

	return &Server{
		h: h,
		srv: &http.Server{
			Handler:           h,
			ReadHeaderTimeout: 5 * time.Second, // Slowloris; the repo has no linter to remind us
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 13,
			ErrorLog:          slog.NewLogLogger(o.log.Handler(), slog.LevelWarn),
		},
		addr: addr,
		log:  o.log,
	}
}

// Start binds the listener, then serves in the background. Calling it twice
// returns an error rather than binding a second port.
func (s *Server) Start() error {
	if s == nil {
		return nil
	}

	// Listen synchronously: ListenAndServe in a goroutine returns nil on a taken
	// port and surfaces the bind error only later, in a log line.
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("health/http: listen %s: %w", s.addr, err)
	}

	if err := s.serve(l); err != nil {
		l.Close()

		return err
	}

	return nil
}

// serve is the seam the tests use to present a non-loopback address.
func (s *Server) serve(l net.Listener) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()

		return errors.New("health/http: server is already started")
	}
	s.started = true
	s.ln = l

	verbose := loopbackOnly(l.Addr())
	s.h.verboseAllowed.Store(verbose)
	s.mu.Unlock()

	if !verbose {
		s.log.Warn("health/http: ?verbose refused, the listener is not loopback", "addr", l.Addr())
	}

	go func() {
		if err := s.srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("health/http: server failed", "err", err)
		}
	}()

	s.log.Info("health/http: listening", "addr", l.Addr())

	return nil
}

// Stop shuts the server down, bounded by ctx and by its own timeout. Stopping a
// server that was never started is a no-op, not an error.
func (s *Server) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	started := s.started
	s.mu.Unlock()

	if !started {
		return nil
	}

	s.log.Info("health/http: stopping")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.srv.Shutdown(ctx)
}

// Addr reports the bound address once listening, otherwise the configured one.
func (s *Server) Addr() string {
	if s == nil {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ln != nil {
		return s.ln.Addr().String()
	}

	return s.addr
}

// loopbackOnly decides from the bound listener, never from the configured
// string: net.Listen has already resolved it, so the address is a concrete IP.
// No name is resolved here — a lookup would be both a side effect and a TOCTOU
// window, which is why example.com is not loopback to this rule.
func loopbackOnly(a net.Addr) bool {
	host, _, err := net.SplitHostPort(a.String())
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}
