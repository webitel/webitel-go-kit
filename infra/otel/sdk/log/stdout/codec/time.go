package codec

import "time"

const (
	// Default timestamp format
	TimeStamp = "2006-01-02T15:04:05.999Z07:00" // time.RFC3339Nano
)

func TimeStampIsValid(layout string, skrew time.Duration) bool {
	src := time.Now().UTC()
	str := src.Format(layout)
	dst, err := time.Parse(layout, str)
	if err != nil {
		return false
	}
	return dst.Sub(src).Abs() < skrew
}

func (conf *Options) timeStamp() string {
	// custom
	if conf != nil {
		return conf.TimeStamp
	}
	// default
	return TimeStamp
}

func (conf *Options) Timestamp(date time.Time) *Timestamp {
	// Zero timestamp ?
	if date.IsZero() || date.Unix() < 1 {
		return nil
	}
	// No output ?
	layout := conf.timeStamp()
	if layout == "" {
		return nil
	}
	// Format
	return &Timestamp{
		// strip monotonic to match Attr behavior
		Time:   date.Round(0),
		Format: layout,
	}
}

// Timestamp. NULLable.
type Timestamp struct {
	Time   time.Time
	Format string
}

func (ts *Timestamp) MarshalText() ([]byte, error) {
	layout := TimeStamp
	if ts.Format != "" {
		layout = ts.Format
	}
	return ts.Time.AppendFormat(
		make([]byte, 0, len(layout)), layout,
	), nil
}

func (ts *Timestamp) String() string {
	b, _ := ts.MarshalText()
	return string(b)
}
