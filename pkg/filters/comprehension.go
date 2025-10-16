package filters

import (
	"fmt"

	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// parseComprehensionExpr handles comprehension expressions.
func parseComprehensionExpr(call *expr.Expr_Comprehension) (*FilterExpr, error) {
	// get loop step to determine the operation
	var (
		parent      = call.GetIterRange().GetIdentExpr().GetName()
		parentAlias = call.GetIterVar()
		loopStep    = call.LoopStep
	)
	if loopStep == nil {
		return nil, fmt.Errorf("comprehension expression missing loop step")
	}
	callExpression := loopStep.GetCallExpr()
	if callExpression == nil {
		return nil, fmt.Errorf("comprehension loop step is not a call expression")
	}
	if len(callExpression.Args) < 2 {
		return nil, fmt.Errorf("comprehension loop step must have 2 arguments")
	}
	for _, arg := range callExpression.Args {
		if arg.GetCallExpr() == nil {
			continue
		}
		return parseExprWithAlias(arg, map[string]string{parentAlias: parent})
	}
	return nil, fmt.Errorf("could not parse comprehension expression")
}

// parseExpr recursively parses a CEL expression into a Filterer structure.
func parseExprWithAlias(s *expr.Expr, alias map[string]string) (*FilterExpr, error) {
	switch e := s.ExprKind.(type) {
	case *expr.Expr_CallExpr:
		return parseCallExprWithAlias(e.CallExpr, alias)
	case *expr.Expr_IdentExpr:
		return nil, fmt.Errorf("standalone identifier not supported")
	case *expr.Expr_ConstExpr:
		return nil, fmt.Errorf("standalone constant not supported")
	case *expr.Expr_ComprehensionExpr:
		return parseComprehensionExpr(e.ComprehensionExpr)
	default:
		return nil, fmt.Errorf("unsupported expression type")
	}
}

// parseCallExpr handles function call expressions and delegates to specific parsers based on the function name.
func parseCallExprWithAlias(call *expr.Expr_Call, alias map[string]string) (*FilterExpr, error) {
	switch call.Function {
	case "_&&_", "_||_":
		return parseLogicalExprWithAlias(call, alias)
	case "_==_", "_!=_", "_>_", "_>=_", "_<_", "_<=_", "like":
		return parseComparisonExprWithAlias(call, alias)
	case "isnull", "notnull":
		return parseNullExpr(call)
	default:
		return nil, fmt.Errorf("unsupported function: %s", call.Function)
	}
}

// parseLogicalExpr parses logical expressions (AND, OR) into a FilterNode.
func parseLogicalExprWithAlias(call *expr.Expr_Call, alias map[string]string) (*FilterExpr, error) {
	if len(call.Args) != 2 {
		return nil, fmt.Errorf("logical expression must have 2 arguments")
	}

	var connection ConnectionType
	if call.Function == "_&&_" {
		connection = And
	} else {
		connection = Or
	}

	left, err := parseExprWithAlias(call.Args[0], alias)
	if err != nil {
		return nil, err
	}

	right, err := parseExprWithAlias(call.Args[1], alias)
	if err != nil {
		return nil, err
	}

	return &FilterExpr{filter: &FilterNode{
		Nodes:      []*FilterExpr{left, right},
		Connection: connection,
	}}, nil
}

// parseComparisonExpr parses comparison expressions into a Filter.
func parseComparisonExprWithAlias(call *expr.Expr_Call, alias map[string]string) (*FilterExpr, error) {
	if len(call.Args) != 2 {
		return nil, fmt.Errorf("comparison expression must have 2 arguments")
	}

	column, err := ExtractIdentifier(call.Args[0], alias)
	if err != nil {
		return nil, err
	}

	value, err := extractConstant(call.Args[1])
	if err != nil {
		return nil, err
	}

	comparison, err := mapCELArrayComparison(call.Function)
	if err != nil {
		return nil, err
	}

	return &FilterExpr{&Filter{
		Column:         column,
		Value:          value,
		ComparisonType: comparison,
	}}, nil
}

// mapCELComparison maps CEL comparison function names to Comparison types.
func mapCELArrayComparison(function string) (Comparison, error) {
	switch function {
	case "_==_":
		return Contains, nil
	case "_!=_":
		return NotContains, nil
	default:
		return 0, fmt.Errorf("unsupported comparison: %s", function)
	}
}
