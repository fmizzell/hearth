package main

import (
	"fmt"
	"time"

	"github.com/fmizzell/hearth"
)

// createTask creates a task and saves it to disk
func createTask(workspaceDir, taskID, title, description string, parentID *string) error {
	// Load hearth with persistence
	h, err := hearth.NewHearth(workspaceDir)
	if err != nil {
		return fmt.Errorf("failed to load hearth: %w", err)
	}

	// Create task event
	event := &hearth.TaskCreated{
		TaskID:      taskID,
		Title:       title,
		Description: description,
		ParentID:    parentID,
		Time:        time.Now(),
	}

	// Process event (auto-persists via FileRepository)
	err = h.Process(event)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}
