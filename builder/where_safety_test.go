package builder

import (
	"testing"

	"github.com/carlosnayan/prisma-go-client/internal/dialect"
)

// =============================================================================
// NIL POINTER SAFETY TESTS
// These tests ensure buildWhereClause handles nil arguments without panicking
// =============================================================================

// TestBuildWhereClause_NilInSlicePosition verifies nil handling when passed via slice
// This was the original bug: integration_keys.RevokedAtIsNull() crash
func TestBuildWhereClause_NilInSlicePosition(t *testing.T) {
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{query: "status = ?", args: []interface{}{nil}}, // nil passed as value
		},
	}

	argIndex := 1
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("buildWhereClause panicked with nil arg: %v", r)
		}
	}()

	whereClause, args := q.buildWhereClause(&argIndex)

	// Should process without panic
	if len(args) != 1 || args[0] != nil {
		t.Errorf("Expected 1 nil arg, got %v", args)
	}

	if !contains(whereClause, "$1") {
		t.Errorf("Expected placeholder in: %s", whereClause)
	}
}

// TestBuildWhereClause_EmptySlice verifies empty slice doesn't cause index issues
func TestBuildWhereClause_EmptySlice(t *testing.T) {
	emptySlice := []int{}
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{query: "id IN (?)", args: []interface{}{emptySlice}},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	// Empty slice should produce () with no args
	if len(args) != 0 {
		t.Errorf("Expected 0 args for empty slice, got %d", len(args))
	}

	if !contains(whereClause, "()") {
		t.Errorf("Expected '()' in WHERE clause, got: %s", whereClause)
	}
}

// TestBuildWhereClause_SliceWithValues verifies normal slice expansion
func TestBuildWhereClause_SliceWithValues(t *testing.T) {
	ids := []int{1, 2, 3}
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{query: "id IN (?)", args: []interface{}{ids}},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	if len(args) != 3 {
		t.Fatalf("Expected 3 args, got %d", len(args))
	}

	// Verify all IDs are present
	for i, expected := range ids {
		if args[i] != expected {
			t.Errorf("Expected arg[%d]=%d, got %v", i, expected, args[i])
		}
	}

	// Should contain ($1, $2, $3)
	if !contains(whereClause, "($1, $2, $3)") {
		t.Errorf("Expected placeholders in: %s", whereClause)
	}
}

// =============================================================================
// WHERE OPERATOR TESTS
// These tests verify all SQL operators generate correct syntax
// =============================================================================

// TestWhereOperators_ISNull verifies IS NULL generates correct SQL (no args!)
func TestWhereOperators_ISNull(t *testing.T) {
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			// IS NULL doesn't have ? placeholder, so no args
			{query: "deleted_at IS NULL", args: []interface{}{}},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	// Should be exactly "deleted_at IS NULL" with NO args
	if whereClause != "deleted_at IS NULL" {
		t.Errorf("Expected 'deleted_at IS NULL', got '%s'", whereClause)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args for IS NULL, got %d: %v", len(args), args)
	}

	// Critical: Must NOT contain IS_NULL bug
	if contains(whereClause, "IS_NULL") {
		t.Error("BUG: Found 'IS_NULL' instead of 'IS NULL'")
	}
}

// TestWhereOperators_ISNotNull verifies IS NOT NULL generates correct SQL
func TestWhereOperators_ISNotNull(t *testing.T) {
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{query: "deleted_at IS NOT NULL", args: []interface{}{}},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	if whereClause != "deleted_at IS NOT NULL" {
		t.Errorf("Expected 'deleted_at IS NOT NULL', got '%s'", whereClause)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}

	if contains(whereClause, "IS_NOT_NULL") {
		t.Error("BUG: Found 'IS_NOT_NULL' instead of 'IS NOT NULL'")
	}
}

// TestWhereOperators_Equals verifies = operator
func TestWhereOperators_Equals(t *testing.T) {
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{query: "email = ?", args: []interface{}{"test@example.com"}},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	if len(args) != 1 || args[0] != "test@example.com" {
		t.Errorf("Expected email arg, got: %v", args)
	}

	if !contains(whereClause, "$1") {
		t.Errorf("Expected $1 placeholder in: %s", whereClause)
	}
}

// TestWhereOperators_CombinedAND verifies AND with mixed operators
func TestWhereOperators_CombinedAND(t *testing.T) {
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{query: "active = ?", args: []interface{}{true}},
			{query: "deleted_at IS NULL", args: []interface{}{}}, // No args for IS NULL!
			{query: "role = ?", args: []interface{}{"admin"}},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	// Should have only 2 args: true and "admin" (IS NULL has no args)
	if len(args) != 2 {
		t.Fatalf("Expected 2 args (IS NULL has none), got %d: %v", len(args), args)
	}

	if args[0] != true || args[1] != "admin" {
		t.Errorf("Expected [true, admin], got: %v", args)
	}

	// Should contain 2 AND operators
	andCount := 0
	for i := 0; i < len(whereClause)-2; i++ {
		if whereClause[i:i+3] == "AND" {
			andCount++
		}
	}
	if andCount != 2 {
		t.Errorf("Expected 2 ANDs, got %d in: %s", andCount, whereClause)
	}

	// Must contain "IS NULL" not "IS_NULL"
	if !contains(whereClause, "IS NULL") {
		t.Error("Expected 'IS NULL' in clause")
	}
	if contains(whereClause, "IS_NULL") {
		t.Error("BUG: Found 'IS_NULL'!")
	}
}

// TestWhereOperators_RealWorldTenantQuery verifies user's exact failing scenario
func TestWhereOperators_RealWorldTenantQuery(t *testing.T) {
	// User's query: tenants.And(tenants.IdTenantEQ(id), tenants.DeleteAtIsNull())
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{
				// applyCondition generates: "id_tenant = ?" and "delete_at IS NULL" separately
				// then joins with AND
				query: "id_tenant = ?",
				args:  []interface{}{"tenant-123"},
			},
			{
				query: "delete_at IS NULL",
				args:  []interface{}{}, // IS NULL has NO args!
			},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	// Should have only 1 arg (the tenant ID)
	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d: %v", len(args), args)
	}

	if args[0] != "tenant-123" {
		t.Errorf("Expected tenant ID, got: %v", args[0])
	}

	// Verify final SQL structure
	// Should produce: id_tenant = $1 AND delete_at IS NULL
	if !contains(whereClause, "id_tenant") ||
		!contains(whereClause, "$1") ||
		!contains(whereClause, "AND") ||
		!contains(whereClause, "IS NULL") {
		t.Errorf("Invalid WHERE structure: %s", whereClause)
	}

	// CRITICAL: Must NOT have IS_NULL
	if contains(whereClause, "IS_NULL") {
		t.Errorf("CRITICAL BUG: Found 'IS_NULL' in: %s", whereClause)
	}
}

// TestWhereOperators_INWithEmptySlice verifies empty IN handling
func TestWhereOperators_INWithEmptySlice(t *testing.T) {
	emptySlice := []string{}
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{query: "status IN (?)", args: []interface{}{emptySlice}},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	if len(args) != 0 {
		t.Errorf("Expected 0 args for empty slice, got %d", len(args))
	}

	if !contains(whereClause, "()") {
		t.Errorf("Expected '()' for empty IN, got: %s", whereClause)
	}
}

// TestWhereOperators_LIKE verifies LIKE operator
func TestWhereOperators_LIKE(t *testing.T) {
	q := &Query{
		dialect: dialect.GetDialect("postgresql"),
		whereConditions: []whereCondition{
			{query: "name LIKE ?", args: []interface{}{"%john%"}},
		},
	}

	argIndex := 1
	whereClause, args := q.buildWhereClause(&argIndex)

	if len(args) != 1 || args[0] != "%john%" {
		t.Errorf("Expected LIKE pattern, got: %v", args)
	}

	if !contains(whereClause, "LIKE") {
		t.Errorf("Expected LIKE in: %s", whereClause)
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
