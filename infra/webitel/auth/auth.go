// Package auth authenticates and authorizes gRPC calls of Webitel services.
//
// The access token (metadata key x-webitel-access) is exchanged for a Session
// through Auth.UserInfo of go.webitel.app. Client.Authenticate only
// authenticates; Authorizer.Authorize also checks the call against the generated
// WebitelAPI map: licenses first, then obac for the object class and action;
// methods missing from the map are denied. Both have the shape of an AuthFunc
// and handlers read the caller with SessionFromContext.
//
// See the package example for wiring.
package auth

import (
	"context"
	"crypto/tls"
	stderrors "errors"
	"fmt"
	"sync"
	"time"

	authgrpc "buf.build/gen/go/webitel/webitel-go/grpc/go/_gogrpc"
	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	resolver "github.com/webitel/webitel-go-kit/infra/transport/gRPC/resolver/discovery"
	"github.com/webitel/webitel-go-kit/pkg/cache"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// TokenHeader carries the access token.
const TokenHeader = "x-webitel-access"

// Defaults of the With* options.
const (
	DefaultCacheTTL    = 10 * time.Second
	DefaultCacheSize   = 10_000
	DefaultCallTimeout = 5 * time.Second
	// DefaultService is how go.webitel.app registers itself in service discovery.
	DefaultService = "go.webitel.app"
)

// UserInfoClient is the part of the generated AuthClient the Client needs.
type UserInfoClient interface {
	UserInfo(ctx context.Context, in *authpb.UserinfoRequest, opts ...grpc.CallOption) (*authpb.Userinfo, error)
}

// Client turns an access token into a Session through Auth.UserInfo, with a cache.
// The owner of the Client must Close it.
type Client struct {
	client UserInfoClient
	opts   options
	// nil when the cache is disabled.
	cache cache.Cache[string, *Session]
	// Set by Dial only; New never closes the caller's connection.
	conn *grpc.ClientConn

	closeOnce sync.Once
	closeErr  error
}

// New returns a Client that resolves sessions through Auth.UserInfo.
func New(c UserInfoClient, opts ...Option) *Client {
	o := defaultOptions()
	for _, f := range opts {
		f(&o)
	}

	return build(c, o)
}

func Dial(_ context.Context, d discovery.Discovery, opts ...DialOption) (*Client, error) {
	o := defaultOptions()
	for _, f := range opts {
		f.apply(&o)
	}
	if d == nil {
		return nil, fmt.Errorf("auth: dial %s: nil discovery", o.service)
	}

	creds := insecure.NewCredentials()
	if o.tls != nil {
		creds = credentials.NewTLS(o.tls)
	}

	// WithInsecure only picks the plain "grpc" endpoint scheme, the only one go.webitel.app registers; TLS comes from creds.
	conn, err := grpc.NewClient("discovery:///"+o.service, append([]grpc.DialOption{
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithTransportCredentials(creds),
		grpc.WithResolvers(resolver.NewBuilder(d, resolver.WithInsecure(true), resolver.PrintDebugLog(false))),
	}, o.dialOpts...)...)
	if err != nil {
		return nil, fmt.Errorf("auth: dial %s: %w", o.service, err)
	}

	c := build(authgrpc.NewAuthClient(conn), o)
	c.conn = conn

	return c, nil
}

// Close releases the connection made by Dial and the cache; it is safe to call more than once.
// The Client must not be used afterwards.
func (a *Client) Close() error {
	a.closeOnce.Do(func() {
		// The connection goes first so that loads still in flight fail instead of feeding a closing cache.
		var errs []error
		if a.conn != nil {
			errs = append(errs, a.conn.Close())
		}
		if a.cache != nil {
			errs = append(errs, a.cache.Close())
		}
		a.closeErr = stderrors.Join(errs...)
	})

	return a.closeErr
}

// Session returns the session of the token, from the cache when possible.
func (a *Client) Session(ctx context.Context, token string) (*Session, error) {
	var (
		s   *Session
		err error
	)
	if a.cache == nil {
		s, err = a.load(ctx, token)
	} else {
		s, _, err = a.cache.Get(ctx, token)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.Wrap(ctx.Err(),
				errors.WithCode(status.FromContextError(ctx.Err()).Code()),
				errors.WithID(idCanceled),
			)
		}

		return nil, err
	}

	return s, nil
}

// Authenticate puts the session of the incoming token into the context; it has the shape of go-grpc-middleware's AuthFunc.
func (a *Client) Authenticate(ctx context.Context) (context.Context, error) {
	token := tokenFromContext(ctx)
	if token == "" {
		return nil, ErrTokenMissing
	}

	s, err := a.Session(ctx, token)
	if err != nil {
		return nil, err
	}
	if s == nil || s.Expired() {
		return nil, ErrSessionExpired
	}

	return ContextWithSession(ctx, s), nil
}

func tokenFromContext(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get(TokenHeader); len(v) > 0 {
		return v[0]
	}

	return ""
}

type options struct {
	cacheTTL    time.Duration
	cacheSize   int
	callTimeout time.Duration

	// Used by Dial only.
	service  string
	tls      *tls.Config
	dialOpts []grpc.DialOption
}

// Option configures New and Dial.
type Option func(*options)

func (f Option) apply(o *options) { f(o) }

// DialOption configures Dial: any Option, WithService, WithTLS or WithDialOptions.
type DialOption interface{ apply(*options) }

type dialOnly func(*options)

func (f dialOnly) apply(o *options) { f(o) }

// WithCacheTTL sets how long a session is reused; a revoked token stays
// effective that long. A non-positive value disables the cache.
func WithCacheTTL(d time.Duration) Option { return func(o *options) { o.cacheTTL = d } }

// WithCacheSize is the ristretto MaxCost of the cache; non-positive keeps the default.
func WithCacheSize(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.cacheSize = n
		}
	}
}

// WithCallTimeout bounds one UserInfo call.
func WithCallTimeout(d time.Duration) Option {
	return func(o *options) {
		if d > 0 {
			o.callTimeout = d
		}
	}
}

// WithService sets the discovery name of the auth service; Dial only.
func WithService(name string) DialOption {
	return dialOnly(func(o *options) {
		if name != "" {
			o.service = name
		}
	})
}

// WithTLS secures the connection of Dial; nil connects without TLS.
func WithTLS(cfg *tls.Config) DialOption { return dialOnly(func(o *options) { o.tls = cfg }) }

func WithDialOptions(opts ...grpc.DialOption) DialOption {
	return dialOnly(func(o *options) { o.dialOpts = append(o.dialOpts, opts...) })
}

func defaultOptions() options {
	return options{cacheTTL: DefaultCacheTTL, cacheSize: DefaultCacheSize, callTimeout: DefaultCallTimeout, service: DefaultService}
}

func build(c UserInfoClient, o options) *Client {
	a := &Client{client: c, opts: o}
	a.cache = a.newCache()

	return a
}
