package redis

import (
	"context"
	_ "embed"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

//go:embed token_bucket.lua
var tokenBucketLuaScript string

// https://pkg.go.dev/github.com/go-redis/redis_rate/v10
// https://github.com/rwz/redis-gcra/blob/master/vendor/perform_gcra_ratelimit.lua
var tokenBucketScript = redis.NewScript(
	tokenBucketLuaScript,
)

func (rc *redisZone) tokenBucket(
	ctx context.Context,
	key string,
	rate ratelimit.Rate,
	burst uint32,
	cost uint32,
) (ratelimit.Status, error) {

	cost = max(1, cost)   // default: (1)
	burst = max(1, burst) // default: (1)

	// ARGV[1] => burst
	// ARGV[2] => rate ; request(s) number per interval
	// ARGV[3] => rate ; time interval, in seconds
	// ARGV[4] => tokens(s) count to take
	params := []any{
		burst,
		rate.Limit,            // rate.limit
		rate.Window.Seconds(), // rate.interval
		cost,
	}

	result, err := tokenBucketScript.Run(
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
		Limit:      burst, // rate,
		Allowed:    uint32(params[0].(int64)),
		Remaining:  uint32(params[1].(int64)),
		RetryAfter: duration(retryAfter),
		ResetAfter: duration(resetAfter),
	}
	return stat, nil
}
