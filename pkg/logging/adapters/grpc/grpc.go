package grpcadapter

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/grpclog"

	"github.com/webitel/webitel-go-kit/pkg/logging"
)

const Component logging.Component = "grpc"

const componentKey = "grpc.component"

// New returns a [grpclog.DepthLoggerV2] writing to the "grpc" component.
func New(l *logging.Logger) grpclog.DepthLoggerV2 {
	return &logger{log: l.WithComponent(Component)}
}

// Use sets New(l) as the gRPC logger. Call it before any other gRPC
// function, as required by [grpclog.SetLoggerV2].
func Use(l *logging.Logger) {
	grpclog.SetLoggerV2(New(l))
}

type logger struct {
	log *logging.Logger
}

var _ grpclog.DepthLoggerV2 = (*logger)(nil)

func (g *logger) Info(args ...any) {
	g.print(logging.LevelInfo, args)
}

func (g *logger) Infoln(args ...any) {
	g.println(logging.LevelInfo, args)
}

func (g *logger) Infof(format string, args ...any) {
	g.printf(logging.LevelInfo, format, args)
}

func (g *logger) InfoDepth(_ int, args ...any) {
	g.depth(logging.LevelInfo, args)
}

func (g *logger) Warning(args ...any) {
	g.print(logging.LevelWarn, args)
}

func (g *logger) Warningln(args ...any) {
	g.println(logging.LevelWarn, args)
}

func (g *logger) Warningf(format string, args ...any) {
	g.printf(logging.LevelWarn, format, args)
}

func (g *logger) WarningDepth(_ int, args ...any) {
	g.depth(logging.LevelWarn, args)
}

func (g *logger) Error(args ...any) {
	g.print(logging.LevelError, args)
}

func (g *logger) Errorln(args ...any) {
	g.println(logging.LevelError, args)
}

func (g *logger) Errorf(format string, args ...any) {
	g.printf(logging.LevelError, format, args)
}

func (g *logger) ErrorDepth(_ int, args ...any) {
	g.depth(logging.LevelError, args)
}

func (g *logger) Fatal(args ...any) {
	g.print(logging.LevelFatal, args)
}

func (g *logger) Fatalln(args ...any) {
	g.println(logging.LevelFatal, args)
}

func (g *logger) Fatalf(format string, args ...any) {
	g.printf(logging.LevelFatal, format, args)
}

func (g *logger) FatalDepth(_ int, args ...any) {
	g.depth(logging.LevelFatal, args)
}

// V reports whether level INFO-n is enabled.
func (g *logger) V(l int) bool {
	return g.log.Enabled(context.Background(), logging.LevelInfo-logging.Level(l))
}

func (g *logger) print(lvl logging.Level, args []any) {
	ctx := context.Background()
	if !g.log.Enabled(ctx, lvl) {
		return
	}
	g.log.Log(ctx, lvl, fmt.Sprint(args...), errorOf(args))
}

func (g *logger) println(lvl logging.Level, args []any) {
	ctx := context.Background()
	if !g.log.Enabled(ctx, lvl) {
		return
	}
	g.log.Log(ctx, lvl, sprintln(args), errorOf(args))
}

func (g *logger) printf(lvl logging.Level, format string, args []any) {
	ctx := context.Background()
	if !g.log.Enabled(ctx, lvl) {
		return
	}
	g.log.Log(ctx, lvl, fmt.Sprintf(format, args...), errorOf(args))
}

func (g *logger) depth(lvl logging.Level, args []any) {
	ctx := context.Background()
	if !g.log.Enabled(ctx, lvl) {
		return
	}

	var component logging.Field
	if len(args) > 1 {
		if name, ok := componentName(args[0]); ok {
			component = logging.Str(componentKey, name)
			args = args[1:]
		}
	}
	g.log.Log(ctx, lvl, sprintln(args), component, errorOf(args))
}

// componentName truncate leading and trailing brackets: [component] -> component
func componentName(arg any) (string, bool) {
	s, ok := arg.(string)
	if !ok || len(s) < 3 || s[0] != '[' || s[len(s)-1] != ']' {
		return "", false
	}
	return s[1 : len(s)-1], true
}

func sprintln(args []any) string {
	return strings.TrimSuffix(fmt.Sprintln(args...), "\n")
}

func errorOf(args []any) logging.Field {
	for _, arg := range args {
		if err, ok := arg.(error); ok {
			return logging.Err(err)
		}
	}
	return logging.Field{}
}
