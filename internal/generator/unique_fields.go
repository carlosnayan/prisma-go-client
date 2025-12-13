package generator

import (
	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

type UniqueConstraint struct {
	Fields       []string
	Name         string
	IsComposite  bool
	IsPrimaryKey bool
}

func getUniqueConstraints(model *parser.Model) []UniqueConstraint {
	var constraints []UniqueConstraint

	for _, field := range model.Fields {
		for _, attr := range field.Attributes {
			if attr.Name == "id" {
				constraints = append(constraints, UniqueConstraint{
					Fields:       []string{field.Name},
					IsPrimaryKey: true,
				})
			}
			if attr.Name == "unique" {
				constraints = append(constraints, UniqueConstraint{
					Fields: []string{field.Name},
				})
			}
		}
	}

	for _, attr := range model.Attributes {
		if attr.Name == "unique" || attr.Name == "id" {
			constraint := UniqueConstraint{
				IsPrimaryKey: attr.Name == "id",
			}
			for _, arg := range attr.Arguments {
				if arg.Name == "" || arg.Name == "fields" {
					if fields, ok := arg.Value.([]interface{}); ok {
						for _, f := range fields {
							if fieldName, ok := f.(string); ok {
								constraint.Fields = append(constraint.Fields, fieldName)
							}
						}
					}
				}
				if arg.Name == "map" {
					if name, ok := arg.Value.(string); ok {
						constraint.Name = name
					}
				}
			}
			if len(constraint.Fields) > 0 {
				constraint.IsComposite = len(constraint.Fields) > 1
				constraints = append(constraints, constraint)
			}
		}
	}

	return constraints
}

func matchesUniqueConstraint(whereFields []string, constraints []UniqueConstraint) *UniqueConstraint {
	for i := range constraints {
		if slicesEqual(whereFields, constraints[i].Fields) {
			return &constraints[i]
		}
	}
	return nil
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
