package parser

import (
	"fmt"
)

// Parser parseia tokens e constrói a AST
type Parser struct {
	lexer     *Lexer
	errors    []string
	curToken  Token
	peekToken Token
}

// NewParser cria um novo parser
func NewParser(lexer *Lexer) *Parser {
	p := &Parser{
		lexer:  lexer,
		errors: []string{},
	}

	// Ler dois tokens para ter curToken e peekToken
	p.nextToken()
	p.nextToken()

	return p
}

// Errors retorna os erros encontrados durante o parsing
func (p *Parser) Errors() []string {
	return p.errors
}

// nextToken avança os tokens
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// expectToken verifica se o token atual é do tipo esperado
func (p *Parser) expectToken(t TokenType) bool {
	if p.curToken.Type == t {
		return true
	}
	p.errors = append(p.errors, fmt.Sprintf("esperado %s, encontrado %s na linha %d, coluna %d", t, p.curToken.Type, p.curToken.Line, p.curToken.Column))
	return false
}

// ParseSchema parseia o schema completo
func (p *Parser) ParseSchema() *Schema {
	schema := &Schema{
		Datasources: []*Datasource{},
		Generators:  []*Generator{},
		Models:      []*Model{},
		Enums:       []*Enum{},
	}

	for p.curToken.Type != TokenEOF {
		switch p.curToken.Type {
		case TokenDatasource:
			ds := p.parseDatasource()
			if ds != nil {
				schema.Datasources = append(schema.Datasources, ds)
			}
		case TokenGenerator:
			gen := p.parseGenerator()
			if gen != nil {
				schema.Generators = append(schema.Generators, gen)
			}
		case TokenModel:
			model := p.parseModel()
			if model != nil {
				schema.Models = append(schema.Models, model)
			}
		case TokenEnum:
			enum := p.parseEnum()
			if enum != nil {
				schema.Enums = append(schema.Enums, enum)
			}
		case TokenIllegal:
			// Ignorar tokens ilegais (comentários já foram tratados pelo lexer)
			p.nextToken()
		case TokenWhitespace, TokenNewline:
			// Ignorar whitespace e newlines
			p.nextToken()
		default:
			p.errors = append(p.errors, fmt.Sprintf("token inesperado: %s na linha %d, coluna %d", p.curToken.Type, p.curToken.Line, p.curToken.Column))
			p.nextToken()
		}
	}

	return schema
}

// parseDatasource parseia um datasource
func (p *Parser) parseDatasource() *Datasource {
	if !p.expectToken(TokenDatasource) {
		return nil
	}
	p.nextToken()

	ds := &Datasource{
		Fields: []*Field{},
	}

	// Nome do datasource
	if p.curToken.Type != TokenIdent {
		p.errors = append(p.errors, fmt.Sprintf("esperado identificador para nome do datasource na linha %d", p.curToken.Line))
		return nil
	}
	ds.Name = p.curToken.Literal
	p.nextToken()

	// {
	if !p.expectToken(TokenLBrace) {
		return nil
	}
	p.nextToken()

	// Campos
	for p.curToken.Type != TokenRBrace && p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenIdent {
			field := p.parseField()
			if field != nil {
				ds.Fields = append(ds.Fields, field)
			}
		} else {
			p.nextToken()
		}
	}

	if p.curToken.Type == TokenRBrace {
		p.nextToken()
	} else if p.curToken.Type != TokenEOF {
		p.errors = append(p.errors, fmt.Sprintf("esperado }, encontrado %s na linha %d", p.curToken.Type, p.curToken.Line))
	}

	return ds
}

// parseGenerator parseia um generator
func (p *Parser) parseGenerator() *Generator {
	if !p.expectToken(TokenGenerator) {
		return nil
	}
	p.nextToken()

	gen := &Generator{
		Fields: []*Field{},
	}

	// Nome do generator
	if p.curToken.Type != TokenIdent {
		p.errors = append(p.errors, fmt.Sprintf("esperado identificador para nome do generator na linha %d", p.curToken.Line))
		return nil
	}
	gen.Name = p.curToken.Literal
	p.nextToken()

	// {
	if !p.expectToken(TokenLBrace) {
		return nil
	}
	p.nextToken()

	// Campos
	for p.curToken.Type != TokenRBrace && p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenIdent {
			field := p.parseField()
			if field != nil {
				gen.Fields = append(gen.Fields, field)
			}
		} else {
			p.nextToken()
		}
	}

	if p.curToken.Type == TokenRBrace {
		p.nextToken()
	} else if p.curToken.Type != TokenEOF {
		p.errors = append(p.errors, fmt.Sprintf("esperado }, encontrado %s na linha %d", p.curToken.Type, p.curToken.Line))
	}

	return gen
}

// parseModel parseia um model
func (p *Parser) parseModel() *Model {
	if !p.expectToken(TokenModel) {
		return nil
	}
	p.nextToken()

	model := &Model{
		Fields:     []*ModelField{},
		Attributes: []*Attribute{},
	}

	// Nome do model
	if p.curToken.Type != TokenIdent {
		p.errors = append(p.errors, fmt.Sprintf("esperado identificador para nome do model na linha %d", p.curToken.Line))
		return nil
	}
	model.Name = p.curToken.Literal
	p.nextToken()

	// {
	if !p.expectToken(TokenLBrace) {
		return nil
	}
	p.nextToken()

	// Campos e atributos
	for p.curToken.Type != TokenRBrace && p.curToken.Type != TokenEOF {
		switch p.curToken.Type {
		case TokenAtAt:
			// Atributo de model (@@)
			p.nextToken()
			attr := p.parseAttribute()
			if attr != nil {
				model.Attributes = append(model.Attributes, attr)
			}
		case TokenIdent, TokenTypeKeyword:
			// Campo do model (TokenTypeKeyword permite campos chamados "type")
			field := p.parseModelField()
			if field != nil {
				model.Fields = append(model.Fields, field)
			}
		case TokenWhitespace, TokenNewline:
			p.nextToken()
		default:
			// Se não é um token esperado, pode ser fim do arquivo ou erro
			if p.curToken.Type == TokenEOF {
				break
			}
			p.nextToken()
		}
	}

	if p.curToken.Type == TokenRBrace {
		p.nextToken()
	} else if p.curToken.Type != TokenEOF {
		p.errors = append(p.errors, fmt.Sprintf("esperado }, encontrado %s na linha %d", p.curToken.Type, p.curToken.Line))
	}

	return model
}

// parseModelField parseia um campo de model
func (p *Parser) parseModelField() *ModelField {
	field := &ModelField{
		Attributes: []*Attribute{},
	}

	// Nome do campo - aceita identificadores e também keywords que podem ser nomes de campo
	// (ex: "type" pode ser usado como nome de campo em Prisma)
	if p.curToken.Type != TokenIdent && p.curToken.Type != TokenTypeKeyword {
		p.errors = append(p.errors, fmt.Sprintf("esperado identificador para nome do campo na linha %d", p.curToken.Line))
		return nil
	}
	field.Name = p.curToken.Literal
	p.nextToken()

	// Tipo do campo
	field.Type = p.parseFieldType()

	// Atributos (@id, @default, etc.)
	for p.curToken.Type == TokenAt {
		p.nextToken()
		attr := p.parseAttribute()
		if attr != nil {
			field.Attributes = append(field.Attributes, attr)
		}
	}

	return field
}

// parseFieldType parseia o tipo de um campo
func (p *Parser) parseFieldType() *FieldType {
	fieldType := &FieldType{}

	// Prisma usa Type[] para arrays, não []Type
	if p.curToken.Type == TokenIdent {
		// Tipo normal ou Unsupported
		if p.curToken.Literal == "Unsupported" {
			fieldType.IsUnsupported = true
			p.nextToken()
			if p.curToken.Type == TokenLParen {
				p.nextToken()
				if p.curToken.Type == TokenString {
					fieldType.UnsupportedValue = p.curToken.Literal
					p.nextToken()
				}
				if !p.expectToken(TokenRParen) {
					return nil
				}
				p.nextToken()
			}
		} else {
			fieldType.Name = p.curToken.Literal
			p.nextToken()
		}
	} else {
		p.errors = append(p.errors, fmt.Sprintf("tipo de campo inválido na linha %d", p.curToken.Line))
		return nil
	}

	// Verificar se é array (Type[])
	if p.curToken.Type == TokenLBracket {
		p.nextToken() // pular '['
		if p.curToken.Type == TokenRBracket {
			fieldType.IsArray = true
			p.nextToken() // pular ']'
		}
	}

	// Verificar se é opcional (?)
	if p.curToken.Type == TokenQuestion {
		fieldType.IsOptional = true
		p.nextToken()
	}

	return fieldType
}

// parseEnum parseia um enum
func (p *Parser) parseEnum() *Enum {
	if !p.expectToken(TokenEnum) {
		return nil
	}
	p.nextToken()

	enum := &Enum{
		Values: []*EnumValue{},
	}

	// Nome do enum
	if p.curToken.Type != TokenIdent {
		p.errors = append(p.errors, fmt.Sprintf("esperado identificador para nome do enum na linha %d", p.curToken.Line))
		return nil
	}
	enum.Name = p.curToken.Literal
	p.nextToken()

	// {
	if !p.expectToken(TokenLBrace) {
		return nil
	}
	p.nextToken()

	// Valores do enum
	for p.curToken.Type != TokenRBrace && p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenIdent {
			value := &EnumValue{
				Name:       p.curToken.Literal,
				Attributes: []*Attribute{},
			}
			p.nextToken()

			// Atributos do valor
			for p.curToken.Type == TokenAt {
				p.nextToken()
				attr := p.parseAttribute()
				if attr != nil {
					value.Attributes = append(value.Attributes, attr)
				}
			}

			enum.Values = append(enum.Values, value)
		} else {
			p.nextToken()
		}
	}

	if p.curToken.Type == TokenRBrace {
		p.nextToken()
	} else if p.curToken.Type != TokenEOF {
		p.errors = append(p.errors, fmt.Sprintf("esperado }, encontrado %s na linha %d", p.curToken.Type, p.curToken.Line))
	}

	return enum
}

// parseAttribute parseia um atributo (@id, @default(...), @db.Uuid, etc.)
func (p *Parser) parseAttribute() *Attribute {
	attr := &Attribute{
		Arguments: []*AttributeArgument{},
	}

	// Nome do atributo
	if p.curToken.Type != TokenIdent {
		p.errors = append(p.errors, fmt.Sprintf("esperado identificador para nome do atributo na linha %d", p.curToken.Line))
		return nil
	}
	attr.Name = p.curToken.Literal
	p.nextToken()

	// Verificar se é atributo composto (ex: @db.Uuid, @db.VarChar)
	for p.curToken.Type == TokenDot {
		p.nextToken() // pular o '.'
		if p.curToken.Type == TokenIdent {
			attr.Name = attr.Name + "." + p.curToken.Literal
			p.nextToken()
		}
	}

	// Argumentos do atributo (se houver)
	if p.curToken.Type == TokenLParen {
		p.nextToken()
		for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
			arg := p.parseAttributeArgument()
			if arg != nil {
				attr.Arguments = append(attr.Arguments, arg)
			}
			if p.curToken.Type == TokenComma {
				p.nextToken()
			}
		}
		if !p.expectToken(TokenRParen) {
			return nil
		}
		p.nextToken()
	}

	return attr
}

// parseAttributeArgument parseia um argumento de atributo
func (p *Parser) parseAttributeArgument() *AttributeArgument {
	arg := &AttributeArgument{}

	// Verificar se é named argument (name: value ou name = value)
	if p.curToken.Type == TokenIdent && (p.peekToken.Type == TokenEqual || p.peekToken.Type == TokenColon) {
		arg.Name = p.curToken.Literal
		p.nextToken() // pular nome
		p.nextToken() // pular = ou :
	}

	// Valor do argumento
	arg.Value = p.parseValue()

	return arg
}

// parseValue parseia um valor (string, número, função, etc.)
func (p *Parser) parseValue() interface{} {
	switch p.curToken.Type {
	case TokenString:
		val := p.curToken.Literal
		p.nextToken()
		return val
	case TokenInt:
		val := p.curToken.Literal
		p.nextToken()
		return val
	case TokenFloat:
		val := p.curToken.Literal
		p.nextToken()
		return val
	case TokenBoolean:
		val := p.curToken.Literal == "true"
		p.nextToken()
		return val
	case TokenLBracket:
		// Array - usado em @@index([field1, field2], ...)
		p.nextToken() // pular '['
		values := []interface{}{}
		for p.curToken.Type != TokenRBracket && p.curToken.Type != TokenEOF {
			val := p.parseValue()
			if val != nil {
				values = append(values, val)
			}
			if p.curToken.Type == TokenComma {
				p.nextToken()
			}
		}
		if p.curToken.Type == TokenRBracket {
			p.nextToken()
		}
		return values
	case TokenIdent:
		// Pode ser uma função (env, autoincrement, now, etc.)
		ident := p.curToken.Literal
		p.nextToken()
		if p.curToken.Type == TokenLParen {
			// É uma função
			p.nextToken()
			args := []interface{}{}
			for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
				args = append(args, p.parseValue())
				if p.curToken.Type == TokenComma {
					p.nextToken()
				}
			}
			if !p.expectToken(TokenRParen) {
				return nil
			}
			p.nextToken()
			return map[string]interface{}{
				"function": ident,
				"args":     args,
			}
		}
		return ident
	default:
		// Avançar o token para evitar loop infinito
		p.nextToken()
		return nil
	}
}

// parseField parseia um campo genérico (usado em datasource e generator)
func (p *Parser) parseField() *Field {
	field := &Field{}

	// Nome do campo
	if p.curToken.Type != TokenIdent {
		p.errors = append(p.errors, fmt.Sprintf("esperado identificador para nome do campo na linha %d", p.curToken.Line))
		return nil
	}
	field.Name = p.curToken.Literal
	p.nextToken()

	// =
	if !p.expectToken(TokenEqual) {
		return nil
	}
	p.nextToken()

	// Valor
	field.Value = p.parseValue()

	return field
}
