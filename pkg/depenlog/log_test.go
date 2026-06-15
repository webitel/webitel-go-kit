package depenlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/logger"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx/fxevent"
)

// newTestLogger builds a logger that writes JSON into buf, bypassing New's
// process-wide side effects (slog.SetDefault, grpc-go global logger).
func newTestLogger(t *testing.T, buf *bytes.Buffer) logger.Logger {
	t.Helper()
	return logger.NewSlog(slog.New(buildPlainHandler(buf, Config{JSON: true})))
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("unmarshal log line %q: %v", buf.String(), err)
	}
	return m
}

// Built-in slog keys must be renamed to the canonical semconv keys, so one
// Loki/ELK query works across services.
func TestSchemaKeys(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(t, &buf)
	l.Info("hello", "k", "v")

	m := decode(t, &buf)
	for _, key := range []string{semconv.TimestampKey, semconv.LevelKey, semconv.MessageKey} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing canonical key %q in %v", key, m)
		}
	}
	if _, ok := m["time"]; ok {
		t.Error("built-in 'time' key should have been renamed to 'date'")
	}
	if _, ok := m["msg"]; ok {
		t.Error("built-in 'msg' key should have been renamed to 'message'")
	}
	if got := m[semconv.MessageKey]; got != "hello" {
		t.Errorf("message = %v, want hello", got)
	}
}

// The common "err" misspelling is normalized to the canonical error key, and
// error values serialize to their message rather than "{}".
func TestErrKeyNormalized(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(t, &buf)
	l.Error("boom", "err", errors.New("bad"))

	m := decode(t, &buf)
	if _, ok := m["err"]; ok {
		t.Errorf("'err' should be renamed to %q", semconv.ErrorKey)
	}
	if got := m[semconv.ErrorKey]; got != "bad" {
		t.Errorf("%s = %v, want bad", semconv.ErrorKey, got)
	}
}

// trace_id/span_id are attached from a context carrying a span, and absent
// otherwise.
func TestTraceInjection(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	var buf bytes.Buffer
	l := newTestLogger(t, &buf)

	l.InfoContext(ctx, "with span")
	m := decode(t, &buf)
	if got := m[semconv.TraceIDKey]; got != traceID.String() {
		t.Errorf("%s = %v, want %s", semconv.TraceIDKey, got, traceID)
	}
	if got := m[semconv.SpanIDKey]; got != spanID.String() {
		t.Errorf("%s = %v, want %s", semconv.SpanIDKey, got, spanID)
	}

	buf.Reset()
	l.InfoContext(context.Background(), "no span")
	m = decode(t, &buf)
	if _, ok := m[semconv.TraceIDKey]; ok {
		t.Errorf("%s should be absent without an active span", semconv.TraceIDKey)
	}
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(t, &buf)
	WithComponent(l, "grpc").Info("x")

	m := decode(t, &buf)
	if got := m[semconv.ComponentKey]; got != "grpc" {
		t.Errorf("%s = %v, want grpc", semconv.ComponentKey, got)
	}
}

// The fx adapter logs failures at error level with the canonical error key and
// component=fx.
func TestFxLoggerError(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(t, &buf)
	FxLogger(l).LogEvent(&fxevent.OnStartExecuted{
		FunctionName: "f", CallerName: "c", Err: errors.New("nope"),
	})

	m := decode(t, &buf)
	if got := m[semconv.LevelKey]; got != "ERROR" {
		t.Errorf("%s = %v, want ERROR", semconv.LevelKey, got)
	}
	if got := m[semconv.ErrorKey]; got != "nope" {
		t.Errorf("%s = %v, want nope", semconv.ErrorKey, got)
	}
	if got := m[semconv.ComponentKey]; got != "fx" {
		t.Errorf("%s = %v, want fx", semconv.ComponentKey, got)
	}
}

// The grpc adapter routes through the unified handler tagged component=grpc.
func TestGRPCLogger(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(t, &buf)
	g := &grpcLogger{log: WithComponent(l, "grpc")}
	g.Error("boom")

	m := decode(t, &buf)
	if got := m[semconv.LevelKey]; got != "ERROR" {
		t.Errorf("%s = %v, want ERROR", semconv.LevelKey, got)
	}
	if got := m[semconv.ComponentKey]; got != "grpc" {
		t.Errorf("%s = %v, want grpc", semconv.ComponentKey, got)
	}
	if g.V(0) != true || g.V(1) != false {
		t.Errorf("V gate: want V(0)=true V(1)=false, got V(0)=%v V(1)=%v", g.V(0), g.V(1))
	}
}
