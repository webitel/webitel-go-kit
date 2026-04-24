package ratelimit

import "time"

// TimeUnit represents time interval
type TimeUnit = time.Duration

// Constant well-known TimeUnit(s)
const (
	Millisecond TimeUnit = time.Millisecond
	Second               = time.Second
	Minute               = time.Minute
	Hour                 = time.Hour
	Day                  = (24 * Hour)
	Week                 = (07 * Day)

	// MinInterval is minimal time interval between events
	MinInterval = Millisecond
)

var (
	timeUnitMap = map[uint64]string{
		uint64(Millisecond): "ms",
		uint64(Second):      "s",
		uint64(Minute):      "m",
		uint64(Hour):        "h",
		uint64(Day):         "d",
		uint64(Week):        "w",
	}
	unitTimeMap = map[string]uint64{
		// "ns": uint64(time.Nanosecond),
		// "us": uint64(time.Microsecond),
		// "µs": uint64(time.Microsecond), // U+00B5 = micro symbol
		// "μs": uint64(time.Microsecond), // U+03BC = Greek letter mu
		"ms": uint64(Millisecond),
		"s":  uint64(Second),
		"m":  uint64(Minute),
		"h":  uint64(Hour),
		"d":  uint64(Day),
		"w":  uint64(Week),
		// extra ..
		"msec": uint64(Millisecond),
		"sec":  uint64(Second),
		"min":  uint64(Minute),
		"hour": uint64(Hour),
		"day":  uint64(Day),
		"week": uint64(Week),
	}
)

func parseTimeUnit(spec string) (time.Duration, bool) {
	if unit, ok := unitTimeMap[spec]; ok {
		return time.Duration(unit), true
	}
	return 0, false
}

func formatTimeUnit(unit time.Duration) string {
	if spec, ok := timeUnitMap[uint64(unit)]; ok {
		return spec
	}
	return "-"
}
