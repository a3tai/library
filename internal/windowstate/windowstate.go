// Package windowstate persists the desktop app's window geometry between
// runs. State lives at <UserConfigDir>/A3T Library/window.json and is
// written atomically (temp file + rename) so a crash mid-write can't
// corrupt it. Existing pre-rename state is copied into the new directory
// once if no new state file exists yet.
package windowstate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	configDirName       = "A3T Library"
	legacyConfigDirName = "Books"
)

// State captures everything we restore on next launch. Width/Height of 0
// (or negative coordinates outside reasonable bounds) means "use defaults".
type State struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximised bool `json:"maximised"`
}

// Path resolves the on-disk location for the window state file. The parent
// directory is created on demand.
func Path() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cfg, configDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "window.json")
	legacyPath := filepath.Join(cfg, legacyConfigDirName, "window.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if data, readErr := os.ReadFile(legacyPath); readErr == nil {
			_ = os.WriteFile(path, data, 0o644)
		}
	}
	return path, nil
}

// Load reads the saved state. Missing files are not errors — the caller
// gets a zero-value State and uses its own defaults.
func Load() (State, error) {
	p, err := Path()
	if err != nil {
		return State{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		// A corrupt state file shouldn't break startup; fall back to defaults.
		return State{}, nil
	}
	return s, nil
}

// Save writes state atomically. Errors are returned for logging but
// callers typically swallow them — losing one frame of position data
// is not worth crashing the app over.
func Save(s State) error {
	p, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Equal reports whether two states would result in identical window
// geometry — used to skip redundant disk writes when nothing changed.
func (s State) Equal(other State) bool {
	return s.X == other.X &&
		s.Y == other.Y &&
		s.Width == other.Width &&
		s.Height == other.Height &&
		s.Maximised == other.Maximised
}

// Valid reports whether the state has plausible dimensions worth
// restoring. Brand-new installs (zero State) return false so callers
// fall back to their defaults instead of opening a 0×0 window.
func (s State) Valid() bool {
	return s.Width >= 320 && s.Height >= 240
}
