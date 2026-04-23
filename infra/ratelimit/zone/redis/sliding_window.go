package redis

import (
	"context"
	_ "embed"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

//go:embed sliding_window.lua
var slidingWindowLuaScript string

// https://blogs.halodoc.io/taming-the-traffic-redis-and-lua-powered-sliding-window-rate-limiter-in-action/
// ARGV[1] => request(s) number ; burst
// ARGV[2] => over window interval
// ARGV[3] => request(s) count
var slidingWindowScript = redis.NewScript(
	slidingWindowLuaScript,
)

func (rc *redisZone) slidingWindow(
	ctx context.Context,
	key string,
	rate ratelimit.Rate,
	cost uint32,
) (ratelimit.Status, error) {

	cost = max(1, cost) // default: (1)

	// ARGV[1] => request(s) number per fixed_window; burst
	// ARGV[2] => window interval, in seconds
	// ARGV[3] => request(s) count
	params := []any{
		rate.Limit,            // burst
		rate.Window.Seconds(), // window
		cost,
	}

	result, err := slidingWindowScript.Run(
		ctx, rc.client, []string{key}, params...,
	).Result()

	if err != nil {
		return ratelimit.Status{}, err
	}

	params = result.([]any)

	retryAfter, err := strconv.ParseFloat(
		params[2].(string), 64,
	)
	if err != nil {
		return ratelimit.Status{}, err
	}

	resetAfter, err := strconv.ParseFloat(
		params[3].(string), 64,
	)
	if err != nil {
		return ratelimit.Status{}, err
	}

	stat := ratelimit.Status{
		Limit:      uint32(rate.Limit),
		Allowed:    uint32(params[0].(int64)),
		Remaining:  uint32(params[1].(int64)),
		RetryAfter: duration(retryAfter),
		ResetAfter: duration(resetAfter),
	}
	return stat, nil
}
