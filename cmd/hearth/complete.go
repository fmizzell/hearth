package main

import (
	"fmt"
	"time"

	"github.com/fmizzell/hearth"
	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete <task-id>",
	Short: "Complete a task",
	Long:  `Mark a task as completed in the current workspace.`,
	Args:  cobra.ExactArgs(1),
	Run:   completeTask,
}

func completeTask(cmd *cobra.Command, args []string) {
	taskID := args[0]

	workspaceDir, err := getWorkspaceDir()
	if err != nil {
		fatal("Failed to get workspace directory: %v", err)
	}

	// Load hearth with persistence
	h, err := hearth.NewHearthWithPersistence(workspaceDir)
	if err != nil {
		fatal("Failed to load hearth: %v", err)
	}

	// Check if task exists
	task := h.GetTask(taskID)
	if task == nil {
		fatal("Task not found: %s", taskID)
	}

	// Check if already completed
	if task.Status == "completed" {
		fmt.Printf("Task %s is already completed.\n", taskID)
		return
	}

	// Process completion event (auto-persists via FileRepository)
	err = h.Process(&hearth.TaskCompleted{
		TaskID: taskID,
		Time:   time.Now(),
	})
	if err != nil {
		fatal("Failed to complete task: %v", err)
	}

	fmt.Printf("✓ Task completed: %s\n", taskID)
	fmt.Printf("  %s\n", task.Title)

	// Check if parent auto-completed
	if task.ParentID != nil {
		parent := h.GetTask(*task.ParentID)
		if parent != nil && parent.Status == "completed" {
			fmt.Printf("  ✓ Parent task also completed: %s (%s)\n", *task.ParentID, parent.Title)
		}
	}
}
