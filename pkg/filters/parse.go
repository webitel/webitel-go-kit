package filters

import (
	"fmt"

	"google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// parseExpr recursively parses a CEL expression into a Filterer structure.
func parseExpr(s *expr.Expr) (*FilterExpr, error) {
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

// parseCallExpr handles function call expressions and delegates to specific parsers based on the function name.
func parseCallExpr(call *expr.Expr_Call) (*FilterExpr, error) {
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

// parseLogicalExpr parses logical expressions (AND, OR) into a FilterNode.
func parseLogicalExpr(call *expr.Expr_Call) (*FilterExpr, error) {
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

	return &FilterExpr{filter: &FilterNode{
		Nodes:      []*FilterExpr{left, right},
		Connection: connection,
	}}, nil
}

// parseComparisonExpr parses comparison expressions into a Filter.
func parseComparisonExpr(call *expr.Expr_Call) (*FilterExpr, error) {
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

	return &FilterExpr{&Filter{
		Column:         column,
		Value:          value,
		ComparisonType: comparison,
	}}, nil
}

// parseLikeExpr parses 'like' expressions into a Filter.
func parseLikeExpr(call *expr.Expr_Call) (*FilterExpr, error) {
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

	return &FilterExpr{&Filter{
		Column:         column,
		Value:          value,
		ComparisonType: Like,
	}}, nil
}

// parseCELASTToFilter converts a CEL AST expression into a Filterer structure.
func parseCELASTToFilter(expr *expr.Expr) (*FilterExpr, error) {
	return parseExpr(expr)
}

// extractIdentifier recursively extracts the full identifier path from a CEL expression.
func extractIdentifier(expression *expr.Expr, depth int) (string, error) {
	if selectExpr := expression.GetSelectExpr(); selectExpr != nil {
		nested, err := extractIdentifier(selectExpr.Operand, depth+1)
		if err != nil {
			return "", err
		}
		if nested == "" {
			return selectExpr.GetField(), nil
		}
		return fmt.Sprintf("%s.%s", nested, selectExpr.GetField()), nil
	} else if identExpr := expression.GetIdentExpr(); identExpr != nil {
		return identExpr.Name, nil
	}
	return "", nil
}

// extractConstant extracts the constant value from a CEL expression.
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

// mapCELComparison maps CEL comparison function names to Comparison types.
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
