package auth

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	authgrpc "buf.build/gen/go/webitel/webitel-go/grpc/go/_gogrpc"
	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/webitel-go-kit/infra/discovery"
)

type authServer struct {
	authgrpc.UnimplementedAuthServer
	calls atomic.Int32
}

func (s *authServer) UserInfo(ctx context.Context, _ *authpb.UserinfoRequest) (*authpb.Userinfo, error) {
	s.calls.Add(1)
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get(TokenHeader)) == 0 {
		return nil, assert.AnError
	}

	return testInfo(), nil
}

// stubDiscovery serves one instance for one service name and records the names asked for.
type stubDiscovery struct {
	discovery.Discovery
	service  string
	addr     string
	watching chan string
}

func (d *stubDiscovery) GetWatcher(_ context.Context, name string) (discovery.Watcher, error) {
	d.watching <- name
	w := &stubWatcher{stop: make(chan struct{})}
	if name == d.service {
		w.ins = []*discovery.ServiceInstance{{Id: "1", Name: name, Endpoints: []string{"grpc://" + d.addr}}}
	}

	return w, nil
}

type stubWatcher struct {
	ins  []*discovery.ServiceInstance
	sent bool
	stop chan struct{}
}

func (w *stubWatcher) Next() ([]*discovery.ServiceInstance, error) {
	if !w.sent {
		w.sent = true

		return w.ins, nil
	}
	<-w.stop

	return nil, context.Canceled
}

func (w *stubWatcher) Stop() error {
	close(w.stop)

	return nil
}

func startServer(t *testing.T) (*authServer, string) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	s := &authServer{}
	authgrpc.RegisterAuthServer(srv, s)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return s, lis.Addr().String()
}

func session(t *testing.T, c *Client) (*Session, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Session(ctx, "token")
}

func TestDial(t *testing.T) {
	tests := []struct {
		name    string
		opts    []DialOption
		service string // registered in discovery and expected as the target
	}{
		{"custom service", []DialOption{WithService("custom.app")}, "custom.app"},
		{"default service", nil, DefaultService},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, addr := startServer(t)
			d := &stubDiscovery{service: tt.service, addr: addr, watching: make(chan string, 4)}

			c, err := Dial(context.Background(), d, tt.opts...)
			require.NoError(t, err)
			t.Cleanup(func() { _ = c.Close() })

			got, err := session(t, c)
			require.NoError(t, err, "round trip through the discovery resolver")
			assert.Equal(t, int64(3), got.DomainID)
			assert.Equal(t, tt.service, <-d.watching)
		})
	}
}

func TestDialCachesAndCloses(t *testing.T) {
	s, addr := startServer(t)
	d := &stubDiscovery{service: DefaultService, addr: addr, watching: make(chan string, 4)}

	c, err := Dial(context.Background(), d)
	require.NoError(t, err)

	_, err = session(t, c)
	require.NoError(t, err)
	// the ristretto cache admits entries asynchronously: poll until a call is served from it
	require.Eventually(t, func() bool {
		before := s.calls.Load()
		_, err := session(t, c)

		return err == nil && s.calls.Load() == before
	}, 5*time.Second, 10*time.Millisecond, "a repeated call is served from the cache")

	require.NoError(t, c.Close())
	assert.NoError(t, c.Close(), "Close is idempotent")
	_, err = c.client.UserInfo(context.Background(), &authpb.UserinfoRequest{})
	assert.Error(t, err, "Close closes the connection")
}

func TestDialNilDiscovery(t *testing.T) {
	c, err := Dial(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, c)
}

func TestNewDoesNotCloseCallerConn(t *testing.T) {
	s, addr := startServer(t)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	c := New(authgrpc.NewAuthClient(conn))
	require.NoError(t, c.Close())

	_, err = authgrpc.NewAuthClient(conn).UserInfo(
		metadata.AppendToOutgoingContext(context.Background(), TokenHeader, "t"), &authpb.UserinfoRequest{})
	require.NoError(t, err, "the caller's connection stays open")
	assert.EqualValues(t, 1, s.calls.Load())
}
