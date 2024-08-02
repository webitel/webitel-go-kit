package model

const (
	CriticalLevel LogLevel = iota
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

var (
	Levels = map[string]LogLevel{
		"critical": CriticalLevel,
		"error":    ErrorLevel,
		"warn":     WarnLevel,
		"info":     InfoLevel,
		"debug":    DebugLevel,
	}
)

type LogLevel int

func (l LogLevel) String() string {
	var lvl string
	switch l {
	case 0:
		lvl = "CRITICAL"
	case 1:
		lvl = "ERROR"
	case 2:
		lvl = "WARN"
	case 4:
		lvl = "DEBUG"
	default:
		lvl = "INFO"
	}
	return lvl
}

func (l LogLevel) Int() int {
	return int(l)
}
