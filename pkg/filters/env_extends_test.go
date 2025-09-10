// go
package filters

import (
	"fmt"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	stubs "github.com/webitel/webitel-go-kit/pkg/filters/test_stubs/gen"
)

func newEnvWithFns(t *testing.T, fns ...string) *cel.Env {
	t.Helper()
	opts := make([]cel.EnvOption, 0, len(fns)+4)
	opts = append(opts, ProtoToCELVariables(&stubs.TestingObject{})...)
	for _, fn := range fns {
		opt, ok := FunctionExtends[fn]
		if !ok {
			t.Fatalf("unknown function: %s", fn)
		}
		opts = append(opts, opt)
	}
	env, err := cel.NewEnv(opts...)
	if err != nil {
		t.Fatalf("cel.NewEnv error: %v", err)
	}
	return env
}

func compile(env *cel.Env, expr string) (*cel.Ast, error) {
	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	return ast, nil
}

func eval(env *cel.Env, ast *cel.Ast, vars map[string]any) (ref.Val, error) {
	prg, err := env.Program(ast)
	if err != nil {
		return nil, err
	}
	out, _, err := prg.Eval(vars)
	return out, err
}

func mustEvalTrue(t *testing.T, env *cel.Env, expr string, vars map[string]any) {
	t.Helper()
	ast, err := compile(env, expr)
	if err != nil {
		t.Fatalf("compile error for %q: %v", expr, err)
	}
	out, err := eval(env, ast, vars)
	if err != nil {
		t.Fatalf("eval error for %q: %v", expr, err)
	}
	if b, ok := out.Value().(bool); !ok || !b {
		t.Fatalf("expected true for %q, got: %v", expr, out)
	}
}

func mustCompileError(t *testing.T, env *cel.Env, expr string) {
	t.Helper()
	if _, err := compile(env, expr); err == nil {
		t.Fatalf("expected compile error for %q, got nil", expr)
	}
}

func Test_FunctionExtends_isnull(t *testing.T) {
	env := newEnvWithFns(t, "isnull")

	// Valid overloads
	mustEvalTrue(t, env, "isnull(description)", map[string]any{"description": "x"})
	mustEvalTrue(t, env, "isnull(state)", map[string]any{"state": true})
	mustEvalTrue(t, env, "isnull(id)", map[string]any{"id": int64(1)})
	mustEvalTrue(t, env, "isnull(timestamp('2020-01-01T00:00:00Z'))", nil)

	// Invalid arity
	mustCompileError(t, env, "isnull()")

	// Invalid type (no message overload)
	mustCompileError(t, env, "isnull(created_by)")
}

func Test_FunctionExtends_notnull(t *testing.T) {
	env := newEnvWithFns(t, "notnull")

	// Valid overloads
	mustEvalTrue(t, env, "notnull(description)", map[string]any{"description": "x"})
	mustEvalTrue(t, env, "notnull(state)", map[string]any{"state": true})
	mustEvalTrue(t, env, "notnull(id)", map[string]any{"id": int64(1)})
	mustEvalTrue(t, env, "notnull(timestamp('2020-01-01T00:00:00Z'))", nil)

	// Invalid arity
	mustCompileError(t, env, "notnull()")

	// Invalid type (no message overload)
	mustCompileError(t, env, "notnull(created_by)")
}

func Test_FunctionExtends_equals(t *testing.T) {
	env := newEnvWithFns(t, "equals")

	// Valid overloads
	mustEvalTrue(t, env, "equals(1, 1)", nil)                                                                 // int_int
	mustEvalTrue(t, env, "equals(id, 1)", map[string]any{"id": int64(1)})                                     // int_int with var
	mustEvalTrue(t, env, "equals(timestamp('2020-01-01T00:00:00Z'), 0)", nil)                                 // timestamp_int
	mustEvalTrue(t, env, "equals(timestamp('2020-01-01T00:00:00Z'), timestamp('2020-01-01T00:00:00Z'))", nil) // timestamp_timestamp

	// Invalid arity
	mustCompileError(t, env, "equals(1)")
	mustCompileError(t, env, "equals(1, 2, 3)")

	// Invalid types (no string\_int or int\_string overload)
	mustCompileError(t, env, "equals('1', 1)")
	mustCompileError(t, env, "equals(1, '1')")
}

func Test_FunctionExtends_like(t *testing.T) {
	env := newEnvWithFns(t, "like")

	// Valid overloads
	mustEvalTrue(t, env, "like('abc', 'a%')", nil)
	mustEvalTrue(t, env, "like(name, 'a%')", map[string]any{"name": "alice"})

	// Invalid arity
	mustCompileError(t, env, "like('a%')")
	mustCompileError(t, env, "like('a', 'b', 'c')")

	// Invalid types
	mustCompileError(t, env, "like(id, '1')")     // int, string
	mustCompileError(t, env, "like(name, 1)")     // string, int
	mustCompileError(t, env, "like(state, 't%')") // bool, string
}

func Test_FunctionExtends_each_function_is_registered(t *testing.T) {
	// Ensure each key in FunctionExtends can build an env without errors in isolation.
	for fn := range FunctionExtends {
		t.Run(fmt.Sprintf("env-%s", fn), func(t *testing.T) {
			_ = newEnvWithFns(t, fn)
		})
	}
}
