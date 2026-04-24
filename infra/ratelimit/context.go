package ratelimit

import (
	"reflect"
)

// Environment used to cache resolved Key.(Value)s
type Env map[string]Value

// Key wraps given [Key] with [Env].
func (e Env) Key(set Key) Key {
	// break Env.Key(!) recursion
	if bind, ok := set.(readEnv); ok {
		a := reflect.ValueOf(e).Pointer()
		b := reflect.ValueOf(bind.env).Pointer()
		if a == b {
			return bind
		}
		// extract ..
		set = bind.key
	}
	return readEnv{key: set, env: e}
}

// func (e Env) Get(ctx context.Context, key Key) Value {
// 	return e.Key(key).Value(ctx)
// }

// func (e Env) Set(ctx context.Context, key Key) {
// 	_ = e.Get(ctx, key)
// }

// Env[Key].(Value) wrapper
type readEnv struct {
	env Env // env.(cache)
	key Key // key.(read)
}

var _ Key = readEnv{}

func (c readEnv) String() string {
	return c.key.String()
}

func (c readEnv) Value(req Request) string {

	key := c.key.String()
	val, ok := c.env[key]

	if ok {
		return val
	}

	// resolve: once
	val = c.key.Value(req)
	c.env[key] = val

	return val
}
