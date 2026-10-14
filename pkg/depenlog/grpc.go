package depenlog

import (
	"fmt"
	"os"

	"github.com/webitel/webitel-go-kit/pkg/logger"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"google.golang.org/grpc/grpclog"
)

// UseGRPC routes grpc-go's framework logs through l, tagged component=grpc, so
// they share the unified schema. grpc-go's global logger is context-free, so
// these records carry no trace_id — per-RPC correlation comes from server
// interceptors, which do have a context.
func UseGRPC(l logger.Logger) {
	grpclog.SetLoggerV2(&grpcLogger{log: WithComponent(l, "grpc")})
}

type grpcLogger struct {
	log logger.Logger
}

var _ grpclog.LoggerV2 = (*grpcLogger)(nil)

type logFunc = func(msg string, args ...any)

func (g *grpcLogger) logArgs(log logFunc, args ...any) {
	log(fmt.Sprint(args...), semconv.GRPCArgsKey, normalizeArgs(args))
}

func (g *grpcLogger) logf(log logFunc, format string, args ...any) {
	log(fmt.Sprintf(format, args...),
		semconv.GRPCFormatKey, format,
		semconv.GRPCArgsKey, normalizeArgs(args))
}

func (g *grpcLogger) fatal() { os.Exit(1) }

func normalizeArgs(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		if err, ok := a.(error); ok {
			out[i] = err.Error()
			continue
		}
		out[i] = a
	}
	return out
}

func (g *grpcLogger) Info(args ...any) {
	g.logArgs(g.log.Info, args...)
}
func (g *grpcLogger) Infoln(args ...any) {
	g.logArgs(g.log.Info, args...)
}
func (g *grpcLogger) Infof(format string, args ...any) {
	g.logf(g.log.Info, format, args...)
}

func (g *grpcLogger) Warning(args ...any) {
	g.logArgs(g.log.Warn, args...)
}
func (g *grpcLogger) Warningln(args ...any) {
	g.logArgs(g.log.Warn, args...)
}
func (g *grpcLogger) Warningf(format string, args ...any) {
	g.logf(g.log.Warn, format, args...)
}

func (g *grpcLogger) Error(args ...any) {
	g.logArgs(g.log.Error, args...)
}
func (g *grpcLogger) Errorln(args ...any) {
	g.logArgs(g.log.Error, args...)
}
func (g *grpcLogger) Errorf(format string, args ...any) {
	g.logf(g.log.Error, format, args...)
}

func (g *grpcLogger) Fatal(args ...any) {
	g.logArgs(g.log.Error, args...)
	g.fatal()
}
func (g *grpcLogger) Fatalln(args ...any) {
	g.logArgs(g.log.Error, args...)
	g.fatal()
}
func (g *grpcLogger) Fatalf(format string, args ...any) {
	g.logf(g.log.Error, format, args...)
	g.fatal()
}

// V mirrors grpc-go's default verbosity gate (verbosity 0): level 0 logs pass,
// higher (more verbose) levels are suppressed to avoid flooding.
func (g *grpcLogger) V(level int) bool { return level <= 0 }
