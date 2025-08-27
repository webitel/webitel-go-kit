package filters

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// ParseFilters parses a CEL expression string into a Filterer tree structure using the provided CEL environment.
func ParseFilters(env *cel.Env, query string) (Filterer, error) {
	ast, iss := env.Compile(query)
	if err := iss.Err(); err != nil {
		return nil, err
	}
	expr, err := cel.AstToCheckedExpr(ast)
	if err != nil {
		return nil, err
	}
	return parseCELASTToFilter(expr.GetExpr())
}

func parseCELASTToFilter(expr *expr.Expr) (Filterer, error) {
	return parseExpr(expr)
}

func parseExpr(s *expr.Expr) (Filterer, error) {
	switch e := s.ExprKind.(type) {
	case *expr.Expr_CallExpr:
		return parseCallExpr(e.CallExpr)
	case *expr.Expr_IdentExpr:
		return nil, fmt.Errorf("standalone identifier not supported")
	case *expr.Expr_ConstExpr:
		return nil, fmt.Errorf("standalone constant not supported")
	default:
		return nil, fmt.Errorf("unsupported expression type")
	}
}

func parseCallExpr(call *expr.Expr_Call) (Filterer, error) {
	switch call.Function {
	case "_&&_", "_||_":
		return parseLogicalExpr(call)
	case "_==_", "_!=_", "_>_", "_>=_", "_<_", "_<=_":
		return parseComparisonExpr(call)
	case "contains", "matches":
		return parseLikeExpr(call)
	default:
		return nil, fmt.Errorf("unsupported function: %s", call.Function)
	}
}

func parseLogicalExpr(call *expr.Expr_Call) (*FilterNode, error) {
	if len(call.Args) != 2 {
		return nil, fmt.Errorf("logical expression must have 2 arguments")
	}

	var connection ConnectionType
	if call.Function == "_&&_" {
		connection = And
	} else {
		connection = Or
	}

	left, err := parseExpr(call.Args[0])
	if err != nil {
		return nil, err
	}

	right, err := parseExpr(call.Args[1])
	if err != nil {
		return nil, err
	}

	return &FilterNode{
		Nodes:      []Filterer{left, right},
		Connection: connection,
	}, nil
}

func parseComparisonExpr(call *expr.Expr_Call) (*Filter, error) {
	if len(call.Args) != 2 {
		return nil, fmt.Errorf("comparison expression must have 2 arguments")
	}

	column, err := ExtractIdentifier(call.Args[0])
	if err != nil {
		return nil, err
	}

	value, err := extractConstant(call.Args[1])
	if err != nil {
		return nil, err
	}

	comparison, err := mapCELComparison(call.Function)
	if err != nil {
		return nil, err
	}

	return &Filter{
		Column:         column,
		Value:          value,
		ComparisonType: comparison,
	}, nil
}

func parseLikeExpr(call *expr.Expr_Call) (*Filter, error) {
	if len(call.Args) != 2 {
		return nil, fmt.Errorf("like expression must have 2 arguments")
	}

	column, err := ExtractIdentifier(call.Args[0])
	if err != nil {
		return nil, err
	}

	value, err := extractConstant(call.Args[1])
	if err != nil {
		return nil, err
	}

	return &Filter{
		Column:         column,
		Value:          value,
		ComparisonType: Like,
	}, nil
}

func ExtractIdentifier(expr *expr.Expr) (string, error) {
	var depth int
	return extractIdentifier(expr, depth)
}

func extractIdentifier(expr *expr.Expr, depth int) (string, error) {
	if ident := expr.GetSelectExpr(); ident != nil {
		nested, err := extractIdentifier(ident.Operand, depth+1)
		if err != nil {
			return "", err
		}
		if nested == "" {
			return ident.GetField(), nil
		}
		return fmt.Sprintf("%s.%s", nested, ident.GetField()), nil
	} else if depth == 0 {
		if ident := expr.GetIdentExpr(); ident != nil {
			return ident.Name, nil
		}
	}
	return "", nil
}

func extractConstant(s *expr.Expr) (any, error) {
	if constant := s.GetConstExpr(); constant != nil {
		switch v := constant.ConstantKind.(type) {
		case *expr.Constant_StringValue:
			return v.StringValue, nil
		case *expr.Constant_Int64Value:
			return v.Int64Value, nil
		case *expr.Constant_DoubleValue:
			return v.DoubleValue, nil
		case *expr.Constant_BoolValue:
			return v.BoolValue, nil
		default:
			return nil, fmt.Errorf("unsupported constant type")
		}
	}
	return nil, fmt.Errorf("expected constant")
}

func mapCELComparison(function string) (Comparison, error) {
	switch function {
	case "_==_":
		return Equal, nil
	case "_!=_":
		return NotEqual, nil
	case "_>_":
		return GreaterThan, nil
	case "_>=_":
		return GreaterThanOrEqual, nil
	case "_<_":
		return LessThan, nil
	case "_<=_":
		return LessThanOrEqual, nil
	default:
		return 0, fmt.Errorf("unsupported comparison: %s", function)
	}
}
