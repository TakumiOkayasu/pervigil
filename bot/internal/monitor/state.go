package monitor

import (
	"encoding/json"
	"os"
)

// FileStateStore persists state to a file
type FileStateStore struct {
	path string
}

// NewFileStateStore creates a new file-based state store
func NewFileStateStore(path string) *FileStateStore {
	return &FileStateStore{path: path}
}

// Load reads the state from file
func (s *FileStateStore) Load() (MonitorState, error) {
	defaultState := MonitorState{TempState: StateNormal, SpeedLimited: false}

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return defaultState, nil
	}
	if err != nil {
		return defaultState, err
	}

	var state MonitorState
	if err := json.Unmarshal(data, &state); err != nil {
		return defaultState, nil
	}

	// Validate TempState
	if state.TempState != StateNormal && state.TempState != StateWarning && state.TempState != StateCritical {
		state.TempState = StateNormal
	}

	return state, nil
}

// Save writes the state to file
func (s *FileStateStore) Save(state MonitorState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
