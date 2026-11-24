package consul

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/webitel/webitel-go-kit/infra/discovery"
)

var (
	// ErrKeyNotFound is returned by [Cluster.KVGet] for a key that does not exist.
	ErrKeyNotFound = errors.New("consul: key not found")
	// ErrSessionNotFound is returned for a session that expired or was destroyed.
	ErrSessionNotFound = errors.New("consul: session not found")
	// ErrNoAddrs is returned by [NewCluster] when it gets no server address.
	ErrNoAddrs = errors.New("consul: no server addresses")
	// ErrOutcomeUnknown marks a write that was sent to a server which then failed to
	// answer: it may or may not have been applied, and it is not retried on another server.
	ErrOutcomeUnknown = errors.New("consul: outcome of the write is unknown")
)

// EndpointError is a failed call to one server of a [Cluster].
type EndpointError struct {
	// Addr is host:port of the server.
	Addr string
	Err  error
}

func (e *EndpointError) Error() string { return fmt.Sprintf("consul %s: %v", e.Addr, e.Err) }

func (e *EndpointError) Unwrap() error { return e.Err }

// Cluster is a handle to one Consul cluster reachable through several servers.
// A call that fails on the current server moves on to the next one, and the
// server that answered becomes the current one. Methods are safe for concurrent use.
//
// A response of 4xx is the answer of a working server and is not retried.
// A write that is not safe to repeat (KVCAS, KVDeleteCAS, Release) is not retried
// either once its request has been sent: a server that applied it and lost the
// response would make the retry report a lost race. Such a call returns an error
// that matches [ErrOutcomeUnknown]. Reads, Acquire, and the calls on sessions are
// repeatable and fail over.
//
// A call that fails everywhere returns the [EndpointError] of every attempt, joined
// with [errors.Join] when there is more than one, so that the caller can classify
// each one.
//
// Like the rest of the package, [NewCluster] builds its clients with [api.DefaultConfig],
// so CONSUL_HTTP_TOKEN, CONSUL_HTTP_TOKEN_FILE, CONSUL_NAMESPACE and CONSUL_PARTITION
// of the environment apply.
type Cluster struct {
	addrs      []string
	endpoints  []endpoint
	timeout    time.Duration
	httpClient *http.Client
	current    atomic.Uint32
}

type endpoint struct {
	addr   string
	client *api.Client
}

// WithRequestTimeout bounds one attempt against one server; zero means only the
// context of the call bounds it.
func WithRequestTimeout(d time.Duration) discovery.Option[*Cluster] {
	return func(c *Cluster) { c.timeout = d }
}

// WithHTTPClient sets the HTTP client shared by all servers of the cluster.
func WithHTTPClient(hc *http.Client) discovery.Option[*Cluster] {
	return func(c *Cluster) { c.httpClient = hc }
}

// NewCluster builds a [Cluster] over HTTP servers given as host:port, in the order
// they are tried.
func NewCluster(addrs []string, opts ...discovery.Option[*Cluster]) (*Cluster, error) {
	if len(addrs) == 0 {
		return nil, ErrNoAddrs
	}

	c := &Cluster{addrs: append([]string(nil), addrs...)}
	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone()}
	}

	for _, addr := range c.addrs {
		cfg := api.DefaultConfig()
		cfg.Address = addr
		cfg.Scheme = "http"
		cfg.HttpClient = c.httpClient

		client, err := api.NewClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("consul client for %s: %w", addr, err)
		}

		c.endpoints = append(c.endpoints, endpoint{addr: addr, client: client})
	}

	return c, nil
}

// Addrs returns the server addresses in the order they were given.
func (c *Cluster) Addrs() []string { return append([]string(nil), c.addrs...) }

// do runs fn against the current server and then the others until one succeeds.
// fn must only publish results on success. A call that is not idempotent stops at the
// first attempt whose request was sent.
func (c *Cluster) do(ctx context.Context, idempotent bool, fn func(ctx context.Context, client *api.Client) error) error {
	var (
		n     = len(c.endpoints)
		start = int(c.current.Load()) % n
		errs  []error
	)

	for i := range n {
		idx := (start + i) % n
		ep := c.endpoints[idx]

		sent, err := c.attempt(ctx, ep, fn)
		if err == nil {
			c.current.Store(uint32(idx))

			return nil
		}

		if final(err) {
			errs = append(errs, &EndpointError{Addr: ep.addr, Err: err})

			break
		}

		if !idempotent && sent {
			errs = append(errs, &EndpointError{Addr: ep.addr, Err: fmt.Errorf("%w: %w", ErrOutcomeUnknown, err)})

			break
		}

		errs = append(errs, &EndpointError{Addr: ep.addr, Err: err})

		if ctx.Err() != nil {
			break
		}
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return errors.Join(errs...)
}

// attempt runs fn once and reports whether its request was completely written.
func (c *Cluster) attempt(ctx context.Context, ep endpoint, fn func(context.Context, *api.Client) error) (bool, error) {
	if c.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	var sent atomic.Bool

	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				sent.Store(true)
			}
		},
	})

	err := fn(ctx, ep.client)

	return sent.Load(), err
}

// final reports whether no other server can answer better: a 4xx, or a session
// that is gone.
func final(err error) bool {
	if errors.Is(err, ErrSessionNotFound) {
		return true
	}

	var se api.StatusError
	if errors.As(err, &se) {
		return se.Code < http.StatusInternalServerError
	}

	return false
}

// invalidSession turns the answer of Consul to a lock call with a session that
// is gone into [ErrSessionNotFound]; the text of the body is the only signal.
func invalidSession(err error) error {
	var se api.StatusError
	if errors.As(err, &se) && strings.Contains(se.Body, "invalid session") {
		return fmt.Errorf("%w: %w", ErrSessionNotFound, err)
	}

	return err
}
