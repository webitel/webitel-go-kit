package ratelimit

import (
	"fmt"
	"strings"
)

// [Key] Value the limit is applied to ..
type Value = string

// An Undefined Value indicates that the [Key] could not be determined.
var Undefined = Value("")

// Key.(Value) resolver
type Key interface {
	// String name of the Key
	String() string // Key.(name)

	// Key [Value] associated with given [Request]
	Value(*Request) Value
}

// MUST return [Key] specific [Value] to apply limit to ..
type ValueFunc func(*Request) Value

// NamedKey represents a ValueFunc as a [Key]
type NamedKey struct {
	Name string
	Read ValueFunc
}

var _ Key = NamedKey{}

func (x NamedKey) String() string {
	if x.Name != "" {
		return x.Name
	}
	// !NOKEY
	return ""
}

func (x NamedKey) Value(req *Request) Value {
	if x.Read != nil {
		return x.Read(req)
	}
	// not implemented
	return Undefined
}

// KeyFunc returns a [Key] for given [read] ValueFunc
func KeyFunc(name string, read ValueFunc) Key {
	return NamedKey{Name: name, Read: read}
}

// KeyValue returns a [Key] for given static [value]
func KeyValue(name string, value Value) Key {
	return NamedKey{
		Name: name,
		Read: func(_ *Request) string {
			return value
		},
	}
}

// NamedKeys registry
type NamedKeys map[string]Key

// KeyName canonizes given [name] for a [Key] registry
func KeyName(name string) string { // (string, error) {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	// return name, nil
	return name
}

func (m NamedKeys) Add(env Key) error {
	if env == nil {
		return fmt.Errorf("register: key(<nil>) undefined")
	}
	name := KeyName(env.String())
	if reg, ok := m[name]; ok {
		if reg == env {
			// already
			return nil
		}
		return fmt.Errorf("register: key(%q) duplicate", name)
	}
	m[name] = env
	return nil
}

func (m NamedKeys) AddFunc(name string, read ValueFunc) error {
	if read == nil {
		return fmt.Errorf("register: key(%q) undefined", name)
	}
	name = KeyName(name)
	return m.Add(KeyFunc(name, read))
}

func (m NamedKeys) Set(name string, env Key) error {
	if env == nil {
		return fmt.Errorf("register key(%q) undefined", name)
	}
	if name == "" {
		name = env.String()
	}
	name = KeyName(name)
	m[name] = env
	return nil
}

func (m NamedKeys) Get(name string) Key {
	name = KeyName(name)
	reg, _ := m[name]
	return reg
}

func (m NamedKeys) Del(name string) (Key, bool) {
	name = KeyName(name)
	if reg, ok := m[name]; ok {
		delete(m, name)
		return reg, true
	}
	return nil, false
}
