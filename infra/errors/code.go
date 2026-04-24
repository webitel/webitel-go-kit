package errors

import (
	"context"
	"io"
	"net/http"
	"os"

	"google.golang.org/grpc/codes"
)

// map[http]grpc code
var codeMap = map[int32]codes.Code{
	0: codes.Unknown,
	// [200]x
	http.StatusOK: codes.OK,
	// [400]x
	http.StatusBadRequest:         codes.InvalidArgument,
	http.StatusUnauthorized:       codes.Unauthenticated,
	http.StatusForbidden:          codes.PermissionDenied,
	http.StatusNotFound:           codes.NotFound,
	http.StatusMethodNotAllowed:   codes.PermissionDenied,
	http.StatusRequestTimeout:     codes.DeadlineExceeded,
	http.StatusConflict:           codes.AlreadyExists,
	http.StatusPreconditionFailed: codes.FailedPrecondition,
	http.StatusTooManyRequests:    codes.ResourceExhausted,
	// [500]x
	http.StatusInternalServerError:           codes.Internal,
	http.StatusNotImplemented:                codes.Unimplemented,
	http.StatusBadGateway:                    codes.Unavailable,
	http.StatusGatewayTimeout:                codes.DeadlineExceeded,
	http.StatusServiceUnavailable:            codes.Unavailable,
	http.StatusNetworkAuthenticationRequired: codes.Unauthenticated,
	// [system]
	// codes.Canceled,
	// codes.Unknown,
	// codes.Aborted,
	// codes.OutOfRange
	// codes.DataLoss
}

func http2grpcCode(code int32) codes.Code {
	if code < 0 {
		code *= -1 // make positive !
	}
	if grpc, ok := codeMap[code]; ok {
		return grpc
	}
	// switch {
	// case code < 100:
	//   return codes.OK
	// case 100 <= code && code < 200: // Informational
	//   return codes.OK
	// case 200 <= code && code < 300: // Successful
	//   return codes.OK
	// case 300 <= code && code < 400: // Redirection
	//   return codes.OK
	// case 400 <= code && code < 500: // Client-side
	//   return codes.InvalidArgument
	// case 500 <= code: // Server-side
	//   return codes.Internal
	// }
	return codes.Unknown
}

// errorStatusCode converts a standard Go error into its canonical code. Note that
// this is only used to translate the error returned by the server applications.
func errorStatusCode(err error) codes.Code {
	switch err {
	case nil:
		return codes.OK
	case io.EOF:
		return codes.OutOfRange
	case io.ErrClosedPipe, io.ErrNoProgress, io.ErrShortBuffer, io.ErrShortWrite, io.ErrUnexpectedEOF:
		return codes.FailedPrecondition
	case os.ErrInvalid:
		return codes.InvalidArgument
	case context.Canceled:
		return codes.Canceled // Aborted
	case context.DeadlineExceeded:
		return codes.DeadlineExceeded
	}
	switch {
	case os.IsExist(err):
		return codes.AlreadyExists
	case os.IsNotExist(err):
		return codes.NotFound
	case os.IsPermission(err):
		return codes.PermissionDenied
	}
	return codes.Unknown
}
