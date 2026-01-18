package updatechecker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CheckCache stores the timestamp of the last update check
type CheckCache struct {
	LastCheck time.Time `json:"last_check"`
}

// getCacheFilePath returns the path to the cache file
func getCacheFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	prismaDir := filepath.Join(homeDir, ".prisma")
	if err := os.MkdirAll(prismaDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(prismaDir, "update_check"), nil
}

// shouldSkipCheck returns true if we checked within the last 24 hours
func shouldSkipCheck() bool {
	cache := loadCache()
	return time.Since(cache.LastCheck) < 24*time.Hour
}

// loadCache loads the cache from disk, returns empty cache on error
func loadCache() CheckCache {
	cachePath, err := getCacheFilePath()
	if err != nil {
		return CheckCache{}
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return CheckCache{}
	}

	var cache CheckCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return CheckCache{}
	}

	return cache
}

// updateLastCheck saves the current timestamp to cache
func updateLastCheck() {
	cachePath, err := getCacheFilePath()
	if err != nil {
		return // Fail silently
	}

	cache := CheckCache{
		LastCheck: time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return
	}

	_ = os.WriteFile(cachePath, data, 0644) // Ignore error - fail silently
}
