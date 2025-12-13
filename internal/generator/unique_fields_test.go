package generator

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/parser"
)

func TestGetUniqueConstraints_SingleUnique(t *testing.T) {
	model := &parser.Model{
		Name: "User",
		Fields: []*parser.ModelField{
			{
				Name: "id",
				Type: &parser.FieldType{Name: "Int"},
				Attributes: []*parser.Attribute{
					{Name: "id"},
				},
			},
			{
				Name: "email",
				Type: &parser.FieldType{Name: "String"},
				Attributes: []*parser.Attribute{
					{Name: "unique"},
				},
			},
			{
				Name: "name",
				Type: &parser.FieldType{Name: "String"},
			},
		},
	}

	constraints := getUniqueConstraints(model)

	if len(constraints) != 2 {
		t.Fatalf("Expected 2 constraints, got %d", len(constraints))
	}

	hasId := false
	hasEmail := false
	for _, c := range constraints {
		if len(c.Fields) == 1 && c.Fields[0] == "id" && c.IsPrimaryKey {
			hasId = true
		}
		if len(c.Fields) == 1 && c.Fields[0] == "email" && !c.IsPrimaryKey {
			hasEmail = true
		}
	}

	if !hasId {
		t.Error("Expected @id constraint on id field")
	}
	if !hasEmail {
		t.Error("Expected @unique constraint on email field")
	}
}

func TestGetUniqueConstraints_CompositeUnique(t *testing.T) {
	model := &parser.Model{
		Name: "Customer",
		Fields: []*parser.ModelField{
			{Name: "id", Type: &parser.FieldType{Name: "Int"}},
			{Name: "id_tenant", Type: &parser.FieldType{Name: "String"}},
			{Name: "customer_reference", Type: &parser.FieldType{Name: "String"}},
		},
		Attributes: []*parser.Attribute{
			{
				Name: "unique",
				Arguments: []*parser.AttributeArgument{
					{
						Name:  "",
						Value: []interface{}{"id_tenant", "customer_reference"},
					},
					{
						Name:  "map",
						Value: "customers_id_tenant_customer_reference_key",
					},
				},
			},
		},
	}

	constraints := getUniqueConstraints(model)

	if len(constraints) != 1 {
		t.Fatalf("Expected 1 constraint, got %d", len(constraints))
	}

	c := constraints[0]
	if !c.IsComposite {
		t.Error("Expected IsComposite to be true")
	}
	if len(c.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(c.Fields))
	}
	if c.Fields[0] != "id_tenant" || c.Fields[1] != "customer_reference" {
		t.Errorf("Expected [id_tenant, customer_reference], got %v", c.Fields)
	}
	if c.Name != "customers_id_tenant_customer_reference_key" {
		t.Errorf("Expected map name, got %s", c.Name)
	}
}

func TestMatchesUniqueConstraint_ExactMatch(t *testing.T) {
	constraints := []UniqueConstraint{
		{Fields: []string{"email"}},
		{Fields: []string{"id_tenant", "customer_reference"}, IsComposite: true},
	}

	match := matchesUniqueConstraint([]string{"email"}, constraints)
	if match == nil {
		t.Error("Expected match for email")
	}

	match = matchesUniqueConstraint([]string{"id_tenant", "customer_reference"}, constraints)
	if match == nil {
		t.Error("Expected match for composite")
	}

	match = matchesUniqueConstraint([]string{"id_tenant"}, constraints)
	if match != nil {
		t.Error("Should not match partial composite")
	}

	match = matchesUniqueConstraint([]string{"email", "name"}, constraints)
	if match != nil {
		t.Error("Should not match with extra fields")
	}
}

func TestSlicesEqual(t *testing.T) {
	if !slicesEqual([]string{"a", "b"}, []string{"a", "b"}) {
		t.Error("Should be equal")
	}
	if slicesEqual([]string{"a", "b"}, []string{"b", "a"}) {
		t.Error("Order matters, should not be equal")
	}
	if slicesEqual([]string{"a"}, []string{"a", "b"}) {
		t.Error("Different lengths, should not be equal")
	}
}
