package builder

import (
	"fmt"
	"reflect"
	"strings"
)

// Validator interface para validações customizadas
type Validator interface {
	Validate(value interface{}) error
}

// ValidationRule representa uma regra de validação
type ValidationRule struct {
	Field    string
	Rule     string
	Value    interface{}
	Message  string
	Validator Validator
}

// ValidationErrors representa erros de validação
type ValidationErrors struct {
	Errors []ValidationError
}

func (ve *ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// ValidationError representa um erro de validação individual
type ValidationError struct {
	Field   string
	Message string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// ValidateStruct valida uma struct usando tags de validação
func ValidateStruct(s interface{}) error {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("ValidateStruct espera uma struct, recebeu %s", v.Kind())
	}

	var errors []ValidationError

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Verificar tag "validate"
		validateTag := field.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// Parsear regras de validação
		rules := strings.Split(validateTag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			
			if err := validateField(field.Name, value, rule); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// validateField valida um campo individual
func validateField(fieldName string, value reflect.Value, rule string) *ValidationError {
	switch {
	case rule == "required":
		if isEmpty(value) {
			return &ValidationError{
				Field:   fieldName,
				Message: "é obrigatório",
			}
		}
	case strings.HasPrefix(rule, "min="):
		min := parseIntRule(rule, "min=")
		if value.Kind() == reflect.String {
			if value.Len() < min {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("deve ter no mínimo %d caracteres", min),
				}
			}
		} else if isNumeric(value.Kind()) {
			if getNumericValue(value) < float64(min) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("deve ser no mínimo %d", min),
				}
			}
		}
	case strings.HasPrefix(rule, "max="):
		max := parseIntRule(rule, "max=")
		if value.Kind() == reflect.String {
			if value.Len() > max {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("deve ter no máximo %d caracteres", max),
				}
			}
		} else if isNumeric(value.Kind()) {
			if getNumericValue(value) > float64(max) {
				return &ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("deve ser no máximo %d", max),
				}
			}
		}
	case strings.HasPrefix(rule, "email"):
		if value.Kind() == reflect.String {
			if !isValidEmail(value.String()) {
				return &ValidationError{
					Field:   fieldName,
					Message: "deve ser um email válido",
				}
			}
		}
	}

	return nil
}

// isEmpty verifica se um valor está vazio
func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
		return v.IsNil()
	case reflect.Bool:
		return !v.Bool()
	}
	return false
}

// isNumeric verifica se o tipo é numérico
func isNumeric(kind reflect.Kind) bool {
	return kind >= reflect.Int && kind <= reflect.Float64
}

// getNumericValue obtém o valor numérico como float64
func getNumericValue(v reflect.Value) float64 {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return v.Float()
	}
	return 0
}

// parseIntRule extrai um inteiro de uma regra
func parseIntRule(rule, prefix string) int {
	rule = strings.TrimPrefix(rule, prefix)
	var result int
	_, _ = fmt.Sscanf(rule, "%d", &result)
	return result
}

// isValidEmail valida formato de email (básico)
func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

