package throttle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestLimiter_allow(t *testing.T) {
	t.Run("track result", func(t *testing.T) {
		l, _ := newTestLimiter(Config{Window: time.Minute, First: 1, Every: 2})

		for _, want := range []struct {
			suppressed int64
			pass       bool
		}{{0, true}, {0, false}, {1, true}} {
			suppressed, pass := l.allow(1)
			assert.Equal(t, want.suppressed, suppressed)
			assert.Equal(t, want.pass, pass)
		}
	})

	t.Run("no report below key bound", func(t *testing.T) {
		var errs []error
		setErrorHandler(t, func(err error) { errs = append(errs, err) })
		l, _ := newTestLimiter(Config{Window: time.Minute, First: 1, Every: 100, MaxKeys: 2})

		for range 5 {
			l.allow(1)
			l.allow(2)
		}

		assert.Empty(t, errs)
	})

	t.Run("overflow reported once per window", func(t *testing.T) {
		var errs []error
		setErrorHandler(t, func(err error) { errs = append(errs, err) })
		l, clk := newTestLimiter(Config{Window: time.Minute, First: 1, Every: 100, MaxKeys: 1})

		l.allow(1)
		l.allow(2)
		l.allow(3)
		require.Len(t, errs, 1)
		assert.ErrorContains(t, errs[0], "1 keys in one window")

		clk.Add(time.Minute)
		l.allow(1)
		l.allow(2)
		assert.Len(t, errs, 2)
	})

	t.Run("overflow reported outside lock", func(t *testing.T) {
		l, _ := newTestLimiter(Config{Window: time.Minute, First: 1, Every: 100, MaxKeys: 1})
		setErrorHandler(t, func(error) { l.allow(3) })
		l.allow(1)

		done := make(chan struct{})
		go func() {
			defer close(done)
			l.allow(2)
		}()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("allow deadlocked on the error handler")
		}
	})
}

func TestLimiter_track(t *testing.T) {
	t.Run("first then every", func(t *testing.T) {
		l, _ := newTestLimiter(Config{Window: time.Minute, First: 2, Every: 3})

		assert.Equal(t, []result{
			{0, true, false},
			{0, true, false},
			{0, false, false},
			{0, false, false},
			{2, true, false},
			{0, false, false},
			{0, false, false},
			{2, true, false},
		}, track(l, 1, 8))
	})

	t.Run("separate keys", func(t *testing.T) {
		l, _ := newTestLimiter(Config{Window: time.Minute, First: 1, Every: 100})

		track(l, 1, 3)

		assert.Equal(t, []result{{0, true, false}}, track(l, 2, 1))
	})

	t.Run("within window", func(t *testing.T) {
		l, clk := newTestLimiter(Config{Window: time.Minute, First: 2, Every: 100})

		track(l, 1, 2)
		clk.Add(time.Minute - time.Nanosecond)

		assert.Equal(t, []result{{0, false, false}}, track(l, 1, 1))
	})

	t.Run("window reset", func(t *testing.T) {
		l, clk := newTestLimiter(Config{Window: time.Minute, First: 2, Every: 100})

		track(l, 1, 5)
		clk.Add(time.Minute)

		assert.Equal(t, []result{{0, true, false}}, track(l, 1, 1), "suppressed count of the previous window is not carried over")
	})

	t.Run("max keys", func(t *testing.T) {
		l, clk := newTestLimiter(Config{Window: time.Minute, First: 1, Every: 100, MaxKeys: 2})

		track(l, 1, 1)
		track(l, 2, 1)
		assert.Equal(t, []result{{0, true, true}, {0, true, false}, {0, true, false}}, track(l, 3, 3))
		assert.Equal(t, []result{{0, false, false}}, track(l, 1, 1))

		clk.Add(time.Minute)
		assert.Equal(t, []result{{0, true, false}, {0, false, false}}, track(l, 3, 2))
	})

	t.Run("overflow once per window", func(t *testing.T) {
		l, clk := newTestLimiter(Config{Window: time.Minute, First: 1, Every: 100, MaxKeys: 1})

		track(l, 1, 1)
		assert.Equal(t, []result{{0, true, true}}, track(l, 2, 1))
		assert.Equal(t, []result{{0, true, false}}, track(l, 3, 1))

		clk.Add(time.Minute)
		track(l, 1, 1)
		assert.Equal(t, []result{{0, true, true}}, track(l, 2, 1))
	})
}

func newTestLimiter(cfg Config) (*limiter, *clock) {
	clk := &clock{now: time.Unix(0, 0)}
	return newLimiter(cfg.withDefaults(), clk.Now), clk
}

func track(l *limiter, key uint64, n int) []result {
	out := make([]result, 0, n)
	for range n {
		suppressed, pass, overflow := l.track(key)
		out = append(out, result{suppressed, pass, overflow})
	}
	return out
}

func setErrorHandler(t *testing.T, h func(error)) {
	t.Helper()
	orig := otel.GetErrorHandler()
	otel.SetErrorHandler(otel.ErrorHandlerFunc(h))
	t.Cleanup(func() { otel.SetErrorHandler(orig) })
}

type result struct {
	suppressed int64
	pass       bool
	overflow   bool
}
