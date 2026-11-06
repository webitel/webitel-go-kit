package auth

import (
	"context"
	"fmt"
	"time"

	authpb "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type exampleClient struct{}

func (exampleClient) UserInfo(context.Context, *authpb.UserinfoRequest, ...grpc.CallOption) (*authpb.Userinfo, error) {
	return &authpb.Userinfo{
		UserId: 7,
		License: []*authpb.LicenseUser{
			{Prod: "AUDIT", ExpiresAt: time.Now().Add(time.Hour).UnixMilli()},
		},
		Scope: []*authpb.Objclass{{Class: "forms", Access: "r", Obac: true, Rbac: true}},
	}, nil
}

// A service with a generated WebitelAPI. infra/auth does not import pkg/interceptors,
// so the server wiring is shown in comments:
//
//	authorizer, err := auth.NewAuthorizer(client, pb.WebitelAPI)
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
	// In a service: New(authgrpc.NewAuthClient(conn)) and the generated pb.WebitelAPI;
	// testAPI has the same shape.
	client := New(exampleClient{})
	defer client.Close()
	authorizer, err := NewAuthorizer(client, testAPI)
	if err != nil {
		panic(err)
	}

	// The interceptor calls Authorize; the method comes from the server context.
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(TokenHeader, "token"))
	ctx = grpc.NewContextWithServerTransportStream(ctx, fakeMethodStream{"/webitel.audit.Forms/Search"})
	ctx, err = authorizer.Authorize(ctx)
	fmt.Println("error:", err)
	handler(ctx)

	// Output:
	// error: <nil>
	// user 7 may read forms: true
}

// A service without WebitelAPI only needs the session: pass client.Authenticate instead:
//
//	interceptors.AuthUnaryServerInterceptor(client.Authenticate)
func ExampleClient_Authenticate() {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(TokenHeader, "token"))
	client := New(exampleClient{})
	defer client.Close()
	ctx, err := client.Authenticate(ctx)
	fmt.Println("error:", err)
	handler(ctx)

	// Output:
	// error: <nil>
	// user 7 may read forms: true
}

// A handler reads the caller from the context. RequiresRBAC(ctx) covers the class
// of the method; s.RequiresRBAC(parentClass, action) is for parent objects.
func ExampleRequiresRBAC() {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(TokenHeader, "token"))
	ctx = grpc.NewContextWithServerTransportStream(ctx, fakeMethodStream{"/webitel.audit.Forms/Search"})
	client := New(exampleClient{})
	defer client.Close()
	authorizer, err := NewAuthorizer(client, testAPI)
	if err != nil {
		panic(err)
	}
	ctx, err = authorizer.Authorize(ctx)
	if err != nil {
		panic(err)
	}
	s, _ := SessionFromContext(ctx)
	if RequiresRBAC(ctx) {
		fmt.Println("restrict records to the roles of", s.ACLRoles())
	}

	// Output:
	// restrict records to the roles of [7]
}

func handler(ctx context.Context) {
	s, _ := SessionFromContext(ctx)
	fmt.Println("user", s.UserID, "may read forms:", s.CanAccess("forms", ActionRead))
}
