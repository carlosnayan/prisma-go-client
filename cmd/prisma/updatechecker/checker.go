package updatechecker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/carlosnayan/prisma-go-client/cmd/prisma/version"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
}

// CheckForUpdate checks GitHub for a newer version
// Returns nil if no update available or on error (fail silently)
func CheckForUpdate() (*UpdateInfo, error) {
	// Check cache first - don't check more than once per 24h
	if shouldSkipCheck() {
		return nil, nil
	}

	// Query GitHub API with timeout
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/carlosnayan/prisma-go-client/releases/latest")
	if err != nil {
		return nil, err // Network error - fail silently
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	// Compare versions
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := version.Version

	if isNewer(latestVersion, currentVersion) {
		// Update found - save check timestamp
		updateLastCheck()

		return &UpdateInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion,
			ReleaseURL:     release.HTMLURL,
		}, nil
	}

	// No update - save check timestamp
	updateLastCheck()
	return nil, nil
}

// Returns true if latest > current
func isNewer(latest, current string) bool {
	latestParts := parseVersion(latest)
	currentParts := parseVersion(current)

	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}
	return false // Versions are equal
}

// parseVersion parses a semantic version string into [major, minor, patch]
func parseVersion(v string) [3]int {
	parts := strings.Split(v, ".")
	result := [3]int{0, 0, 0}

	for i := 0; i < len(parts) && i < 3; i++ {
		if num, err := strconv.Atoi(parts[i]); err == nil {
			result[i] = num
		}
	}

	return result
}
