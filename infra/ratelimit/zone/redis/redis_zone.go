package redis

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

type redisZone struct {
	options ratelimit.Options
	client  *redis.Client
}

func newZone(client *redis.Client, options ratelimit.Options) *redisZone {
	return &redisZone{options: options, client: client}
}

var _ ratelimit.Zone = (*redisZone)(nil)

// Zone Options
func (rc *redisZone) Options() ratelimit.Options {
	return rc.options
}

// Limit single request with context
// Returns the time duration to wait before the request can be processed.
func (rc *redisZone) LimitRequest(req ratelimit.Request) (res ratelimit.Status, err error) {
	// panic("not implemented")
	var (
		zone = &rc.options // req.Zone
		// rate  = &zone.Rate
		vkey = req.Get(rc.options.Key)
		// burst uint32
		cost  = max(1, req.Cost)
		burst = max(1, zone.Burst)
	)

	defer func() {

		level := slog.LevelDebug
		if err != nil || !res.OK() {
			level = slog.LevelError
		}

		req.Log(
			// zone hit ..
			level, "| ⌙ (redis)",
			// args: deferred
			"", ratelimit.LogValue(func() slog.Value {
				return slog.GroupValue(
					// slog.Group("req",
					slog.String(zone.Key.String(), vkey),
					// slog.String("zone", zone.Name),
					// slog.String("key", zone.Key.String()),
					// ),
					// slog.String("zone", zone.Name),
					slog.Group("zone",
						slog.String("name", zone.Name),
						slog.String("algo", zone.Algo),
						slog.String("rate", zone.Rate.String()),
						// slog.Int64("burst", int64(burst)),
					),
					slog.Any("limit", &res),
				)
			}),
		)

	}()

	ctx := req.Context
	key := limitKey(rc.options.Name, vkey)

	switch zone.Algo {
	case ratelimit.AlgoTokenBucket:
		res, err = rc.tokenBucket(ctx, key, zone.Rate, burst, cost)
	case ratelimit.AlgoFixedWindow:
		res, err = rc.fixedWindow(ctx, key, zone.Rate, cost)
	case ratelimit.AlgoSlidingWindow:
		res, err = rc.slidingWindow(ctx, key, zone.Rate, cost)
	default:
		// not implemented yet
	}

	return // stat, err
}

func limitKey(path string, key any) string {
	pkey, ok := key.(string)
	if !ok {
		pkey = fmt.Sprintf("%s", key)
	}
	return path + ":" + pkey
}

func duration(f float64) time.Duration {
	if f == -1 {
		return -1
	}
	return time.Duration(f * float64(time.Second))
}
