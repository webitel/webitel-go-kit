package interceptors

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ctxKey struct{}

// handlers report back the ctx value they see, via the grpc status message.
var testDesc = grpc.ServiceDesc{
	ServiceName: "t.S",
	HandlerType: (*any)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Open", Handler: unaryHandler("/t.S/Open")},
		{MethodName: "Closed", Handler: unaryHandler("/t.S/Closed")},
	},
	Streams: []grpc.StreamDesc{{StreamName: "Watch", ServerStreams: true, Handler: func(_ any, ss grpc.ServerStream) error {
		return status.Error(codes.AlreadyExists, ss.Context().Value(ctxKey{}).(string))
	}}},
}

func unaryHandler(full string) func(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error) {
	return func(srv any, ctx context.Context, dec func(any) error, ic grpc.UnaryServerInterceptor) (any, error) {
		call := func(ctx context.Context, _ any) (any, error) {
			v, _ := ctx.Value(ctxKey{}).(string)
			return nil, status.Error(codes.AlreadyExists, v)
		}
		if ic == nil {
			return call(ctx, nil)
		}
		return ic(ctx, &emptypb.Empty{}, &grpc.UnaryServerInfo{Server: srv, FullMethod: full}, call)
	}
}

func dial(t *testing.T, opts ...grpc.ServerOption) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(opts...)
	srv.RegisterService(&testDesc, struct{}{})
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func authWithValue(ctx context.Context) (context.Context, error) {
	return context.WithValue(ctx, ctxKey{}, "authed"), nil
}

func authFails(context.Context) (context.Context, error) {
	return nil, errors.Unauthenticated("no token")
}

func watch(conn *grpc.ClientConn) error {
	s, err := conn.NewStream(context.Background(), &grpc.StreamDesc{ServerStreams: true}, "/t.S/Watch")
	if err != nil {
		return err
	}
	if err = s.SendMsg(&emptypb.Empty{}); err != nil {
		return err
	}
	_ = s.CloseSend()
	return s.RecvMsg(&emptypb.Empty{})
}

func invokeClosed(conn *grpc.ClientConn) error {
	return conn.Invoke(context.Background(), "/t.S/Closed", &emptypb.Empty{}, &emptypb.Empty{})
}

func TestAuth(t *testing.T) {
	tests := []struct {
		name     string
		opts     []grpc.ServerOption
		call     func(*grpc.ClientConn) error
		wantCode codes.Code
		wantMsg  string // the handler reports the ctx value it saw; empty when the handler must not run
	}{
		{
			name:     "unary passes the AuthFunc ctx to the handler",
			opts:     []grpc.ServerOption{grpc.ChainUnaryInterceptor(AuthUnaryServerInterceptor(authWithValue))},
			call:     invokeClosed,
			wantCode: codes.AlreadyExists,
			wantMsg:  "authed",
		},
		{
			name:     "unary error stops the handler",
			opts:     []grpc.ServerOption{grpc.ChainUnaryInterceptor(UnaryServerErrorInterceptor(), AuthUnaryServerInterceptor(authFails))},
			call:     invokeClosed,
			wantCode: codes.Unauthenticated,
		},
		{
			// ss.Context() carries the value, so the library replaces the stream ctx itself.
			name:     "stream passes the AuthFunc ctx to the handler",
			opts:     []grpc.ServerOption{grpc.ChainStreamInterceptor(AuthStreamServerInterceptor(authWithValue))},
			call:     watch,
			wantCode: codes.AlreadyExists,
			wantMsg:  "authed",
		},
		{
			// Known behaviour until WTEL-10860: the pkg/errors error is not mapped for streams.
			name:     "stream error reaches the client as Unknown",
			opts:     []grpc.ServerOption{grpc.ChainStreamInterceptor(AuthStreamServerInterceptor(authFails))},
			call:     watch,
			wantCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(dial(t, tt.opts...))
			if status.Code(err) != tt.wantCode {
				t.Fatalf("got %v, want code %v", err, tt.wantCode)
			}
			if tt.wantMsg != "" && status.Convert(err).Message() != tt.wantMsg {
				t.Fatalf("got message %q, want %q", status.Convert(err).Message(), tt.wantMsg)
			}
		})
	}
}

func TestAuthUnarySelector(t *testing.T) {
	needsAuth := selector.MatchFunc(func(_ context.Context, c interceptors.CallMeta) bool {
		return !strings.HasPrefix(c.FullMethod(), "/t.S/Open")
	})
	deny := func(context.Context) (context.Context, error) { return nil, status.Error(codes.Unauthenticated, "") }
	conn := dial(t, grpc.ChainUnaryInterceptor(selector.UnaryServerInterceptor(AuthUnaryServerInterceptor(deny), needsAuth)))
	ctx := context.Background()
	if err := conn.Invoke(ctx, "/t.S/Open", &emptypb.Empty{}, &emptypb.Empty{}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("open method: got %v", err)
	}
	if err := conn.Invoke(ctx, "/t.S/Closed", &emptypb.Empty{}, &emptypb.Empty{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("closed method: got %v", err)
	}
}
