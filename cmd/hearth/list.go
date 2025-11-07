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

	// Get tasks in depth-first execution order
	orderedTasks := getTasksInDepthFirstOrder(filteredTasks)

	// Display task tree
	fmt.Println("📋 Tasks:")
	fmt.Println()

	// Display each task in depth-first order with proper indentation
	for _, task := range orderedTasks {
		indent := calculateIndent(task, filteredTasks)
		displayTaskLine(task, indent)
	}
}

// getTasksInDepthFirstOrder returns tasks sorted in depth-first execution order
func getTasksInDepthFirstOrder(tasks map[string]*hearth.Task) []*hearth.Task {
	// Convert map to slice
	var taskSlice []*hearth.Task
	for _, task := range tasks {
		taskSlice = append(taskSlice, task)
	}

	// Find roots and sort by creation time
	var roots []*hearth.Task
	for _, task := range taskSlice {
		if task.ParentID == nil {
			roots = append(roots, task)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].CreatedAt.Before(roots[j].CreatedAt)
	})

	// Collect all tasks in depth-first order
	var ordered []*hearth.Task
	taskMap := tasks // Use the original map
	for _, root := range roots {
		collectDepthFirst(root, taskMap, &ordered)
	}

	return ordered
}

// collectDepthFirst recursively collects tasks in depth-first order
func collectDepthFirst(task *hearth.Task, taskMap map[string]*hearth.Task, result *[]*hearth.Task) {
	*result = append(*result, task)

	// Get children and sort by creation time
	var children []*hearth.Task
	for _, t := range taskMap {
		if t.ParentID != nil && *t.ParentID == task.ID {
			children = append(children, t)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].CreatedAt.Before(children[j].CreatedAt)
	})

	// Recursively collect children
	for _, child := range children {
		collectDepthFirst(child, taskMap, result)
	}
}

// calculateIndent calculates the indentation level for a task based on its depth
func calculateIndent(task *hearth.Task, taskMap map[string]*hearth.Task) int {
	depth := 0
	current := task
	for current.ParentID != nil {
		parent := taskMap[*current.ParentID]
		if parent == nil {
			break
		}
		depth++
		current = parent
	}
	return depth
}

// displayTaskLine displays a single task line with proper formatting
func displayTaskLine(task *hearth.Task, indent int) {
	prefix := strings.Repeat("  ", indent)

	// Status icon
	statusIcon := "○" // todo
	if task.Status == "completed" {
		statusIcon = "✓"
	} else if task.Status == "in-progress" {
		statusIcon = "→"
	}

	fmt.Printf("%s%s [%s] %s\n", prefix, statusIcon, task.ID, task.Title)
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
