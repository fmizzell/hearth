package hearth

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

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

// SaveToFile saves the event log to a file with proper locking and merging
func (h *Hearth) SaveToFile(filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file for read/write, create if not exists
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock file: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	// Read existing events from disk
	var existingEvents []atmos.Event
	fileInfo, _ := file.Stat()
	if fileInfo.Size() > 0 {
		data := make([]byte, fileInfo.Size())
		_, err := file.Read(data)
		if err != nil {
			return fmt.Errorf("failed to read existing events: %w", err)
		}

		// Unmarshal existing events
		temp := NewHearth()
		existingEvents, err = temp.engine.UnmarshalEvents(data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal existing events: %w", err)
		}
	}

	// Merge events: keep existing + add new ones we have that aren't in existing
	merged := mergeEvents(existingEvents, h.engine.GetEvents())

	// Marshal merged events
	data, err := h.engine.MarshalEvents(merged)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Truncate and write
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// mergeEvents combines two event lists, using existing as base and adding any new events from current
func mergeEvents(existing, current []atmos.Event) []atmos.Event {
	// Build map of existing event types and timestamps to detect duplicates
	existingMap := make(map[string]bool)
	for _, e := range existing {
		key := fmt.Sprintf("%s-%d", e.Type(), e.Timestamp().UnixNano())
		existingMap[key] = true
	}

	// Start with existing events
	merged := make([]atmos.Event, len(existing))
	copy(merged, existing)

	// Add any new events from current that aren't in existing
	for _, e := range current {
		key := fmt.Sprintf("%s-%d", e.Type(), e.Timestamp().UnixNano())
		if !existingMap[key] {
			merged = append(merged, e)
		}
	}

	return merged
}
