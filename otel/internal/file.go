package internal

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileWriter with rotation
type FileWriter = lumberjack.Logger

var fileparams = map[string]func(*FileWriter, string) error{
	"max-size": func(c *FileWriter, s string) error {
		Mb, err := strconv.Atoi(s)
		if err == nil {
			c.MaxSize = Mb
		} else {
			err = fmt.Errorf("file.max-size=%s ; accept: [ int ] Mb ; default: %d", s, c.MaxSize)
		}
		return err
	},
	"max-age": func(c *FileWriter, s string) error {
		days, err := strconv.Atoi(s)
		if err == nil {
			c.MaxAge = days
		} else {
			err = fmt.Errorf("file.max-age=%s ; accept: [ int ] days ; default: %d", s, c.MaxAge)
		}
		return err
	},
	"backups": func(c *FileWriter, s string) error {
		count, err := strconv.Atoi(s)
		if err == nil {
			c.MaxBackups = count
		} else {
			err = fmt.Errorf("file.backups=%s ; accept: [ int ] count ; default: %d", s, c.MaxBackups)
		}
		return err
	},
	"localtime": func(c *FileWriter, s string) error {
		var (
			err error
			UTC = strings.EqualFold(s, "UTC")
		)
		if !UTC {
			UTC, err = strconv.ParseBool(s)
			if err != nil {
				// log unaceptable value ; using default: localtime
				err = fmt.Errorf("file.localtime=%s ; accept: [ bool | \"utc\" ]; default: false", s)
			}
		}
		c.LocalTime = !UTC
		return err
	},
	"compress": func(c *FileWriter, s string) error {
		var (
			err  error
			gzip = strings.EqualFold(s, "gzip")
		)
		if !gzip {
			gzip, err = strconv.ParseBool(s)
			if err != nil {
				err = fmt.Errorf("file.compress=%s ; accept: [ bool | \"gzip\" ]; default: false", s)
			}
		}
		c.Compress = gzip
		return err
	},
}

func FileDSN(rawDSN string) (*FileWriter, error) {
	scheme, rawDSN, err := GetScheme(rawDSN)
	if err != nil {
		return nil, err
	}
	_ = scheme // "file"
	path, params, err := ParseDSN(rawDSN)
	if err != nil {
		return nil, err
	}
	// validate
	switch filepath.Base(path) {
	case ".", string(filepath.Separator):
		return nil, fmt.Errorf("file:path expected")
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("file:path ; %v", err)
	}
	dsn := &FileWriter{
		Filename:   path,
		MaxSize:    100,   // Mb.
		MaxAge:     30,    // day(s) count
		MaxBackups: 3,     // file(s) count
		LocalTime:  true,  // !UTC
		Compress:   false, // gzip ?
	}
	for cn, s := range params {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			continue
		}
		cn = strings.ToLower(cn)
		setup := fileparams[cn]
		if setup == nil {
			// unknown ;param= spec
			slog.Default().Warn("Unknown file.param= spec", "param", cn, cn, s)
			continue
		}
		err = setup(dsn, s)
		if err != nil {
			// WARN: non critical !
			slog.Default().Warn(err.Error())
		}
	}
	return dsn, nil
}
