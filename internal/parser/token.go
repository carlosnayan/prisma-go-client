package parser

// TokenType representa o tipo de token
type TokenType string

const (
	// Tokens especiais
	TokenEOF        TokenType = "EOF"
	TokenIllegal    TokenType = "ILLEGAL"
	TokenWhitespace TokenType = "WHITESPACE"
	TokenNewline    TokenType = "NEWLINE"

	// Identificadores e literais
	TokenIdent   TokenType = "IDENT"
	TokenString  TokenType = "STRING"
	TokenInt     TokenType = "INT"
	TokenFloat   TokenType = "FLOAT"
	TokenBoolean TokenType = "BOOLEAN"

	// Operadores e símbolos
	TokenAt        TokenType = "@"  // @
	TokenAtAt      TokenType = "@@" // @@
	TokenLParen    TokenType = "("  // (
	TokenRParen    TokenType = ")"  // )
	TokenLBrace    TokenType = "{"  // {
	TokenRBrace    TokenType = "}"  // }
	TokenLBracket  TokenType = "["  // [
	TokenRBracket  TokenType = "]"  // ]
	TokenEqual     TokenType = "="  // =
	TokenColon     TokenType = ":"  // :
	TokenQuestion  TokenType = "?"  // ?
	TokenComma     TokenType = ","  // ,
	TokenSemicolon TokenType = ";"  // ;
	TokenDot       TokenType = "."  // .

	// Keywords
	TokenModel       TokenType = "model"
	TokenEnum        TokenType = "enum"
	TokenDatasource  TokenType = "datasource"
	TokenGenerator   TokenType = "generator"
	TokenTypeKeyword TokenType = "type"
)

// Token representa um token do lexer
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// IsKeyword verifica se uma string é uma keyword
func IsKeyword(ident string) bool {
	keywords := map[string]TokenType{
		"model":      TokenModel,
		"enum":       TokenEnum,
		"datasource": TokenDatasource,
		"generator":  TokenGenerator,
		"type":       TokenTypeKeyword,
	}
	_, ok := keywords[ident]
	return ok
}

// LookupIdent retorna o TokenType para um identificador
func LookupIdent(ident string) TokenType {
	if tok, ok := map[string]TokenType{
		"model":      TokenModel,
		"enum":       TokenEnum,
		"datasource": TokenDatasource,
		"generator":  TokenGenerator,
		"type":       TokenTypeKeyword,
	}[ident]; ok {
		return tok
	}
	return TokenIdent
}
