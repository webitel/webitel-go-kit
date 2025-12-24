package interceptors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type rpcError struct {
	Id     string `json:"id"`
	Detail string `json:"detail"`
	Code   int32  `json:"code"`
	Status string `json:"status"`
}

func UnaryServerErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		resp, err = handler(ctx, req)
		if err != nil {
			return nil, logAndReturnGRPCError(ctx, err, info)
		}
		return resp, nil
	}
}

func logAndReturnGRPCError(ctx context.Context, err error, info *grpc.UnaryServerInfo) error {
	slog.WarnContext(ctx, fmt.Sprintf("method %s, error: %v", info.FullMethod, err.Error()))

	span := trace.SpanFromContext(ctx)
	span.RecordError(err)

	var (
		grpcCode codes.Code
		httpCode int
		id       string
	)

	slog.ErrorContext(ctx, err.Error())

	switch grpcCode = errors.Code(err); grpcCode {
	case codes.Unauthenticated:
		httpCode = http.StatusUnauthorized
		id = "api.process.unauthenticated"
	case codes.PermissionDenied:
		httpCode = http.StatusForbidden
		id = "api.process.unauthorized"
	case codes.NotFound:
		httpCode = http.StatusNotFound
		id = "api.process.not_found"
	case codes.Aborted, codes.InvalidArgument, codes.AlreadyExists:
		httpCode = http.StatusBadRequest
		id = "api.process.bad_args"
	default:
		httpCode = http.StatusInternalServerError
		id = "api.process.internal"
	}

	grpcErr := &rpcError{
		Id:     id,
		Detail: err.Error(),
		Code:   int32(httpCode),
		Status: http.StatusText(httpCode),
	}

	marshaledErr, _ := json.Marshal(grpcErr)
	return status.Error(grpcCode, string(marshaledErr))
}
