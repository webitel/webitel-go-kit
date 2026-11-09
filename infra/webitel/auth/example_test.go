package auth

import (
	"context"
	"fmt"

	authgrpc "buf.build/gen/go/webitel/webitel-go/grpc/go/_gogrpc"
	"google.golang.org/grpc"
)

// Full service wiring; the interceptors come from pkg/interceptors, which this package does not import:
//
//	client := auth.New(authgrpc.NewAuthClient(conn))
//	defer client.Close()
//
//	authorizer, err := auth.NewAuthorizer(client, webitelAPI)
//	if err != nil {
//		return err
//	}
//
//	srv := grpc.NewServer(
//		grpc.ChainUnaryInterceptor(
//			interceptors.UnaryServerErrorInterceptor(),
//			interceptors.AuthUnaryServerInterceptor(authorizer.Authorize),
//		),
//		grpc.ChainStreamInterceptor(interceptors.AuthStreamServerInterceptor(authorizer.Authorize)),
//	)
//
// Public methods such as health checks are left out with go-grpc-middleware's selector.
func ExampleNewAuthorizer() {
	var conn grpc.ClientConnInterface // connection to go.webitel.app
	var webitelAPI any                // the generated pb.WebitelAPI

	client := New(authgrpc.NewAuthClient(conn))
	defer func() { _ = client.Close() }()
	authorizer, err := NewAuthorizer(client, webitelAPI)
	if err != nil {
		panic(err)
	}
	_ = authorizer.Authorize
}

// A service without WebitelAPI passes client.Authenticate to the interceptors instead.
func ExampleClient_Authenticate() {
	var conn grpc.ClientConnInterface // connection to go.webitel.app

	client := New(authgrpc.NewAuthClient(conn))
	defer func() { _ = client.Close() }()
	_ = client.Authenticate
}

// A handler reads the caller from the context; s.RequiresRBAC(parentClass, action) is for parent objects.
func ExampleSessionFromContext() {
	handler := func(ctx context.Context) error {
		s, ok := SessionFromContext(ctx)
		if !ok || !s.CanAccess("forms", ActionRead) {
			return ErrPermission
		}
		if RequiresRBAC(ctx) {
			fmt.Println("restrict records to the roles of", s.ACLRoles())
		}

		return nil
	}
	_ = handler
}
