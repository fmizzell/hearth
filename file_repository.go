package hearth

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/cumulusrpg/atmos"
)

// FileRepository implements atmos.EventRepository as a pure storage layer
// No caching - always reads/writes to file. File locking prevents races.
type FileRepository struct {
	filePath string
}

// NewFileRepository creates a new file-based event repository
func NewFileRepository(workspaceDir string) (*FileRepository, error) {
	filePath := filepath.Join(workspaceDir, ".hearth", "events.json")

	// Create .hearth directory if it doesn't exist
	hearthDir := filepath.Dir(filePath)
	if err := os.MkdirAll(hearthDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .hearth directory: %w", err)
	}

	return &FileRepository{
		filePath: filePath,
	}, nil
}

// Add commits a new event to storage
// Lock → Read all → Append → Write → Unlock
func (r *FileRepository) Add(engine *atmos.Engine, event atmos.Event) error {
	return r.withFileLock(func(file *os.File) error {
		// Read existing events
		existing, err := r.readEvents(file, engine)
		if err != nil {
			return err
		}

		// Append new event
		existing = append(existing, event)

		// Write all events back
		return r.writeEvents(file, engine, existing)
	})
}

// GetAll returns all events from file
// Lock → Read → Unmarshal → Unlock → Return
func (r *FileRepository) GetAll(engine *atmos.Engine) []atmos.Event {
	var events []atmos.Event

	err := r.withFileLock(func(file *os.File) error {
		var err error
		events, err = r.readEvents(file, engine)
		return err
	})

	if err != nil {
		// Return empty slice on error (file might not exist yet)
		return []atmos.Event{}
	}

	return events
}

// SetAll atomically replaces all events
// Lock → Write → Unlock
func (r *FileRepository) SetAll(engine *atmos.Engine, events []atmos.Event) error {
	return r.withFileLock(func(file *os.File) error {
		return r.writeEvents(file, engine, events)
	})
}

// withFileLock executes a function with the file locked
func (r *FileRepository) withFileLock(fn func(*os.File) error) error {
	// Open file for read/write, create if not exists
	file, err := os.OpenFile(r.filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock file: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	// Execute function with locked file
	return fn(file)
}

// readEvents reads and unmarshals events from a file
func (r *FileRepository) readEvents(file *os.File, engine *atmos.Engine) ([]atmos.Event, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Empty file
	if fileInfo.Size() == 0 {
		return []atmos.Event{}, nil
	}

	// Read file contents
	data := make([]byte, fileInfo.Size())
	if _, err := file.ReadAt(data, 0); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal events
	events, err := engine.UnmarshalEvents(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	return events, nil
}

// writeEvents marshals and writes events to a file
func (r *FileRepository) writeEvents(file *os.File, engine *atmos.Engine, events []atmos.Event) error {
	// Marshal events
	data, err := engine.MarshalEvents(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Truncate file
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}

	// Seek to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// Write data
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
