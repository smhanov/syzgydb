package query

import (
	"fmt"
	"strconv"
	"strings"
)

type Node interface {
	String() string
}

type ExpressionNode struct {
	Left     Node
	Operator string
	Right    Node
}

func (n *ExpressionNode) String() string {
	if n.Left == nil {
		return fmt.Sprintf("%s(%s)", n.Operator, n.Right.String())
	}
	return fmt.Sprintf("%s(%s, %s)", n.Operator, n.Left.String(), n.Right.String())
}

type IdentifierNode struct {
	Name string
}

func (n *IdentifierNode) String() string {
	return n.Name
}

type ValueNode struct {
	Value interface{}
}

func (n *ValueNode) String() string {
	switch v := n.Value.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

type FunctionNode struct {
	Name      string
	Arguments []Node
}

func (n *FunctionNode) String() string {
	args := make([]string, len(n.Arguments))
	for i, arg := range n.Arguments {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", n.Name, strings.Join(args, ", "))
}

type ParameterNode struct {
	Name string
}

func (n *ParameterNode) String() string {
	return ":" + n.Name
}

type ArrayNode struct {
	Elements []Node
}

func (n *ArrayNode) String() string {
	elements := make([]string, len(n.Elements))
	for i, elem := range n.Elements {
		elements[i] = elem.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

type AnyNode struct {
	Array     Node
	Condition Node
}

func (n *AnyNode) String() string {
	return fmt.Sprintf("ANY(%s %s)", n.Array.String(), n.Condition.String())
}

type AllNode struct {
	Array     Node
	Condition Node
}

func (n *AllNode) String() string {
	return fmt.Sprintf("ALL(%s %s)", n.Array.String(), n.Condition.String())
}

type ArrayStarNode struct {
	Array Node
}

func (n *ArrayStarNode) String() string {
	return fmt.Sprintf("%s[*]", n.Array.String())
}

type Parser struct {
	lexer        *Lexer
	currentToken Token
	peekToken    Token
}

func NewParser(lexer *Lexer) *Parser {
	p := &Parser{lexer: lexer}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) Parse() (Node, error) {
	return p.parseExpression()
}

// Eexpression := LogicalExpression
func (p *Parser) parseExpression() (Node, error) {
	return p.parseOr()
}

// OrExpression := AndExpression (OR AndExpression)*
func (p *Parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.currentToken.Type == TokenOr {
		p.nextToken()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &ExpressionNode{Left: left, Operator: "OR", Right: right}
	}

	return left, nil
}

// AndExpression := NotExpression (AND NotExpression)*
func (p *Parser) parseAnd() (Node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.currentToken.Type == TokenAnd {
		p.nextToken()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &ExpressionNode{Left: left, Operator: "AND", Right: right}
	}

	return left, nil
}

// NotExpression := NOT? ComparisonExpression
func (p *Parser) parseNot() (Node, error) {
	if p.currentToken.Type == TokenNot {
		p.nextToken()
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &ExpressionNode{Left: nil, Operator: "NOT", Right: expr}, nil
	}
	return p.parsePrimary()
}

// PrimaryExpression := Identifier | Value | Parameter | GroupedExpression | ArrayLiteral
func (p *Parser) parsePrimary() (Node, error) {
	switch p.currentToken.Type {
	case TokenIdentifier:
		return p.parseIdentifierOrFunction()
	case TokenNumber:
		return p.parseNumber()
	case TokenString:
		tok := p.currentToken.Literal
		p.nextToken()
		return &ValueNode{Value: tok}, nil
	case TokenBoolean:
		return p.parseBoolean()
	case TokenNull:
		return &ValueNode{Value: nil}, nil
	case TokenLeftParen:
		return p.parseGroupedExpression()
	case TokenLeftBracket:
		return p.parseArrayLiteral()
	case TokenColon:
		return p.parseParameter()
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.currentToken.Literal)
	}
}

// IdentifierOrFunction := Identifier (DOT Identifier)* (LEFT_PAREN FunctionCallArguments? RIGHT_PAREN)? |
//
//	Identifier EXISTS | Identifier DOES_NOT_EXIST
func (p *Parser) parseIdentifierOrFunction() (Node, error) {
	expr, err := p.parseArrayAccessOrIdentifier()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type == TokenIN || p.currentToken.Type == TokenNot {
		return p.parseIn(expr)
	}

	if p.currentToken.Type == TokenLeftParen {
		return p.parseFunction(expr)
	}

	if p.currentToken.Type == TokenEXISTS {
		p.nextToken()
		return &FunctionNode{Name: "EXISTS", Arguments: []Node{expr}}, nil
	}

	if p.currentToken.Type == TokenDOESNOTEXIST {
		p.nextToken()
		return &FunctionNode{Name: "DOES_NOT_EXIST", Arguments: []Node{expr}}, nil
	}

	return expr, nil
}

// FunctionCall := Identifier LEFT_PAREN FunctionCallArguments? RIGHT_PAREN
func (p *Parser) parseFunction(expr Node) (Node, error) {
	if p.currentToken.Type != TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name, got %s", p.currentToken.Literal)
	}
	p.nextToken() // consume '('

	funcName, ok := expr.(*IdentifierNode)
	if !ok {
		return nil, fmt.Errorf("expected function name, got %T", expr)
	}

	args := []Node{}
	if p.currentToken.Type != TokenRightParen {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		for p.currentToken.Type == TokenComma {
			p.nextToken() // consume ','
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}

	if p.currentToken.Type != TokenRightParen {
		return nil, fmt.Errorf("expected ')' after function arguments, got %s", p.currentToken.Literal)
	}
	p.nextToken() // consume ')'

	return &FunctionNode{Name: funcName.Name, Arguments: args}, nil
}

/*
	func (p *Parser) parseAnyAllFunction() (Node, error) {
		funcName := p.currentToken.Literal
		p.nextToken() // consume 'ANY' or 'ALL'

		if p.currentToken.Type != TokenLeftParen {
			return nil, fmt.Errorf("expected '(' after %s, got %s", funcName, p.currentToken.Literal)
		}
		p.nextToken() // consume '('

		arrayExpr, err := p.parseArrayExpression()
		if err != nil {
			return nil, err
		}

		if p.currentToken.Type != TokenRightParen {
			condition, err := p.parseExpression()
			if err != nil {
				return nil, err
			}

			if p.currentToken.Type != TokenRightParen {
				return nil, fmt.Errorf("expected ')' after condition in %s function, got %s", funcName, p.currentToken.Literal)
			}
			p.nextToken() // consume ')'

			if funcName == "ANY" {
				return &AnyNode{Array: arrayExpr, Condition: condition}, nil
			} else {
				return &AllNode{Array: arrayExpr, Condition: condition}, nil
			}
		}

		return nil, fmt.Errorf("expected condition after array in %s function", funcName)
	}

func (p *Parser) parseArrayExpression() (Node, error) {
	expr, err := p.parseArrayAccessOrIdentifier()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type == TokenArrayStar {
		p.nextToken() // consume '[*]'
		return &ArrayStarNode{Array: expr}, nil
	}

	return expr, nil
}*/

func (p *Parser) parseArrayAccessOrIdentifier() (Node, error) {
	expr, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}

	for p.currentToken.Type == TokenLeftBracket || p.currentToken.Type == TokenDot {
		if p.currentToken.Type == TokenLeftBracket {
			p.nextToken() // consume '['
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.currentToken.Type != TokenRightBracket {
				return nil, fmt.Errorf("expected ']', got %s", p.currentToken.Literal)
			}
			p.nextToken() // consume ']'
			expr = &ExpressionNode{Left: expr, Operator: "[]", Right: index}
		} else if p.currentToken.Type == TokenDot {
			p.nextToken() // consume '.'
			if p.currentToken.Type != TokenIdentifier {
				return nil, fmt.Errorf("expected identifier after '.', got %s", p.currentToken.Literal)
			}
			fieldName := p.currentToken.Literal
			p.nextToken()
			expr = &ExpressionNode{Left: expr, Operator: ".", Right: &IdentifierNode{Name: fieldName}}
		}
	}

	return expr, nil
}

func (p *Parser) parseIdentifier() (Node, error) {
	if p.currentToken.Type != TokenIdentifier {
		return nil, fmt.Errorf("expected identifier, got %s", p.currentToken.Literal)
	}
	identifier := &IdentifierNode{Name: p.currentToken.Literal}
	p.nextToken()
	return identifier, nil
}

func (p *Parser) parseIn(expr Node) (Node, error) {
	operator := p.currentToken.Type
	p.nextToken() // consume 'IN' or 'NOT IN'

	// Change TokenIN to TokenNOTIN if the operator is "NOT IN"
	if operator == TokenNot && p.currentToken.Type == TokenIN {
		operator = TokenNOTIN
		p.nextToken() // consume 'NOT'
	}

	if p.currentToken.Type != TokenLeftBracket {
		return nil, fmt.Errorf("expected '[' after IN/NOT IN, got %s", p.currentToken.Literal)
	}

	arrayNode, err := p.parseArrayLiteral()
	if err != nil {
		return nil, err
	}

	operatorStr := "IN"
	if operator == TokenNOTIN {
		operatorStr = "NOT_IN"
	}

	return &ExpressionNode{
		Left:     expr,
		Operator: operatorStr,
		Right:    arrayNode,
	}, nil
}

// ComparisonExpression := NotExpression ComparisonOperator NotExpression
func (p *Parser) parseComparison() (Node, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	if p.isComparisonOperator(p.currentToken.Type) {
		operator := p.currentToken.Literal
		p.nextToken()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &ExpressionNode{Left: left, Operator: operator, Right: right}, nil
	}

	return left, nil
}

func (p *Parser) parseArrayLiteral() (Node, error) {
	p.nextToken() // consume '['
	elements := []Node{}

	if p.currentToken.Type != TokenRightBracket {
		element, err := p.parseArrayElement()
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)

		for p.currentToken.Type == TokenComma {
			p.nextToken() // consume ','
			element, err := p.parseArrayElement()
			if err != nil {
				return nil, err
			}
			elements = append(elements, element)
		}
	}

	if p.currentToken.Type != TokenRightBracket {
		return nil, fmt.Errorf("expected ']', got %s", p.currentToken.Literal)
	}
	p.nextToken() // consume ']'

	return &ArrayNode{Elements: elements}, nil
}

func (p *Parser) parseArrayElement() (Node, error) {
	switch p.currentToken.Type {
	case TokenNumber:
		return p.parseNumber()
	case TokenString:
		value := p.currentToken.Literal
		p.nextToken()
		return &ValueNode{Value: value}, nil
	default:
		return nil, fmt.Errorf("expected number or string in array, got %s", p.currentToken.Literal)
	}
}

func (p *Parser) parseParameter() (Node, error) {
	p.nextToken() // consume ':'
	if p.currentToken.Type != TokenIdentifier {
		return nil, fmt.Errorf("expected identifier after ':', got %s", p.currentToken.Literal)
	}
	param := &ParameterNode{Name: p.currentToken.Literal}
	p.nextToken()
	return param, nil
}

func (p *Parser) parseNumber() (Node, error) {
	value, err := strconv.ParseFloat(p.currentToken.Literal, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse number: %s", p.currentToken.Literal)
	}
	p.nextToken()

	return &ValueNode{Value: value}, nil
}

func (p *Parser) parseBoolean() (Node, error) {
	value, err := strconv.ParseBool(p.currentToken.Literal)
	if err != nil {
		return nil, fmt.Errorf("could not parse boolean: %s", p.currentToken.Literal)
	}
	p.nextToken()

	return &ValueNode{Value: value}, nil
}

func (p *Parser) parseGroupedExpression() (Node, error) {
	p.nextToken() // consume '('
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != TokenRightParen {
		return nil, fmt.Errorf("expected ')', got %s", p.currentToken.Literal)
	}
	p.nextToken() // consume ')'

	return expr, nil
}

func (p *Parser) isComparisonOperator(tokenType TokenType) bool {
	return tokenType == TokenEqual || tokenType == TokenNotEqual ||
		tokenType == TokenGreater || tokenType == TokenGreaterEqual ||
		tokenType == TokenLess || tokenType == TokenLessEqual ||
		tokenType == TokenIN || tokenType == TokenNOTIN ||
		tokenType == TokenCONTAINS || tokenType == TokenSTARTSWITH ||
		tokenType == TokenENDSWITH || tokenType == TokenMATCHES ||
		tokenType == TokenEXISTS || tokenType == TokenDOESNOTEXIST
}
