package parser

import (
	"fmt"
)

// Validator valida um schema
type Validator struct {
	schema *Schema
	errors []string
}

// Validate valida o schema completo
func Validate(schema *Schema) []string {
	v := &Validator{
		schema: schema,
		errors: []string{},
	}

	v.validateSchema()
	return v.errors
}

// validateSchema valida o schema completo
func (v *Validator) validateSchema() {
	// Validar datasources
	for _, ds := range v.schema.Datasources {
		v.validateDatasource(ds)
	}

	// Validar generators
	for _, gen := range v.schema.Generators {
		v.validateGenerator(gen)
	}

	// Validar models
	modelNames := make(map[string]bool)
	for _, model := range v.schema.Models {
		if modelNames[model.Name] {
			v.errors = append(v.errors, fmt.Sprintf("model '%s' duplicado", model.Name))
		}
		modelNames[model.Name] = true
		v.validateModel(model)
	}

	// Validar enums
	enumNames := make(map[string]bool)
	for _, enum := range v.schema.Enums {
		if enumNames[enum.Name] {
			v.errors = append(v.errors, fmt.Sprintf("enum '%s' duplicado", enum.Name))
		}
		enumNames[enum.Name] = true
		v.validateEnum(enum)
	}

	// Validar relacionamentos
	v.validateRelations()
}

// validateDatasource valida um datasource
func (v *Validator) validateDatasource(ds *Datasource) {
	hasProvider := false
	for _, field := range ds.Fields {
		if field.Name == "provider" {
			hasProvider = true
			provider, ok := field.Value.(string)
			if ok {
				validProviders := map[string]bool{
					"postgresql": true,
					"mysql":      true,
					"sqlite":     true,
				}
				if !validProviders[provider] {
					v.errors = append(v.errors, fmt.Sprintf("provider inválido no datasource '%s': %s", ds.Name, provider))
				}
			}
		}
	}
	if !hasProvider {
		v.errors = append(v.errors, fmt.Sprintf("datasource '%s' deve ter um campo 'provider'", ds.Name))
	}
}

// validateGenerator valida um generator
func (v *Validator) validateGenerator(gen *Generator) {
	hasProvider := false
	for _, field := range gen.Fields {
		if field.Name == "provider" {
			hasProvider = true
		}
	}
	if !hasProvider {
		v.errors = append(v.errors, fmt.Sprintf("generator '%s' deve ter um campo 'provider'", gen.Name))
	}
}

// validateModel valida um model
func (v *Validator) validateModel(model *Model) {
	if model.Name == "" {
		v.errors = append(v.errors, "model sem nome")
		return
	}

	fieldNames := make(map[string]bool)

	for _, field := range model.Fields {
		// Verificar nomes duplicados
		if fieldNames[field.Name] {
			v.errors = append(v.errors, fmt.Sprintf("campo '%s' duplicado no model '%s'", field.Name, model.Name))
		}
		fieldNames[field.Name] = true

		// Validar tipo do campo
		v.validateFieldType(field.Type, model.Name, field.Name)

		// Validar atributos do campo
		for _, attr := range field.Attributes {
			v.validateFieldAttribute(attr, model.Name, field.Name)
		}
	}

	// Validar atributos do model
	for _, attr := range model.Attributes {
		v.validateModelAttribute(attr, model.Name)
	}

	// Note: Primary key validation is optional, so we don't enforce it here
	// If needed in the future, add validation to check for @id or @@id attributes
}

// validateFieldType valida o tipo de um campo
func (v *Validator) validateFieldType(fieldType *FieldType, modelName, fieldName string) {
	if fieldType == nil {
		v.errors = append(v.errors, fmt.Sprintf("campo '%s' no model '%s' não tem tipo", fieldName, modelName))
		return
	}

	if fieldType.IsUnsupported {
		return // Unsupported é sempre válido
	}

	validTypes := map[string]bool{
		"String":      true,
		"Int":         true,
		"BigInt":      true,
		"Float":       true,
		"Decimal":     true,
		"Boolean":     true,
		"DateTime":    true,
		"Json":        true,
		"Bytes":       true,
		"Unsupported": true,
	}

	// Verificar se é um tipo válido ou um enum/model
	if !validTypes[fieldType.Name] {
		// Pode ser um enum ou model (será validado depois)
		// Por enquanto, apenas verificar se não está vazio
		if fieldType.Name == "" {
			v.errors = append(v.errors, fmt.Sprintf("tipo inválido para campo '%s' no model '%s'", fieldName, modelName))
		}
	}
}

// validateFieldAttribute valida um atributo de campo
func (v *Validator) validateFieldAttribute(attr *Attribute, modelName, fieldName string) {
	// Note: Unknown attributes are allowed (may be custom attributes)
	// If strict validation is needed in the future, add validation here

	// Validações específicas
	switch attr.Name {
	case "id":
		// @id não precisa de argumentos
	case "default":
		// @default deve ter pelo menos um argumento
		if len(attr.Arguments) == 0 {
			v.errors = append(v.errors, fmt.Sprintf("@default no campo '%s' do model '%s' deve ter um valor", fieldName, modelName))
		}
	case "relation":
		// @relation pode ter argumentos fields e references, mas não é obrigatório
		// no lado "um" de relações um-para-muitos (onde o campo é um array)
		// Apenas validar se tiver um, deve ter o outro
		hasFields := false
		hasReferences := false
		for _, arg := range attr.Arguments {
			if arg.Name == "fields" {
				hasFields = true
			}
			if arg.Name == "references" {
				hasReferences = true
			}
		}
		// Se tem um mas não o outro, é erro
		if (hasFields && !hasReferences) || (!hasFields && hasReferences) {
			v.errors = append(v.errors, fmt.Sprintf("@relation no campo '%s' do model '%s' deve ter ambos 'fields' e 'references' ou nenhum", fieldName, modelName))
		}
	}
}

// validateModelAttribute valida um atributo de model
func (v *Validator) validateModelAttribute(attr *Attribute, modelName string) {
	validAttributes := map[string]bool{
		"id":     true,
		"unique": true,
		"index":  true,
		"map":    true,
	}

	// Note: Unknown attributes are allowed (may be custom attributes)
	// If strict validation is needed in the future, add validation here
	_ = validAttributes[attr.Name]
}

// validateEnum valida um enum
func (v *Validator) validateEnum(enum *Enum) {
	if enum.Name == "" {
		v.errors = append(v.errors, "enum sem nome")
		return
	}

	if len(enum.Values) == 0 {
		v.errors = append(v.errors, fmt.Sprintf("enum '%s' não tem valores", enum.Name))
	}

	valueNames := make(map[string]bool)
	for _, value := range enum.Values {
		if valueNames[value.Name] {
			v.errors = append(v.errors, fmt.Sprintf("valor '%s' duplicado no enum '%s'", value.Name, enum.Name))
		}
		valueNames[value.Name] = true
	}
}

// validateRelations valida relacionamentos entre models
func (v *Validator) validateRelations() {
	// Criar mapa de models
	models := make(map[string]*Model)
	for _, model := range v.schema.Models {
		models[model.Name] = model
	}

	// Validar relacionamentos
	for _, model := range v.schema.Models {
		for _, field := range model.Fields {
			for _, attr := range field.Attributes {
				if attr.Name == "relation" {
					v.validateRelation(attr, model.Name, field.Name, models)
				}
			}
		}
	}
}

// validateRelation validates a relationship
func (v *Validator) validateRelation(attr *Attribute, modelName, fieldName string, models map[string]*Model) {
	var relationModel string
	var fields []string
	var references []string

	for _, arg := range attr.Arguments {
		switch arg.Name {
		case "fields":
			if vals, ok := arg.Value.([]interface{}); ok {
				for _, val := range vals {
					if str, ok := val.(string); ok {
						fields = append(fields, str)
					}
				}
			}
		case "references":
			if vals, ok := arg.Value.([]interface{}); ok {
				for _, val := range vals {
					if str, ok := val.(string); ok {
						references = append(references, str)
					}
				}
			}
		case "model":
			if str, ok := arg.Value.(string); ok {
				relationModel = str
			}
		}
	}

	if relationModel == "" {
		model := models[modelName]
		if model != nil {
			for _, f := range model.Fields {
				if f.Name == fieldName && f.Type != nil {
					relationModel = f.Type.Name
				}
			}
		}
	}

	if relationModel != "" {
		if _, exists := models[relationModel]; !exists {
			v.errors = append(v.errors, fmt.Sprintf("relacionamento no campo '%s' do model '%s' referencia model inexistente '%s'", fieldName, modelName, relationModel))
		}
	}

	if len(fields) > 0 && len(references) > 0 {
		if len(fields) != len(references) {
			v.errors = append(v.errors, fmt.Sprintf("@relation no campo '%s' do model '%s' deve ter o mesmo número de 'fields' e 'references'", fieldName, modelName))
		}
	}
}

// GetValidTypes retorna a lista de tipos válidos
func GetValidTypes() []string {
	return []string{
		"String", "Int", "BigInt", "Float", "Decimal",
		"Boolean", "DateTime", "Json", "Bytes", "Unsupported",
	}
}

// IsValidType verifica se um tipo é válido
func IsValidType(typeName string) bool {
	validTypes := GetValidTypes()
	for _, t := range validTypes {
		if t == typeName {
			return true
		}
	}
	return false
}

// GetTypeGoMapping retorna o mapeamento de tipos Prisma para Go
func GetTypeGoMapping() map[string]string {
	return map[string]string{
		"String":      "string",
		"Int":         "int",
		"BigInt":      "int64",
		"Float":       "float64",
		"Decimal":     "float64", // ou usar uma biblioteca de decimal
		"Boolean":     "bool",
		"DateTime":    "time.Time",
		"Json":        "json.RawMessage",
		"Bytes":       "[]byte",
		"Unsupported": "string",
	}
}

// GetTypeGoMappingNullable retorna o mapeamento de tipos Prisma para Go (nullable)
func GetTypeGoMappingNullable() map[string]string {
	mapping := GetTypeGoMapping()
	nullable := make(map[string]string)
	for k, v := range mapping {
		// Tipos que não devem ser ponteiros
		if k == "Json" || k == "Bytes" {
			nullable[k] = v
		} else {
			nullable[k] = "*" + v
		}
	}
	return nullable
}
