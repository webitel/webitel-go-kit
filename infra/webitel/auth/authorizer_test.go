package auth

import (
	"context"
	"net"
	"testing"
	"testing/synctest"
	"time"

	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// fakeMethodStream gives a test context the method grpc.Method reads inside a server.
type fakeMethodStream struct{ method string }

func (f fakeMethodStream) Method() string             { return f.method }
func (fakeMethodStream) SetHeader(metadata.MD) error  { return nil }
func (fakeMethodStream) SendHeader(metadata.MD) error { return nil }
func (fakeMethodStream) SetTrailer(metadata.MD) error { return nil }

func serverCtx(ctx context.Context, method string) context.Context {
	return grpc.NewContextWithServerTransportStream(ctx, fakeMethodStream{method})
}

func newTestAuthorizer(t *testing.T, c UserInfoClient, api any) *Authorizer {
	t.Helper()

	a, err := NewAuthorizer(newClient(t, c), api)
	require.NoError(t, err)

	return a
}

func TestAuthorize(t *testing.T) {
	noLicense := testInfo()
	noLicense.License = nil
	noScope := testInfo()
	noScope.Scope = nil
	expired := testInfo()
	expired.ExpiresAt = time.Now().Add(-time.Minute).UnixMilli()

	const search = "/webitel.audit.Forms/Search"
	tests := []struct {
		name       string
		method     string
		ctx        context.Context
		resp       *authpb.Userinfo
		err        error
		wantCode   codes.Code
		wantID     string
		wantIs     error
		wantAuthed bool // go.webitel.app must have been asked
	}{
		{name: "allowed", method: search, ctx: withIncomingToken("tok"), resp: testInfo(), wantAuthed: true},
		{name: "unknown service", method: "/webitel.audit.Nope/Search", ctx: withIncomingToken("tok"), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "unknown method", method: "/webitel.audit.Forms/Nope", ctx: withIncomingToken("tok"), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "service without class", method: "/webitel.audit.NoClass/Do", ctx: withIncomingToken("tok"), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "access out of range", method: "/webitel.audit.Forms/Bad", ctx: withIncomingToken("tok"), wantCode: codes.PermissionDenied, wantID: "auth.method.unknown", wantIs: ErrMethodUnknown},
		{name: "no token", method: search, ctx: context.Background(), wantCode: codes.Unauthenticated, wantID: "auth.token.missing", wantIs: ErrTokenMissing},
		{name: "empty token", method: search, ctx: withIncomingToken(""), wantCode: codes.Unauthenticated, wantID: "auth.token.missing", wantIs: ErrTokenMissing},
		{name: "invalid token", method: search, ctx: withIncomingToken("tok"), err: status.Error(codes.Unauthenticated, "bad"), wantCode: codes.Unauthenticated, wantID: "auth.token.invalid", wantIs: ErrInvalidToken, wantAuthed: true},
		{name: "auth service down", method: search, ctx: withIncomingToken("tok"), err: status.Error(codes.Unavailable, "down"), wantCode: codes.Unavailable, wantID: "auth.provider.unavailable", wantIs: ErrUnavailable, wantAuthed: true},
		{name: "expired session", method: search, ctx: withIncomingToken("tok"), resp: expired, wantCode: codes.Unauthenticated, wantID: "auth.session.expired", wantIs: ErrSessionExpired, wantAuthed: true},
		{name: "missing license", method: search, ctx: withIncomingToken("tok"), resp: noLicense, wantCode: codes.PermissionDenied, wantID: "auth.license", wantIs: ErrLicense, wantAuthed: true},
		{name: "obac denies create", method: "/webitel.audit.Forms/Create", ctx: withIncomingToken("tok"), resp: testInfo(), wantCode: codes.PermissionDenied, wantID: "auth.permission", wantIs: ErrPermission, wantAuthed: true},
		{name: "class not in scope", method: search, ctx: withIncomingToken("tok"), resp: noScope, wantCode: codes.PermissionDenied, wantID: "auth.permission", wantIs: ErrPermission, wantAuthed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeClient{resp: tt.resp, err: tt.err}
			a := newTestAuthorizer(t, f, testAPI)

			ctx, err := a.Authorize(serverCtx(tt.ctx, tt.method))

			assert.Equal(t, tt.wantAuthed, f.count() > 0)
			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				s, ok := SessionFromContext(ctx)
				require.True(t, ok)
				assert.EqualValues(t, 7, s.UserID)

				return
			}
			require.ErrorIs(t, err, tt.wantIs)
			assert.Nil(t, ctx)
			assert.Equal(t, tt.wantCode, errors.Code(err))
			assert.Equal(t, tt.wantID, errors.ID(err))
		})
	}
}

func TestRequiresRBACFromContext(t *testing.T) {
	info := func(rbac bool, perms ...string) *authpb.Userinfo {
		i := testInfo()
		i.Permissions = nil
		for _, p := range perms {
			i.Permissions = append(i.Permissions, &authpb.Permission{Id: p})
		}
		i.Scope = []*authpb.Objclass{{Class: "forms", Access: "rwxd", Obac: true, Rbac: rbac}}

		return i
	}
	tests := []struct {
		name, method string
		resp         *authpb.Userinfo
		want         bool
	}{
		{"rbac on", "Search", info(true), true},
		{"rbac off", "Search", info(false), false},
		{"super read", "Search", info(true, "read"), false},
		{"super write", "Update", info(true, "write"), false},
		{"super add", "Create", info(true, "add"), false},
		{"super delete", "Delete", info(true, "delete"), false},
		{"other super keeps rbac", "Search", info(true, "write"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestAuthorizer(t, &fakeClient{resp: tt.resp}, testAPI)

			ctx, err := a.Authorize(serverCtx(withIncomingToken("tok"), "/webitel.audit.Forms/"+tt.method))

			require.NoError(t, err)
			assert.Equal(t, tt.want, RequiresRBAC(ctx))
		})
	}
}

func TestRequiresRBACFailsSafe(t *testing.T) {
	// Authenticate stores no rule; the session alone is not enough to answer.
	ctx, err := newClient(t, &fakeClient{resp: testInfo()}).Authenticate(withIncomingToken("tok"))
	require.NoError(t, err)
	_, ok := SessionFromContext(ctx)
	require.True(t, ok)
	assert.True(t, RequiresRBAC(ctx), "session without rule")

	assert.True(t, RequiresRBAC(context.Background()), "bare context")
	assert.True(t, RequiresRBAC(contextWithRule(context.Background(), rule{class: "forms"})), "rule without session")
}

func TestAuthorizeKeepsCachedSession(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newClient(t, &fakeClient{resp: testInfo()})
		a, err := NewAuthorizer(c, testAPI)
		require.NoError(t, err)

		cached, err := c.Session(context.Background(), "tok")
		require.NoError(t, err)
		synctest.Wait() // the ristretto cache admits entries asynchronously
		before := *cached

		ctx, err := a.Authorize(serverCtx(withIncomingToken("tok"), "/webitel.audit.Forms/Search"))
		require.NoError(t, err)

		s, _ := SessionFromContext(ctx)
		assert.Same(t, cached, s)
		assert.Equal(t, before, *cached)
	})
}

func TestAuthorizeOutsideServer(t *testing.T) {
	f := &fakeClient{resp: testInfo()}

	_, err := newTestAuthorizer(t, f, testAPI).Authorize(withIncomingToken("tok"))

	assert.ErrorIs(t, err, ErrMethodUnknown)
	assert.Zero(t, f.count())
}

func TestAuthorizePassesToken(t *testing.T) {
	f := &fakeClient{resp: testInfo()}

	_, err := newTestAuthorizer(t, f, testAPI).Authorize(serverCtx(withIncomingToken("secret"), "/webitel.audit.Forms/Search"))

	require.NoError(t, err)
	require.Len(t, f.md, 1)
	assert.Equal(t, []string{"secret"}, f.md[0].Get(TokenHeader))
}

func TestAuthorizeListsAllMissingLicenses(t *testing.T) {
	api := genServicesInfo{"Forms": {
		ObjClass:           "forms",
		AdditionalLicenses: []string{"AUDIT", "OTHER"},
		WebitelMethods:     map[string]genMethod{"Search": {Access: 1}},
	}}
	info := testInfo()
	info.License = nil

	_, err := newTestAuthorizer(t, &fakeClient{resp: info}, api).Authorize(serverCtx(withIncomingToken("tok"), "/webitel.audit.Forms/Search"))

	require.ErrorIs(t, err, ErrLicense)
	assert.Equal(t, []string{"AUDIT", "OTHER"}, errors.Value(err, ValueLicenses))
}

func TestAuthorizePermissionCarriesClass(t *testing.T) {
	_, err := newTestAuthorizer(t, &fakeClient{resp: testInfo()}, testAPI).Authorize(serverCtx(withIncomingToken("tok"), "/webitel.audit.Forms/Create"))

	require.ErrorIs(t, err, ErrPermission)
	assert.Equal(t, "forms", errors.Value(err, ValueClass))
}

// A server may hand back a session that has already expired; it is cached like any other but refused on every call.
func TestAuthorizeRefusesExpiredSessionFromServer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		info := testInfo()
		info.ExpiresAt = time.Now().Add(-time.Minute).UnixMilli()
		f := &fakeClient{resp: info}
		a := newTestAuthorizer(t, f, testAPI)

		for range 2 {
			_, err := a.Authorize(serverCtx(withIncomingToken("tok"), "/webitel.audit.Forms/Search"))

			require.ErrorIs(t, err, ErrSessionExpired)
			assert.Equal(t, "auth.session.expired", errors.ID(err))
			synctest.Wait() // ristretto admits the entry asynchronously
		}
		assert.Equal(t, 1, f.count())
	})
}

func TestNewAuthorizerRejectsBadInput(t *testing.T) {
	_, err := NewAuthorizer(newClient(t, &fakeClient{}), "nope")
	assert.Error(t, err)

	_, err = NewAuthorizer(newClient(t, &fakeClient{}), genServicesInfo{})
	assert.Error(t, err)

	_, err = NewAuthorizer(nil, testAPI)
	assert.Error(t, err)
}

// An interceptor like the one in pkg/interceptors: grpc.Method must work inside a real server.
func TestAuthorizeInRealServer(t *testing.T) {
	a := newTestAuthorizer(t, &fakeClient{resp: testInfo()}, testAPI)

	var seen *Session
	var seenRBAC bool
	desc := grpc.ServiceDesc{
		ServiceName: "webitel.audit.Forms",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "Search",
			Handler: func(_ any, ctx context.Context, dec func(any) error, ic grpc.UnaryServerInterceptor) (any, error) {
				in := new(emptypb.Empty)
				if err := dec(in); err != nil {
					return nil, err
				}
				h := func(ctx context.Context, _ any) (any, error) {
					seen, _ = SessionFromContext(ctx)
					seenRBAC = RequiresRBAC(ctx)

					return new(emptypb.Empty), nil
				}
				if ic == nil {
					return h(ctx, in)
				}

				return ic(ctx, in, &grpc.UnaryServerInfo{FullMethod: "/webitel.audit.Forms/Search"}, h)
			},
		}},
	}

	srv := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		ctx, err := a.Authorize(ctx)
		if err != nil {
			return nil, status.Error(errors.Code(err), err.Error())
		}

		return h(ctx, req)
	}))
	srv.RegisterService(&desc, struct{}{})
	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.DialContext(context.Background()) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	invoke := func(ctx context.Context) error {
		return conn.Invoke(ctx, "/webitel.audit.Forms/Search", new(emptypb.Empty), new(emptypb.Empty))
	}

	err = invoke(context.Background())
	assert.Equal(t, codes.Unauthenticated, status.Code(err), "no token")
	assert.Nil(t, seen)

	require.NoError(t, invoke(metadata.AppendToOutgoingContext(context.Background(), TokenHeader, "tok")))
	require.NotNil(t, seen)
	assert.EqualValues(t, 7, seen.UserID)
	assert.False(t, seenRBAC, "testInfo has the super read permission")
}
