package ratelimit

import (
	"context"
	"net/http"
	"strconv"

	"github.com/webitel/webitel-go-kit/infra/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// // GrpcAddress [Value]
// func GrpcRemoteIP(ctx context.Context) (ip netip.Addr, ok bool) {
// 	peer, _ := peer.FromContext(ctx)
// 	if peer == nil {
// 		return // netip.Addr{}, false
// 	}
// 	switch addr := peer.Addr.(type) {
// 	case *net.IPAddr:
// 		{
// 			ip, ok = netip.AddrFromSlice(addr.IP)
// 		}
// 	case *net.TCPAddr:
// 		{
// 			ip, ok = netip.AddrFromSlice(addr.IP)
// 		}
// 	case *net.UDPAddr:
// 		{
// 			ip, ok = netip.AddrFromSlice(addr.IP)
// 		}
// 	case *net.UnixAddr:
// 	}
// 	return // ip, ok
// }

func GrpcUnaryServerInterceptor(front Handler) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, args any, info *grpc.UnaryServerInfo, invoke grpc.UnaryHandler) (resp any, err error) {

		if front != nil {

			http, err := http.NewRequestWithContext(
				ctx, http.MethodPost, info.FullMethod,
				http.NoBody,
			)
			// emit.RequestURI =
			// emit.RemoteAddr =

			// emit.TLS = ???
			http.Proto = "HTTP/2.0"
			http.ProtoMajor = 2
			http.ProtoMinor = 0

			head2, _ := metadata.FromIncomingContext(ctx)
			for h, vs := range head2 {
				switch h {
				case ":authority":
					http.Host = vs[0]
					continue // for ..
				}
				for _, v := range vs {
					http.Header.Add(h, v)
				}
			}

			req := NewRequest(
				ctx, func(req *Request) {
					req.Http = http
				},
			)

			status, err := front.LimitRequest(&req)
			if err != nil {
				// Bad Gateway ; [front] error !
				return nil, errors.BadGateway(
					errors.Message(err.Error()),
				)
			}

			// if !status.OK() {}

			ctx, err = GrpcResponse(ctx, status)
			if err != nil {
				// Denied
				return nil, err
			}

		}

		// passthrough ..
		return invoke(ctx, args)
	}
}

func GrpcResponse(ctx context.Context, res Status) (context.Context, error) {

	if res.Limit == 0 {
		// No [RATE_LIMIT] assigned !
		if res.Allowed > 0 {
			// +ALLOW[ed] !
			return ctx, nil
		}
		// DENIED for all !
		return ctx, ErrForbidden
	}

	header := make([]string, 0, (4 * 2)) // kv.. == 2
	headQuota := func(key string, quota int64) {
		if quota == 0 {
			return
		}
		header = append(header, key, strconv.FormatInt(quota, 10))
	}

	headQuota(H2LimitQuota, int64(res.Limit))
	headQuota(H2LimitRemaining, int64(res.Remaining))
	headQuota(H2LimitResetAfter, MinSeconds(res.ResetAfter))

	if !res.OK() {
		headQuota(H2RetryAfter, MinSeconds(res.RetryAfter))
	}

	// populate [X-RateLimit-*] status details
	if len(header) > 0 {
		ctx = metadata.AppendToOutgoingContext(
			ctx, header...,
		)
	}

	// respond
	return ctx, res.Err()
}
