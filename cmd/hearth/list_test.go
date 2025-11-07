package main

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/fmizzell/hearth"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// setupTestWorkspace creates a temporary workspace for testing
func setupTestWorkspace(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "hearth-test-*")
	assert.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// captureOutput captures stdout during command execution
func captureOutput(f func()) string {
	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
	return buf.String()
}

// TestListWithStatusFilter_Todo tests filtering for todo tasks
func TestListWithStatusFilter_Todo(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with some tasks
	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Add tasks with different statuses
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T1",
		Title:       "Todo task 1",
		Description: "This is pending",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T2",
		Title:       "Todo task 2",
		Description: "This is also pending",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T3",
		Title:       "Completed task",
		Description: "This is done",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Complete T3
	err = h.Process(&hearth.TaskCompleted{
		TaskID: "T3",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Test filtering by todo status
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	// Set status filter to "todo"
	oldStatusFilter := statusFilter
	statusFilter = "todo"
	defer func() { statusFilter = oldStatusFilter }()

	// Capture output
	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify output contains only todo tasks
	assert.Contains(t, output, "T1")
	assert.Contains(t, output, "T2")
	assert.NotContains(t, output, "T3")
	assert.Contains(t, output, "Todo task 1")
	assert.Contains(t, output, "Todo task 2")
	assert.NotContains(t, output, "Completed task")
}

// TestListWithStatusFilter_Pending tests filtering with "pending" alias
func TestListWithStatusFilter_Pending(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with some tasks
	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Add tasks
	err = h.Process(&hearth.TaskCreated{
		TaskID: "T1",
		Title:  "Pending task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID: "T2",
		Title:  "Completed task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCompleted{
		TaskID: "T2",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Test filtering by "pending" (alias for "todo")
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	oldStatusFilter := statusFilter
	statusFilter = "pending"
	defer func() { statusFilter = oldStatusFilter }()

	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify "pending" works as alias for "todo"
	assert.Contains(t, output, "T1")
	assert.NotContains(t, output, "T2")
	assert.Contains(t, output, "Pending task")
}

// TestListWithStatusFilter_InProgress tests filtering for in-progress tasks
func TestListWithStatusFilter_InProgress(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with some tasks
	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Add tasks with different statuses
	err = h.Process(&hearth.TaskCreated{
		TaskID: "T1",
		Title:  "Todo task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID: "T2",
		Title:  "In progress task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Start T2 to put it in-progress
	err = h.Process(&hearth.TaskStarted{
		TaskID: "T2",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Test filtering by in-progress status
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	oldStatusFilter := statusFilter
	statusFilter = "in-progress"
	defer func() { statusFilter = oldStatusFilter }()

	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify output contains only in-progress tasks
	assert.NotContains(t, output, "T1")
	assert.Contains(t, output, "T2")
	assert.Contains(t, output, "In progress task")
}

// TestListWithStatusFilter_Completed tests filtering for completed tasks
func TestListWithStatusFilter_Completed(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with some tasks
	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Add tasks
	err = h.Process(&hearth.TaskCreated{
		TaskID: "T1",
		Title:  "Todo task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID: "T2",
		Title:  "Completed task 1",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID: "T3",
		Title:  "Completed task 2",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Complete T2 and T3
	err = h.Process(&hearth.TaskCompleted{
		TaskID: "T2",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCompleted{
		TaskID: "T3",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Test filtering by completed status
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	oldStatusFilter := statusFilter
	statusFilter = "completed"
	defer func() { statusFilter = oldStatusFilter }()

	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify output contains only completed tasks
	assert.NotContains(t, output, "T1")
	assert.Contains(t, output, "T2")
	assert.Contains(t, output, "T3")
	assert.Contains(t, output, "Completed task 1")
	assert.Contains(t, output, "Completed task 2")
}

// TestListEmptyWorkspace tests behavior when no tasks exist at all
func TestListEmptyWorkspace(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with NO tasks
	_, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Set workspace to test directory
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	// No filters applied
	oldStatusFilter := statusFilter
	statusFilter = ""
	defer func() { statusFilter = oldStatusFilter }()

	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify appropriate message when no tasks exist
	assert.Contains(t, output, "No tasks found.")
}

// TestListWithStatusFilter_NoMatches tests behavior when no tasks match filter
func TestListWithStatusFilter_NoMatches(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with only todo tasks
	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID: "T1",
		Title:  "Todo task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Test filtering by completed status (should find none)
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	oldStatusFilter := statusFilter
	statusFilter = "completed"
	defer func() { statusFilter = oldStatusFilter }()

	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify appropriate message when no matches
	assert.Contains(t, output, "No tasks found with status: completed")
	assert.NotContains(t, output, "T1")
}

// TestListWithStatusFilter_HierarchicalTasks tests filtering with parent/child tasks
func TestListWithStatusFilter_HierarchicalTasks(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with hierarchical tasks
	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Create parent task
	err = h.Process(&hearth.TaskCreated{
		TaskID: "T1",
		Title:  "Parent task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Create child tasks with different statuses
	parentID := "T1"
	err = h.Process(&hearth.TaskCreated{
		TaskID:   "T2",
		Title:    "Child todo task",
		ParentID: &parentID,
		Time:     time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID:   "T3",
		Title:    "Child completed task",
		ParentID: &parentID,
		Time:     time.Now(),
	})
	assert.NoError(t, err)

	// Complete T3
	err = h.Process(&hearth.TaskCompleted{
		TaskID: "T3",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Test filtering by todo status - should show parent and todo child
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	oldStatusFilter := statusFilter
	statusFilter = "todo"
	defer func() { statusFilter = oldStatusFilter }()

	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify hierarchical display with filtering
	assert.Contains(t, output, "T1")
	assert.Contains(t, output, "T2")
	assert.NotContains(t, output, "T3")
	assert.Contains(t, output, "Parent task")
	assert.Contains(t, output, "Child todo task")
	assert.NotContains(t, output, "Child completed task")
}

// TestListWithoutStatusFilter tests that no filter shows all tasks
func TestListWithoutStatusFilter(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with tasks of different statuses
	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID: "T1",
		Title:  "Todo task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID: "T2",
		Title:  "Completed task",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCompleted{
		TaskID: "T2",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Test without any filter
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	oldStatusFilter := statusFilter
	statusFilter = ""
	defer func() { statusFilter = oldStatusFilter }()

	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify all tasks are shown
	assert.Contains(t, output, "T1")
	assert.Contains(t, output, "T2")
	assert.Contains(t, output, "Todo task")
	assert.Contains(t, output, "Completed task")
}

// TestMatchesStatus tests the matchesStatus helper function
func TestMatchesStatus(t *testing.T) {
	tests := []struct {
		taskStatus string
		filter     string
		expected   bool
	}{
		{"todo", "todo", true},
		{"todo", "pending", true}, // pending is alias for todo
		{"todo", "completed", false},
		{"completed", "completed", true},
		{"completed", "todo", false},
		{"in-progress", "in-progress", true},
		{"in-progress", "todo", false},
		{"in-progress", "completed", false},
	}

	for _, tt := range tests {
		t.Run(tt.taskStatus+"_"+tt.filter, func(t *testing.T) {
			result := matchesStatus(tt.taskStatus, tt.filter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestListBasicIntegration is a comprehensive integration test for the basic 'hearth list' command
// with no filters. It verifies all tasks are displayed correctly with proper formatting.
func TestListBasicIntegration(t *testing.T) {
	tmpDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create hearth with a comprehensive set of tasks
	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Create a todo task
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T1",
		Title:       "Implement user authentication",
		Description: "Add JWT-based authentication to the API",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create a completed task
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T2",
		Title:       "Setup database schema",
		Description: "Create initial migration for users table",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCompleted{
		TaskID: "T2",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Create a task with dependency
	dependsOn := "T2"
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T3",
		Title:       "Seed test data",
		Description: "Add sample users to database",
		DependsOn:   &dependsOn,
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create a parent task with children
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T4",
		Title:       "Build frontend components",
		Description: "Create reusable UI components",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create child tasks
	parentID := "T4"
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T5",
		Title:       "Create button component",
		Description: "",
		ParentID:    &parentID,
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T6",
		Title:       "Create form component",
		Description: "",
		ParentID:    &parentID,
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Complete one child task
	err = h.Process(&hearth.TaskCompleted{
		TaskID: "T5",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Execute list command with no filters
	oldWorkspaceFlag := workspaceFlag
	workspaceFlag = tmpDir
	defer func() { workspaceFlag = oldWorkspaceFlag }()

	oldStatusFilter := statusFilter
	statusFilter = ""
	defer func() { statusFilter = oldStatusFilter }()

	output := captureOutput(func() {
		cmd := &cobra.Command{}
		listTasks(cmd, []string{})
	})

	// Verify header
	assert.Contains(t, output, "📋 Tasks:")

	// Verify all task IDs are present
	assert.Contains(t, output, "T1")
	assert.Contains(t, output, "T2")
	assert.Contains(t, output, "T3")
	assert.Contains(t, output, "T4")
	assert.Contains(t, output, "T5")
	assert.Contains(t, output, "T6")

	// Verify all task titles are present
	assert.Contains(t, output, "Implement user authentication")
	assert.Contains(t, output, "Setup database schema")
	assert.Contains(t, output, "Seed test data")
	assert.Contains(t, output, "Build frontend components")
	assert.Contains(t, output, "Create button component")
	assert.Contains(t, output, "Create form component")

	// Verify status icons
	assert.Contains(t, output, "○") // Todo status icon
	assert.Contains(t, output, "✓") // Completed status icon

	// Verify the output contains the proper task IDs in brackets format
	assert.Contains(t, output, "[T1]")
	assert.Contains(t, output, "[T2]")
	assert.Contains(t, output, "[T3]")
	assert.Contains(t, output, "[T4]")
	assert.Contains(t, output, "[T5]")
	assert.Contains(t, output, "[T6]")
}

func strPtr(s string) *string {
	return &s
}
