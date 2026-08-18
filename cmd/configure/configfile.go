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
