package throttle

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"

	"github.com/webitel/webitel-go-kit/infra/otel/correlation"
	"github.com/webitel/webitel-go-kit/pkg/logging/logtest"
)

func TestThrottleLogger_Emit(t *testing.T) {
	t.Run("suppressed count", func(t *testing.T) {
		p, rec, _ := newTestProvider(Config{Window: time.Minute, First: 2, Every: 3})
		l := p.Logger("scope")

		emit(l, context.Background(), 8, "retry failed")

		recs := rec.Records()
		require.Len(t, recs, 4)

		want := []struct {
			count int64
			ok    bool
		}{{0, false}, {0, false}, {2, true}, {2, true}}
		for i, w := range want {
			n, ok := suppressed(t, recs[i])
			assert.Equal(t, w.ok, ok, "record %d", i)
			assert.Equal(t, w.count, n, "record %d", i)
		}
	})

	t.Run("key parts", func(t *testing.T) {
		ctx := context.Background()

		tests := []struct {
			name string
			emit func(p *LoggerProvider)
		}{
			{
				name: "body",
				emit: func(p *LoggerProvider) {
					emit(p.Logger("scope"), ctx, 5, "a")
					emit(p.Logger("scope"), ctx, 5, "b")
				},
			},
			{
				name: "scope",
				emit: func(p *LoggerProvider) {
					emit(p.Logger("one"), ctx, 5, "a")
					emit(p.Logger("two"), ctx, 5, "b")
				},
			},
			{
				name: "severity",
				emit: func(p *LoggerProvider) {
					l := p.Logger("scope")
					for range 5 {
						l.Emit(ctx, record("a", log.SeverityWarn))
						l.Emit(ctx, record("b", log.SeverityError))
					}
				},
			},
			{
				name: "event name",
				emit: func(p *LoggerProvider) {
					l := p.Logger("scope")
					for _, name := range []string{"a", "b"} {
						for range 5 {
							var r log.Record
							r.SetSeverity(log.SeverityInfo)
							r.SetEventName(name)
							l.Emit(ctx, r)
						}
					}
				},
			},
			{
				name: "tenant",
				emit: func(p *LoggerProvider) {
					emit(p.Logger("scope"), withDomain(ctx, 1), 5, "a")
					emit(p.Logger("scope"), withDomain(ctx, 2), 5, "b")
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p, rec, _ := newTestProvider(Config{Window: time.Minute, First: 2, Every: 100})
				tt.emit(p)

				recs := rec.Records()
				require.Len(t, recs, 4)
				for _, r := range recs {
					_, ok := suppressed(t, r)
					assert.False(t, ok)
				}
			})
		}
	})

	t.Run("separate tenant quota", func(t *testing.T) {
		p, rec, _ := newTestProvider(Config{Window: time.Minute, First: 2, Every: 100})
		l := p.Logger("scope")
		ctx := context.Background()

		emit(l, withDomain(ctx, 1), 50, "noisy")
		emit(l, withDomain(ctx, 2), 2, "noisy")
		emit(l, ctx, 2, "noisy")

		assert.Len(t, rec.Records(), 6)
	})

	t.Run("caller record unchanged", func(t *testing.T) {
		p, rec, _ := newTestProvider(Config{Window: time.Minute, First: 1, Every: 2})
		l := p.Logger("scope")

		r := record("a", log.SeverityError)
		r.AddAttributes(attribute.Int("n", 1))
		for range 3 {
			l.Emit(context.Background(), r)
		}

		assert.Equal(t, 1, r.AttributesLen())
		recs := rec.Records()
		require.Len(t, recs, 2)
		n, ok := suppressed(t, recs[1])
		assert.True(t, ok)
		assert.Equal(t, int64(1), n)
	})

	t.Run("concurrent", func(t *testing.T) {
		p, rec, _ := newTestProvider(Config{Window: time.Minute, First: 10, Every: 10})
		l := p.Logger("scope")

		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				emit(l, context.Background(), 100, "a")
			})
		}
		wg.Wait()

		recs := rec.Records()
		require.Len(t, recs, 10+(800-10)/10)

		var total int64
		for _, r := range recs {
			n, _ := suppressed(t, r)
			total += n
		}
		assert.Equal(t, int64(800-len(recs)), total)
	})
}

func TestThrottleLogger_Enabled(t *testing.T) {
	t.Run("passes through", func(t *testing.T) {
		rec := logtest.NewRecorder(logtest.WithEnabled(func(_ context.Context, _ logtest.Scope, param log.EnabledParameters) bool {
			return param.Severity >= log.SeverityWarn
		}))
		l := NewLoggerProvider(rec, Config{First: 1, Every: 100}).Logger("scope")

		for range 3 {
			assert.True(t, l.Enabled(context.Background(), log.EnabledParameters{Severity: log.SeverityError}))
		}
		assert.False(t, l.Enabled(context.Background(), log.EnabledParameters{Severity: log.SeverityInfo}))
	})
}

func newTestProvider(cfg Config) (*LoggerProvider, *logtest.Recorder, *clock) {
	rec := logtest.NewRecorder()
	clk := &clock{now: time.Unix(0, 0)}
	p := NewLoggerProvider(rec, cfg)
	p.limiter = newLimiter(cfg.withDefaults(), clk.Now)
	return p, rec, clk
}

func record(body string, sev log.Severity) log.Record {
	var r log.Record
	r.SetBody(attribute.StringValue(body))
	r.SetSeverity(sev)
	return r
}

func emit(l log.Logger, ctx context.Context, n int, body string) {
	for range n {
		l.Emit(ctx, record(body, log.SeverityError))
	}
}

func suppressed(t *testing.T, r logtest.Record) (int64, bool) {
	t.Helper()
	for _, kv := range r.Attributes() {
		if kv.Key == SuppressedCountKey {
			return kv.Value.AsInt64(), true
		}
	}
	return 0, false
}

func withDomain(ctx context.Context, id int64) context.Context {
	return correlation.WithIdentity(ctx, id, 0)
}

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
