package query

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"reflect"
	"regexp"
	"strings"
)

type CompiledExpression func(data interface{}) (interface{}, error)

func CompileExpression(node Node) CompiledExpression {
	switch n := node.(type) {
	case *ExpressionNode:
		left := CompileExpression(n.Left)
		var right CompiledExpression
		if n.Operator == "." {
			// For the '.' operator, we don't compile the right side
			// as it should be treated as a string identifier
			right = func(data interface{}) (interface{}, error) {
				if identNode, ok := n.Right.(*IdentifierNode); ok {
					return identNode.Name, nil
				}
				return nil, fmt.Errorf("right side of '.' must be an identifier")
			}
		} else {
			right = CompileExpression(n.Right)
		}
		return func(data interface{}) (interface{}, error) {
			lval, err := left(data)
			if err != nil {
				return nil, err
			}
			rval, err := right(data)
			if err != nil {
				return nil, err
			}
			var ret interface{}
			ret, err = evaluateOperation(n.Operator, lval, rval)
			//log.Printf("Evaluated operation: `%v, %v, %v` to `%v, %v`", n.Operator, lval, rval, ret, err)
			return ret, err
		}
	case *IdentifierNode:
		return func(data interface{}) (interface{}, error) {
			return getField(data, strings.Split(n.Name, "."))
		}
	case *ValueNode:
		return func(data interface{}) (interface{}, error) {
			//log.Printf("Returning n.Value of %v", n.Value)
			return n.Value, nil
		}
	case *FunctionNode:
		args := make([]CompiledExpression, len(n.Arguments))
		for i, arg := range n.Arguments {
			args[i] = CompileExpression(arg)
		}
		return func(data interface{}) (interface{}, error) {
			if n.Name == "DOES_NOT_EXIST" {
				if len(n.Arguments) != 1 {
					return nil, fmt.Errorf("DOES_NOT_EXIST function requires exactly one argument")
				}

				identifier, ok := n.Arguments[0].(*IdentifierNode)
				if !ok {
					return nil, fmt.Errorf("DOES_NOT_EXIST function argument must be an identifier but got %T", n.Arguments[0])
				}
				dataMap, ok := data.(map[string]interface{})
				if !ok {
					return false, nil // If data is not a map, the key doesn't exist
				}
				_, exists := dataMap[identifier.Name]
				return !exists, nil
			}
			return evaluateFunction(n.Name, args, data)
		}
	case *ParameterNode:
		return func(data interface{}) (interface{}, error) {
			params, ok := data.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("parameters not provided")
			}
			value, exists := params[n.Name]
			if !exists {
				return nil, fmt.Errorf("parameter %s not provided", n.Name)
			}
			return value, nil
		}
	case *ArrayNode:
		elements := make([]CompiledExpression, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = CompileExpression(elem)
		}
		return func(data interface{}) (interface{}, error) {
			result := make([]interface{}, len(elements))
			for i, elem := range elements {
				val, err := elem(data)
				if err != nil {
					return nil, err
				}
				result[i] = val
			}
			return result, nil
		}
	case *AnyNode:
		arrayExpr := CompileExpression(n.Array)
		condition := CompileExpression(n.Condition)
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
		arrayExpr := CompileExpression(n.Array)
		condition := CompileExpression(n.Condition)
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
		arrayExpr := CompileExpression(n.Array)
		return func(data interface{}) (interface{}, error) {
			return arrayExpr(data)
		}
	case nil:
		return func(data interface{}) (interface{}, error) {
			return nil, nil
		}
	default:
		return func(data interface{}) (interface{}, error) {
			return nil, fmt.Errorf("unsupported node type: %T", n)
		}
	}
}

func evaluateOperation(operator string, left, right interface{}) (interface{}, error) {
	//log.Printf("Evaluating operation: %s, %v, %v", operator, left, right)
	//log.Printf("Type of left: %T", left)
	//log.Printf("Type of right: %T", right)
	switch operator {
	case "==":
		return reflect.DeepEqual(left, right), nil
	case "!=":
		return !reflect.DeepEqual(left, right), nil
	case ">", ">=", "<", "<=":
		return compareValues(operator, left, right)
	case "AND":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("AND operation requires boolean operands")
		}
		return l && r, nil
	case "OR":
		l, lok := left.(bool)
		if !lok {
			return nil, fmt.Errorf("OR operation requires boolean operands, got %T for left operand", left)
		}
		if l {
			return true, nil // Short-circuit if left operand is true
		}
		r, rok := right.(bool)
		if !rok {
			return nil, fmt.Errorf("OR operation requires boolean operands, got %T for right operand", right)
		}
		return r, nil
	case "NOT":
		r, rok := right.(bool)
		if !rok {
			return nil, fmt.Errorf("NOT operation requires a boolean operand")
		}
		return !r, nil
	case "IN":
		return evaluateIn(left, right)
	case "NOT_IN":
		inResult, err := evaluateIn(left, right)
		if err != nil {
			return nil, err
		}
		return !inResult, nil
	case "CONTAINS":
		return evaluateContains(left, right)
	case "STARTS_WITH":
		return evaluateStartsWith(left, right)
	case "ENDS_WITH":
		return evaluateEndsWith(left, right)
	case "MATCHES":
		return evaluateMatches(left, right)
	case ".":
		switch leftVal := left.(type) {
		case map[string]interface{}:
			rightIdent, ok := right.(string)
			if !ok {
				return nil, fmt.Errorf("right operand of '.' must be a string identifier, got %T", right)
			}
			value, exists := leftVal[rightIdent]
			if !exists {
				return nil, fmt.Errorf("key '%s' not found in map", rightIdent)
			}
			return value, nil
		case []interface{}:
			rightIdent, ok := right.(string)
			if !ok {
				return nil, fmt.Errorf("right operand of '.' must be a string identifier, got %T", right)
			}
			if rightIdent == "length" {
				return float64(len(leftVal)), nil
			}
			return nil, fmt.Errorf("invalid operation on array: %s", rightIdent)
		default:
			return nil, fmt.Errorf("left operand of '.' must be a map or array, got %T", left)
		}
	case "[]":
		leftArray, ok := left.([]interface{})
		if !ok {
			return nil, fmt.Errorf("left operand of '[]' must be an array, got %T", left)
		}

		index, err := toFloat64(right)
		if err != nil {
			return nil, fmt.Errorf("right operand of '[]' must be a number, got %T", right)
		}

		roundedIndex := int(math.Round(index))
		if roundedIndex < 0 || roundedIndex >= len(leftArray) {
			return nil, nil // Out of bounds, return nil
		}

		return leftArray[roundedIndex], nil
	default:
		return nil, fmt.Errorf("unsupported operator: %s", operator)
	}
}

func compareValues(operator string, left, right interface{}) (bool, error) {
	lv := reflect.ValueOf(left)

	switch lv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		l := lv.Int()
		r, err := toInt64(right)
		if err != nil {
			return false, err
		}
		switch operator {
		case ">":
			return l > r, nil
		case ">=":
			return l >= r, nil
		case "<":
			return l < r, nil
		case "<=":
			return l <= r, nil
		}
	case reflect.Float32, reflect.Float64:
		l := lv.Float()
		r, err := toFloat64(right)
		if err != nil {
			return false, err
		}
		switch operator {
		case ">":
			return l > r, nil
		case ">=":
			return l >= r, nil
		case "<":
			return l < r, nil
		case "<=":
			return l <= r, nil
		}
	case reflect.String:
		l := lv.String()
		r, ok := right.(string)
		if !ok {
			return false, fmt.Errorf("cannot compare string with non-string")
		}
		switch operator {
		case ">":
			return l > r, nil
		case ">=":
			return l >= r, nil
		case "<":
			return l < r, nil
		case "<=":
			return l <= r, nil
		}
	}
	return false, fmt.Errorf("unsupported comparison: %v %s %v", left, operator, right)
}

func evaluateFunction(name string, args []CompiledExpression, data interface{}) (interface{}, error) {
	switch name {
	case "LENGTH":
		if len(args) != 1 {
			return nil, fmt.Errorf("LENGTH function requires exactly one argument")
		}
		arg, err := args[0](data)
		if err != nil {
			return nil, err
		}
		return evaluateLength(arg)
	case "EXISTS":
		if len(args) != 1 {
			return nil, fmt.Errorf("EXISTS function requires exactly one argument")
		}
		_, err := args[0](data)
		return err == nil, nil
	case "DOES_NOT_EXIST":
		if len(args) != 1 {
			return nil, fmt.Errorf("DOES_NOT_EXIST function requires exactly one argument")
		}
		arg, err := args[0](data)
		if err != nil {
			return nil, err
		}
		identifier, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("DOES_NOT_EXIST function argument must be an identifier")
		}
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return false, nil // If data is not a map, the key doesn't exist
		}
		_, exists := dataMap[identifier]
		return !exists, nil
	default:
		return nil, fmt.Errorf("unsupported function: %s", name)
	}
}

func evaluateLength(arg interface{}) (int, error) {
	switch v := arg.(type) {
	case string:
		return len(v), nil
	case []interface{}:
		return len(v), nil
	case map[string]interface{}:
		return len(v), nil
	default:
		return 0, fmt.Errorf("LENGTH function not supported for type %T", arg)
	}
}

func evaluateIn(left, right interface{}) (bool, error) {
	list, ok := right.([]interface{})
	if !ok {
		return false, fmt.Errorf("IN operator requires a list on the right side")
	}
	log.Printf("Evaluating IN: %v, %v", left, list)
	for _, item := range list {
		log.Printf("Comparing %v(type: %T) with %v(type: %T)", left, left, item, item)
		if reflect.DeepEqual(left, item) {
			return true, nil
		}
	}
	log.Printf("IN evaluation result: %v", false)
	return false, nil
}

func evaluateContains(left, right interface{}) (bool, error) {
	l, lok := left.(string)
	r, rok := right.(string)
	if !lok || !rok {
		return false, fmt.Errorf("CONTAINS operation requires string operands")
	}
	return strings.Contains(l, r), nil
}

func evaluateStartsWith(left, right interface{}) (bool, error) {
	l, lok := left.(string)
	r, rok := right.(string)
	if !lok || !rok {
		return false, fmt.Errorf("STARTS_WITH operation requires string operands")
	}
	return strings.HasPrefix(l, r), nil
}

func evaluateEndsWith(left, right interface{}) (bool, error) {
	l, lok := left.(string)
	r, rok := right.(string)
	if !lok || !rok {
		return false, fmt.Errorf("ENDS_WITH operation requires string operands")
	}
	return strings.HasSuffix(l, r), nil
}

func evaluateMatches(left, right interface{}) (bool, error) {
	l, lok := left.(string)
	r, rok := right.(string)
	if !lok || !rok {
		return false, fmt.Errorf("MATCHES operation requires string operands")
	}
	matched, err := regexp.MatchString(r, l)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %v", err)
	}
	return matched, nil
}

func getField(data interface{}, path []string) (interface{}, error) {
	current := data
	for _, key := range path {
		switch v := current.(type) {
		case map[string]interface{}:
			return v[key], nil
		case []interface{}:
			if key == "*" {
				return v, nil
			}
			return nil, fmt.Errorf("cannot use dot notation on array")
		default:
			return nil, fmt.Errorf("cannot access field %s on %T", key, current)
		}
	}
	return current, nil
}

func toInt64(v interface{}) (int64, error) {
	switch i := v.(type) {
	case int:
		return int64(i), nil
	case int64:
		return i, nil
	case float64:
		return int64(i), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

func toFloat64(v interface{}) (float64, error) {
	switch f := v.(type) {
	case float64:
		return f, nil
	case int:
		return float64(f), nil
	case int64:
		return float64(f), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func CreateFilterFunction(compiledExpr CompiledExpression) func([]byte) (bool, error) {
	return func(record []byte) (bool, error) {
		var data interface{}
		err := json.Unmarshal(record, &data)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal JSON: %v", err)
		}

		result, err := compiledExpr(data)
		if err != nil {
			return false, err
		}

		boolResult, ok := result.(bool)
		if !ok {
			return false, fmt.Errorf("query result is not a boolean: %v", result)
		}

		return boolResult, nil
	}
}
