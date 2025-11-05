package hearth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cumulusrpg/atmos"
)

// NewHearthWithPersistence creates a Hearth instance, loading events from .hearth/events.json if it exists
func NewHearthWithPersistence(workspaceDir string) (*Hearth, error) {
	eventsFile := filepath.Join(workspaceDir, ".hearth", "events.json")

	// Create .hearth directory if it doesn't exist
	hearthDir := filepath.Dir(eventsFile)
	if err := os.MkdirAll(hearthDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .hearth directory: %w", err)
	}

	// Check if state file exists
	if _, err := os.Stat(eventsFile); err == nil {
		// Read state file
		data, err := os.ReadFile(eventsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read state file: %w", err)
		}

		// Create temp hearth to unmarshal events
		temp := NewHearth()
		events, err := temp.engine.UnmarshalEvents(data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		// Create hearth instance with events pre-loaded
		return NewHearth(atmos.WithEvents(events)), nil
	}

	// No existing state, create fresh instance
	return NewHearth(), nil
}

// SaveToFile saves the event log to a file
func (h *Hearth) SaveToFile(filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal events
	data, err := h.engine.MarshalEvents(h.engine.GetEvents())
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
