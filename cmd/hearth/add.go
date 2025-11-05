package main

import (
	"fmt"
	"time"

	"github.com/fmizzell/hearth"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	addTitle       string
	addDescription string
	addParent      string
	addDependsOn   string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new task",
	Long:  `Add a new task to the current workspace with optional parent/dependency relationships.`,
	Run:   addTask,
}

func init() {
	addCmd.Flags().StringVarP(&addTitle, "title", "t", "", "Task title (required)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Task description")
	addCmd.Flags().StringVarP(&addParent, "parent", "p", "", "Parent task ID (for hierarchical tasks)")
	addCmd.Flags().StringVarP(&addDependsOn, "depends-on", "D", "", "Task ID this task depends on")
	addCmd.MarkFlagRequired("title")
}

func addTask(cmd *cobra.Command, args []string) {
	workspaceDir, err := getWorkspaceDir()
	if err != nil {
		fatal("Failed to get workspace directory: %v", err)
	}

	// Load hearth with persistence
	h, err := hearth.NewHearthWithPersistence(workspaceDir)
	if err != nil {
		fatal("Failed to load hearth: %v", err)
	}

	// Generate task ID
	taskID := generateTaskID()

	// Build event
	event := &hearth.TaskCreated{
		TaskID:      taskID,
		Title:       addTitle,
		Description: addDescription,
		Time:        time.Now(),
	}

	if addParent != "" {
		event.ParentID = &addParent
	}

	if addDependsOn != "" {
		event.DependsOn = &addDependsOn
	}

	// Process event
	err = h.Process(event)
	if err != nil {
		fatal("Failed to create task: %v", err)
	}

	// Save to file
	eventsFile := workspaceDir + "/.hearth/events.json"
	err = h.SaveToFile(eventsFile)
	if err != nil {
		fatal("Failed to save events: %v", err)
	}

	// Print confirmation
	fmt.Printf("✓ Task created: %s\n", taskID)
	fmt.Printf("  Title: %s\n", addTitle)
	if addDescription != "" {
		fmt.Printf("  Description: %s\n", addDescription)
	}
	if addParent != "" {
		fmt.Printf("  Parent: %s\n", addParent)
	}
	if addDependsOn != "" {
		fmt.Printf("  Depends on: %s\n", addDependsOn)
	}
}

func generateTaskID() string {
	// Generate short UUID-based ID
	return "T-" + uuid.New().String()[:8]
}
