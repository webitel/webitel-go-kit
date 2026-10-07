package rabbitmq

import (
	"context"
	"maps"
	"reflect"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	testTraceID     = "4bf92f3577b34da6a3ce929d0e0e4736"
	testSpanID      = "00f067aa0ba902b7"
	testTraceparent = "00-" + testTraceID + "-" + testSpanID + "-01"
	testBaggage     = "webitel.domain.id=7"
)

func TestHeaderCarrierGet(t *testing.T) {
	carrier := HeaderCarrier{
		"string": "value",
		"bytes":  []byte("value"),
		"int":    int32(7),
		"nil":    nil,
	}

	tests := []struct {
		key  string
		want string
	}{
		{key: "string", want: "value"},
		{key: "bytes", want: "value"},
		{key: "int", want: ""},
		{key: "nil", want: ""},
		{key: "missing", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			require.Equal(t, tt.want, carrier.Get(tt.key))
		})
	}

	require.Empty(t, HeaderCarrier(nil).Get("missing"))
}

func TestHeaderCarrierSetKeys(t *testing.T) {
	carrier := HeaderCarrier{"a": int32(1)}
	carrier.Set("b", "2")
	carrier.Set("a", "1")

	require.Equal(t, "1", carrier.Get("a"))
	require.Equal(t, "2", carrier.Get("b"))
	require.ElementsMatch(t, []string{"a", "b"}, carrier.Keys())
	require.Empty(t, HeaderCarrier(nil).Keys())
}

func TestInjectHeadersNoopPropagator(t *testing.T) {
	setPropagator(t, propagation.NewCompositeTextMapPropagator())

	headers := amqp.Table{"a": "b"}
	got := injectHeaders(testContext(t), headers)

	require.True(t, sameTable(headers, got))
	require.Equal(t, amqp.Table{"a": "b"}, got)
	require.Nil(t, injectHeaders(testContext(t), nil))
}

func TestInjectHeaders(t *testing.T) {
	setW3CPropagator(t)

	headers := amqp.Table{"a": "b"}
	got := injectHeaders(testContext(t), headers)

	require.False(t, sameTable(headers, got))
	require.Equal(t, amqp.Table{"a": "b"}, headers)
	require.Equal(t, amqp.Table{
		"a":           "b",
		"traceparent": testTraceparent,
		"baggage":     testBaggage,
	}, got)
}

func TestInjectHeadersNil(t *testing.T) {
	setW3CPropagator(t)

	got := injectHeaders(testContext(t), nil)

	require.Equal(t, amqp.Table{
		"traceparent": testTraceparent,
		"baggage":     testBaggage,
	}, got)
}

func TestInjectHeadersEmptyContext(t *testing.T) {
	setW3CPropagator(t)

	headers := amqp.Table{"a": "b"}
	got := injectHeaders(context.Background(), headers)

	require.Equal(t, amqp.Table{"a": "b"}, got)
	require.Equal(t, amqp.Table{"a": "b"}, headers)
}

func TestInjectHeadersReplacesFields(t *testing.T) {
	setW3CPropagator(t)

	headers := amqp.Table{
		"a":           "b",
		"traceparent": "00-11111111111111111111111111111111-2222222222222222-01",
		"tracestate":  "vendor=value",
		"baggage":     []byte("webitel.domain.id=9"),
	}
	want := maps.Clone(headers)

	got := injectHeaders(testContext(t), headers)

	require.Equal(t, want, headers)
	require.Equal(t, amqp.Table{
		"a":           "b",
		"traceparent": testTraceparent,
		"baggage":     testBaggage,
	}, got)
}

func TestPublishParentWithoutFields(t *testing.T) {
	setW3CPropagator(t)

	ctx := testContext(t)

	require.Equal(t, ctx, publishParent(ctx, nil))
	require.Equal(t, ctx, publishParent(ctx, amqp.Table{"a": "b"}))
}

func TestPublishParentFromHeaders(t *testing.T) {
	setW3CPropagator(t)

	const (
		otherTraceID = "11111111111111111111111111111111"
		otherSpanID  = "2222222222222222"
	)

	tests := []struct {
		name      string
		headers   amqp.Table
		wantTrace string
		wantSpan  string
		wantBag   string
	}{
		{
			name:      "traceparent",
			headers:   amqp.Table{"traceparent": "00-" + otherTraceID + "-" + otherSpanID + "-01"},
			wantTrace: otherTraceID,
			wantSpan:  otherSpanID,
		},
		{
			name:    "baggage",
			headers: amqp.Table{"baggage": []byte("webitel.domain.id=9")},
			wantBag: "9",
		},
		{
			name:      "both",
			headers:   amqp.Table{"traceparent": "00-" + otherTraceID + "-" + otherSpanID + "-01", "baggage": "webitel.domain.id=9"},
			wantTrace: otherTraceID,
			wantSpan:  otherSpanID,
			wantBag:   "9",
		},
		{
			name:    "empty value",
			headers: amqp.Table{"traceparent": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, cancel := context.WithCancel(testContext(t))
			defer cancel()

			ctx := publishParent(parent, tt.headers)

			sc := trace.SpanContextFromContext(ctx)
			if tt.wantTrace == "" {
				require.False(t, sc.IsValid())
			} else {
				require.True(t, sc.IsRemote())
				require.Equal(t, tt.wantTrace, sc.TraceID().String())
				require.Equal(t, tt.wantSpan, sc.SpanID().String())
			}
			require.Equal(t, tt.wantBag, baggage.FromContext(ctx).Member("webitel.domain.id").Value())

			cancel()
			require.ErrorIs(t, ctx.Err(), context.Canceled)
		})
	}
}

func TestExtractHeaders(t *testing.T) {
	setW3CPropagator(t)

	tests := []struct {
		name    string
		headers amqp.Table
	}{
		{name: "string", headers: amqp.Table{"traceparent": testTraceparent, "baggage": testBaggage}},
		{name: "bytes", headers: amqp.Table{"traceparent": []byte(testTraceparent), "baggage": []byte(testBaggage)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := extractHeaders(context.Background(), tt.headers)

			sc := trace.SpanContextFromContext(ctx)
			require.True(t, sc.IsValid())
			require.True(t, sc.IsRemote())
			require.True(t, sc.IsSampled())
			require.Equal(t, testTraceID, sc.TraceID().String())
			require.Equal(t, testSpanID, sc.SpanID().String())
			require.Equal(t, "7", baggage.FromContext(ctx).Member("webitel.domain.id").Value())
		})
	}
}

func TestExtractHeadersEmpty(t *testing.T) {
	setW3CPropagator(t)

	for _, headers := range []amqp.Table{nil, {}, {"traceparent": int32(1)}} {
		ctx := extractHeaders(context.Background(), headers)

		require.False(t, trace.SpanContextFromContext(ctx).IsValid())
		require.Zero(t, baggage.FromContext(ctx).Len())
	}
}

func TestPropagationRoundTrip(t *testing.T) {
	setW3CPropagator(t)

	src := testContext(t)
	ctx := extractHeaders(context.Background(), injectHeaders(src, nil))

	require.Equal(t, trace.SpanContextFromContext(src).WithRemote(true), trace.SpanContextFromContext(ctx))
	require.Equal(t, baggage.FromContext(src).String(), baggage.FromContext(ctx).String())
}

func setPropagator(t *testing.T, p propagation.TextMapPropagator) {
	t.Helper()

	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(p)
	t.Cleanup(func() {
		otel.SetTextMapPropagator(prev)
	})
}

func setW3CPropagator(t *testing.T) {
	t.Helper()

	setPropagator(t, propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

func testContext(t *testing.T) context.Context {
	t.Helper()

	traceID, err := trace.TraceIDFromHex(testTraceID)
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex(testSpanID)
	require.NoError(t, err)

	bag, err := baggage.Parse(testBaggage)
	require.NoError(t, err)

	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))

	return baggage.ContextWithBaggage(ctx, bag)
}

func sameTable(a, b amqp.Table) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}
