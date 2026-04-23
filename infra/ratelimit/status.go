package ratelimit

import (
	"math"
	"strconv"
	"time"
)

// Status of the Limit usage
type Status struct {
	Date       time.Time     // Date of request
	Limit      uint32        // Limit quota applied
	Allowed    uint32        // Taken token(-s) count. Cost of request. Zoro indicates failure !
	Remaining  uint32        // Remaining token(-s) count. Still available for use
	ResetAfter time.Duration // Time to wait for token(-s) limit to refresh
	RetryAfter time.Duration // Time to wait for next attempt. Non-Zero indicates failure !
}

// ALLOW Request. [Status] defaults.
// Optionally assign a [status.Limit] quota value to indicate that the limit is applied.
func Allow(req *Request) Status {
	return Status{
		Date:    req.Date,
		Limit:   0, // no constraint affected
		Allowed: max(req.Cost, 1),
	}
}

// DENY Request. [Status] defaults.
// By default this Status represents Forbidden (forever: no Limit qouta defined).
// Optionally assign a [status.Limit] quota value to reflect a temporary ban.
func Deny(req *Request) Status {
	return Status{
		Date:    req.Date,
		Allowed: 0,
	}
}

func (res *Status) OK() bool {
	return res == nil || res.Allowed > 0
	// return stat == nil || stat.RetryAfter < 1
}

func (res *Status) Err() error {
	if !res.OK() {
		return &Error{*res}
	}
	// [ OK ]
	return nil
}

const (

	// textproto.CanonicalMIMEHeaderKey()
	H1LimitQuota      = "X-RateLimit-Limit"     // containing the requests quota in the time window
	H1LimitRemaining  = "X-RateLimit-Remaining" // containing the remaining requests quota in the current window
	H1LimitResetAfter = "X-RateLimit-Reset"     // containing the time remaining in the current window, specified in seconds
	H1RetryAfter      = "Retry-After"           // in seconds

	// strings.ToLower()
	H2LimitQuota      = "x-ratelimit-limit"     // containing the requests quota in the time window
	H2LimitRemaining  = "x-ratelimit-remaining" // containing the remaining requests quota in the current window
	H2LimitResetAfter = "x-ratelimit-reset"     // containing the time remaining in the current window, specified in seconds
	H2RetryAfter      = "retry-after"           // in seconds
)

func MinSeconds(in time.Duration) (sec int64) {
	if in == 0 {
		return 0
	}
	// Because float64 uses a 52-bit mantissa (plus one implicit bit),
	// it can represent integers exactly only up to maximum safe integer (2^53)
	// const maxSafeInt = float64(1 << 53)
	return int64(math.Ceil(in.Seconds()))
}

func Response(res Status) (head map[string]string, err error) {

	if res.Limit == 0 {
		// No [RATE_LIMIT] assigned !
		if res.Allowed > 0 {
			// +ALLOW[ed] !
			return nil, nil
		}
		// DENIED for all !
		err = ErrForbidden
		return // nil, err
	}

	head = make(map[string]string, 4)
	headQuota := func(key string, quota int64) {
		if quota == 0 {
			return
		}
		head[key] = strconv.FormatInt(quota, 10)
	}

	headQuota(H2LimitQuota, int64(res.Limit))
	headQuota(H2LimitRemaining, int64(res.Remaining))
	headQuota(H2LimitResetAfter, MinSeconds(res.ResetAfter))

	if !res.OK() {
		headQuota(H2RetryAfter, MinSeconds(res.RetryAfter))
	}

	err = res.Err()
	return // head, err?
}
