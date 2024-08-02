package jsonl_provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/webitel/webitel-go-kit/logs/model"
	"os"
	"time"
)

const (
	defaultFilePath = "/var/log/webitel"
)

type JsonlProvider struct {
	config   *config
	fileTime time.Time
	file     *os.File
}

func New(opts ...Option) (*JsonlProvider, error) {
	var (
		conf config
	)
	for _, opt := range opts {
		opt.apply(&conf)
	}
	if conf.filePath == "" {
		conf.filePath = defaultFilePath
	}
	if conf.serviceName == "" {
		return nil, errors.New("service name required")
	}
	provider := &JsonlProvider{config: &conf, fileTime: time.Now()}
	err := provider.openOrCreateNewFile()
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (o *JsonlProvider) getFileName(neededTime time.Time) string {
	return fmt.Sprintf("%s-%v-%v-%v.jsonl", o.config.serviceName, neededTime.Year(), neededTime.Month(), neededTime.Day())
}

func (o *JsonlProvider) Info(ctx context.Context, s *model.Record) error {
	if s.Level != model.InfoLevel.String() {
		return nil
	}
	return o.writeToFile(s)
}

func (o *JsonlProvider) Debug(ctx context.Context, s *model.Record) error {
	if s.Level != model.DebugLevel.String() {
		return nil
	}
	return o.writeToFile(s)
}

func (o *JsonlProvider) Warn(ctx context.Context, s *model.Record) error {
	if s.Level != model.WarnLevel.String() {
		return nil
	}
	return o.writeToFile(s)
}

func (o *JsonlProvider) Critical(ctx context.Context, s *model.Record) error {
	if s.Level != model.CriticalLevel.String() {
		return nil
	}
	return o.writeToFile(s)
}

func (o *JsonlProvider) Error(ctx context.Context, s *model.Record) error {
	if s.Level != model.ErrorLevel.String() {
		return nil
	}
	return o.writeToFile(s)
}

func (o *JsonlProvider) SetAsGlobal() {
	return
}

func (o *JsonlProvider) writeToFile(r *model.Record) error {
	j, err := r.Jsonify()
	if err != nil {
		return fmt.Errorf("could not json marshal data: %w", err)
	}

	_, err = o.file.Write(j)

	if err != nil {
		return fmt.Errorf("could not write json data to underlying io.Writer: %w", err)
	}

	_, err = o.file.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("could not write newline to underlying io.Writer: %w", err)
	}

	return nil
}

func (o *JsonlProvider) isFileOutdated() bool {
	if time.Now().Year() != o.fileTime.Year() || time.Now().Month() != o.fileTime.Month() || time.Now().Day() != o.fileTime.Day() {
		return true
	}
	return false
}

func (o *JsonlProvider) openOrCreateNewFile() error {
	currentTime := time.Now()
	fileName := o.getFileName(currentTime)
	filePath := o.config.filePath
	if err := os.Mkdir(filePath, os.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	fullPath := fmt.Sprintf("%s/%s", filePath, fileName)
	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		file, err = os.Create(fullPath)
		if err != nil {
			return err
		}
		o.fileTime = currentTime
	}
	o.file = file
	return nil
}
