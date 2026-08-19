package configure

import (
	"os"
	"path/filepath"
)

// writeOpencodeConfig merges the Pangolin AI gateway baseURL into OpenCode's
// global config (~/.config/opencode/opencode.json, respecting $XDG_CONFIG_HOME
// if set) and, when keyed, the credential into its auth store
// (~/.local/share/opencode/auth.json, respecting $XDG_DATA_HOME if set),
// preserving any other existing keys.
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
		authPath, err := opencodeDataPath(home)
		if err != nil {
			return nil, err
		}
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
// $XDG_CONFIG_HOME if set (per OpenCode's documented XDG support). OpenCode
// uses this same literal ~/.config path as its default on every OS, including
// Windows (it doesn't map to %APPDATA%/%LOCALAPPDATA% by default), so no
// GOOS-specific fallback is needed here.
func opencodeConfigPath(home string) (string, error) {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "opencode", "opencode.json"), nil
	}
	return filepath.Join(home, ".config", "opencode", "opencode.json"), nil
}

// opencodeDataPath resolves OpenCode's auth store file path, honoring
// $XDG_DATA_HOME if set, for the same reason opencodeConfigPath honors
// $XDG_CONFIG_HOME: OpenCode's data directory follows the same
// (Windows-including) XDG resolution as its config directory.
func opencodeDataPath(home string) (string, error) {
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "opencode", "auth.json"), nil
	}
	return filepath.Join(home, ".local", "share", "opencode", "auth.json"), nil
}

// resetOpencodeConfig removes the baseURL overrides writeOpencodeConfig adds
// from OpenCode's global config, and the credential it stores from OpenCode's
// auth store, preserving every other existing key in both.
func resetOpencodeConfig() ([]string, error) {
	home, err := homeDir()
	if err != nil {
		return nil, err
	}

	configPath, err := opencodeConfigPath(home)
	if err != nil {
		return nil, err
	}
	authPath, err := opencodeDataPath(home)
	if err != nil {
		return nil, err
	}

	var paths []string

	config, exists, err := readExistingJSONMap(configPath)
	if err != nil {
		return nil, err
	}
	if exists {
		changed := false
		if provider, ok := config["provider"].(map[string]interface{}); ok {
			for _, name := range []string{"anthropic", "openai"} {
				entry, ok := provider[name].(map[string]interface{})
				if !ok {
					continue
				}
				if options, ok := entry["options"].(map[string]interface{}); ok {
					if deleteKey(options, "baseURL") {
						changed = true
					}
					pruneEmptyMap(entry, "options")
				}
				pruneEmptyMap(provider, name)
			}
			pruneEmptyMap(config, "provider")
		}
		if changed {
			if err := writeJSONMap(configPath, config); err != nil {
				return nil, err
			}
			paths = append(paths, configPath)
		}
	}

	authData, exists, err := readExistingJSONMap(authPath)
	if err != nil {
		return nil, err
	}
	if exists && deleteKey(authData, "anthropic") {
		if err := writeJSONMap(authPath, authData); err != nil {
			return nil, err
		}
		paths = append(paths, authPath)
	}

	return paths, nil
}
