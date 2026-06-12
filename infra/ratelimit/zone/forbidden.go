package limitzone

import (
	"log/slog"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// Forbidden zone helper
type Forbidden Options

var _ Zone = (*Forbidden)(nil)

// Options of zone configuration
func (c *Forbidden) Options() Options {
	return Options(*c) // shallowcopy
}

// LimitRequest implements ratelimit.Handler interface.
func (c *Forbidden) LimitRequest(req *ratelimit.Request) (*ratelimit.Status, error) {
	ratelimit.Log(
		req.Context, req.Logger,
		slog.LevelWarn, "| ✕ (forbidden)",
		slog.String("zone.name", c.Name),
	)
	return ratelimit.Deny(req), nil
}
