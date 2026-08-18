package configure

import (
	"os"
	"path/filepath"
)

// writeOpencodeConfig merges the Pangolin AI gateway baseURL into OpenCode's
// global config (~/.config/opencode/opencode.json, respecting $XDG_CONFIG_HOME
// if set) and, when keyed, the credential into its auth store
// (~/.local/share/opencode/auth.json), preserving any other existing keys.
func writeOpencodeConfig(endpoint string, auth Auth) ([]string, error) {
	home, err := homeDir()
	if err != nil {
		return nil, err
	}

	configPath, err := opencodeConfigPath(home)
	if err != nil {
		return nil, err
	}

	config, err := readJSONMap(configPath)
	if err != nil {
		return nil, err
	}

	if _, ok := config["$schema"]; !ok {
		config["$schema"] = "https://opencode.ai/config.json"
	}

	provider := ensureMap(config, "provider")
	for _, name := range []string{"anthropic", "openai"} {
		options := ensureMap(ensureMap(provider, name), "options")
		options["baseURL"] = endpoint + "/v1"
	}

	if err := writeJSONMap(configPath, config); err != nil {
		return nil, err
	}
	paths := []string{configPath}

	if auth.Mode == AuthModeKeyed {
		authPath := filepath.Join(home, ".local", "share", "opencode", "auth.json")
		authData, err := readJSONMap(authPath)
		if err != nil {
			return nil, err
		}
		authData["anthropic"] = map[string]interface{}{
			"type": "api",
			"key":  auth.Key,
		}
		if err := writeJSONMap(authPath, authData); err != nil {
			return nil, err
		}
		paths = append(paths, authPath)
	}

	return paths, nil
}

// opencodeConfigPath resolves OpenCode's global config file path, honoring
// $XDG_CONFIG_HOME if set (per OpenCode's documented XDG support).
func opencodeConfigPath(home string) (string, error) {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "opencode", "opencode.json"), nil
	}
	return filepath.Join(home, ".config", "opencode", "opencode.json"), nil
}
