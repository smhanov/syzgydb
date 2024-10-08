package query

import (
	"log"
	"testing"
)

func TestParser(t *testing.T) {
	testCases := []struct {
		input    string
		expected string // This would be a string representation of the expected AST
	}{
		{
			input:    "age >= 18 AND status == 'active'",
			expected: "AND(>=(age, 18), ==(status, 'active'))",
		},
		{
			input:    "name STARTS_WITH 'J' OR name ENDS_WITH 'son'",
			expected: "OR(STARTS_WITH(name, 'J'), ENDS_WITH(name, 'son'))",
		},
		{
			input:    "tags CONTAINS 'urgent' AND priority > 5",
			expected: "AND(CONTAINS(tags, 'urgent'), >(priority, 5))",
		},
		{
			input:    "NOT (status == 'inactive' OR lastLogin < '2023-01-01')",
			expected: "NOT(OR(==(status, 'inactive'), <(lastLogin, '2023-01-01')))",
		},
		{
			input:    "age IN [18, 21, 25] AND country NOT IN ['US', 'CA']",
			expected: "AND(IN(age, [18, 21, 25]), NOT_IN(country, ['US', 'CA']))",
		},
		{
			input:    "middleName EXISTS AND nickname DOES NOT EXIST",
			expected: "AND(EXISTS(middleName), DOES_NOT_EXIST(nickname))",
		},
		/*{
			input:    "ANY(orders[*] > 1000) AND ALL(ratings[*] >= 4)",
			expected: "AND(ANY(>(orders[*], 1000)), ALL(>=(ratings[*], 4)))",
		},*/
		{
			input:    "items.length > 0 AND items[0].price < 100",
			expected: "AND(>(.(items, length), 0), <(.([](items, 0), price), 100))",
		},
		{
			input:    "user.profile.completed == true AND user.age >= :minAge",
			expected: "AND(==(.(.(user, profile), completed), true), >=(.(user, age), :minAge))",
		},
		{
			input:    "(status == 'active' AND age >= 18) OR role == 'admin'",
			expected: "OR(AND(==(status, 'active'), >=(age, 18)), ==(role, 'admin'))",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			log.Printf("Parsing input: %s", tc.input)
			lexer := NewLexer(tc.input)
			parser := NewParser(lexer)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parser error: %v", err)
			}
			result := ast.String() // Assume we have a String() method on Node interface
			if result != tc.expected {
				t.Errorf("Expected AST %s, got %s", tc.expected, result)
			}
		})
	}
}
