package ratelimit

import (
	"fmt"
	"time"
)

// Limit [Rate] specification, e.g.: 10r/s
type Rate struct {
	// Limit the maximum number of tokens, events, requests ..
	Limit int
	// Time Limit window, period, interval ..
	Window TimeUnit
}

func (v *Rate) String() string {
	if v == nil {
		return "0r/s" // FORBIDDEN
	}
	return fmt.Sprintf(
		"%dr/%s", v.Limit,
		formatTimeUnit(v.Window),
	)
}

// Minimum time interval between tokens
func (v *Rate) Every() time.Duration {
	if v != nil && v.Limit > 0 && v.Window > 0 {
		return time.Duration(uint64(v.Window) / uint64(v.Limit))
	}
	// Forbidden(!)
	return 0
}

// IsValid reports whether [v] represents a valid time-based Rate Limit.
// False MUST be treated as Forbidden.
func (v *Rate) IsValid() bool {
	return (v.Every() / MinInterval) > 0 // >= 1r/ms
	// return v != nil && v.Max > 0 && v.Per > 0 &&
	// 	uint64(v.Max) <= uint64(v.Per / Millisecond) // MinUnit
}

func (v *Rate) MarshalText() (text []byte, err error) {
	if v == nil {
		return nil, nil // NULL
	}
	text = fmt.Appendf(
		text, "%dr/%s",
		v.Limit, formatTimeUnit(v.Window),
	)
	return text, nil
}

func (v *Rate) UnmarshalText(text []byte) error {
	v1, ok := ParseRate(string(text))
	if (*v) = v1; !ok {
		return ErrRateInvalid
	}
	return nil
}

var ErrRateInvalid = fmt.Errorf("rate: invalid spec")

func ParseRate(spec string) (rate Rate, ok bool) {
	var (
		max int
		per string
	)
	n, err := fmt.Sscanf(spec, "%dr/%s", &max, &per)
	if err != nil {
		// invalid rate specification
		return // Rate{}, false
	}
	_ = n // expect: 2
	unit, ok := parseTimeUnit(per)
	if !ok {
		// invalid time unit interval
		return // Rate{}, false
	}
	// if max < 1 {
	// 	// FIXME: negative means NO limit ; invalid rate spec
	// 	max = 1
	// }
	rate.Limit = max
	rate.Window = time.Duration(unit)
	return rate, true
}

func MustRate(spec string) Rate {
	v, ok := ParseRate(spec)
	if !ok {
		panic(ErrRateInvalid)
	}
	return v
}
