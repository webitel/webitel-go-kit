package authfx

import (
	"crypto/tls"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"github.com/webitel/webitel-go-kit/infra/webitel/auth"
)

// fakeDiscovery is never called: gRPC builds the resolver on the first RPC, and
// the tests make none.
type fakeDiscovery struct{ discovery.DiscoveryProvider }

type fakeAPI map[string]struct {
	ObjClass       string
	WebitelMethods map[string]struct{ Access int }
}

var api = fakeAPI{"Forms": {ObjClass: "forms", WebitelMethods: map[string]struct{ Access int }{"Search": {Access: 1}}}}

func supplyDiscovery() fx.Option {
	return fx.Provide(func() discovery.DiscoveryProvider { return fakeDiscovery{} })
}

func TestModule_Lifecycle(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "defaults"},
		{name: "custom", cfg: Config{Service: "other.app", CacheSize: 5, CallTimeout: time.Second}},
		{name: "tls", cfg: Config{TLS: &tls.Config{MinVersion: tls.VersionTLS12}}},
		{name: "cache ttl", cfg: Config{CacheTTL: 30 * time.Second}},
		{name: "cache disabled", cfg: Config{DisableCache: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fxtest.New(t, supplyDiscovery(), Module(tt.cfg, api)).RequireStart().RequireStop()
		})
	}
}

func TestModule_ProvidesClientAndAuthorizer(t *testing.T) {
	var (
		c *auth.Client
		a *auth.Authorizer
	)

	app := fxtest.New(t, supplyDiscovery(), Module(Config{}, api), fx.Populate(&c, &a))
	defer app.RequireStart().RequireStop()

	if c == nil || a == nil {
		t.Fatalf("client = %v, authorizer = %v, want both provided", c, a)
	}
}

func TestModule_WithoutWebitelAPI(t *testing.T) {
	var c *auth.Client

	app := fxtest.New(t, supplyDiscovery(), Module(Config{}, nil), fx.Populate(&c))
	defer app.RequireStart().RequireStop()

	if c == nil {
		t.Fatal("client was not provided")
	}

	err := fx.ValidateApp(supplyDiscovery(), Module(Config{}, nil), fx.Invoke(func(*auth.Authorizer) {}))
	if err == nil {
		t.Error("an Authorizer must not be provided without a WebitelAPI")
	}
}

func TestModule_RejectsBadAPI(t *testing.T) {
	if err := fx.ValidateApp(supplyDiscovery(), Module(Config{}, "not a map"), fx.Invoke(func(*auth.Authorizer) {})); err != nil {
		t.Fatalf("graph is wired incorrectly: %v", err)
	}

	app := fx.New(fx.NopLogger, supplyDiscovery(), Module(Config{}, "not a map"))
	if app.Err() == nil {
		t.Error("an invalid WebitelAPI must fail at startup")
	}
}
