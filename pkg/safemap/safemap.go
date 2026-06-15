package safemap

import (
	"errors"
	"maps"
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

// Range iterates over all key-value pairs in the map, calling the callback function for each pair.
// If the callback function returns an error, iteration stops and the error is returned.
func (s *SafeMap[Key, Value]) Range(callback func(Key, Value) error) error {
	if s == nil {
		return errors.New("safeMap is nil")
	}
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.store {
		if err := callback(k, v); err != nil {
			return err
		}
	}
	return nil
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
	return maps.Clone(s.store), nil
}

func (s *SafeMap[Key, Value]) Len() int {
	if s == nil {
		return 0
	}
	s.RLock()
	defer s.RUnlock()
	return len(s.store)
}
