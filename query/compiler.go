package query

import (
	"fmt"
)

func compileExpression(node Node) CompiledExpression {
	switch n := node.(type) {
	case *ExpressionNode:
		left := compileExpression(n.Left)
		right := compileExpression(n.Right)
		return func(data interface{}) (interface{}, error) {
			lval, err := left(data)
			if err != nil {
				return nil, err
			}
			rval, err := right(data)
			if err != nil {
				return nil, err
			}
			// Perform operation based on n.Operator
			// This is a simplified example, you'll need to implement the actual operations
			return nil, fmt.Errorf("operator %s not implemented", n.Operator)
		}
	case *IdentifierNode:
		return func(data interface{}) (interface{}, error) {
			// Access the field in data
			// This is a simplified example, you'll need to implement the actual field access
			return nil, fmt.Errorf("field access not implemented")
		}
	case *ValueNode:
		return func(data interface{}) (interface{}, error) {
			return n.Value, nil
		}
	case *AnyNode:
		arrayExpr := compileExpression(n.Array)
		condition := compileExpression(n.Condition)
		return func(data interface{}) (interface{}, error) {
			arr, err := arrayExpr(data)
			if err != nil {
				return false, err
			}
			slice, ok := arr.([]interface{})
			if !ok {
				return false, fmt.Errorf("expected array, got %T", arr)
			}
			for _, item := range slice {
				match, err := condition(item)
				if err != nil {
					return false, err
				}
				if m, ok := match.(bool); ok && m {
					return true, nil
				}
			}
			return false, nil
		}
	case *AllNode:
		arrayExpr := compileExpression(n.Array)
		condition := compileExpression(n.Condition)
		return func(data interface{}) (interface{}, error) {
			arr, err := arrayExpr(data)
			if err != nil {
				return false, err
			}
			slice, ok := arr.([]interface{})
			if !ok {
				return false, fmt.Errorf("expected array, got %T", arr)
			}
			for _, item := range slice {
				match, err := condition(item)
				if err != nil {
					return false, err
				}
				if m, ok := match.(bool); ok && !m {
					return false, nil
				}
			}
			return true, nil
		}
	case *ArrayStarNode:
		arrayExpr := compileExpression(n.Array)
		return func(data interface{}) (interface{}, error) {
			return arrayExpr(data)
		}
	default:
		return func(data interface{}) (interface{}, error) {
			return nil, fmt.Errorf("unsupported node type: %T", n)
		}
	}
}

type CompiledExpression func(data interface{}) (interface{}, error)
