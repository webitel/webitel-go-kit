// Package auth authenticates and authorizes gRPC calls of Webitel services.
//
// The access token (metadata key x-webitel-access) is exchanged for a Session
// through Auth.UserInfo of go.webitel.app. The interceptor checks the call
// against the generated WebitelAPI map: licenses first, then obac for the
// object class and action; methods missing from the map are denied. Handlers
// read the caller with SessionFromContext.
//
// See the package example for wiring.
package auth

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

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

// UserInfoClient is the part of the generated AuthClient the Authorizer needs.
type UserInfoClient interface {
	UserInfo(ctx context.Context, in *authpb.UserinfoRequest, opts ...grpc.CallOption) (*authpb.Userinfo, error)
}

// Authorizer turns an access token into a Session.
type Authorizer interface {
	Authorize(ctx context.Context, token string) (*Session, error)
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

// WithCacheSize bounds cached sessions; non-positive keeps the default.
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

type cacheEntry struct {
	session *Session
	until   time.Time
}

type authorizer struct {
	client UserInfoClient
	opts   options
	group  singleflight.Group

	mu    sync.Mutex
	cache map[[sha256.Size]byte]cacheEntry
}

// New returns an Authorizer that resolves sessions through Auth.UserInfo.
func New(c UserInfoClient, opts ...Option) Authorizer {
	a := &authorizer{
		client: c,
		opts:   options{cacheTTL: DefaultCacheTTL, cacheSize: DefaultCacheSize, callTimeout: DefaultCallTimeout},
		cache:  make(map[[sha256.Size]byte]cacheEntry),
	}
	for _, o := range opts {
		o(&a.opts)
	}

	return a
}

func (a *authorizer) Authorize(ctx context.Context, token string) (*Session, error) {
	// Hashed so raw tokens never sit in the map keys.
	key := sha256.Sum256([]byte(token))
	if s := a.cached(key); s != nil {
		return s, nil
	}

	ch := a.group.DoChan(string(key[:]), func() (any, error) { return a.fetch(ctx, key, token) })
	select {
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}

		return res.Val.(*Session), nil
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(),
			errors.WithCode(status.FromContextError(ctx.Err()).Code()),
			errors.WithID("auth.request.canceled"),
		)
	}
}

// fetch must not die with the context of the caller that started the flight.
func (a *authorizer) fetch(ctx context.Context, key [sha256.Size]byte, token string) (any, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.opts.callTimeout)
	defer cancel()

	// Only the token is forwarded, so the cached result depends on nothing else.
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(TokenHeader, token))

	info, err := a.client.UserInfo(ctx, &authpb.UserinfoRequest{})
	if err != nil {
		return nil, mapError(err)
	}

	s := newSession(info)
	a.store(key, s)

	return s, nil
}

func (a *authorizer) cached(key [sha256.Size]byte) *Session {
	a.mu.Lock()
	defer a.mu.Unlock()

	e, ok := a.cache[key]
	if !ok {
		return nil
	}
	if !time.Now().Before(e.until) {
		delete(a.cache, key)

		return nil
	}

	return e.session
}

func (a *authorizer) store(key [sha256.Size]byte, s *Session) {
	if a.opts.cacheTTL <= 0 {
		return
	}

	now := time.Now()
	until := now.Add(a.opts.cacheTTL)
	if !s.ExpiresAt.IsZero() && s.ExpiresAt.Before(until) {
		until = s.ExpiresAt
	}
	if !until.After(now) {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, replace := a.cache[key]; !replace && len(a.cache) >= a.opts.cacheSize {
		for k, e := range a.cache {
			if !now.Before(e.until) {
				delete(a.cache, k)
			}
		}
		// shortcut: a full cache stops caching new tokens until entries expire, upgrade to LRU if the limit is hit.
		if len(a.cache) >= a.opts.cacheSize {
			return
		}
	}
	a.cache[key] = cacheEntry{session: s, until: until}
}

// mapError keeps an outage of the auth service from looking like a bad token.
func mapError(err error) error {
	switch status.Code(err) {
	case codes.Unauthenticated, codes.PermissionDenied, codes.NotFound:
		return errors.Wrap(ErrInvalidToken, errors.WithCause(err))
	default:
		return errors.Wrap(ErrUnavailable, errors.WithCause(err))
	}
}

// millis converts milliseconds; 0 stays the zero time, which means no bound.
func millis(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}

	return time.UnixMilli(ms)
}

func newSession(info *authpb.Userinfo) *Session {
	s := &Session{
		UserID:   info.GetUserId(),
		DomainID: info.GetDc(),
		Name:     info.GetName(),
		Username: info.GetUsername(),
		Scopes:   make(map[string]Scope, len(info.GetScope())),
	}
	s.ExpiresAt = millis(info.GetExpiresAt())
	for _, r := range info.GetRoles() {
		s.RoleIDs = append(s.RoleIDs, r.GetId())
	}
	for _, p := range info.GetPermissions() {
		s.Permissions = append(s.Permissions, p.GetId())
	}
	for _, l := range info.GetLicense() {
		s.Licenses = append(s.Licenses, License{
			Prod:      l.GetProd(),
			IssuedAt:  millis(l.GetIssuedAt()),
			ExpiresAt: millis(l.GetExpiresAt()),
		})
	}
	for _, sc := range info.GetScope() {
		s.Scopes[sc.GetClass()] = Scope{
			Class:  sc.GetClass(),
			Access: sc.GetAccess(),
			OBAC:   sc.GetObac(),
			RBAC:   sc.GetRbac(),
		}
	}

	return s
}
