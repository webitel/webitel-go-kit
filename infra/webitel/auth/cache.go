package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/webitel-go-kit/pkg/cache"
)

// cacheKey hashes the token so raw tokens never sit in the cache.
func cacheKey(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

// load is the cache loader; the cache already detaches ctx from the first caller's cancellation.
func (a *Client) load(ctx context.Context, token string) (*Session, error) {
	ctx, cancel := context.WithTimeout(ctx, a.opts.callTimeout)
	defer cancel()

	// Only the token is forwarded, so the cached result depends on nothing else.
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(TokenHeader, token))

	info, err := a.client.UserInfo(ctx, &authpb.UserinfoRequest{})
	if err != nil {
		return nil, mapError(err)
	}

	return newSession(info), nil
}

// newCache returns nil when the cache is disabled.
func (a *Client) newCache() cache.Cache[string, *Session] {
	if a.opts.cacheTTL <= 0 {
		return nil
	}

	// Cannot fail: size and TTL are positive and there is one layer.
	return cache.New[string, *Session]().
		KeyFunc(cacheKey).
		L1(cache.RistrettoConfig{MaxCost: int64(a.opts.cacheSize), TTL: a.opts.cacheTTL}).
		WithLoader(a.load).
		MustBuild()
}
