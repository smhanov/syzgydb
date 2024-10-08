package query

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
	// New token types
	TokenLeftBracket  // '['
	TokenRightBracket // ']'
	TokenColon        // ':'
	TokenDot          // '.'
	TokenArrayStar    // '[*]'
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

	if l.ch == 0 {
		return Token{Type: TokenEOF, Literal: "", Line: l.line, Column: l.column}
	}

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
	case '[':
		if l.peekChar() == '*' && l.peekNextChar() == ']' {
			tok = Token{Type: TokenArrayStar, Literal: "[*]", Line: l.line, Column: l.column}
			l.readChar() // consume '*'
			l.readChar() // consume ']'
		} else {
			tok = Token{Type: TokenLeftBracket, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case ']':
		tok = Token{Type: TokenRightBracket, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ':':
		tok = Token{Type: TokenColon, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '.':
		tok = Token{Type: TokenDot, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '"':
		fallthrough
	case '\'':
		tok.Literal = l.readString(l.ch)
		tok.Type = TokenString
		return tok
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifierOrKeyword()
			tok.Type = lookupIdentifier(tok.Literal)
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else if isDigit(l.ch) {
			tok.Literal = l.readNumber()
			tok.Type = TokenNumber
			return tok
		} else {
			tok = Token{Type: TokenOperator, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readIdentifierOrKeyword() string {
	position := l.position
	l.readIdentifier()

	// Check if it's the start of "DOES NOT EXIST"
	if l.input[position:l.position] == "DOES" && l.ch == ' ' {
		l.readChar() // consume the space
		if l.ch == 'N' {
			restOfKeyword := l.readWord()
			if restOfKeyword == "NOT" {
				l.readChar() // consume the space
				if l.readWord() == "EXIST" {
					return "DOES NOT EXIST"
				}
			}
		}
		// If it's not "DOES NOT EXIST", reset the position
		l.position = position
		l.readPosition = position + 1
		l.ch = l.input[position]
	}

	// Read the rest of the identifier
	l.readIdentifier()

	return l.input[position:l.position]
}

func (l *Lexer) readIdentifier() {
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
}

func (l *Lexer) readWord() string {
	position := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
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
	case "DOES NOT EXIST":
		return TokenDOESNOTEXIST
	case "EXISTS":
		return TokenEXISTS
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
	case "null":
		return TokenNull
	case "true", "false":
		return TokenBoolean
	default:
		return TokenIdentifier
	}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readNumber() string {
	position := l.position
	isHex := false
	isFloat := false

	// Check for hexadecimal prefix
	if l.ch == '0' && (l.peekChar() == 'x' || l.peekChar() == 'X') {
		isHex = true
		l.readChar() // consume '0'
		l.readChar() // consume 'x' or 'X'
	}

	for {
		if isHex {
			if !isHexDigit(l.ch) {
				break
			}
		} else if isDigit(l.ch) || (l.ch == '.' && !isFloat) {
			if l.ch == '.' {
				isFloat = true
			}
		} else {
			break
		}
		l.readChar()
	}

	// Handle exponent for floating-point numbers
	if !isHex && (l.ch == 'e' || l.ch == 'E') {
		l.readChar() // consume 'e' or 'E'
		if l.ch == '+' || l.ch == '-' {
			l.readChar() // consume '+' or '-'
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position]
}

// Add this helper function
func isHexDigit(ch byte) bool {
	return isDigit(ch) || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) peekNextChar() byte {
	if l.readPosition+1 >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition+1]
}

func (l *Lexer) readString(quotechar byte) string {
	var result []byte
	for {
		l.readChar()
		if l.ch == quotechar || l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case 'r':
				result = append(result, '\r')
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			case 0:
				// syntax error; unterminated string
			default:
				result = append(result, '\\', l.ch)
			}
		} else {
			result = append(result, l.ch)
		}
	}
	if l.ch == quotechar {
		l.readChar()
	}
	return string(result)
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
