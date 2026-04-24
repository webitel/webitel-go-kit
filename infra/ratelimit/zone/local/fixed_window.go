package local

import (
	"sync"
	"time"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

type fixedWindow struct {
	mx    sync.Mutex
	limit ratelimit.Rate
	reset time.Time // when window resets
	taken uint32    // token(s) USED within window
}

func newFixedWindow(limit ratelimit.Rate) *fixedWindow {
	return &fixedWindow{
		limit: limit,
		taken: 0,
	}
}

func (e *fixedWindow) TokensAt(date time.Time) int {

	e.mx.Lock()
	defer e.mx.Unlock()
	// window has started ?
	if e.reset.IsZero() {
		// Not yet ; Available MAX !
		return e.limit.Limit
	}

	offset := e.reset.Sub(date)
	if 0 < offset {
		// date AFTER this window
		return e.limit.Limit
	}

	if e.limit.Window < offset {
		// past window !
		return 0
	}
	// X-RateLimit-Remaining ; MAY: negative
	return e.limit.Limit - int(e.taken)
}

func (e *fixedWindow) takeN(date time.Time, cost uint32, hard bool) (allow, remain uint32, reset time.Duration) {

	e.mx.Lock()
	defer e.mx.Unlock()

	if e.reset.Before(date) {
		// start NEW window at date !
		e.reset = date.Add(e.limit.Window)
		e.taken = 0 // reset !
	}
	// remaining ?
	tokens := e.limit.Limit - int(e.taken)
	// excess ?
	if tokens < 1 {
		return 0, 0, e.reset.Sub(date)
	}
	// partial ?
	if uint32(tokens) < cost && hard {
		return 0, 0, e.reset.Sub(date)
	}
	// advance !
	e.taken += cost
	allow = uint32(tokens)
	if cost < allow {
		allow = cost
	}
	return allow, (uint32(tokens) - cost), e.reset.Sub(date)
}

func (c *fixedWindow) requestAt(date time.Time, cost uint32) (res ratelimit.Status) {
	res.Date = date
	if c.limit.Limit < 1 {
		return // DENY ALL
	}
	res.Limit = uint32(c.limit.Limit)
	res.Allowed, res.Remaining, res.ResetAfter = c.takeN(date, cost, true)
	if res.Allowed < 1 {
		res.RetryAfter = res.ResetAfter
	}
	return // res
}
