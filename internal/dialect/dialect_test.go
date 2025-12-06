package dialect

import (
	"testing"
)

// TestDialect_PostgreSQL tests PostgreSQL-specific features
func TestDialect_PostgreSQL(t *testing.T) {
	d := GetDialect("postgresql")
	if d == nil {
		t.Fatal("GetDialect returned nil for postgresql")
	}

	// Test type mapping
	tests := []struct {
		input    string
		expected string
	}{
		{"String", "TEXT"},
		{"Int", "INTEGER"},
		{"Boolean", "BOOLEAN"},
		{"DateTime", "TIMESTAMP"},
		{"UUID", "UUID"},
	}

	for _, tt := range tests {
		result := d.MapType(tt.input, false)
		if result != tt.expected {
			t.Errorf("MapType(%s, false) = %s, want %s", tt.input, result, tt.expected)
		}
	}

	// Test identifier quoting
	quoted := d.QuoteIdentifier("user")
	if quoted != `"user"` {
		t.Errorf("QuoteIdentifier('user') = %s, want \"user\"", quoted)
	}

	// Test placeholder format
	placeholder := d.GetPlaceholder(1)
	if placeholder != "$1" {
		t.Errorf("GetPlaceholder(1) = %s, want $1", placeholder)
	}
}

// TestDialect_MySQL tests MySQL-specific features
func TestDialect_MySQL(t *testing.T) {
	d := GetDialect("mysql")
	if d == nil {
		t.Fatal("GetDialect returned nil for mysql")
	}

	// Test type mapping
	tests := []struct {
		input    string
		expected string
	}{
		{"String", "VARCHAR(191)"},
		{"Int", "INT"},
		{"Boolean", "TINYINT(1)"},
		{"DateTime", "DATETIME"},
	}

	for _, tt := range tests {
		result := d.MapType(tt.input, false)
		if result != tt.expected {
			t.Errorf("MapType(%s, false) = %s, want %s", tt.input, result, tt.expected)
		}
	}

	// Test identifier quoting
	quoted := d.QuoteIdentifier("user")
	if quoted != "`user`" {
		t.Errorf("QuoteIdentifier('user') = %s, want `user`", quoted)
	}

	// Test placeholder format
	placeholder := d.GetPlaceholder(1)
	if placeholder != "?" {
		t.Errorf("GetPlaceholder(1) = %s, want ?", placeholder)
	}
}

// TestDialect_SQLite tests SQLite-specific features
func TestDialect_SQLite(t *testing.T) {
	d := GetDialect("sqlite")
	if d == nil {
		t.Fatal("GetDialect returned nil for sqlite")
	}

	// Test type mapping
	tests := []struct {
		input    string
		expected string
	}{
		{"String", "TEXT"},
		{"Int", "INTEGER"},
		{"Boolean", "INTEGER"},
		{"DateTime", "TEXT"},
	}

	for _, tt := range tests {
		result := d.MapType(tt.input, false)
		if result != tt.expected {
			t.Errorf("MapType(%s, false) = %s, want %s", tt.input, result, tt.expected)
		}
	}

	// Test identifier quoting
	quoted := d.QuoteIdentifier("user")
	if quoted != `"user"` {
		t.Errorf("QuoteIdentifier('user') = %s, want \"user\"", quoted)
	}

	// Test placeholder format
	placeholder := d.GetPlaceholder(1)
	if placeholder != "?" {
		t.Errorf("GetPlaceholder(1) = %s, want ?", placeholder)
	}
}

