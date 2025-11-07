package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fmizzell/hearth"
	"github.com/stretchr/testify/assert"
)

// TestExecuteTasks_StoresResult tests that executing tasks stores Claude's response to files
func TestExecuteTasks_StoresResult(t *testing.T) {
	tmpDir := t.TempDir()

	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Create a task
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T-test-123",
		Title:       "Analyze code",
		Description: "Check for issues in auth.go",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Mock Claude's response
	expectedResponse := "Found 3 issues:\n1. Null pointer in auth.go:42\n2. Missing error check\n3. Race condition"

	mockClaudeCaller := func(prompt, workDir string) (string, error) {
		return expectedResponse, nil
	}

	// Execute all tasks (runs the loop) - nil events channel for tests
	err = executeTasks(h, tmpDir, mockClaudeCaller, nil)
	assert.NoError(t, err)

	// Verify result file was created
	resultPath := filepath.Join(tmpDir, ".hearth", "results", "T-test-123.md")
	assert.FileExists(t, resultPath, "Result file should be created")

	// Verify result content matches Claude's response
	content, err := os.ReadFile(resultPath)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, string(content))

	// Verify task was marked completed (FileRepository auto-reloads from disk)
	task := h.GetTask("T-test-123")
	assert.Equal(t, "completed", task.Status)
}

// TestExecuteTasks_WithSubtasks tests that tasks with children don't get marked complete
func TestExecuteTasks_WithSubtasks(t *testing.T) {
	tmpDir := t.TempDir()

	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Create parent task
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T-parent",
		Title:       "Big task",
		Description: "Do analysis",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	task := h.GetTask("T-parent")
	parentID := task.ID

	callCount := 0
	mockClaudeCaller := func(prompt, workDir string) (string, error) {
		callCount++
		// First call: parent creates subtask
		if callCount == 1 {
			h2, _ := hearth.NewHearthWithPersistence(workDir)
			_ = h2.Process(&hearth.TaskCreated{
				TaskID:   "T-child",
				Title:    "Subtask",
				ParentID: &parentID,
				Time:     time.Now(),
			})
			return "Created subtask T-child", nil
		}
		// Second call: child executes (doesn't create more children)
		if callCount == 2 {
			return "Child task completed", nil
		}
		// Third call: parent summary
		if callCount == 3 {
			return "Summary: All subtasks completed successfully", nil
		}
		return "Unexpected call", nil
	}

	// Execute all tasks (processes parent, finds child, processes child) - nil events channel for tests
	err = executeTasks(h, tmpDir, mockClaudeCaller, nil)
	assert.NoError(t, err)

	// Verify result files were created for both parent and child
	parentResultPath := filepath.Join(tmpDir, ".hearth", "results", "T-parent.md")
	assert.FileExists(t, parentResultPath)

	childResultPath := filepath.Join(tmpDir, ".hearth", "results", "T-child.md")
	assert.FileExists(t, childResultPath)

	// FileRepository auto-reloads, so we can check status directly
	parent := h.GetTask("T-parent")
	assert.Equal(t, "completed", parent.Status, "Parent should complete after child completes")

	// Verify child exists and is completed
	child := h.GetTask("T-child")
	assert.NotNil(t, child)
	assert.Equal(t, "T-parent", *child.ParentID)
	assert.Equal(t, "completed", child.Status)
}

// TestExecuteTasks_ParentSummary tests that when all children complete, parent result is re-generated with summary
func TestExecuteTasks_ParentSummary(t *testing.T) {
	tmpDir := t.TempDir()

	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Create parent task
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T-parent",
		Title:       "Analyze project",
		Description: "Create a full analysis report",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	task := h.GetTask("T-parent")
	parentID := task.ID

	callCount := 0
	mockClaudeCaller := func(prompt, workDir string) (string, error) {
		callCount++

		// First call: parent creates two children
		if callCount == 1 {
			h2, _ := hearth.NewHearthWithPersistence(workDir)
			_ = h2.Process(&hearth.TaskCreated{
				TaskID:   "T-child-1",
				Title:    "Count files",
				ParentID: &parentID,
				Time:     time.Now(),
			})
			_ = h2.Process(&hearth.TaskCreated{
				TaskID:   "T-child-2",
				Title:    "List packages",
				ParentID: &parentID,
				Time:     time.Now(),
			})
			return "Created two subtasks", nil
		}

		// Second call: child 1 executes
		if callCount == 2 {
			return "Found 42 Go files", nil
		}

		// Third call: child 2 executes
		if callCount == 3 {
			return "Main packages: cmd/hearth, hearth, prompts", nil
		}

		// Fourth call: parent summary (should mention children's results)
		if callCount == 4 {
			// Verify prompt contains child result references
			if !strings.Contains(prompt, ".hearth/results/T-child-1.md") {
				t.Errorf("Summary prompt should reference child 1 result file")
			}
			if !strings.Contains(prompt, ".hearth/results/T-child-2.md") {
				t.Errorf("Summary prompt should reference child 2 result file")
			}
			return "SUMMARY: Project has 42 Go files across packages cmd/hearth, hearth, and prompts", nil
		}

		return "Unexpected call", nil
	}

	// Execute all tasks
	err = executeTasks(h, tmpDir, mockClaudeCaller, nil)
	assert.NoError(t, err)

	// Verify parent result was overwritten with summary
	parentResultPath := filepath.Join(tmpDir, ".hearth", "results", "T-parent.md")
	content, err := os.ReadFile(parentResultPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "SUMMARY:", "Parent result should contain the summary")
	assert.NotContains(t, string(content), "Created two subtasks", "Parent result should be replaced, not appended")

	// Verify parent is completed
	parent := h.GetTask("T-parent")
	assert.Equal(t, "completed", parent.Status)

	// Should have made exactly 4 calls to Claude
	assert.Equal(t, 4, callCount, "Should call Claude 4 times: parent, child1, child2, parent summary")
}

// TestExecuteTasks_ContextInjection tests that child tasks receive parent chain and sibling context
func TestExecuteTasks_ContextInjection(t *testing.T) {
	tmpDir := t.TempDir()

	h, err := hearth.NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	// Create root task
	err = h.Process(&hearth.TaskCreated{
		TaskID:      "T-root",
		Title:       "Root task",
		Description: "The ultimate goal",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	rootID := "T-root"

	callCount := 0
	mockClaudeCaller := func(prompt, workDir string) (string, error) {
		callCount++

		// First call: root creates two children
		if callCount == 1 {
			h2, _ := hearth.NewHearthWithPersistence(workDir)
			_ = h2.Process(&hearth.TaskCreated{
				TaskID:   "T-child-1",
				Title:    "First child",
				ParentID: &rootID,
				Time:     time.Now(),
			})
			_ = h2.Process(&hearth.TaskCreated{
				TaskID:   "T-child-2",
				Title:    "Second child",
				ParentID: &rootID,
				Time:     time.Now(),
			})
			return "Created two children", nil
		}

		// Second call: child 1 executes - should see parent chain but NO siblings yet
		if callCount == 2 {
			assert.Contains(t, prompt, "ROOT TASK:", "Should show root task")
			assert.Contains(t, prompt, "The ultimate goal", "Should show root description")
			assert.NotContains(t, prompt, "PREVIOUS SIBLING", "First child has no previous siblings")
			return "Child 1 result", nil
		}

		// Third call: child 2 executes - should see parent chain AND child 1's result
		if callCount == 3 {
			assert.Contains(t, prompt, "ROOT TASK:", "Should show root task")
			assert.Contains(t, prompt, "The ultimate goal", "Should show root description")
			assert.Contains(t, prompt, "PREVIOUS SIBLING RESULTS:", "Should show sibling section")
			assert.Contains(t, prompt, "T-child-1", "Should reference child 1")
			assert.Contains(t, prompt, "First child", "Should show child 1 title")
			assert.Contains(t, prompt, ".hearth/results/T-child-1.md", "Should point to child 1 result file")
			return "Child 2 result", nil
		}

		// Fourth call: parent summary
		if callCount == 4 {
			return "Summary of both children", nil
		}

		return "Unexpected call", nil
	}

	// Execute all tasks
	err = executeTasks(h, tmpDir, mockClaudeCaller, nil)
	assert.NoError(t, err)

	assert.Equal(t, 4, callCount)
}
