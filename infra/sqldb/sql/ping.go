package sql

import (
	"context"
)

const (
	metricMethodPing = "go.sql.ping"
	traceMethodPing  = "db.Ping"
)

// pingFuncMiddleware is a type for pingFunc middleware.
type pingFuncMiddleware = middleware[pingFunc]

// pingFunc is a callback for pingFunc.
type pingFunc func(ctx context.Context) error

// nopPing pings nothing.
func nopPing(_ context.Context) error {
	return nil
}

// // pingTrace traces ping.
// func pingTrace(t MethodTracer) pingFuncMiddleware {
// 	return func(next pingFunc) pingFunc {
// 		return func(ctx context.Context) (err error) {
// 			ctx, end := t.Trace(ctx, traceMethodPing)
//
// 			defer func() {
// 				end(err)
// 			}()
//
// 			return next(ctx)
// 		}
// 	}
// }

func makePingFuncMiddlewares(t MethodTracer) []pingFuncMiddleware {
	middlewares := make([]pingFuncMiddleware, 0, 2)
	if t != nil {
		// TODO: enable only metrics for ping
		// middlewares = append(middlewares, pingTrace(t))
	}

	return middlewares
}
