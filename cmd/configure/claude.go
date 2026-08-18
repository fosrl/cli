package configure

import (
	"fmt"
	"path/filepath"
)

// writeClaudeConfig merges the Pangolin AI gateway settings into
// ~/.claude/settings.json, preserving any other existing keys.
func writeClaudeConfig(endpoint string, auth Auth) ([]string, error) {
	home, err := homeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".claude", "settings.json")

	m, err := readJSONMap(path)
	if err != nil {
		return nil, err
	}

	if auth.Mode == AuthModeKeyed {
		m["apiKeyHelper"] = fmt.Sprintf("echo '%s'", auth.Key)
	}

	env := ensureMap(m, "env")
	env["ANTHROPIC_BASE_URL"] = endpoint

	if err := writeJSONMap(path, m); err != nil {
		return nil, err
	}

	return []string{path}, nil
}
