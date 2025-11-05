package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fmizzell/hearth"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long:  `List all tasks in the current workspace with their status.`,
	Run:   listTasks,
}

func listTasks(cmd *cobra.Command, args []string) {
	workspaceDir, err := getWorkspaceDir()
	if err != nil {
		fatal("Failed to get workspace directory: %v", err)
	}

	// Load hearth with persistence
	h, err := hearth.NewHearthWithPersistence(workspaceDir)
	if err != nil {
		fatal("Failed to load hearth: %v", err)
	}

	tasks := h.GetTasks()
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	// Sort tasks by ID for consistent display
	taskIDs := make([]string, 0, len(tasks))
	for id := range tasks {
		taskIDs = append(taskIDs, id)
	}
	sort.Strings(taskIDs)

	// Display task tree
	fmt.Println("📋 Tasks:")
	fmt.Println()

	// First, display root tasks (no parent)
	for _, id := range taskIDs {
		task := tasks[id]
		if task.ParentID == nil {
			displayTask(h, task, 0)
		}
	}
}

func displayTask(h *hearth.Hearth, task *hearth.Task, indent int) {
	prefix := strings.Repeat("  ", indent)

	// Status icon
	statusIcon := "○" // todo
	if task.Status == "completed" {
		statusIcon = "✓"
	} else if task.Status == "in-progress" {
		statusIcon = "→"
	}

	// Build display line
	fmt.Printf("%s%s [%s] %s\n", prefix, statusIcon, task.ID, task.Title)

	if task.Description != "" && indent == 0 {
		fmt.Printf("%s   %s\n", prefix, task.Description)
	}

	// Display metadata for root tasks
	if indent == 0 {
		if task.DependsOn != nil {
			fmt.Printf("%s   Depends on: %s\n", prefix, *task.DependsOn)
		}
	}

	// Display children recursively
	children := h.GetChildTasks(task.ID)
	for _, child := range children {
		displayTask(h, child, indent+1)
	}

	if indent == 0 {
		fmt.Println()
	}
}
