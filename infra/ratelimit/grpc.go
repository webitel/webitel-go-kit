package ratelimit

import (
	"context"
	"net/http"
	"strconv"

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
	return func(ctx context.Context, args any, grpc *grpc.UnaryServerInfo, invoke grpc.UnaryHandler) (resp any, err error) {

		http, err := http.NewRequestWithContext(
			ctx, http.MethodPost, grpc.FullMethod,
			http.NoBody,
		)
		// emit.RequestURI =
		// emit.RemoteAddr =

		// emit.TLS = ???
		http.Proto = "HTTP/2.0"
		http.ProtoMajor = 2
		http.ProtoMinor = 0

		head, _ := metadata.FromIncomingContext(ctx)
		for h, vs := range head {
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
			// // Bad Gateway ; [front] error !
			// return nil, errors.BadGateway(
			// 	errors.Message(err.Error()),
			// )
		}

		// if !status.OK() {}

		ctx, err = GrpcResponse(ctx, status)
		if err != nil {
			// Forbidden / Denied !
			return nil, err
		}

		// Not Affected / Allow[ed] / Passthrough ..
		return invoke(ctx, args)
	}
}

func GrpcResponse(ctx context.Context, res *Status) (context.Context, error) {

	if res == nil {
		// Not affected !
		return ctx, nil
	}

	if res.Limit == 0 {
		// No [Limit] applied !
		// Check permitted ?
		if res.Allowed > 0 {
			// +[ALLOWED]
			return ctx, nil
		}
		// -[DENIED] for ALL !
		return ctx, ErrForbidden
	}

	head := make([]string, 0, (4 * 2)) // kv.. == 2
	headQuota := func(key string, quota int64) {
		if quota == 0 {
			return
		}
		head = append(head, key, strconv.FormatInt(quota, 10))
	}

	headQuota(H2LimitQuota, int64(res.Limit))
	// headQuota("X-RateLimit-Allowed", 1) // ~ res.Allowed
	headQuota(H2LimitRemaining, int64(res.Remaining))
	headQuota(H2LimitResetAfter, MinSeconds(res.ResetAfter))

	if !res.OK() {
		headQuota(H2RetryAfter, MinSeconds(res.RetryAfter))
	}

	// populate [X-RateLimit-*] status details
	if len(head) > 0 {
		ctx = metadata.AppendToOutgoingContext(
			ctx, head...,
		)
	}

	// respond
	return ctx, res.Err()
}
