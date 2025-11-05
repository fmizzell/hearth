package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fmizzell/hearth"
)

// createTask creates a task and saves it to disk
func createTask(workspaceDir, taskID, title, description string, parentID, dependsOn *string) error {
	// Load hearth with persistence
	h, err := hearth.NewHearthWithPersistence(workspaceDir)
	if err != nil {
		return fmt.Errorf("failed to load hearth: %w", err)
	}

	// Create task event
	event := &hearth.TaskCreated{
		TaskID:      taskID,
		Title:       title,
		Description: description,
		ParentID:    parentID,
		DependsOn:   dependsOn,
		Time:        time.Now(),
	}

	// Process event
	err = h.Process(event)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Save to file
	eventsFile := filepath.Join(workspaceDir, ".hearth", "events.json")
	err = h.SaveToFile(eventsFile)
	if err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	return nil
}
