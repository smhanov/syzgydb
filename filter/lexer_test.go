package query

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `age >= 18 AND status == "active"`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIdentifier, "age"},
		{TokenGreaterEqual, ">="},
		{TokenNumber, "18"},
		{TokenAnd, "AND"},
		{TokenIdentifier, "status"},
		{TokenEqual, "=="},
		{TokenString, "active"},
		{TokenEOF, ""},
	}

	lexer := NewLexer(input)

	for i, tt := range tests {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - token type wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLexerAdditionalCases(t *testing.T) {
	input := `name != "John" AND (age < 30 OR status IN ("active", "pending"))`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIdentifier, "name"},
		{TokenNotEqual, "!="},
		{TokenString, "John"},
		{TokenAnd, "AND"},
		{TokenLeftParen, "("},
		{TokenIdentifier, "age"},
		{TokenLess, "<"},
		{TokenNumber, "30"},
		{TokenOr, "OR"},
		{TokenIdentifier, "status"},
		{TokenIN, "IN"},
		{TokenLeftParen, "("},
		{TokenString, "active"},
		{TokenComma, ","},
		{TokenString, "pending"},
		{TokenRightParen, ")"},
		{TokenRightParen, ")"},
		{TokenEOF, ""},
	}

	lexer := NewLexer(input)

	for i, tt := range tests {
		tok := lexer.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - token type wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}
