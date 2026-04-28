package ratelimit

import (
	// "net"
	"net/http"
	"strconv"

	"github.com/webitel/webitel-go-kit/infra/errors"
	"google.golang.org/protobuf/encoding/protojson"
)

// // HttpAddress [Value]
// func HttpRemoteIP(req *http.Request) (ip netip.Addr, ok bool) {
// 	if req == nil || req.RemoteAddr == "" {
// 		// err = fmt.Errorf("!http.Request.RemoteAddr")
// 		return // netip.Addr{}, false
// 	}
// 	peer, err := netip.ParseAddrPort(req.RemoteAddr)
// 	if ok = (err == nil); ok {
// 		return peer.Addr(), true
// 	}
// 	ip, re := netip.ParseAddr(req.RemoteAddr)
// 	if ok = (re == nil); ok {
// 		return ip, true
// 	}
// 	// invalid: addr:port
// 	return // netip.Addr{}, false
// }

// HTTP middlware
func HttpMiddleware(front Handler, back http.Handler) http.Handler {

	if back == nil {
		// DEFAULT
		back = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// (200) OK
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if front != nil {

			ctx := r.Context()
			req := NewRequest(
				ctx, func(req *Request) {
					req.Http = r
				},
			)

			status, err := front.LimitRequest(&req)

			if err != nil {
				HttpWriteError(w, err)
				return // terminate
			}

			if !HttpWriteStatus(w, status) {
				// Forbidden | Denied
				return // terminate
			}

			// +passthrough
		}

		// [ OK ] invoke ..
		back.ServeHTTP(w, r)
	})
}

func HttpWriteStatus(w http.ResponseWriter, res Status) (ok bool) {

	if res.Limit == 0 {
		// No [RATE_LIMIT] assigned !
		if res.Allowed > 0 {
			// [+ALLOWED]
			return true
		}
		// [-DENIED] for all !
		HttpWriteError(w, ErrForbidden)
		return // false // terminate !
	}

	header := w.Header()
	headQuota := func(key string, value int64) {
		if value == 0 {
			return // undefined
		}
		header.Set(key, strconv.FormatInt(int64(value), 10))
	}
	headQuota(H1LimitQuota, int64(res.Limit))
	// sendHeader("X-RateLimit-Allowed", 1) // ~ res.Allowed
	headQuota(H1LimitRemaining, int64(res.Remaining))
	headQuota(H1LimitResetAfter, MinSeconds(res.ResetAfter))

	if !res.OK() {
		// RateLimit quota exceeded !
		headQuota(H1RetryAfter, MinSeconds(res.RetryAfter))
		HttpWriteError(w, ErrFloodWait)
		return false // terminate
	}

	// +passthrough
	return true
}

func HttpWriteError(w http.ResponseWriter, err error) {

	const mediatype = "application/json; charset=utf-8"
	codec := protojson.MarshalOptions{
		Multiline:         true,
		Indent:            "  ",
		AllowPartial:      true,
		UseProtoNames:     true,
		UseEnumNumbers:    false,
		EmitUnpopulated:   false,
		EmitDefaultValues: false,
		Resolver:          nil,
	}

	res, _ := errors.FromError(err)
	data, err := codec.Marshal(res.Proto())
	if err != nil {
		HttpWriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", mediatype)
	w.WriteHeader(int(res.Code))

	_, _ = w.Write(data)
}
