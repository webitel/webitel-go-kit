package filters

import (
	"fmt"

	"github.com/google/cel-go/cel"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	ErrEmptyQuery     = fmt.Errorf("empty query")
	ErrNilEnvironment = fmt.Errorf("nil CEL environment provided")
)

// ParseFilters parses a CEL expression string into a Filterer tree structure using the provided CEL environment.
func ParseFilters(env *cel.Env, query string) (*FilterExpr, error) {
	if query == "" {
		return nil, ErrEmptyQuery
	}
	if env == nil {
		return nil, ErrNilEnvironment
	}
	extensions := make([]cel.EnvOption, 0, len(FunctionExtends))
	for _, ext := range FunctionExtends {
		extensions = append(extensions, ext)
	}
	env, err := env.Extend(extensions...)
	if err != nil {
		return nil, err
	}
	ast, iss := env.Compile(query)
	if err := iss.Err(); err != nil {
		return nil, err
	}
	checkedExpr, err := cel.AstToCheckedExpr(ast)
	if err != nil {
		return nil, err
	}
	return parseCELASTToFilter(checkedExpr.GetExpr())
}

// ExtractIdentifier extracts the full identifier path from a CEL expression.
func ExtractIdentifier(expr *expr.Expr) (string, error) {
	var depth int
	return extractIdentifier(expr, depth)
}

// ProtoToCELVariables converts a protobuf message's fields into CEL variable declarations.
// Also registers the message type itself in the CEL environment.
func ProtoToCELVariables(msg proto.Message) []cel.EnvOption {
	fields := msg.ProtoReflect().Descriptor().Fields()
	var opts []cel.EnvOption
	opts = append(opts, cel.Types(msg))
	for i := 0; i < fields.Len(); i++ {
		descriptor := fields.Get(i)
		var celType *cel.Type
		switch descriptor.Kind() {
		case protoreflect.StringKind:
			celType = cel.StringType
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
			protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
			protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
			protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			celType = cel.IntType
		case protoreflect.BoolKind:
			celType = cel.BoolType
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			celType = cel.DoubleType
		case protoreflect.MessageKind:
			name := descriptor.Message().FullName()
			celType = cel.ObjectType(string(name))
		}
		opts = append(opts, cel.Variable(string(descriptor.Name()), celType))
	}
	return opts
}
