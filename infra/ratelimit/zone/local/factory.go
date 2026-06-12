package local

import (
	limitzone "github.com/webitel/webitel-go-kit/infra/ratelimit/zone"
)

func New(dataSource string, options limitzone.Options) (limitzone.Zone, error) {
	// TODO: move local:[size=] to dataSource instead of zone options ?
	_ = dataSource // no affect yet
	return newZone(options), nil
}

func init() {
	// builtin & DEFAULT
	limitzone.Register(New, "local", "memory", "")
}
