package configure

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fosrl/cli/internal/logger"
	toml "github.com/pelletier/go-toml/v2"
)

// writeCodexConfig merges a "pangolin" model provider into
// $CODEX_HOME/config.toml (CODEX_HOME defaults to ~/.codex, matching Codex's
// own resolution), preserving any other existing keys. NOTE: go-toml/v2
// doesn't preserve comments/formatting on round-trip.
func writeCodexConfig(endpoint string, auth Auth) ([]string, error) {
	path, err := codexConfigPath()
	if err != nil {
		return nil, err
	}

	m, err := readTOMLMap(path)
	if err != nil {
		return nil, err
	}

	m["model_provider"] = "pangolin"

	providers, ok := m["model_providers"].(map[string]interface{})
	if !ok {
		providers = map[string]interface{}{}
		m["model_providers"] = providers
	}

	pangolin := map[string]interface{}{
		"name":     "Pangolin AI Gateway",
		"base_url": endpoint + "/v1",
		"wire_api": "responses",
	}
	if auth.Mode == AuthModeKeyed {
		pangolin["env_key"] = "PANGOLIN_API_KEY"
	}
	providers["pangolin"] = pangolin

	data, err := toml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", path, err)
	}
	if err := writeFile(path, data); err != nil {
		return nil, err
	}

	if auth.Mode == AuthModeKeyed {
		logger.Info("Codex reads its API key from the PANGOLIN_API_KEY environment variable; run:")
		logger.Info("  %s", exportEnvVarCommand("PANGOLIN_API_KEY", auth.Key))
	}

	return []string{path}, nil
}

// codexConfigPath resolves Codex's config file path, honoring $CODEX_HOME if
// set (Codex's own override for relocating its whole config directory);
// otherwise it defaults to ~/.codex/config.toml, which is Codex's default on
// every OS (e.g. %USERPROFILE%\.codex\config.toml on Windows, not an AppData
// path).
func codexConfigPath() (string, error) {
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		return filepath.Join(codexHome, "config.toml"), nil
	}
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// exportEnvVarCommand formats a shell command to set an environment variable
// for the user's current session, using the syntax for their platform's
// default shell (PowerShell on Windows, POSIX export elsewhere).
func exportEnvVarCommand(name, value string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`$env:%s = "%s"`, name, value)
	}
	return fmt.Sprintf("export %s=%s", name, value)
}

func readTOMLMap(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}

	var m map[string]interface{}
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse %s as TOML: %w", path, err)
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	return m, nil
}

// resetCodexConfig removes the model provider writeCodexConfig adds from
// $CODEX_HOME/config.toml, preserving every other existing key. NOTE:
// go-toml/v2 doesn't preserve comments/formatting on round-trip.
func resetCodexConfig() ([]string, error) {
	path, err := codexConfigPath()
	if err != nil {
		return nil, err
	}

	m, exists, err := readExistingTOMLMap(path)
	if err != nil || !exists {
		return nil, err
	}

	changed := false
	// Only clear the active provider if it's still pointing at ours - the
	// user may have switched back to another provider by hand.
	if m["model_provider"] == "pangolin" {
		delete(m, "model_provider")
		changed = true
	}
	if providers, ok := m["model_providers"].(map[string]interface{}); ok {
		if deleteKey(providers, "pangolin") {
			changed = true
		}
		pruneEmptyMap(m, "model_providers")
	}

	if !changed {
		return nil, nil
	}

	data, err := toml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", path, err)
	}
	if err := writeFile(path, data); err != nil {
		return nil, err
	}

	logger.Info("If you set PANGOLIN_API_KEY for Codex, unset it; run:")
	logger.Info("  %s", unsetEnvVarCommand("PANGOLIN_API_KEY"))

	return []string{path}, nil
}

// unsetEnvVarCommand formats the counterpart to exportEnvVarCommand, in the
// syntax for the platform's default shell.
func unsetEnvVarCommand(name string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`Remove-Item Env:\%s`, name)
	}
	return fmt.Sprintf("unset %s", name)
}

// readExistingTOMLMap reads path like readTOMLMap, but reports whether the
// file exists at all so reset can skip files it would otherwise create.
func readExistingTOMLMap(path string) (map[string]interface{}, bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to read %s: %w", path, err)
	}

	m, err := readTOMLMap(path)
	if err != nil {
		return nil, false, err
	}
	return m, true, nil
}
