package notice

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fosrl/cli/internal/config"
)

const noticesFileName = "notices.json"

type state struct {
	Shown map[string]bool `json:"shown"`
}

func noticesFilePath() (string, error) {
	dir, err := config.GetPangolinConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, noticesFileName), nil
}

func loadState() (*state, error) {
	path, err := noticesFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{Shown: map[string]bool{}}, nil
		}
		return nil, fmt.Errorf("read notices state: %w", err)
	}

	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse notices state: %w", err)
	}
	if s.Shown == nil {
		s.Shown = map[string]bool{}
	}
	return &s, nil
}

func saveState(s *state) error {
	dir, err := config.GetPangolinConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	path, err := noticesFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal notices state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write notices state: %w", err)
	}
	return nil
}

func (s *state) wasShown(id string) bool {
	return s.Shown[id]
}

func (s *state) markShown(id string) {
	s.Shown[id] = true
}
