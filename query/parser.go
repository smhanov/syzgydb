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

type Parser struct {
	lexer        *Lexer
	currentToken Token
	peekToken    Token
	errors       []string
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

func (p *Parser) parseExpression() (Node, error) {
	return p.parseLogicalExpression()
}

func (p *Parser) parseLogicalExpression() (Node, error) {
	left, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	for p.currentToken.Type == TokenAnd || p.currentToken.Type == TokenOr {
		operator := p.currentToken.Literal
		p.nextToken()

		right, err := p.parseOr()
		if err != nil {
			return nil, err
		}

		left = &ExpressionNode{Left: left, Operator: operator, Right: right}
	}

	return left, nil
}

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

func (p *Parser) parseAnd() (Node, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.currentToken.Type == TokenAnd {
		p.nextToken()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &ExpressionNode{Left: left, Operator: "AND", Right: right}
	}

	return left, nil
}

func (p *Parser) parseNot() (Node, error) {
	if p.currentToken.Type == TokenNot {
		p.nextToken()
		expr, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		return &ExpressionNode{Left: nil, Operator: "NOT", Right: expr}, nil
	}
	return p.parseComparison()
}

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

func (p *Parser) parsePrimary() (Node, error) {
	switch p.currentToken.Type {
	case TokenIdentifier:
		return p.parseIdentifierOrFunction()
	case TokenNumber:
		return p.parseNumber()
	case TokenString:
		return &ValueNode{Value: p.currentToken.Literal}, nil
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
	case TokenNot:
		return p.parseNotExpression()
	case TokenEXISTS, TokenDOESNOTEXIST, TokenANY, TokenALL, TokenLENGTH:
		return p.parseFunction()
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.currentToken.Literal)
	}
}

func (p *Parser) parsePrimary() (Node, error) {
	switch p.currentToken.Type {
	case TokenIdentifier:
		return p.parseIdentifierOrFunction()
	case TokenNumber:
		return p.parseNumber()
	case TokenString:
		return &ValueNode{Value: p.currentToken.Literal}, nil
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
	case TokenNot:
		return p.parseNotExpression()
	case TokenEXISTS, TokenDOESNOTEXIST, TokenANY, TokenALL, TokenLENGTH:
		return p.parseFunction()
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.currentToken.Literal)
	}
}

func (p *Parser) parseNotExpression() (Node, error) {
	p.nextToken() // consume NOT
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ExpressionNode{Operator: "NOT", Right: expr}, nil
}

func (p *Parser) parseFunction() (Node, error) {
	var funcName string
	if p.currentToken.Type == TokenIdentifier {
		funcName = p.currentToken.Literal
		p.nextToken() // consume function name
	}

	if p.currentToken.Type != TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after %s, got %s", funcName, p.currentToken.Literal)
	}
	p.nextToken() // consume '('

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

	return &FunctionNode{Name: funcName, Arguments: args}, nil
}

func (p *Parser) parseArrayLiteral() (Node, error) {
	p.nextToken() // consume '['
	elements := []Node{}

	if p.currentToken.Type != TokenRightBracket {
		element, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)

		for p.currentToken.Type == TokenComma {
			p.nextToken() // consume ','
			element, err := p.parseExpression()
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

	return &FunctionNode{Name: "ARRAY", Arguments: elements}, nil
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

func (p *Parser) parseIdentifierOrFunction() (Node, error) {
	identifier := p.currentToken.Literal
	p.nextToken()

	for p.currentToken.Type == TokenDot {
		p.nextToken() // consume '.'
		if p.currentToken.Type != TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after '.', got %s", p.currentToken.Literal)
		}
		identifier += "." + p.currentToken.Literal
		p.nextToken()
	}

	if p.currentToken.Type == TokenLeftParen {
		return p.parseFunction()
	}

	if p.currentToken.Type == TokenEXISTS {
		p.nextToken()
		return &FunctionNode{Name: "EXISTS", Arguments: []Node{&IdentifierNode{Name: identifier}}}, nil
	}

	if p.currentToken.Type == TokenDOESNOTEXIST {
		p.nextToken()
		return &FunctionNode{Name: "DOES_NOT_EXIST", Arguments: []Node{&IdentifierNode{Name: identifier}}}, nil
	}

	return &IdentifierNode{Name: identifier}, nil
}

func (p *Parser) parseNumber() (Node, error) {
	value, err := strconv.ParseFloat(p.currentToken.Literal, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse number: %s", p.currentToken.Literal)
	}
	return &ValueNode{Value: value}, nil
}

func (p *Parser) parseBoolean() (Node, error) {
	value, err := strconv.ParseBool(p.currentToken.Literal)
	if err != nil {
		return nil, fmt.Errorf("could not parse boolean: %s", p.currentToken.Literal)
	}
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
