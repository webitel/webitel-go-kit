package redis

import (
	"cmp"
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
	limitzone "github.com/webitel/webitel-go-kit/infra/ratelimit/zone"
)

func New(dataSource string, options ratelimit.Options) (ratelimit.Zone, error) {
	// TODO: "redis:///.." => "unix:/.."
	dsn, err := redis.ParseURL(dataSource)
	if err != nil {
		return nil, err
	}
	dsn.ClientName = cmp.Or(dsn.ClientName, "rate-limit")
	// TODO: map[*redis.Options]*redis.Client !!!
	client := redis.NewClient(dsn)

	ctx, cancel := context.WithTimeout(
		context.Background(), (5 * time.Second),
	)
	defer cancel()

	return newZone(client, options), client.Ping(ctx).Err()
}

func init() {
	// Register REDIS factory driver scheme(s) for usage ..
	limitzone.Register(New, "redis", "rediss", "unix")
}
