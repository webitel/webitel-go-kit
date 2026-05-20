package ratelimit

import (
	"fmt"
	"slices"
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

func EqualKeys(k1, k2 Key) bool {
	return k1 != nil && k2 != nil &&
		(k1 == k2 || k1.String() == k2.String())
}

// MultiKey resolver ; [0] = prime ; [1:] = salt
func MultiKey(sep string, pkey Key, salt ...Key) Key {
	if len(salt) == 0 {
		return pkey
	}
	return MultiKeys(
		sep, append([]Key{pkey}, salt...)...,
	)
}

// MultiKey resolver ; [0] = prime ; [1:] = salt
func MultiKeys(sep string, keys ...Key) Key {
	keys = slices.DeleteFunc(
		keys, func(x Key) bool {
			return x == nil
		},
	)
	switch len(keys) {
	case 0:
		panic("multi: keys missing")
	case 1:
		return keys[0]
	}
	// default:
	// if sep == "" {
	// 	sep = "+" // "_" // ";"
	// }
	var name strings.Builder
	defer name.Reset()
	for _, x := range keys {
		name.WriteString(sep + x.String())
	}
	return KeyFunc(
		name.String(), func(req *Request) Value {
			pk := keys[0]
			vs := req.Get(pk)
			if vs == Undefined {
				return Undefined
			}
			for _, sk := range keys[1:] {
				salt := req.Get(sk)
				if salt == Undefined {
					continue
				}
				vs += (sep + salt)
			}
			return vs
		},
	)
}

// // MultiValue returns a [key] for given static [value]
// func MultiValue(pkey Value, salt ...Value) Value {
// 	if pkey == Undefined {
// 		return Undefined
// 	}
// 	salt = slices.DeleteFunc(salt,
// 		func(salt Value) bool {
// 			return salt == Undefined
// 		},
// 	)
// 	const delim = ";"
// 	for _, salt := range salt {
// 		pkey += (";" + salt)
// 	}
// 	return pkey
// }

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
