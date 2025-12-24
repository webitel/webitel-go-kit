package interceptors

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc"
)

var errPanicReceived = errors.New("panic occurred, please contact our support", errors.WithID("interceptor.panic"))

type Logger interface {
	Error(msg string, args ...any)
}

type panicErr struct {
	panic any
	stack []byte
}

func (e *panicErr) Error() string {
	return fmt.Sprintf("panic caught: %v\n\n%s", e.panic, e.stack)
}

type recoveryHandlerFuncContext func(ctx context.Context, p any) (err error)

func RecoveryUnaryServerInterceptor(log Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = recoverFrom(ctx, r, grpcPanicRecoveryHandler(log))
			}
		}()

		return handler(ctx, req)
	}
}

func recoverFrom(ctx context.Context, p any, r recoveryHandlerFuncContext) error {
	if r != nil {
		return r(ctx, p)
	}

	stack := make([]byte, 64<<10)
	stack = stack[:runtime.Stack(stack, false)]

	return &panicErr{panic: p, stack: stack}
}

func grpcPanicRecoveryHandler(log Logger) func(context.Context, any) error {
	return func(ctx context.Context, p any) (err error) {
		log.Error(fmt.Sprintf("recovered from panic: %s", debug.Stack()))

		return errors.Wrap(errPanicReceived, errors.WithValue("stack", p))
	}
}
