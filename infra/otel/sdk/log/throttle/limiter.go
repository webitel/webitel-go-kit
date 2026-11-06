package throttle

import (
	"fmt"
	"hash/maphash"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
)

type count struct {
	seen       int
	suppressed int64
}

type limiter struct {
	cfg  Config
	now  func() time.Time
	seed maphash.Seed

	mu       sync.Mutex
	start    time.Time
	counts   map[uint64]*count
	overflow bool
}

func newLimiter(cfg Config, now func() time.Time) *limiter {
	return &limiter{
		cfg:    cfg,
		now:    now,
		seed:   maphash.MakeSeed(),
		start:  now(),
		counts: make(map[uint64]*count),
	}
}

func (l *limiter) allow(key uint64) (int64, bool) {
	suppressed, pass, keysOverflow := l.track(key)
	if keysOverflow {
		otel.Handle(fmt.Errorf("otel/sdk/log/throttle: %d keys in one window, records with new keys pass unthrottled", l.cfg.MaxKeys))
	}
	return suppressed, pass
}

// track counts the record with key. overflow is true only for the first
// record of a window that finds the key bound reached, so it is reported once.
func (l *limiter) track(key uint64) (suppressed int64, pass, keysOverflow bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if now := l.now(); now.Sub(l.start) >= l.cfg.Window {
		clear(l.counts)
		l.start = now
		l.overflow = false
	}

	c, ok := l.counts[key]
	if !ok {
		if len(l.counts) >= l.cfg.MaxKeys {
			keysOverflow = !l.overflow // return true only for first entry when l.overflow is false
			l.overflow = true
			return 0, true, keysOverflow
		}
		c = &count{}
		l.counts[key] = c
	}

	c.seen++
	if c.seen <= l.cfg.First {
		return 0, true, false
	}
	if (c.seen-l.cfg.First)%l.cfg.Every != 0 {
		c.suppressed++
		return 0, false, false
	}

	suppressed = c.suppressed
	c.suppressed = 0
	return suppressed, true, false
}
