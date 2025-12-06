package driver

import "os"

// getTestDatabaseURL gets test database URL from environment variables
//
//nolint:unused // Used by test files with build tags
func getTestDatabaseURL(provider string) string {
	var envVar string
	switch provider {
	case "postgresql":
		envVar = os.Getenv("TEST_DATABASE_URL_POSTGRESQL")
		if envVar == "" {
			envVar = os.Getenv("TEST_DATABASE_URL")
		}
	case "mysql":
		envVar = os.Getenv("TEST_DATABASE_URL_MYSQL")
		if envVar == "" {
			envVar = os.Getenv("TEST_DATABASE_URL")
		}
	case "sqlite":
		envVar = os.Getenv("TEST_DATABASE_URL_SQLITE")
		if envVar == "" {
			envVar = os.Getenv("TEST_DATABASE_URL")
		}
	}
	return envVar
}

// replaceDatabaseName replaces database name in URL
//
//nolint:unused // Used by test files with build tags
func replaceDatabaseName(url, dbName string) string {
	if url == "" {
		return url
	}

	lastSlash := -1
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash == -1 {
		return url + "/" + dbName
	}

	queryStart := -1
	for i := lastSlash; i < len(url); i++ {
		if url[i] == '?' {
			queryStart = i
			break
		}
	}

	if queryStart != -1 {
		return url[:lastSlash+1] + dbName + url[queryStart:]
	}

	return url[:lastSlash+1] + dbName
}

// removeDatabaseFromURL removes database name from URL
//
//nolint:unused // Used by test files with build tags
func removeDatabaseFromURL(url string) string {
	lastSlash := -1
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash == -1 {
		return url
	}

	queryStart := -1
	for i := lastSlash; i < len(url); i++ {
		if url[i] == '?' {
			queryStart = i
			break
		}
	}

	if queryStart != -1 {
		return url[:lastSlash+1] + url[queryStart:]
	}

	return url[:lastSlash+1]
}
