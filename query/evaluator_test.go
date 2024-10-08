package query

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCompileExpression(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		data     string
		expected interface{}
	}{
		{
			name:     "Simple Equality",
			node:     &ExpressionNode{Left: &IdentifierNode{Name: "age"}, Operator: "==", Right: &ValueNode{Value: 30}},
			data:     `{"age": 30}`,
			expected: true,
		},
		{
			name:     "Simple Inequality",
			node:     &ExpressionNode{Left: &IdentifierNode{Name: "age"}, Operator: "!=", Right: &ValueNode{Value: 25}},
			data:     `{"age": 30}`,
			expected: true,
		},
		{
			name:     "Greater Than",
			node:     &ExpressionNode{Left: &IdentifierNode{Name: "age"}, Operator: ">", Right: &ValueNode{Value: 25}},
			data:     `{"age": 30}`,
			expected: true,
		},
		{
			name:     "Less Than or Equal",
			node:     &ExpressionNode{Left: &IdentifierNode{Name: "age"}, Operator: "<=", Right: &ValueNode{Value: 30}},
			data:     `{"age": 30}`,
			expected: true,
		},
		{
			name: "Logical AND",
			node: &ExpressionNode{
				Left:     &ExpressionNode{Left: &IdentifierNode{Name: "age"}, Operator: ">", Right: &ValueNode{Value: 25}},
				Operator: "AND",
				Right:    &ExpressionNode{Left: &IdentifierNode{Name: "status"}, Operator: "==", Right: &ValueNode{Value: "active"}},
			},
			data:     `{"age": 30, "status": "active"}`,
			expected: true,
		},
		{
			name: "Logical OR",
			node: &ExpressionNode{
				Left:     &ExpressionNode{Left: &IdentifierNode{Name: "age"}, Operator: "<", Right: &ValueNode{Value: 25}},
				Operator: "OR",
				Right:    &ExpressionNode{Left: &IdentifierNode{Name: "status"}, Operator: "==", Right: &ValueNode{Value: "active"}},
			},
			data:     `{"age": 30, "status": "active"}`,
			expected: true,
		},
		{
			name: "Logical NOT",
			node: &ExpressionNode{
				Left:     nil,
				Operator: "NOT",
				Right:    &ExpressionNode{Left: &IdentifierNode{Name: "status"}, Operator: "==", Right: &ValueNode{Value: "inactive"}},
			},
			data:     `{"status": "active"}`,
			expected: true,
		},
		{
			name: "IN Operator",
			node: &ExpressionNode{
				Left:     &IdentifierNode{Name: "status"},
				Operator: "IN",
				Right:    &ArrayNode{Elements: []Node{&ValueNode{Value: "active"}, &ValueNode{Value: "pending"}}},
			},
			data:     `{"status": "active"}`,
			expected: true,
		},
		{
			name: "NOT IN Operator",
			node: &ExpressionNode{
				Left:     &IdentifierNode{Name: "status"},
				Operator: "NOT_IN",
				Right:    &ArrayNode{Elements: []Node{&ValueNode{Value: "inactive"}, &ValueNode{Value: "pending"}}},
			},
			data:     `{"status": "active"}`,
			expected: true,
		},
		{
			name: "CONTAINS Operator",
			node: &ExpressionNode{
				Left:     &IdentifierNode{Name: "description"},
				Operator: "CONTAINS",
				Right:    &ValueNode{Value: "urgent"},
			},
			data:     `{"description": "This is an urgent message"}`,
			expected: true,
		},
		{
			name: "STARTS_WITH Operator",
			node: &ExpressionNode{
				Left:     &IdentifierNode{Name: "filename"},
				Operator: "STARTS_WITH",
				Right:    &ValueNode{Value: "report_"},
			},
			data:     `{"filename": "report_2023.pdf"}`,
			expected: true,
		},
		{
			name: "ENDS_WITH Operator",
			node: &ExpressionNode{
				Left:     &IdentifierNode{Name: "email"},
				Operator: "ENDS_WITH",
				Right:    &ValueNode{Value: "@example.com"},
			},
			data:     `{"email": "user@example.com"}`,
			expected: true,
		},
		{
			name: "MATCHES Operator",
			node: &ExpressionNode{
				Left:     &IdentifierNode{Name: "username"},
				Operator: "MATCHES",
				Right:    &ValueNode{Value: "^[a-z0-9_]{3,16}$"},
			},
			data:     `{"username": "john_doe123"}`,
			expected: true,
		},
		{
			name: "EXISTS Function",
			node: &FunctionNode{
				Name:      "EXISTS",
				Arguments: []Node{&IdentifierNode{Name: "optional_field"}},
			},
			data:     `{"optional_field": "value"}`,
			expected: true,
		},
		{
			name: "DOES_NOT_EXIST Function",
			node: &FunctionNode{
				Name:      "DOES_NOT_EXIST",
				Arguments: []Node{&IdentifierNode{Name: "optional_field"}},
			},
			data:     `{"other_field": "value"}`,
			expected: true,
		},
		{
			name: "LENGTH Function",
			node: &ExpressionNode{
				Left: &FunctionNode{
					Name:      "LENGTH",
					Arguments: []Node{&IdentifierNode{Name: "tags"}},
				},
				Operator: ">=",
				Right:    &ValueNode{Value: 3},
			},
			data:     `{"tags": ["red", "green", "blue", "yellow"]}`,
			expected: true,
		},
		{
			name: "ANY Function",
			node: &AnyNode{
				Array: &IdentifierNode{Name: "items"},
				Condition: &ExpressionNode{
					Left:     &IdentifierNode{Name: "quantity"},
					Operator: ">",
					Right:    &ValueNode{Value: 100},
				},
			},
			data:     `{"items": [{"quantity": 50}, {"quantity": 150}, {"quantity": 75}]}`,
			expected: true,
		},
		{
			name: "ALL Function",
			node: &AllNode{
				Array: &IdentifierNode{Name: "grades"},
				Condition: &ExpressionNode{
					Left:     &IdentifierNode{Name: ""},
					Operator: ">=",
					Right:    &ValueNode{Value: 60},
				},
			},
			data:     `{"grades": [75, 80, 90, 65]}`,
			expected: true,
		},
		{
			name: "Nested Field Access",
			node: &ExpressionNode{
				Left:     &IdentifierNode{Name: "user.profile.completed"},
				Operator: "==",
				Right:    &ValueNode{Value: true},
			},
			data:     `{"user": {"profile": {"completed": true}}}`,
			expected: true,
		},
		{
			name: "Array Access",
			node: &ExpressionNode{
				Left:     &IdentifierNode{Name: "items[0].name"},
				Operator: "==",
				Right:    &ValueNode{Value: "first item"},
			},
			data:     `{"items": [{"name": "first item"}, {"name": "second item"}]}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiledExpr := CompileExpression(tt.node)
			var data interface{}
			err := json.Unmarshal([]byte(tt.data), &data)
			if err != nil {
				t.Fatalf("Failed to unmarshal test data: %v", err)
			}

			result, err := compiledExpr(data)
			if err != nil {
				t.Fatalf("Evaluation failed: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCreateFilterFunction(t *testing.T) {
	tests := []struct {
		name  string
		query string
		data  string
		want  bool
	}{
		{
			name:  "Simple equality",
			query: "age == 30",
			data:  `{"age": 30}`,
			want:  true,
		},
		{
			name:  "Complex condition",
			query: "(age >= 18 AND status == 'active') OR role == 'admin'",
			data:  `{"age": 25, "status": "active", "role": "user"}`,
			want:  true,
		},
		{
			name:  "Array operation",
			query: "ANY(scores[*] > 90)",
			data:  `{"scores": [85, 92, 78, 95]}`,
			want:  true,
		},
		{
			name:  "Nested field and string operation",
			query: "user.email ENDS_WITH '@example.com'",
			data:  `{"user": {"email": "john@example.com"}}`,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the query string into an AST
			lexer := NewLexer(tt.query)
			parser := NewParser(lexer)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			// Compile the AST
			compiledExpr := CompileExpression(ast)

			// Create the filter function
			filterFunc := CreateFilterFunction(compiledExpr)

			// Test the filter function
			got, err := filterFunc([]byte(tt.data))
			if err != nil {
				t.Fatalf("Filter function failed: %v", err)
			}

			if got != tt.want {
				t.Errorf("Filter function returned %v, want %v", got, tt.want)
			}
		})
	}
}
