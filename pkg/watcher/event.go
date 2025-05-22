package watcher

import "errors"

type EventType string

const (
	EventTypeCreate         EventType = "create"
	EventTypeDelete         EventType = "remove"
	EventTypeUpdate         EventType = "update"
	EventTypeResolutionTime EventType = "resolution_time"
)

var ErrUnknownType = errors.New("unknown event type")
