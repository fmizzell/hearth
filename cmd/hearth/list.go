package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fmizzell/hearth"
	"github.com/spf13/cobra"
)

var (
	statusFilter string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long:  `List all tasks in the current workspace with their status.`,
	Run:   listTasks,
}

func init() {
	listCmd.Flags().StringVarP(&statusFilter, "status", "s", "", "Filter tasks by status (todo, in-progress, completed)")
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

	// Apply status filter if specified
	filteredTasks := tasks
	if statusFilter != "" {
		filteredTasks = make(map[string]*hearth.Task)
		for id, task := range tasks {
			if matchesStatus(task.Status, statusFilter) {
				filteredTasks[id] = task
			}
		}
	}

	if len(filteredTasks) == 0 {
		fmt.Printf("No tasks found with status: %s\n", statusFilter)
		return
	}

	// Sort tasks by ID for consistent display
	taskIDs := make([]string, 0, len(filteredTasks))
	for id := range filteredTasks {
		taskIDs = append(taskIDs, id)
	}
	sort.Strings(taskIDs)

	// Display task tree
	fmt.Println("📋 Tasks:")
	fmt.Println()

	// First, display root tasks (no parent)
	for _, id := range taskIDs {
		task := filteredTasks[id]
		if task.ParentID == nil {
			displayTaskFiltered(h, task, 0, filteredTasks)
		}
	}
}

// matchesStatus checks if a task status matches the filter
// Supports "pending" as an alias for "todo"
func matchesStatus(taskStatus, filter string) bool {
	// Normalize filter - support "pending" as alias for "todo"
	normalizedFilter := filter
	if filter == "pending" {
		normalizedFilter = "todo"
	}

	return taskStatus == normalizedFilter
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

	// Build display line - just show title for clean hierarchy view
	fmt.Printf("%s%s [%s] %s\n", prefix, statusIcon, task.ID, task.Title)

	// Display children recursively
	children := h.GetChildTasks(task.ID)
	for _, child := range children {
		displayTask(h, child, indent+1)
	}
}

func displayTaskFiltered(h *hearth.Hearth, task *hearth.Task, indent int, filteredTasks map[string]*hearth.Task) {
	prefix := strings.Repeat("  ", indent)

	// Status icon
	statusIcon := "○" // todo
	if task.Status == "completed" {
		statusIcon = "✓"
	} else if task.Status == "in-progress" {
		statusIcon = "→"
	}

	// Build display line - just show title for clean hierarchy view
	fmt.Printf("%s%s [%s] %s\n", prefix, statusIcon, task.ID, task.Title)

	// Display children recursively (only if they're in the filtered set)
	children := h.GetChildTasks(task.ID)
	for _, child := range children {
		if _, inFilter := filteredTasks[child.ID]; inFilter {
			displayTaskFiltered(h, child, indent+1, filteredTasks)
		}
	}

	if indent == 0 {
		fmt.Println()
	}
}
