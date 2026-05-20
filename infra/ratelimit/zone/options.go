package limitzone

import (
	"cmp"
	"context"
	"log/slog"

	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// An area used to store and track the status for each unique key (value) record.
type Zone interface {
	// Zone options
	Options() Options
	// Zone storage implementation
	ratelimit.Handler
}

// Options to define zone used to store the state of each Key.(Value) record Limiter.(Algo + Rate)
type Options struct {

	// Key characteristic against which the limit is applied
	Key ratelimit.Key
	// Associate [Key] for limit with the API route.(URL).path that request to this zone is defined within ?
	//
	// This option allows you to separate zone keys based on the API route.
	// Otherwise, the keys will aggregate limits regardless of the API route where they are defined.
	Path bool
	// Zone (reference) name
	Name string // Zone name, e.g.: table name, directory path, prefix ..
	// Maximum memory usage size
	Size ratelimit.ByteUnit // Maximum number of records in storage ...

	// Rate Limit of this Zone
	Rate ratelimit.Rate
	// Algorithm used to limit [Rate] for each unique [Key].(Value) of this Zone.
	Algo string
	// Burst is the maximum number of tokens a bucket (-like algorithms) can hold,
	// allowing a temporary, rapid spike in traffic to exceed the average rate limit instantly
	Burst uint32

	// // The Delay parameter specifies a limit at which excessive requests become delayed.
	// // Default value is zero, i.e. all excessive requests are delayed.
	// // Nil value means NoDelay, otherwise all excessive requests (after N) are delayed
	// Delay *uint32
	// NoDelay bool

	// Logger to Debug requests
	Logger *slog.Logger

	// Context for implementation depended Option bindings ..
	Context context.Context
}

// Option to setup zone Options
type Option func(zone *Options)

func NewOptions(opts ...Option) Options {
	zone := Options{
		// DEFAULT
		Rate: ratelimit.Rate{}, // FORBIDDEN
		Algo: ratelimit.AlgoTokenBucket,
		// Burst: nil,
		// Name:  "",
		// Size:  0,
		// Logger:  nil,
		Context: context.Background(),
	}
	zone.setup(opts)
	return zone
}

func (zone *Options) setup(opts []Option) {
	for _, option := range opts {
		option(zone)
	}
	// normalize ; defaults
	zone.Algo = cmp.Or(zone.Algo, ratelimit.AlgoTokenBucket)
}

// NamedZones registry
type NamedZones map[string]Zone
