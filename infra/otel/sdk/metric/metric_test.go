package metric

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"go.opentelemetry.io/otel"
)

var seq atomic.Int64

// scheme returns a name no other test has registered.
func scheme(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, seq.Add(1))
}

func captureHandled(t *testing.T) *[]error {
	t.Helper()
	var errs []error
	prev := otel.GetErrorHandler()
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) { errs = append(errs, err) }))
	t.Cleanup(func() { otel.SetErrorHandler(prev) })
	return &errs
}

func noopOptions(context.Context, string) ([]Option, error) { return nil, nil }

func TestRegister(t *testing.T) {
	errs := captureHandled(t)
	ctx := context.Background()

	ok := scheme("ok")
	Register(ok, noopOptions)
	if _, err := NewOptions(ctx, strings.ToUpper(ok)+":x"); err != nil {
		t.Fatalf("lookup is case-insensitive: %v", err)
	}

	space, upper, nilCtor := " "+scheme("space"), strings.ToUpper(scheme("upper")), scheme("nil")
	Register(space, noopOptions)
	Register(upper, noopOptions)
	Register("", noopOptions)
	Register(nilCtor, nil)
	for _, dsn := range []string{strings.TrimSpace(space), strings.ToLower(upper), nilCtor} {
		if _, err := NewOptions(ctx, dsn); err == nil {
			t.Errorf("%q must not be registered", dsn)
		}
	}
	Register(nilCtor, noopOptions) // a rejected registration leaves the name free
	if _, err := NewOptions(ctx, nilCtor); err != nil {
		t.Fatal(err)
	}

	dup := scheme("dup")
	Register(dup, func(context.Context, string) ([]Option, error) { return nil, errors.New("first") })
	Register(dup, func(context.Context, string) ([]Option, error) { return nil, errors.New("second") })
	if _, err := NewOptions(ctx, dup); err == nil || err.Error() != "first" {
		t.Fatalf("duplicate must keep the first registration, got %v", err)
	}

	if len(*errs) != 5 {
		t.Fatalf("handled %d errors, want 5: %v", len(*errs), *errs)
	}
}

func TestNewOptionsErrors(t *testing.T) {
	for dsn, want := range map[string]string{"nope:x": "unknown", ":x": "missing scheme"} {
		if _, err := NewOptions(context.Background(), dsn); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("NewOptions(%q) = %v, want %q", dsn, err, want)
		}
	}
}
