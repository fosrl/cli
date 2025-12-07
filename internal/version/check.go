package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/viper"
)

const (
	// GitHubRepoOwner is the GitHub repository owner
	GitHubRepoOwner = "fosrl"
	// GitHubRepoName is the GitHub repository name
	GitHubRepoName = "cli"
	// GitHubAPIBaseURL is the base URL for GitHub API
	GitHubAPIBaseURL = "https://api.github.com"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	URL     string `json:"html_url"`
}

// GetLatestRelease fetches the latest release from GitHub
func GetLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", GitHubAPIBaseURL, GitHubRepoOwner, GitHubRepoName)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "pangolin-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch release: status %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &release, nil
}

// normalizeVersion removes 'v' prefix from version string if present
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// CompareVersions compares the current version with the latest version
// Returns:
// - -1 if current < latest
// - 0 if current == latest
// - 1 if current > latest
// - error if versions cannot be parsed
func CompareVersions(current, latest string) (int, error) {
	currentNorm := normalizeVersion(current)
	latestNorm := normalizeVersion(latest)

	currentVer, err := semver.NewVersion(currentNorm)
	if err != nil {
		return 0, fmt.Errorf("failed to parse current version %s: %w", current, err)
	}

	latestVer, err := semver.NewVersion(latestNorm)
	if err != nil {
		return 0, fmt.Errorf("failed to parse latest version %s: %w", latest, err)
	}

	return currentVer.Compare(latestVer), nil
}

// CheckForUpdate checks if there's an update available
// Returns the latest release if an update is available, nil otherwise
func CheckForUpdate() (*GitHubRelease, error) {
	latest, err := GetLatestRelease()
	if err != nil {
		return nil, err
	}

	comparison, err := CompareVersions(Version, latest.TagName)
	if err != nil {
		return nil, err
	}

	// If current version is less than latest, update is available
	if comparison < 0 {
		return latest, nil
	}

	return nil, nil
}

// CheckForUpdateAsync checks for updates asynchronously and displays a message if available.
// It waits a short time for the check to complete so the message is shown even for fast commands.
// It respects the cache interval and only checks once per day.
func CheckForUpdateAsync(showMessage func(*GitHubRelease)) {
	// Check if update checking is disabled in config
	if viper.GetBool("disable_update_check") {
		return
	}

	// First, check if we have cached info that shows an update
	if cachedRelease, ok := getCachedUpdateInfo(); ok {
		showMessage(cachedRelease)
		return
	}

	// If we shouldn't check yet (based on cache interval), skip
	if !shouldCheckForUpdate() {
		return
	}

	// Channel to signal when the check is complete
	done := make(chan *GitHubRelease, 1)

	// Check for updates in the background
	go func() {
		latest, err := CheckForUpdate()
		if err != nil {
			// Silently fail - don't show errors for update checks
			done <- nil
			return
		}

		// Cache the result (even if no update, to avoid checking too frequently)
		cacheUpdateInfo(latest)
		done <- latest
	}()

	// Wait a short time for the check to complete (200ms should be enough for most cases)
	select {
	case release := <-done:
		// Check completed, show message if update is available
		if release != nil {
			showMessage(release)
		}
	case <-time.After(1000 * time.Millisecond):
		// Timeout - continue without showing message (check continues in background)
		// This ensures fast commands don't get blocked
	}
}
