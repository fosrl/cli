//go:build windows

package update

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/version"
	"github.com/spf13/cobra"
)

const windowsInstallerAssetName = "pangolin-cli_windows_installer.msi"

var windowsUpdateRepo = "fosrl/cli"

type githubReleaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type githubReleaseResponse struct {
	TagName string               `json:"tag_name"`
	HTMLURL string               `json:"html_url"`
	Assets  []githubReleaseAsset `json:"assets"`
}

func UpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Pangolin CLI to the latest version",
		Long:  "Update Pangolin CLI to the latest version by downloading the new installer from GitHub",
		Run: func(cmd *cobra.Command, args []string) {
			if err := updateMain(windowsUpdateRepo); err != nil {
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVar(&windowsUpdateRepo, "repo", windowsUpdateRepo, "GitHub repository in owner/name format")

	return cmd
}

func updateMain(repo string) error {
	logger.Info("Checking for latest Pangolin CLI Windows installer...")

	release, err := getLatestRelease(repo)
	if err != nil {
		logger.Error("Failed to fetch latest release: %v", err)
		return err
	}

	installerURL, err := getInstallerURL(release.Assets)
	if err != nil {
		logger.Error("%v", err)
		logger.Info("Release page: %s", release.HTMLURL)
		return err
	}

	logger.Info("This will download the latest version to a temporary folder, then start the Windows installer.")
	logger.Info("Press Enter to confirm...")
	if err := waitForEnter(); err != nil {
		logger.Error("Failed to read confirmation input: %v", err)
		return err
	}

	tempDir, err := os.MkdirTemp("", "pangolin-cli-update-*")
	if err != nil {
		logger.Error("Failed to create temp dir: %v", err)
		return err
	}

	installerPath := filepath.Join(tempDir, windowsInstallerAssetName)
	logger.Info("Downloading %s from release %s...", windowsInstallerAssetName, release.TagName)
	if err := downloadFile(installerURL, installerPath); err != nil {
		logger.Error("Failed to download installer: %v", err)
		return err
	}

	logger.Info("Launching installer: %s", installerPath)
	msiExecPath := filepath.Join(os.Getenv("WINDIR"), "System32", "msiexec.exe")
	if _, statErr := os.Stat(msiExecPath); statErr != nil {
		msiExecPath = "msiexec.exe"
	}

	// Start detached so the update command can exit while installer continues.
	installCmd := exec.Command(msiExecPath, "/i", installerPath)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Start(); err != nil {
		logger.Error("Failed to start installer: %v", err)
		return err
	}

	logger.Success("Installer launched. Follow the MSI prompts to complete update.")

	return nil
}

func waitForEnter() error {
	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadString('\n')
	return err
}

func getLatestRelease(repo string) (*githubReleaseResponse, error) {
	repoParts := strings.Split(repo, "/")
	if len(repoParts) != 2 || repoParts[0] == "" || repoParts[1] == "" {
		return nil, fmt.Errorf("invalid repo %q, expected owner/name", repo)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", version.GitHubAPIBaseURL, repoParts[0], repoParts[1])

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create release request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pangolin-cli-update")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub releases API returned %d: %s", resp.StatusCode, string(body))
	}

	var release githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release response: %w", err)
	}

	return &release, nil
}

func getInstallerURL(assets []githubReleaseAsset) (string, error) {
	for _, asset := range assets {
		if asset.Name == windowsInstallerAssetName && asset.URL != "" {
			return asset.URL, nil
		}
	}

	return "", fmt.Errorf("latest release does not include %s", windowsInstallerAssetName)
}

func downloadFile(url string, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "pangolin-cli-update")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download installer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("installer download failed with %d: %s", resp.StatusCode, string(body))
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create installer file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write installer file: %w", err)
	}

	return nil
}
