package configure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

// promptOpencodeProviders asks which OpenCode provider(s) to point at the
// Pangolin AI gateway. OpenCode configures baseURL and API keys per provider
// (unlike Claude/Codex, which each speak for a single fixed provider), so the
// user types a comma-separated list rather than picking from a fixed set -
// this works for any provider id OpenCode recognizes, not just the two most
// common ones.
func promptOpencodeProviders() ([]string, error) {
	input := "anthropic,openai"

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Which OpenCode provider(s) should use this gateway?").
				Description("Comma-separated provider ids, e.g. anthropic,openai").
				Value(&input).
				Validate(func(s string) error {
					if len(parseProviderList(s)) == 0 {
						return fmt.Errorf("enter at least one provider id")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("error selecting providers: %w", err)
	}

	return parseProviderList(input), nil
}

// parseProviderList splits a comma-separated provider list into trimmed,
// lowercased, deduplicated, non-empty provider ids.
func parseProviderList(s string) []string {
	seen := map[string]bool{}
	var providers []string
	for _, part := range strings.Split(s, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		providers = append(providers, name)
	}
	return providers
}

// writeOpencodeConfig merges the Pangolin AI gateway baseURL into OpenCode's
// global config (~/.config/opencode/opencode.json, respecting $XDG_CONFIG_HOME
// if set) and, when keyed, the credential into its auth store
// (~/.local/share/opencode/auth.json, respecting $XDG_DATA_HOME if set), for
// each provider the user chooses, preserving any other existing keys.
func writeOpencodeConfig(endpoint string, auth Auth) ([]string, error) {
	providers, err := promptOpencodeProviders()
	if err != nil {
		return nil, err
	}

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
	for _, name := range providers {
		options := ensureMap(ensureMap(provider, name), "options")
		options["baseURL"] = endpoint + "/v1"
	}

	if err := writeJSONMap(configPath, config); err != nil {
		return nil, err
	}
	paths := []string{configPath}

	// The credential is written even for keyless resources: OpenCode refuses
	// to start a provider with no key at all ("OpenAI API key is missing"),
	// so it gets an inert placeholder instead of an omitted entry.
	authPath, err := opencodeDataPath(home)
	if err != nil {
		return nil, err
	}
	authData, err := readJSONMap(authPath)
	if err != nil {
		return nil, err
	}
	for _, name := range providers {
		authData[name] = map[string]interface{}{
			"type": "api",
			"key":  keyOrPlaceholder(auth),
		}
	}
	if err := writeJSONMap(authPath, authData); err != nil {
		return nil, err
	}
	paths = append(paths, authPath)

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
	var touchedProviders []string

	config, exists, err := readExistingJSONMap(configPath)
	if err != nil {
		return nil, err
	}
	if exists {
		changed := false
		if provider, ok := config["provider"].(map[string]interface{}); ok {
			// Providers are user-chosen at configure time (see
			// promptOpencodeProviders), so reset can't know their names in
			// advance - scan whatever's present instead of a fixed list.
			for name, raw := range provider {
				entry, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				if options, ok := entry["options"].(map[string]interface{}); ok {
					if deleteKey(options, "baseURL") {
						changed = true
						touchedProviders = append(touchedProviders, name)
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
	if exists {
		changed := false
		for _, name := range touchedProviders {
			if deleteKey(authData, name) {
				changed = true
			}
		}
		if changed {
			if err := writeJSONMap(authPath, authData); err != nil {
				return nil, err
			}
			paths = append(paths, authPath)
		}
	}

	return paths, nil
}
