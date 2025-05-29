package watcher

type WatchMarshaller interface {
	GetArgs() map[string]any
}

type Observer interface {
	Update(EventType, map[string]any) error
	GetId() string
}
