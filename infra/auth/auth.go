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
	"time"

	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

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
}

// New returns a Client that resolves sessions through Auth.UserInfo.
func New(c UserInfoClient, opts ...Option) *Client {
	a := &Client{
		client: c,
		opts:   options{cacheTTL: DefaultCacheTTL, cacheSize: DefaultCacheSize, callTimeout: DefaultCallTimeout},
	}
	for _, o := range opts {
		o(&a.opts)
	}
	a.cache = a.newCache()

	return a
}

// Close releases the cache; the Client must not be used afterwards.
func (a *Client) Close() error {
	if a.cache == nil {
		return nil
	}

	return a.cache.Close()
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
}

// Option configures New.
type Option func(*options)

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
