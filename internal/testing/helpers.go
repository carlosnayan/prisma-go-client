package testing

import (
	"os"
	"testing"
)

// SkipIfNoDatabase skips the test if database is not available
func SkipIfNoDatabase(t *testing.T, provider string) {
	url := GetTestDatabaseURL(provider)
	if url == "" {
		t.Skipf("TEST_DATABASE_URL_%s not set, skipping %s test", provider, provider)
	}
}

// RequireDatabase fails the test if database is not available
func RequireDatabase(t *testing.T, provider string) {
	url := GetTestDatabaseURL(provider)
	if url == "" {
		t.Fatalf("TEST_DATABASE_URL_%s not set, %s test requires database", provider, provider)
	}
}

// GetProviderFromEnv gets provider from environment or returns default
func GetProviderFromEnv() string {
	provider := os.Getenv("TEST_PROVIDER")
	if provider == "" {
		return "postgresql" // Default
	}
	return provider
}


