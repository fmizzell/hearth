package hearth

import (
	"fmt"

	"github.com/cumulusrpg/atmos"
)

// NewHearthWithPersistence creates a Hearth instance with file-based persistence
// Events automatically persist to disk on every Process() call
func NewHearthWithPersistence(workspaceDir string) (*Hearth, error) {
	// Create file repository (pure storage layer, no caching)
	repo, err := NewFileRepository(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create file repository: %w", err)
	}

	// Create hearth with the repository
	// Repository loads events from disk on first GetAll() call
	// and persists on every Add() call
	return NewHearth(atmos.WithRepository(repo)), nil
}
