package depenlog

import (
	"fmt"
	"os"

	"github.com/webitel/webitel-go-kit/pkg/logger"
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

func (g *grpcLogger) Info(args ...any) {
	g.log.Info(fmt.Sprint(args...))
}

func (g *grpcLogger) Infoln(args ...any) {
	g.log.Info(fmt.Sprint(args...))
}

func (g *grpcLogger) Infof(format string, args ...any) {
	g.log.Info(fmt.Sprintf(format, args...))
}

func (g *grpcLogger) Warning(args ...any) {
	g.log.Warn(fmt.Sprint(args...))
}

func (g *grpcLogger) Warningln(args ...any) {
	g.log.Warn(fmt.Sprint(args...))
}

func (g *grpcLogger) Warningf(format string, args ...any) {
	g.log.Warn(fmt.Sprintf(format, args...))
}

func (g *grpcLogger) Error(args ...any) {
	g.log.Error(fmt.Sprint(args...))
}

func (g *grpcLogger) Errorln(args ...any) {
	g.log.Error(fmt.Sprint(args...))
}

func (g *grpcLogger) Errorf(format string, args ...any) {
	g.log.Error(fmt.Sprintf(format, args...))
}

func (g *grpcLogger) Fatal(args ...any) {
	g.log.Error(fmt.Sprint(args...))
	os.Exit(1)
}
func (g *grpcLogger) Fatalln(args ...any) {
	g.log.Error(fmt.Sprint(args...))
	os.Exit(1)
}
func (g *grpcLogger) Fatalf(format string, args ...any) {
	g.log.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// V mirrors grpc-go's default verbosity gate (verbosity 0): level 0 logs pass,
// higher (more verbose) levels are suppressed to avoid flooding.
func (g *grpcLogger) V(level int) bool { return level <= 0 }
