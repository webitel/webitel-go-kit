package logging

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/log"
)

type Level slog.Level

const (
	LevelTrace = Level(slog.LevelDebug - 4)
	LevelDebug = Level(slog.LevelDebug)
	LevelInfo  = Level(slog.LevelInfo)
	LevelWarn  = Level(slog.LevelWarn)
	LevelError = Level(slog.LevelError)
	LevelFatal = Level(slog.LevelError + 4)
)

const severityOffset = Level(log.SeverityDebug) - LevelDebug

var _ slog.Leveler = LevelInfo

func (l Level) Level() slog.Level {
	return slog.Level(l)
}

func (l Level) Severity() log.Severity {
	return log.Severity(l + severityOffset)
}

func (l Level) String() string {
	name := func(base string, off Level) string {
		if off == 0 {
			return base
		}
		return fmt.Sprintf("%s%+d", base, off)
	}

	switch {
	case l < LevelDebug:
		return name("TRACE", l-LevelTrace)
	case l < LevelInfo:
		return name("DEBUG", l-LevelDebug)
	case l < LevelWarn:
		return name("INFO", l-LevelInfo)
	case l < LevelError:
		return name("WARN", l-LevelWarn)
	case l < LevelFatal:
		return name("ERROR", l-LevelError)
	default:
		return name("FATAL", l-LevelFatal)
	}
}

func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func (l *Level) UnmarshalText(data []byte) (err error) {
	s := string(data)
	defer func() {
		if err != nil {
			err = fmt.Errorf("logging: level %q: %w", s, err)
		}
	}()

	name, offset := s, 0
	if i := strings.IndexAny(s, "+-"); i >= 0 {
		name = s[:i]
		if offset, err = strconv.Atoi(s[i:]); err != nil {
			return err
		}
	}

	var base Level
	switch strings.ToUpper(name) {
	case "TRACE":
		base = LevelTrace
	case "DEBUG":
		base = LevelDebug
	case "INFO":
		base = LevelInfo
	case "WARN":
		base = LevelWarn
	case "ERROR":
		base = LevelError
	case "FATAL":
		base = LevelFatal
	default:
		return errors.New("unknown name")
	}

	*l = base + Level(offset)
	return nil
}
