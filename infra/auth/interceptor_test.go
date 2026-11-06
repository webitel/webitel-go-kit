package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type fakeAuthorizer struct {
	s      *Session
	err    error
	calls  int
	tokens []string
}

func (f *fakeAuthorizer) Authorize(_ context.Context, token string) (*Session, error) {
	f.calls++
	f.tokens = append(f.tokens, token)

	return f.s, f.err
}

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f fakeStream) Context() context.Context { return f.ctx }

func validSession() *Session {
	now := time.Now()

	return &Session{
		UserID:   7,
		Licenses: []License{{Prod: "AUDIT", IssuedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}},
		Scopes:   map[string]Scope{"forms": {Class: "forms", OBAC: true, Access: "r"}},
	}
}

// call runs one RPC through the unary or stream interceptor and reports the
// session the handler saw (nil when the handler was not reached).
type call func(t *testing.T, i *Interceptor, ctx context.Context, method string) (*Session, error)

var calls = map[string]call{
	"unary": func(t *testing.T, i *Interceptor, ctx context.Context, method string) (*Session, error) {
		var seen *Session
		_, err := i.Unary()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method}, func(ctx context.Context, _ any) (any, error) {
			seen, _ = SessionFromContext(ctx)

			return nil, nil
		})

		return seen, err
	},
	"stream": func(t *testing.T, i *Interceptor, ctx context.Context, method string) (*Session, error) {
		var seen *Session
		err := i.Stream()(nil, fakeStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: method}, func(_ any, ss grpc.ServerStream) error {
			seen, _ = SessionFromContext(ss.Context())

			return nil
		})

		return seen, err
	},
}

func withToken(token string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(TokenHeader, token))
}

func TestInterceptor(t *testing.T) {
	expired := validSession()
	expired.ExpiresAt = time.Now().Add(-time.Minute)
	noLicense := validSession()
	noLicense.Licenses = nil

	tests := []struct {
		name       string
		method     string
		ctx        context.Context
		session    *Session
		authErr    error
		wantCode   codes.Code
		wantID     string
		wantIs     error
		wantAuthed bool // the Authorizer must have been asked
	}{
		{name: "allowed", method: "/webitel.audit.Forms/Search", ctx: withToken("tok"), session: validSession(), wantAuthed: true},
		{name: "health is public", method: "/grpc.health.v1.Health/Check", ctx: context.Background()},
		{name: "reflection is not public", method: "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo", ctx: context.Background(), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "unknown service", method: "/webitel.audit.Nope/Search", ctx: withToken("tok"), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "unknown method", method: "/webitel.audit.Forms/Nope", ctx: withToken("tok"), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "service without class", method: "/webitel.audit.NoClass/Do", ctx: withToken("tok"), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "access out of range", method: "/webitel.audit.Forms/Bad", ctx: withToken("tok"), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "no token", method: "/webitel.audit.Forms/Search", ctx: context.Background(), wantCode: codes.Unauthenticated, wantID: "auth.token.missing", wantIs: ErrTokenMissing},
		{name: "empty token", method: "/webitel.audit.Forms/Search", ctx: withToken(""), wantCode: codes.Unauthenticated, wantID: "auth.token.missing", wantIs: ErrTokenMissing},
		{name: "invalid token", method: "/webitel.audit.Forms/Search", ctx: withToken("tok"), authErr: errors.Unauthenticated("bad", errors.WithID("auth.token.invalid")), wantCode: codes.Unauthenticated, wantID: "auth.token.invalid", wantAuthed: true},
		{name: "auth service down", method: "/webitel.audit.Forms/Search", ctx: withToken("tok"), authErr: errors.Unavailable("down", errors.WithID("auth.provider.unavailable")), wantCode: codes.Unavailable, wantID: "auth.provider.unavailable", wantAuthed: true},
		{name: "authorizer returns no session", method: "/webitel.audit.Forms/Search", ctx: withToken("tok"), wantCode: codes.Unauthenticated, wantID: "auth.session.expired", wantIs: ErrSessionExpired, wantAuthed: true},
		{name: "expired session", method: "/webitel.audit.Forms/Search", ctx: withToken("tok"), session: expired, wantCode: codes.Unauthenticated, wantID: "auth.session.expired", wantIs: ErrSessionExpired, wantAuthed: true},
		{name: "missing license", method: "/webitel.audit.Forms/Search", ctx: withToken("tok"), session: noLicense, wantCode: codes.PermissionDenied, wantID: "auth.license", wantIs: ErrLicense, wantAuthed: true},
		{name: "obac denies create", method: "/webitel.audit.Forms/Create", ctx: withToken("tok"), session: validSession(), wantCode: codes.PermissionDenied, wantID: "auth.permission", wantIs: ErrPermission, wantAuthed: true},
	}

	for kind, run := range calls {
		for _, tt := range tests {
			t.Run(kind+"/"+tt.name, func(t *testing.T) {
				a := &fakeAuthorizer{s: tt.session, err: tt.authErr}
				i, err := NewInterceptor(a, testAPI)
				require.NoError(t, err)

				seen, err := run(t, i, tt.ctx, tt.method)

				assert.Equal(t, tt.wantAuthed, a.calls > 0)
				if tt.wantCode == codes.OK {
					require.NoError(t, err)
					if tt.session != nil {
						assert.Same(t, tt.session, seen, "handler must see the session")
					}

					return
				}
				require.Error(t, err)
				assert.Nil(t, seen, "handler must not run")
				if kind == "stream" {
					// pkg/errors carries no gRPC status, so the stream interceptor sets one.
					assert.Equal(t, tt.wantCode, status.Code(err))

					return
				}
				assert.Equal(t, tt.wantCode, errors.Code(err))
				assert.Equal(t, tt.wantID, errors.ID(err))
				if tt.wantIs != nil {
					assert.ErrorIs(t, err, tt.wantIs)
				}
			})
		}
	}
}

func TestInterceptorPassesToken(t *testing.T) {
	a := &fakeAuthorizer{s: validSession()}
	i, err := NewInterceptor(a, testAPI)
	require.NoError(t, err)

	_, err = calls["unary"](t, i, withToken("secret"), "/webitel.audit.Forms/Search")

	require.NoError(t, err)
	assert.Equal(t, []string{"secret"}, a.tokens)
}

func TestInterceptorListsAllMissingLicenses(t *testing.T) {
	api := genServicesInfo{"Forms": {
		ObjClass:           "forms",
		AdditionalLicenses: []string{"AUDIT", "OTHER"},
		WebitelMethods:     map[string]genMethod{"Search": {Access: 1}},
	}}
	s := validSession()
	s.Licenses = nil
	i, err := NewInterceptor(&fakeAuthorizer{s: s}, api)
	require.NoError(t, err)

	_, err = calls["unary"](t, i, withToken("tok"), "/webitel.audit.Forms/Search")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLicense)
	assert.Equal(t, []string{"AUDIT", "OTHER"}, errors.Value(err, ValueLicenses))
}

func TestInterceptorPermissionCarriesClass(t *testing.T) {
	i, err := NewInterceptor(&fakeAuthorizer{s: validSession()}, testAPI)
	require.NoError(t, err)

	_, err = calls["unary"](t, i, withToken("tok"), "/webitel.audit.Forms/Create")

	require.ErrorIs(t, err, ErrPermission)
	assert.Equal(t, "forms", errors.Value(err, ValueClass))
}

func TestInterceptorPublicMethods(t *testing.T) {
	a := &fakeAuthorizer{}
	i, err := NewInterceptor(a, testAPI, WithPublicMethods("/webitel.audit.Public/"))
	require.NoError(t, err)

	_, err = calls["unary"](t, i, context.Background(), "/webitel.audit.Public/Ping")
	require.NoError(t, err)
	_, err = calls["unary"](t, i, context.Background(), "/grpc.health.v1.Health/Check")
	require.NoError(t, err, "defaults stay public")
	assert.Zero(t, a.calls)
}

func TestInterceptorPublicReflectionOptIn(t *testing.T) {
	a := &fakeAuthorizer{}
	i, err := NewInterceptor(a, testAPI, WithPublicMethods("/grpc.reflection.v1.ServerReflection/"))
	require.NoError(t, err)

	_, err = calls["unary"](t, i, context.Background(), "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo")
	require.NoError(t, err)
	assert.Zero(t, a.calls)
}

func TestInterceptorPublicPrefixNeedsSlash(t *testing.T) {
	// "/" would open every method without a token; the others open siblings or nothing.
	for _, p := range []string{"/webitel.audit.Pub", "/", "//", "", "pkg.Pub/", "/a/b/"} {
		_, err := NewInterceptor(&fakeAuthorizer{}, testAPI, WithPublicMethods(p))
		require.Error(t, err, "prefix %q", p)
		assert.Contains(t, err.Error(), "/pkg.Service/")
	}

	// A sibling service of a correct prefix stays protected.
	a := &fakeAuthorizer{}
	i, err := NewInterceptor(a, testAPI, WithPublicMethods("/webitel.audit.Pub/"))
	require.NoError(t, err)

	_, err = calls["unary"](t, i, context.Background(), "/webitel.audit.PubAdmin/Drop")
	require.Error(t, err)
	assert.Equal(t, "auth.method.unknown", errors.ID(err))
}

// A server may hand back a session that has already expired; the interceptor,
// not the cache, must refuse it.
func TestInterceptorRefusesExpiredSessionFromServer(t *testing.T) {
	info := testInfo()
	info.ExpiresAt = time.Now().Add(-time.Minute).UnixMilli()
	client := &fakeClient{resp: info}
	i, err := NewInterceptor(New(client), testAPI)
	require.NoError(t, err)

	for range 2 {
		_, err = calls["unary"](t, i, withToken("tok"), "/webitel.audit.Forms/Search")

		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, errors.Code(err))
		assert.Equal(t, "auth.session.expired", errors.ID(err))
	}
	assert.Equal(t, 2, client.count(), "an expired session must not be cached")
}

func TestNewInterceptorRejectsBadAPI(t *testing.T) {
	_, err := NewInterceptor(&fakeAuthorizer{}, "nope")
	assert.Error(t, err)

	_, err = NewInterceptor(&fakeAuthorizer{}, genServicesInfo{})
	assert.Error(t, err)
}
