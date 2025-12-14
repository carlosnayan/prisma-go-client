package config

import (
	"bufio"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func decodeTOML(data string, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("decodeTOML: v must be a non-nil pointer")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("decodeTOML: v must be a pointer to struct")
	}

	var currentSection string
	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			continue
		}

		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = unquote(value)

		if currentSection == "" {
			setField(rv, key, value)
		} else {
			sectionField := findField(rv, currentSection)
			if sectionField.IsValid() && sectionField.Kind() == reflect.Ptr {
				if sectionField.IsNil() {
					sectionField.Set(reflect.New(sectionField.Type().Elem()))
				}
				setField(sectionField.Elem(), key, value)
			}
		}
	}

	return scanner.Err()
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func findField(v reflect.Value, name string) reflect.Value {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("toml")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		tagName := strings.Split(tag, ",")[0]
		if tagName == name || strings.EqualFold(field.Name, name) {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

func setField(v reflect.Value, key, value string) {
	field := findField(v, key)
	if !field.IsValid() || !field.CanSet() {
		return
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int64:
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			field.SetInt(i)
		}
	case reflect.Bool:
		field.SetBool(value == "true" || value == "1")
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			values := parseStringArray(value)
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(v)
			}
			field.Set(slice)
		}
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		elem := field.Elem()
		if elem.Kind() == reflect.String {
			elem.SetString(value)
		}
	}
}

func parseStringArray(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return nil
	}
	s = s[1 : len(s)-1]
	if s == "" {
		return nil
	}

	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		item = unquote(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
