package safemap

import (
	"errors"
	"sync"
)

// New creates a new thread-safe map.
func New[Key comparable, Value any](init map[Key]Value) (*SafeMap[Key, Value], error) {
	if init == nil {
		init = make(map[Key]Value)
	}
	return &SafeMap[Key, Value]{
		store: init,
	}, nil
}

// SafeMap is a thread-safe map implementation.
type SafeMap[Key comparable, Value any] struct {
	sync.RWMutex
	store map[Key]Value
}

// Get retrieves a value from the map by key.
func (s *SafeMap[Key, Value]) Get(key Key) (Value, bool) {
	if s == nil {
		var zero Value
		return zero, false
	}
	s.RLock()
	defer s.RUnlock()
	enc, ok := s.store[key]
	return enc, ok
}

// Set adds a new key-value pair to the map.
func (s *SafeMap[Key, Value]) Set(key Key, value Value) {
	if s == nil {
		return
	}
	s.Lock()
	defer s.Unlock()
	// Ensure the map is initialized in case the zero value of ValueEncoderStorage is used.
	if s.store == nil {
		s.store = make(map[Key]Value)
	}
	s.store[key] = value
	return
}

// Remove deletes a key-value pair from the map by key.
func (s *SafeMap[Key, Value]) Remove(key Key) {
	if s == nil {
		return
	}
	s.Lock()
	defer s.Unlock()
	delete(s.store, key)
	return
}

// Copy creates a shallow copy of the map.
func (s *SafeMap[Key, Value]) Copy() (map[Key]Value, error) {
	if s == nil {
		return nil, errors.New("safeMap is nil")
	}
	s.Lock()
	defer s.Unlock()
	copied := make(map[Key]Value, len(s.store))
	for k, v := range s.store {
		copied[k] = v
	}
	return copied, nil
}
