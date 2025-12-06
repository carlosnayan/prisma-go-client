package parser

// Node representa um nó da AST
type Node interface {
	String() string
}

// Schema representa o schema completo
type Schema struct {
	Datasources []*Datasource
	Generators  []*Generator
	Models      []*Model
	Enums       []*Enum
}

// Datasource representa um datasource
type Datasource struct {
	Name   string
	Fields []*Field
}

// Generator representa um generator
type Generator struct {
	Name   string
	Fields []*Field
}

// Model representa um model (tabela)
type Model struct {
	Name       string
	Fields     []*ModelField
	Attributes []*Attribute // @@attributes
}

// ModelField representa um campo de um model
type ModelField struct {
	Name       string
	Type       *FieldType
	Attributes []*Attribute // @attributes
}

// FieldType representa o tipo de um campo
type FieldType struct {
	Name             string // String, Int, Boolean, etc.
	IsArray          bool   // [] prefix
	IsOptional       bool   // ? suffix
	IsUnsupported    bool   // Unsupported("...")
	UnsupportedValue string
}

// Enum representa um enum
type Enum struct {
	Name   string
	Values []*EnumValue
}

// EnumValue representa um valor de enum
type EnumValue struct {
	Name       string
	Attributes []*Attribute
}

// Attribute representa um atributo (@id, @default(...), etc.)
type Attribute struct {
	Name      string // id, default, unique, etc.
	Arguments []*AttributeArgument
}

// AttributeArgument representa um argumento de atributo
type AttributeArgument struct {
	Name  string      // opcional, para named arguments
	Value interface{} // string, int, float, bool, ou função (env, autoincrement, etc.)
}

// Field representa um campo genérico (usado em datasource e generator)
type Field struct {
	Name  string
	Value interface{} // string, int, float, bool, ou função
}

// String implementa Node para Schema
func (s *Schema) String() string {
	result := "Schema{\n"
	for _, ds := range s.Datasources {
		result += "  " + ds.String() + "\n"
	}
	for _, gen := range s.Generators {
		result += "  " + gen.String() + "\n"
	}
	for _, model := range s.Models {
		result += "  " + model.String() + "\n"
	}
	for _, enum := range s.Enums {
		result += "  " + enum.String() + "\n"
	}
	result += "}"
	return result
}

// String implementa Node para Datasource
func (d *Datasource) String() string {
	return "Datasource(" + d.Name + ")"
}

// String implementa Node para Generator
func (g *Generator) String() string {
	return "Generator(" + g.Name + ")"
}

// String implementa Node para Model
func (m *Model) String() string {
	return "Model(" + m.Name + ")"
}

// String implementa Node para Enum
func (e *Enum) String() string {
	return "Enum(" + e.Name + ")"
}
