package local

import (
	"time"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
	"golang.org/x/time/rate"
)

type tokenBucket struct {
	limit  ratelimit.Rate
	bucket *rate.Limiter
}

func newTokenBucket(limit ratelimit.Rate, burst uint32) *tokenBucket {
	// minimum: 1
	burst = max(burst, 1)
	// interval := limit.Per
	emission := time.Duration(0)
	if limit.Limit > 0 {
		emission = time.Duration(
			int64(limit.Window) / int64(limit.Limit),
		)
	}
	return &tokenBucket{
		limit: limit,
		bucket: rate.NewLimiter(
			rate.Every(emission), int(burst),
		),
	}
}

func (c *tokenBucket) requestAt(date time.Time, cost uint32) (res ratelimit.Status) {
	// minimum: 1
	cost = max(cost, 1)
	req := c.bucket.ReserveN(date, int(cost))
	retry := req.DelayFrom(date)
	allow := (retry == 0)
	if !allow {
		// DENIED ; Cancel the Reservation !
		req.CancelAt(date)
	}
	rate := &c.limit
	burst := c.bucket.Burst() // int(req.Burst)
	every := rate.Window / time.Duration(rate.Limit)
	if !allow {
		// DENIED
		res = ratelimit.Status{
			Date:       date,
			Limit:      uint32(burst), // *rate,
			Allowed:    0,             // uint32(),
			Remaining:  0,             // uint32(rc.TokensAt(date)),
			RetryAfter: retry,
			ResetAfter: (every * time.Duration(burst)) - retry,
		}
	} else {
		tokens := c.bucket.TokensAt(date)
		taken := float64(burst) - tokens
		reset := time.Duration(float64(every) * taken)
		res = ratelimit.Status{
			Limit:      uint32(burst), // *rate,
			Allowed:    cost,
			Remaining:  uint32(tokens),
			RetryAfter: 0,
			ResetAfter: reset,
		}
	}
	return // res, nil
}
