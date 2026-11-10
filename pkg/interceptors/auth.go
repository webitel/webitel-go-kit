package interceptors

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
)

// AuthFunc authenticates a call and returns the context passed on to the handler.
type AuthFunc = auth.AuthFunc

// AuthUnaryServerInterceptor runs fn before each unary handler; to leave public methods open wrap it in
// selector.UnaryServerInterceptor (go-grpc-middleware/v2/interceptors/selector) with selector.MatchFunc.
func AuthUnaryServerInterceptor(fn AuthFunc) grpc.UnaryServerInterceptor {
	return auth.UnaryServerInterceptor(fn)
}

// AuthStreamServerInterceptor runs fn before each stream handler; the status for stream errors is not formed here.
func AuthStreamServerInterceptor(fn AuthFunc) grpc.StreamServerInterceptor {
	return auth.StreamServerInterceptor(fn)
}
