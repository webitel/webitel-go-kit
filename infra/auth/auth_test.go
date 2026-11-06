package auth

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// fakeClient is a hand-written UserInfoClient. If block is set, UserInfo waits
// for it to close, so tests can hold callers inside the call.
type fakeClient struct {
	mu    sync.Mutex
	calls int
	md    []metadata.MD
	resp  *authpb.Userinfo
	err   error
	block chan struct{}
}

func (f *fakeClient) UserInfo(ctx context.Context, _ *authpb.UserinfoRequest, _ ...grpc.CallOption) (*authpb.Userinfo, error) {
	f.mu.Lock()
	f.calls++
	md, _ := metadata.FromOutgoingContext(ctx)
	f.md = append(f.md, md)
	resp, err, block := f.resp, f.err, f.block
	f.mu.Unlock()

	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return resp, err
}

func (f *fakeClient) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls
}

func testInfo() *authpb.Userinfo {
	return &authpb.Userinfo{
		Dc:          3,
		UserId:      7,
		Name:        "Ann",
		Username:    "ann",
		Roles:       []*authpb.ObjectId{{Id: 10, Name: "admin"}, {Id: 11}},
		Permissions: []*authpb.Permission{{Id: "read"}, {Id: "playback_record_file"}},
		License: []*authpb.LicenseUser{
			{Id: "lic-1", Prod: "AUDIT", IssuedAt: time.Now().Add(-time.Hour).UnixMilli(), ExpiresAt: time.Now().Add(time.Hour).UnixMilli()},
		},
		Scope: []*authpb.Objclass{{Class: "forms", Access: "rw", Obac: true, Rbac: true}},
	}
}

func TestSessionBuildsSession(t *testing.T) {
	exp := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	info := testInfo()
	info.ExpiresAt = exp.UnixMilli()

	s, err := New(&fakeClient{resp: info}).Session(context.Background(), "tok")
	require.NoError(t, err)

	assert.Equal(t, int64(7), s.UserID)
	assert.Equal(t, int64(3), s.DomainID)
	assert.Equal(t, "Ann", s.Name)
	assert.Equal(t, "ann", s.Username)
	assert.Equal(t, []int64{10, 11}, s.RoleIDs)
	assert.Equal(t, []string{"read", "playback_record_file"}, s.Permissions)
	assert.True(t, s.HasLicense("AUDIT"))
	assert.Equal(t, Scope{Class: "forms", Access: "rw", OBAC: true, RBAC: true}, s.Scopes["forms"])
	assert.True(t, exp.Equal(s.ExpiresAt))
	assert.False(t, s.Expired())
}

func TestSessionUnboundedLicense(t *testing.T) {
	info := testInfo()
	info.License = []*authpb.LicenseUser{{Prod: "AUDIT"}}

	s, err := New(&fakeClient{resp: info}).Session(context.Background(), "tok")
	require.NoError(t, err)

	assert.True(t, s.Licenses[0].IssuedAt.IsZero())
	assert.True(t, s.Licenses[0].ExpiresAt.IsZero())
	assert.True(t, s.HasLicense("AUDIT"))
}

func TestSessionNoExpiryWhenZero(t *testing.T) {
	s, err := New(&fakeClient{resp: testInfo()}).Session(context.Background(), "tok")
	require.NoError(t, err)

	assert.True(t, s.ExpiresAt.IsZero())
	assert.False(t, s.Expired())
}

func TestSessionForwardsOnlyToken(t *testing.T) {
	f := &fakeClient{resp: testInfo()}
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-real-ip", "1.2.3.4", TokenHeader, "other"))

	_, err := New(f).Session(ctx, "tok")
	require.NoError(t, err)

	require.Len(t, f.md, 1)
	assert.Equal(t, metadata.Pairs(TokenHeader, "tok"), f.md[0])
}

func TestSessionSingleflight(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeClient{resp: testInfo(), block: make(chan struct{})}
		a := New(f)

		var wg sync.WaitGroup
		var failed atomic.Int32
		for range 50 {
			wg.Go(func() {
				if _, err := a.Session(context.Background(), "tok"); err != nil {
					failed.Add(1)
				}
			})
		}
		synctest.Wait()
		close(f.block)
		wg.Wait()

		assert.Equal(t, 1, f.count())
		assert.Zero(t, failed.Load())
	})
}

func TestSessionFirstCallerCancelDoesNotBreakOthers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeClient{resp: testInfo(), block: make(chan struct{})}
		a := New(f)

		firstCtx, cancel := context.WithCancel(context.Background())
		var firstErr, secondErr error
		var wg sync.WaitGroup
		wg.Go(func() { _, firstErr = a.Session(firstCtx, "tok") })
		synctest.Wait() // the first caller started the flight
		wg.Go(func() { _, secondErr = a.Session(context.Background(), "tok") })
		synctest.Wait() // the second caller joined it

		cancel()
		synctest.Wait()
		close(f.block)
		wg.Wait()

		require.Error(t, firstErr)
		assert.Equal(t, codes.Canceled, errors.Code(firstErr))
		assert.NoError(t, secondErr)
		assert.Equal(t, 1, f.count())
	})
}

func TestCacheTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeClient{resp: testInfo()}
		a := New(f, WithCacheTTL(10*time.Second))
		ctx := context.Background()

		_, err := a.Session(ctx, "tok")
		require.NoError(t, err)
		time.Sleep(9 * time.Second)
		_, err = a.Session(ctx, "tok")
		require.NoError(t, err)
		assert.Equal(t, 1, f.count(), "served from cache inside the TTL")

		time.Sleep(2 * time.Second)
		_, err = a.Session(ctx, "tok")
		require.NoError(t, err)
		assert.Equal(t, 2, f.count(), "refetched after the TTL")
	})
}

func TestCacheLimitedByTokenExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		info := testInfo()
		info.ExpiresAt = time.Now().Add(3 * time.Second).UnixMilli()
		f := &fakeClient{resp: info}
		a := New(f, WithCacheTTL(time.Minute))
		ctx := context.Background()

		_, err := a.Session(ctx, "tok")
		require.NoError(t, err)
		time.Sleep(4 * time.Second)
		_, err = a.Session(ctx, "tok")
		require.NoError(t, err)

		assert.Equal(t, 2, f.count())
	})
}

func TestCacheDisabled(t *testing.T) {
	f := &fakeClient{resp: testInfo()}
	a := New(f, WithCacheTTL(0))

	for range 3 {
		_, err := a.Session(context.Background(), "tok")
		require.NoError(t, err)
	}

	assert.Equal(t, 3, f.count())
}

func TestCacheIsBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeClient{resp: testInfo()}
		a := New(f, WithCacheSize(2))

		for i := range 5 {
			_, err := a.Session(context.Background(), fmt.Sprint("tok", i))
			require.NoError(t, err)
		}
		assert.Len(t, a.cache, 2)

		time.Sleep(DefaultCacheTTL + time.Second)
		_, err := a.Session(context.Background(), "fresh")
		require.NoError(t, err)
		assert.Len(t, a.cache, 1, "expired entries make room")
	})
}

func TestErrorsAreNotCached(t *testing.T) {
	f := &fakeClient{err: status.Error(codes.Unavailable, "down")}
	a := New(f)

	_, err := a.Session(context.Background(), "tok")
	require.Error(t, err)

	f.mu.Lock()
	f.err, f.resp = nil, testInfo()
	f.mu.Unlock()

	_, err = a.Session(context.Background(), "tok")
	require.NoError(t, err)
	assert.Equal(t, 2, f.count())
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
		id   string
	}{
		{"unauthenticated", status.Error(codes.Unauthenticated, "x"), codes.Unauthenticated, "auth.token.invalid"},
		{"permission denied", status.Error(codes.PermissionDenied, "x"), codes.Unauthenticated, "auth.token.invalid"},
		{"not found", status.Error(codes.NotFound, "x"), codes.Unauthenticated, "auth.token.invalid"},
		{"unavailable", status.Error(codes.Unavailable, "x"), codes.Unavailable, "auth.provider.unavailable"},
		{"deadline", status.Error(codes.DeadlineExceeded, "x"), codes.Unavailable, "auth.provider.unavailable"},
		{"internal", status.Error(codes.Internal, "x"), codes.Unavailable, "auth.provider.unavailable"},
		{"plain error", fmt.Errorf("boom"), codes.Unavailable, "auth.provider.unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(&fakeClient{err: tt.err}).Session(context.Background(), "tok")

			require.Error(t, err)
			assert.Equal(t, tt.code, errors.Code(err))
			assert.Equal(t, tt.id, errors.ID(err))
			if tt.code == codes.Unavailable {
				assert.ErrorIs(t, err, ErrUnavailable)
			} else {
				assert.ErrorIs(t, err, ErrInvalidToken)
			}
		})
	}
}

func TestCallTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeClient{block: make(chan struct{})}

		_, err := New(f, WithCallTimeout(time.Second)).Session(context.Background(), "tok")

		require.Error(t, err)
		assert.Equal(t, codes.Unavailable, errors.Code(err))
	})
}

func withIncomingToken(tok string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(TokenHeader, tok))
}

func TestAuthenticate(t *testing.T) {
	t.Run("puts the session into the context and caches it", func(t *testing.T) {
		f := &fakeClient{resp: testInfo()}
		c := New(f)
		ctx := withIncomingToken("tok")

		got, err := c.Authenticate(ctx)
		require.NoError(t, err)
		want, err := c.Session(ctx, "tok")
		require.NoError(t, err)
		s, ok := SessionFromContext(got)
		require.True(t, ok)
		assert.Same(t, want, s)

		_, err = c.Authenticate(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, f.count(), "second call served from the cache")
	})

	t.Run("token missing", func(t *testing.T) {
		c := New(&fakeClient{resp: testInfo()})

		_, err := c.Authenticate(context.Background())
		assert.ErrorIs(t, err, ErrTokenMissing)
		_, err = c.Authenticate(withIncomingToken(""))
		assert.ErrorIs(t, err, ErrTokenMissing)
	})

	t.Run("UserInfo errors keep their sentinel", func(t *testing.T) {
		_, err := New(&fakeClient{err: status.Error(codes.Unauthenticated, "bad")}).Authenticate(withIncomingToken("tok"))
		assert.ErrorIs(t, err, ErrInvalidToken)
		_, err = New(&fakeClient{err: status.Error(codes.Unavailable, "down")}).Authenticate(withIncomingToken("tok"))
		assert.ErrorIs(t, err, ErrUnavailable)
	})

	t.Run("expired session", func(t *testing.T) {
		info := testInfo()
		info.ExpiresAt = time.Now().Add(-time.Minute).UnixMilli()
		f := &fakeClient{resp: info}
		c := New(f)

		for range 2 {
			_, err := c.Authenticate(withIncomingToken("tok"))
			assert.ErrorIs(t, err, ErrSessionExpired)
		}
		assert.Equal(t, 2, f.count(), "an expired session is not cached")
	})
}
