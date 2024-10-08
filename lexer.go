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
    TokenComma
    TokenEOF
    // ... other token types
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
    case '=':
        if l.peekChar() == '=' {
            ch := l.ch
            l.readChar()
            tok = Token{Type: TokenOperator, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
        } else {
            tok = Token{Type: TokenOperator, Literal: string(l.ch), Line: l.line, Column: l.column}
        }
    case '+':
        tok = Token{Type: TokenOperator, Literal: string(l.ch), Line: l.line, Column: l.column}
    case 0:
        tok.Literal = ""
        tok.Type = TokenEOF
    default:
        if isLetter(l.ch) {
            tok.Literal = l.readIdentifier()
            tok.Type = TokenIdentifier
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

func isLetter(ch byte) bool {
    return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
    return '0' <= ch && ch <= '9'
}
