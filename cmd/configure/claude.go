package configure

import (
	"fmt"
	"path/filepath"
	"runtime"
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

	// Keyless resources still get a helper emitting a placeholder: a non-empty,
	// unusable value forces Claude to invoke the helper at all instead of
	// silently falling back to whatever account/key the user already has
	// configured.
	m["apiKeyHelper"] = apiKeyHelperEcho(keyOrPlaceholder(auth))

	env := ensureMap(m, "env")
	env["ANTHROPIC_BASE_URL"] = endpoint

	if err := writeJSONMap(path, m); err != nil {
		return nil, err
	}

	return []string{path}, nil
}

// apiKeyHelperEcho formats an `echo` command that prints value verbatim, in
// the shell Claude Code actually runs apiKeyHelper through on each platform.
// Claude Code runs it via cmd.exe on Windows (not PowerShell or Git Bash),
// and cmd.exe's echo doesn't strip quote characters at all - wrapping value
// in bash-style single quotes there would leak the quotes into the value
// Claude reads. POSIX shells (macOS/Linux) get single-quoted for safety
// against shell interpretation instead.
func apiKeyHelperEcho(value string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("echo %s", value)
	}
	return fmt.Sprintf("echo '%s'", value)
}

// resetClaudeConfig removes the keys writeClaudeConfig sets from
// ~/.claude/settings.json, preserving every other existing key.
func resetClaudeConfig() ([]string, error) {
	home, err := homeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".claude", "settings.json")

	m, exists, err := readExistingJSONMap(path)
	if err != nil || !exists {
		return nil, err
	}

	changed := deleteKey(m, "apiKeyHelper")
	if env, ok := m["env"].(map[string]interface{}); ok {
		if deleteKey(env, "ANTHROPIC_BASE_URL") {
			changed = true
		}
		pruneEmptyMap(m, "env")
	}

	if !changed {
		return nil, nil
	}
	if err := writeJSONMap(path, m); err != nil {
		return nil, err
	}

	return []string{path}, nil
}
