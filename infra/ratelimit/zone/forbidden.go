package limitzone

import (
	"log/slog"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// Forbidden zone helper
type Forbidden ratelimit.Options

var _ ratelimit.Zone = (*Forbidden)(nil)

// Options of zone configuration
func (c *Forbidden) Options() ratelimit.Options {
	return ratelimit.Options(*c)
}

// LimitRequest implements ratelimit.Handler interface.
func (c *Forbidden) LimitRequest(req *ratelimit.Request) (ratelimit.Status, error) {
	ratelimit.Log(
		req.Context, req.Logger,
		slog.LevelWarn, "| ✕ (forbidden)",
		slog.String("zone.name", (*ratelimit.Options)(c).Name),
	)
	return ratelimit.Deny(req), nil
}
