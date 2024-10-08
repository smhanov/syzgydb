package query

import (
    "fmt"
)

type Node interface{}

type ExpressionNode struct {
    Left     Node
    Operator string
    Right    Node
}

type IdentifierNode struct {
    Name string
}

type ValueNode struct {
    Value interface{}
}

type FunctionNode struct {
    Name      string
    Arguments []Node
}

type ParameterNode struct {
    Name string
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
    left, err := p.parsePrimary()
    if err != nil {
        return nil, err
    }

    for p.currentToken.Type == TokenOperator {
        operator := p.currentToken.Literal
        p.nextToken()
        right, err := p.parsePrimary()
        if err != nil {
            return nil, err
        }
        left = &ExpressionNode{Left: left, Operator: operator, Right: right}
    }

    return left, nil
}

func (p *Parser) parsePrimary() (Node, error) {
    switch p.currentToken.Type {
    case TokenIdentifier:
        return &IdentifierNode{Name: p.currentToken.Literal}, nil
    case TokenNumber:
        return &ValueNode{Value: p.currentToken.Literal}, nil
    default:
        return nil, fmt.Errorf("unexpected token: %s", p.currentToken.Literal)
    }
}
