package query

import (
	"unicode"
)

type TokenType int

const (
	TokenIdentifier TokenType = iota
	TokenString
	TokenNumber
	TokenBoolean
	TokenNull
	TokenOperator
	TokenParenthesis
	TokenLeftParen    // '('
	TokenRightParen   // ')'
	TokenComma        // ','
	TokenEqual        // '=='
	TokenNotEqual     // '!='
	TokenGreater      // '>'
	TokenGreaterEqual // '>='
	TokenLess         // '<'
	TokenLessEqual    // '<='
	TokenAnd          // 'AND'
	TokenOr           // 'OR'
	TokenNot          // 'NOT'
	TokenIN
	TokenNOTIN
	TokenEXISTS
	TokenDOESNOTEXIST
	TokenCONTAINS
	TokenSTARTSWITH
	TokenENDSWITH
	TokenMATCHES
	TokenLENGTH
	TokenANY
	TokenALL
	TokenEOF
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	switch l.ch {
	case '(':
		tok = Token{Type: TokenLeftParen, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ')':
		tok = Token{Type: TokenRightParen, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ',':
		tok = Token{Type: TokenComma, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenEqual, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = Token{Type: TokenOperator, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenNotEqual, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenGreaterEqual, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = Token{Type: TokenGreater, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenLessEqual, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = Token{Type: TokenLess, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '"':
		tok.Literal = l.readString()
		tok.Type = TokenString
		return tok
	case 0:
		tok.Literal = ""
		tok.Type = TokenEOF
		tok.Literal = l.readString()
		tok.Type = TokenString
		return tok
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdentifier(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Literal = l.readNumber()
			tok.Type = TokenNumber
			return tok
		} else {
			tok = Token{Type: TokenEOF, Literal: "", Line: l.line, Column: l.column}
		}
	}

	l.readChar()
	return tok
}

func lookupIdentifier(ident string) TokenType {
	switch ident {
	case "AND":
		return TokenAnd
	case "OR":
		return TokenOr
	case "NOT":
		return TokenNot
	case "IN":
		return TokenIN
	case "NOT":
		return TokenNot
	case "IN":
		return TokenIN
	case "DOES NOT EXIST":
		return TokenDOESNOTEXIST
	case "CONTAINS":
		return TokenCONTAINS
	case "STARTS_WITH":
		return TokenSTARTSWITH
	case "ENDS_WITH":
		return TokenENDSWITH
	case "MATCHES":
		return TokenMATCHES
	case "LENGTH":
		return TokenLENGTH
	case "ANY":
		return TokenANY
	case "ALL":
		return TokenALL
	default:
		return TokenIdentifier
	}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
