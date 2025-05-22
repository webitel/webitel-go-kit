package watcher

import (
	"fmt"
	"log/slog"
)

type Watcher interface {
	Attach(EventType, Observer)
	Detach(EventType, Observer)
	OnEvent(EventType, WatchMarshaller) error
}

type DefaultWatcher struct {
	observers map[EventType][]Observer
}

func NewDefaultWatcher() *DefaultWatcher {
	return &DefaultWatcher{
		observers: make(map[EventType][]Observer),
	}
}

func (dw *DefaultWatcher) Attach(et EventType, o Observer) {
	dw.observers[et] = append(dw.observers[et], o)
}

func (dw *DefaultWatcher) Detach(et EventType, o Observer) {
	for i, v := range dw.observers[et] {
		if v.GetId() == o.GetId() {
			dw.observers[et] = append(dw.observers[et][:i], dw.observers[et][i+1:]...)
			break
		}
	}
}

func (dw *DefaultWatcher) Notify(et EventType, entity WatchMarshaller) error {
	var err error
	for _, o := range dw.observers[et] {
		err = o.Update(et, entity.GetArgs())
		if err != nil {
			slog.Error(fmt.Sprintf("observer %s: %s", o.GetId(), err.Error()))
		}
	}
	return nil
}

func (dw *DefaultWatcher) OnEvent(et EventType, entity WatchMarshaller) error {
	switch et {
	case EventTypeCreate:
		return dw.Notify(EventTypeCreate, entity)
	case EventTypeDelete:
		return dw.Notify(EventTypeDelete, entity)
	case EventTypeUpdate:
		return dw.Notify(EventTypeUpdate, entity)
	case EventTypeResolutionTime:
		return dw.Notify(EventTypeResolutionTime, entity)
	default:
		return ErrUnknownType
	}
}
