package parser

import (
	"unicode"
	"unicode/utf8"
)

// Lexer tokeniza o arquivo schema.prisma
type Lexer struct {
	input        string
	position     int  // posição atual no input (aponta para o próximo caractere)
	readPosition int  // posição atual de leitura no input (após o caractere atual)
	ch           byte // caractere atual sendo examinado
	line         int  // linha atual
	column       int  // coluna atual
}

// NewLexer cria um novo lexer
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 1,
	}
	l.readChar()
	return l
}

// readChar lê o próximo caractere e avança a posição
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII NUL representa EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	if l.ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
}

// peekChar retorna o próximo caractere sem avançar
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken retorna o próximo token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '@':
		if l.peekChar() == '@' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenAtAt, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(TokenAt, l.ch, tok.Line, tok.Column)
		}
	case '(':
		tok = newToken(TokenLParen, l.ch, tok.Line, tok.Column)
	case ')':
		tok = newToken(TokenRParen, l.ch, tok.Line, tok.Column)
	case '{':
		tok = newToken(TokenLBrace, l.ch, tok.Line, tok.Column)
	case '}':
		tok = newToken(TokenRBrace, l.ch, tok.Line, tok.Column)
	case '[':
		tok = newToken(TokenLBracket, l.ch, tok.Line, tok.Column)
	case ']':
		tok = newToken(TokenRBracket, l.ch, tok.Line, tok.Column)
	case '=':
		tok = newToken(TokenEqual, l.ch, tok.Line, tok.Column)
	case ':':
		tok = newToken(TokenColon, l.ch, tok.Line, tok.Column)
	case '?':
		tok = newToken(TokenQuestion, l.ch, tok.Line, tok.Column)
	case ',':
		tok = newToken(TokenComma, l.ch, tok.Line, tok.Column)
	case ';':
		tok = newToken(TokenSemicolon, l.ch, tok.Line, tok.Column)
	case '.':
		tok = newToken(TokenDot, l.ch, tok.Line, tok.Column)
	case '"':
		tok.Line = l.line
		tok.Column = l.column
		tok.Type = TokenString
		tok.Literal = l.readString()
		l.readChar() // consumir a aspas de fechamento
		return tok
	case '\n':
		tok = newToken(TokenNewline, l.ch, tok.Line, tok.Column)
	case 0:
		tok.Literal = ""
		tok.Type = TokenEOF
	default:
		if isLetter(l.ch) {
			tok.Line = l.line
			tok.Column = l.column
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Line = l.line
			tok.Column = l.column
			tok.Type, tok.Literal = l.readNumber()
			return tok
		} else {
			tok = newToken(TokenIllegal, l.ch, tok.Line, tok.Column)
		}
	}

	l.readChar()
	return tok
}

// skipWhitespace pula espaços em branco e comentários
func (l *Lexer) skipWhitespace() {
	for {
		// Pular espaços em branco
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
			l.readChar()
		}

		// Pular comentários de linha (//)
		if l.ch == '/' && l.peekChar() == '/' {
			l.skipLineComment()
			continue
		}

		// Pular comentários de bloco (/* */)
		if l.ch == '/' && l.peekChar() == '*' {
			l.skipBlockComment()
			continue
		}

		break
	}
}

// skipLineComment pula um comentário de linha
func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// skipBlockComment pula um comentário de bloco
func (l *Lexer) skipBlockComment() {
	l.readChar() // pular '/'
	l.readChar() // pular '*'
	for {
		if l.ch == 0 {
			return // EOF sem fechar comentário
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // pular '*'
			l.readChar() // pular '/'
			return
		}
		l.readChar()
	}
}

// readIdentifier lê um identificador
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber lê um número (int ou float)
func (l *Lexer) readNumber() (TokenType, string) {
	position := l.position
	tokenType := TokenInt

	for isDigit(l.ch) {
		l.readChar()
	}

	// Verificar se é float
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = TokenFloat
		l.readChar() // pular '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return tokenType, l.input[position:l.position]
}

// readString lê uma string entre aspas
func (l *Lexer) readString() string {
	position := l.position + 1 // pular a aspas inicial
	for {
		l.readChar()
		if l.ch == '"' {
			break
		}
		if l.ch == 0 || l.ch == '\n' {
			break // EOF ou newline sem fechar string
		}
		if l.ch == '\\' {
			l.readChar() // pular o caractere escapado
		}
	}
	return l.input[position:l.position]
}

// newToken cria um novo token
func newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{
		Type:    tokenType,
		Literal: string(ch),
		Line:    line,
		Column:  column,
	}
}

// isLetter verifica se o caractere é uma letra
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(rune(ch))
}

// isDigit verifica se o caractere é um dígito
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
