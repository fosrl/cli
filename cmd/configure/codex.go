package configure

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fosrl/cli/internal/logger"
	toml "github.com/pelletier/go-toml/v2"
)

// writeCodexConfig merges a "pangolin" model provider into ~/.codex/config.toml,
// preserving any other existing keys. NOTE: go-toml/v2 doesn't preserve
// comments/formatting on round-trip.
func writeCodexConfig(endpoint string, auth Auth) ([]string, error) {
	home, err := homeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".codex", "config.toml")

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
		logger.Info("  export PANGOLIN_API_KEY=%s", auth.Key)
	}

	return []string{path}, nil
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
