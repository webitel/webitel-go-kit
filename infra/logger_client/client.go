package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

type Action string

func (a Action) String() string {
	return string(a)
}

const (
	CreateAction Action = "create"
	UpdateAction Action = "update"
	DeleteAction Action = "delete"
	ReadAction   Action = "read"

	ExhchangeName = "logger"
)

type Logger struct {
	publisher Publisher
}
type ObjectedLogger struct {
	objClass string
	parent   *Logger
}
type LoggerOpts func(*Logger) error

type Publisher interface {
	Publish(ctx context.Context, exchange string, routingKey string, body []byte, headers map[string]any) error
}

func WithPublisher(publisher Publisher) LoggerOpts {
	return func(client *Logger) error {
		if publisher == nil {
			return errors.New("publisher is nil")
		}
		client.publisher = publisher
		return nil
	}
}

func New(pub Publisher, opts ...LoggerOpts) (*Logger, error) {
	var err error
	logger := &Logger{}
	for _, opt := range opts {
		err = opt(logger)
		if err != nil {
			return nil, err
		}
	}
	if logger.publisher == nil {
		return nil, errors.New("publisher is nil")
	}
	return logger, nil
}

func (l *Logger) GetObjectedLogger(object string) (*ObjectedLogger, error) {
	if object == "" {
		return nil, errors.New("object required")
	}
	return &ObjectedLogger{
		objClass: object,
		parent:   l,
	}, nil
}

func (l *Logger) SendContext(ctx context.Context, domainId int64, object string, message *Message) (operationId string, err error) {
	if object == "" {
		return "", errors.New("no object")
	}
	if message == nil {
		return "", errors.New("message required")
	}
	if domainId <= 0 {
		return "", errors.New("domain required")
	}
	err = ValidateMessage(message)
	if err != nil {
		return "", err
	}
	message.OperationId = uuid.NewString()
	body, err := json.Marshal(message)
	if err != nil {
		return operationId, err
	}
	err = l.publisher.Publish(ctx, ExhchangeName, formatKey(domainId, object), body, nil)
	if err != nil {
		return operationId, err
	}
	return operationId, nil
}

func (l *ObjectedLogger) SendContext(ctx context.Context, domainId int64, message *Message) (operationId string, err error) {
	if l == nil {
		return "", errors.New("logger is nil")
	}
	if l.parent == nil {
		return "", errors.New("no parent logger")
	}

	return l.parent.SendContext(ctx, domainId, l.objClass, message)
}

func (l *ObjectedLogger) GetObjClass() string {
	return l.objClass
}

// region UTILITY

func formatKey(domainId int64, objClass string) string {
	return fmt.Sprintf("logger.%d.%s", domainId, objClass)
}

func ValidateMessage(message *Message) error {
	var errs []error
	if message == nil {
		return errors.New("message required")
	}
	if message.UserIp == "" {
		errs = append(errs, errors.New("user ip required"))
	}
	if message.UserId <= 0 {
		errs = append(errs, errors.New("user id required"))
	}
	if message.Date <= 0 {
		errs = append(errs, errors.New("date required"))
	}
	switch message.Action {
	case CreateAction.String():
		fallthrough
	case UpdateAction.String():
		if message.Records == nil {
			errs = append(errs, errors.New("records required"))
		}
	case DeleteAction.String():
	default:
		return errors.New("invalid action")
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

// endregion
