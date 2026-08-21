package configure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// readJSONMap reads and parses a JSON object file into a map, preserving
// unknown keys. A missing file yields an empty map, not an error.
func readJSONMap(path string) (map[string]interface{}, error) {
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
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse %s as JSON: %w", path, err)
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	return m, nil
}

// writeJSONMap writes m as indented JSON to path, creating parent directories
// as needed. Permissions are tightened to 0600 since these files often hold
// secrets.
func writeJSONMap(path string, m map[string]interface{}) error {
	data, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to encode %s: %w", path, err)
	}
	return writeFile(path, data)
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// ensureMap returns the map value at key in parent, creating (and storing) an
// empty one if it's absent or of the wrong type.
func ensureMap(parent map[string]interface{}, key string) map[string]interface{} {
	if existing, ok := parent[key].(map[string]interface{}); ok {
		return existing
	}
	created := map[string]interface{}{}
	parent[key] = created
	return created
}

func homeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return home, nil
}

// readExistingJSONMap reads path like readJSONMap, but reports whether the
// file exists at all so reset can skip files it would otherwise create.
func readExistingJSONMap(path string) (map[string]interface{}, bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to read %s: %w", path, err)
	}

	m, err := readJSONMap(path)
	if err != nil {
		return nil, false, err
	}
	return m, true, nil
}

// deleteKey removes key from parent, reporting whether it was there, so
// callers can tell an actual removal from a no-op.
func deleteKey(parent map[string]interface{}, key string) bool {
	if _, ok := parent[key]; !ok {
		return false
	}
	delete(parent, key)
	return true
}

// pruneEmptyMap deletes key from parent when it holds a now-empty map, so
// reset doesn't leave behind the containers configure created.
func pruneEmptyMap(parent map[string]interface{}, key string) {
	if child, ok := parent[key].(map[string]interface{}); ok && len(child) == 0 {
		delete(parent, key)
	}
}
