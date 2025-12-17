package version

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fosrl/cli/internal/config"
)

const (
	// UpdateCheckInterval is how often we check for updates
	UpdateCheckInterval = 12 * time.Hour
	// UpdateCheckCacheFile is the name of the cache file
	UpdateCheckCacheFile = "pangolin-update-check.json"
)

// UpdateCheckCache stores the last update check information
type UpdateCheckCache struct {
	LastCheckTime time.Time `json:"last_check_time"`
	LatestVersion string    `json:"latest_version,omitempty"`
	UpdateURL     string    `json:"update_url,omitempty"`
}

// getCacheFilePath returns the path to the update check cache file
func getCacheFilePath() (string, error) {
	pangolinDir, err := config.GetPangolinConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get .pangolin directory: %w", err)
	}
	return filepath.Join(pangolinDir, UpdateCheckCacheFile), nil
}

// readCache reads the update check cache from disk
func readCache() (*UpdateCheckCache, error) {
	cachePath, err := getCacheFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache doesn't exist yet, return empty cache
			return &UpdateCheckCache{}, nil
		}
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var cache UpdateCheckCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	return &cache, nil
}

// writeCache writes the update check cache to disk
func writeCache(cache *UpdateCheckCache) error {
	cachePath, err := getCacheFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return nil
}

// shouldCheckForUpdate returns true if we should check for updates
// (i.e., if it's been more than UpdateCheckInterval since last check)
func shouldCheckForUpdate() bool {
	cache, err := readCache()
	if err != nil {
		// If we can't read cache, check anyway
		return true
	}

	// If we've never checked, check now
	if cache.LastCheckTime.IsZero() {
		return true
	}

	// Check if enough time has passed
	return time.Since(cache.LastCheckTime) >= UpdateCheckInterval
}

// getCachedUpdateInfo returns cached update information if available
func getCachedUpdateInfo() (*GitHubRelease, bool) {
	cache, err := readCache()
	if err != nil {
		return nil, false
	}

	// If we have cached info and it's recent (within interval), use it
	if cache.LatestVersion != "" && time.Since(cache.LastCheckTime) < UpdateCheckInterval {
		// Check if cached version is newer than current
		comparison, err := CompareVersions(Version, cache.LatestVersion)
		if err == nil && comparison < 0 {
			return &GitHubRelease{
				TagName: cache.LatestVersion,
				URL:     cache.UpdateURL,
			}, true
		}
	}

	return nil, false
}

// cacheUpdateInfo stores the update check result in the cache
func cacheUpdateInfo(release *GitHubRelease) error {
	cache := &UpdateCheckCache{
		LastCheckTime: time.Now(),
	}

	if release != nil {
		cache.LatestVersion = release.TagName
		cache.UpdateURL = release.URL
	}

	return writeCache(cache)
}
