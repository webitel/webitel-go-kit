package grpclog

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/webitel-go-kit/infra/otel/log/bridge"
	"go.opentelemetry.io/otel/log"
	"google.golang.org/grpc/grpclog"
)

type Handler struct {
	*bridge.Handler
}

var _ grpclog.LoggerV2 = (*Handler)(nil)

func NewLoggerV2(opts ...bridge.Option) grpclog.LoggerV2 {
	return &Handler{
		Handler: bridge.NewHandler("grpc", opts...),
	}
}

// Info logs to INFO log. Arguments are handled in the manner of fmt.Print.
func (h *Handler) Info(args ...any) {
	h.emit(log.SeverityInfo, fmt.Sprint(args...))
}

// Infoln logs to INFO log. Arguments are handled in the manner of fmt.Println.
func (h *Handler) Infoln(args ...any) {
	h.emit(log.SeverityInfo, fmt.Sprint(args...))
}

// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func (h *Handler) Infof(format string, args ...any) {
	h.emit(log.SeverityInfo, fmt.Sprintf(format, args...))
}

// Warning logs to WARNING log. Arguments are handled in the manner of fmt.Print.
func (h *Handler) Warning(args ...any) {
	h.emit(log.SeverityWarn, fmt.Sprint(args...))
}

// Warningln logs to WARNING log. Arguments are handled in the manner of fmt.Println.
func (h *Handler) Warningln(args ...any) {
	h.emit(log.SeverityWarn, fmt.Sprint(args...))
}

// Warningf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func (h *Handler) Warningf(format string, args ...any) {
	h.emit(log.SeverityWarn, fmt.Sprintf(format, args...))
}

// Error logs to ERROR log. Arguments are handled in the manner of fmt.Print.
func (h *Handler) Error(args ...any) {
	h.emit(log.SeverityError, fmt.Sprint(args...))
}

// Errorln logs to ERROR log. Arguments are handled in the manner of fmt.Println.
func (h *Handler) Errorln(args ...any) {
	h.emit(log.SeverityError, fmt.Sprint(args...))
}

// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func (h *Handler) Errorf(format string, args ...any) {
	h.emit(log.SeverityError, fmt.Sprintf(format, args...))
}

// Fatal logs to ERROR log. Arguments are handled in the manner of fmt.Print.
// gRPC ensures that all Fatal logs will exit with os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func (h *Handler) Fatal(args ...any) {
	h.emit(log.SeverityFatal, fmt.Sprint(args...))
}

// Fatalln logs to ERROR log. Arguments are handled in the manner of fmt.Println.
// gRPC ensures that all Fatal logs will exit with os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func (h *Handler) Fatalln(args ...any) {
	h.emit(log.SeverityFatal, fmt.Sprint(args...))
}

// Fatalf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
// gRPC ensures that all Fatal logs will exit with os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func (h *Handler) Fatalf(format string, args ...any) {
	h.emit(log.SeverityFatal, fmt.Sprintf(format, args...))
}

const (
	// infoLog indicates Info severity.
	infoLog int = iota
	// warningLog indicates Warning severity.
	warningLog
	// errorLog indicates Error severity.
	errorLog
	// fatalLog indicates Fatal severity.
	fatalLog
)

func severity(level int) log.Severity {
	switch level {
	case infoLog:
		return log.SeverityInfo
	case warningLog:
		return log.SeverityWarn
	case errorLog:
		return log.SeverityError
	case fatalLog:
		return log.SeverityFatal
	}
	return log.SeverityDebug
}

// V reports whether verbosity level l is at least the requested verbose level.
// https://github.com/grpc/grpc-go/blob/v1.65.0/grpclog/loggerv2.go#L79
func (h *Handler) V(level int) bool {
	var test log.Record
	test.SetSeverity(severity(level))
	return h.Logger.Enabled(
		context.Background(), test,
	)
}

func (h *Handler) emit(lvl log.Severity, msg string) {
	var rec log.Record
	rec.SetTimestamp(time.Now())
	rec.SetSeverity(lvl)
	ctx := context.TODO()
	if !h.Logger.Enabled(ctx, rec) {
		return // ignore
	}
	rec.SetBody(log.StringValue(msg))
	h.Logger.Emit(ctx, rec)
}
