package configure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// writeGeminiConfig sets GOOGLE_GEMINI_BASE_URL and GEMINI_API_KEY in
// ~/.gemini/.env, which Gemini CLI loads automatically on every run (it's the
// last stop in its .env search path, so this applies globally regardless of
// which directory Gemini CLI is started from), preserving any other lines.
func writeGeminiConfig(endpoint string, auth Auth) ([]string, error) {
	path, err := geminiEnvPath()
	if err != nil {
		return nil, err
	}

	lines, err := readEnvLines(path)
	if err != nil {
		return nil, err
	}

	lines = setEnvLine(lines, "GOOGLE_GEMINI_BASE_URL", endpoint)
	lines = setEnvLine(lines, "GEMINI_API_KEY", keyOrPlaceholder(auth))

	if err := writeFile(path, []byte(strings.Join(lines, "\n"))); err != nil {
		return nil, err
	}

	return []string{path}, nil
}

// resetGeminiConfig removes the keys writeGeminiConfig sets from
// ~/.gemini/.env, preserving every other existing line.
func resetGeminiConfig() ([]string, error) {
	path, err := geminiEnvPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	lines, err := readEnvLines(path)
	if err != nil {
		return nil, err
	}

	lines, changedURL := unsetEnvLine(lines, "GOOGLE_GEMINI_BASE_URL")
	lines, changedKey := unsetEnvLine(lines, "GEMINI_API_KEY")
	if !changedURL && !changedKey {
		return nil, nil
	}

	if err := writeFile(path, []byte(strings.Join(lines, "\n"))); err != nil {
		return nil, err
	}

	return []string{path}, nil
}

// geminiEnvPath resolves Gemini CLI's global .env file path: ~/.gemini/.env,
// Gemini CLI's default on every OS.
func geminiEnvPath() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", ".env"), nil
}

// readEnvLines reads path as a list of raw lines (no trailing blank line for
// a trailing newline), for a dotenv-style KEY=value file. A missing file
// yields no lines, not an error.
func readEnvLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// setEnvLine replaces the first line assigning key (a line starting with
// "key=") with key=value, or appends key=value if key isn't already set.
// Other lines, including comments, are left untouched.
func setEnvLine(lines []string, key, value string) []string {
	prefix := key + "="
	newLine := fmt.Sprintf("%s=%s", key, value)
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = newLine
			return lines
		}
	}
	return append(lines, newLine)
}

// unsetEnvLine removes every line assigning key (a line starting with
// "key="), reporting whether it removed anything.
func unsetEnvLine(lines []string, key string) ([]string, bool) {
	prefix := key + "="
	out := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			changed = true
			continue
		}
		out = append(out, line)
	}
	return out, changed
}
