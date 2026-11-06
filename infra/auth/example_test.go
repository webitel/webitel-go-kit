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
		Scope: []*authpb.Objclass{{Class: "forms", Access: "r", Obac: true}},
	}, nil
}

// Wire the interceptors into a server and read the caller in a handler.
func Example() {
	// In a service: New(authgrpc.NewAuthClient(conn)) and the generated
	// pb.WebitelAPI; testAPI has the same shape. Put UnaryServerErrorInterceptor
	// of pkg/interceptors before Unary() so clients get gRPC status codes.
	interceptor, err := NewInterceptor(New(exampleClient{}), testAPI)
	if err != nil {
		panic(err)
	}

	_ = grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor.Unary()),
		grpc.ChainStreamInterceptor(interceptor.Stream()),
	)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(TokenHeader, "token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/webitel.audit.Forms/Search"}
	_, err = interceptor.Unary()(ctx, nil, info, func(ctx context.Context, _ any) (any, error) {
		s, _ := SessionFromContext(ctx)
		fmt.Println("user", s.UserID, "may read forms:", s.CanAccess("forms", ActionRead))

		return nil, nil
	})
	fmt.Println("error:", err)

	// Output:
	// user 7 may read forms: true
	// error: <nil>
}
