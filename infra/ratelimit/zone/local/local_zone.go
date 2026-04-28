package local

import (
	"fmt"
	"log/slog"
	"reflect"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/webitel/webitel-go-kit/infra/ratelimit"
)

// internal record limiter interface
type limiter interface {
	requestAt(date time.Time, cost uint32) ratelimit.Status
}

// memoryZone implements local (memory) LRU cache::table of [key::state] limits.
type memoryZone struct {
	opts  ratelimit.Options
	table *lru.Cache[limitkey, limiter]
}

// https://nginx.org/en/docs/http/ngx_http_limit_req_module.html
//
// .. A client IP address serves as a key.
// Note that instead of $remote_addr, the $binary_remote_addr variable is used here.
// The $binary_remote_addr variable’s size is always 4 bytes for IPv4 addresses or 16 bytes for IPv6 addresses.
// The stored state always occupies 64 bytes on 32-bit platforms and 128 bytes on 64-bit platforms.
// One megabyte zone can keep about 16 thousand 64-byte states or about 8 thousand 128-byte states.
//

// Default memory size of new(local.Zone) = 1Mb ;
//
//	TokenBucket(size=96b): ~[ 11 000 ] records ;
//	FixedWindow(size=56b): ~[ 19 000 ] records ;
const DefaultSize = (1 * ratelimit.Megabyte)

func newZone(opts ratelimit.Options) *memoryZone {

	bytes := DefaultSize
	if opts.Size > 0 {
		bytes = opts.Size
	}

	var unit ratelimit.ByteUnit
	switch opts.Algo {
	case ratelimit.AlgoTokenBucket:
		{
			unit = unitTokenBucket
		}
	// case ratelimit.AlgoLeakyBucket:
	case ratelimit.AlgoFixedWindow:
		{
			unit = unitFixedWindow
		}
		// case ratelimit.AlgoSlidingWindow:
		// default:
	}
	// size = int(sizeInRecordsNumber(uint64(zone.Size), unit))
	size := int(bytes.Capacity(unit)) + 1

	table, _ := lru.New[limitkey, limiter](size)
	return &memoryZone{opts: opts, table: table}
}

type limitkey struct {
	Path string
	PKey any
}

func canhash(typ reflect.Kind) bool {
	switch typ {
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8, // byte
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.String:
		return true
	}
	return false
}

func hashable(v any) any {
	var (
		rval = reflect.Indirect(reflect.ValueOf(v))
		rtyp = rval.Type()
	)
	if canhash(rtyp.Kind()) {
		return rval.Interface()
	}
	// https://stackoverflow.com/questions/29175247/hash-with-key-as-an-array-type
	// .. Array types (unlike slices) in Go are comparable ..
	// .. you cannot use slices as keys in go ..
	switch rtyp.Kind() {
	case reflect.Array:
		if etyp := rtyp.Elem(); canhash(etyp.Kind()) {
			if htyp := reflect.ArrayOf(rval.Len(), etyp); rtyp != htyp {
				rval = rval.Convert(htyp)
			}
			return rval.Interface()
		}
	case reflect.Slice:
		if etyp := rtyp.Elem(); canhash(etyp.Kind()) {
			htyp := reflect.ArrayOf(rval.Len(), etyp)
			hval := reflect.New(htyp).Elem()
			_ = reflect.Copy(hval, rval)
			return hval.Interface()
		}
	}
	return fmt.Sprintf("%v", v)
}

var _ ratelimit.Zone = (*memoryZone)(nil)

func (c *memoryZone) Options() ratelimit.Options {
	return c.opts
}

func (c *memoryZone) LimitRequest(req *ratelimit.Request) (res ratelimit.Status, err error) {

	var (
		// ctx = req.Context
		date = req.Date
		vkey = req.Get(c.opts.Key)
		pkey = limitkey{
			Path: c.opts.Name,
			PKey: hashable(vkey),
		}
		zone = &c.opts
		algo = zone.Algo
		// rate  = &zone.Rate
		cost  = max(1, req.Cost)
		burst = max(1, zone.Burst)
		// key.Value(?) was determined ?
		bypass = (vkey == ratelimit.Undefined)
	)

	defer func() {

		level := slog.LevelDebug
		if bypass {
			vkey = "$bypass" // indicates: NO key.Value("") was determined !
			level = slog.LevelWarn
		}
		if err != nil || !res.OK() {
			level = slog.LevelError
		}

		req.Log(
			// zone hit ..
			level, "| ⌙ (local)",
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

	if bypass {
		// bypass: NO key.Value("") for limit was determined !
		return ratelimit.Allow(req), nil
	}

	record, ok := c.table.Get(pkey)
	if !ok {
		switch algo {
		case ratelimit.AlgoTokenBucket:
			{
				record = newTokenBucket(zone.Rate, burst)
			}
		// case ratelimit.AlgoLeakyBucket:
		case ratelimit.AlgoFixedWindow:
			{
				record = newFixedWindow(zone.Rate)
			}
			// case ratelimit.AlgoSlidingWindow:
			// default:
		}
		if record != nil {
			_ = c.table.Add(pkey, record)
		}
	}

	if record == nil {
		res.Date = date
		err = fmt.Errorf("local( algo: %s ); not implemented", algo)
		return // res, err
	}

	res = record.requestAt(date, cost)
	res.Date = date
	return // res, nil
}
