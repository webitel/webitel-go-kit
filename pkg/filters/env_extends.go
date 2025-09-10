package filters

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var (
	// FunctionExtends provides common CEL environment functions
	FunctionExtends = map[string]cel.EnvOption{
		"isnull": cel.Function("isnull",
			cel.Overload("timestamp_null",
				[]*cel.Type{cel.TimestampType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
			cel.Overload("bool_null",
				[]*cel.Type{cel.BoolType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
			cel.Overload("int_null",
				[]*cel.Type{cel.IntType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
			cel.Overload("string_null",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
		),
		"notnull": cel.Function("notnull",
			cel.Overload("timestamp_null",
				[]*cel.Type{cel.TimestampType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
			cel.Overload("bool_null",
				[]*cel.Type{cel.BoolType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
			cel.Overload("int_null",
				[]*cel.Type{cel.IntType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
			cel.Overload("string_null",
				[]*cel.Type{cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
		),
		// timestamp is not considered as built-in CEL type, so we need to define equals overloads for it and comparison with unix
		"equals": cel.Function("equals",
			cel.Overload("timestamp_int",
				[]*cel.Type{cel.TimestampType, cel.IntType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
			cel.Overload("timestamp_timestamp",
				[]*cel.Type{cel.TimestampType, cel.TimestampType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
			cel.Overload("int_int",
				[]*cel.Type{cel.IntType, cel.IntType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
		),
		"like": cel.Function("like",
			cel.Overload("string_string",
				[]*cel.Type{cel.StringType, cel.StringType},
				cel.BoolType,
				cel.FunctionBinding(func(values ...ref.Val) ref.Val {
					return types.True
				}),
			),
		),
	}
)
