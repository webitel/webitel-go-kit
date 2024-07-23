package tracing

import (
	"fmt"
	"math"
	"sync"
	"time"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type rateLimiter struct {
	sync.Mutex

	description string
	rps         float64
	balance     float64
	maxBalance  float64
	lastTick    time.Time
	now         func() time.Time
}

func newRateLimiter(rps float64) *rateLimiter {
	return &rateLimiter{
		rps:         rps,
		description: fmt.Sprintf("RateLimitingSampler{%g}", rps),
		balance:     math.Max(rps, 1),
		maxBalance:  math.Max(rps, 1),
		lastTick:    time.Now(),
		now:         time.Now,
	}
}

func (rl *rateLimiter) ShouldSample(p tracesdk.SamplingParameters) tracesdk.SamplingResult {
	rl.Lock()
	defer rl.Unlock()
	psc := trace.SpanContextFromContext(p.ParentContext)
	if rl.balance >= 1 {
		rl.balance -= 1

		return tracesdk.SamplingResult{Decision: tracesdk.RecordAndSample, Tracestate: psc.TraceState()}
	}
	currentTime := rl.now()
	elapsedTime := currentTime.Sub(rl.lastTick).Seconds()
	rl.lastTick = currentTime
	rl.balance = math.Min(rl.maxBalance, rl.balance+elapsedTime*rl.rps)
	if rl.balance >= 1 {
		rl.balance -= 1

		return tracesdk.SamplingResult{Decision: tracesdk.RecordAndSample, Tracestate: psc.TraceState()}
	}

	return tracesdk.SamplingResult{Decision: tracesdk.Drop, Tracestate: psc.TraceState()}
}

func (rl *rateLimiter) Description() string { return rl.description }
