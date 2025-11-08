package hearth

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fmizzell/hearth/prompts"
)

// ClaudeCaller is an interface for calling Claude (allows mocking in tests)
type ClaudeCaller interface {
	Call(prompt, workDir string) (string, error)
}

// DefaultClaudeCaller uses the claude CLI
type DefaultClaudeCaller struct{}

func (c *DefaultClaudeCaller) Call(prompt, workDir string) (string, error) {
	cmd := exec.Command("claude",
		"--print",                        // Non-interactive output
		"--dangerously-skip-permissions", // Skip permission prompts (safe: sandboxed to workDir)
		prompt,
	)

	// Set Claude's working directory
	cmd.Dir = workDir

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// ExecuteTask handles task execution: builds context, calls Claude, stores result
// This is the business logic extracted from cmd/hearth/run.go for reuse in orchestration
func ExecuteTask(taskID string, tasks map[string]*Task, workspaceDir string, claudeCaller ClaudeCaller) (string, error) {
	task := tasks[taskID]
	if task == nil {
		return "", fmt.Errorf("task not found: %s", taskID)
	}

	// Build context: parent chain + sibling results
	contextInfo := BuildTaskContext(taskID, tasks, workspaceDir)

	// Build full prompt with task context and instructions
	taskContext := fmt.Sprintf(`
CURRENT TASK: %s
CURRENT TASK ID: %s

IMPORTANT: Before starting work, assess if this task should be broken into subtasks.
If this task involves multiple steps or can be parallelized, you MUST create subtasks first.

`, task.Title, task.ID)

	prompt := task.Description
	if prompt == "" {
		prompt = task.Title // Fallback to title if no description
	}

	fullPrompt := contextInfo + taskContext + prompt + "\n" + prompts.TaskSystemInstructions

	// Call Claude with the task description as the prompt
	response, err := claudeCaller.Call(fullPrompt, workspaceDir)
	if err != nil {
		return "", fmt.Errorf("failed to call Claude: %w", err)
	}

	// Store result to .hearth/results/<task-id>.md
	resultPath, err := StoreTaskResult(workspaceDir, task.ID, response)
	if err != nil {
		return "", fmt.Errorf("failed to store result: %w", err)
	}

	return resultPath, nil
}

// BuildTaskContext builds context for a task including parent chain and sibling results
func BuildTaskContext(taskID string, tasks map[string]*Task, workspaceDir string) string {
	task := tasks[taskID]
	if task == nil {
		return ""
	}

	var context strings.Builder

	// Build parent chain to root
	parentChain := buildParentChain(task, tasks)
	if len(parentChain) > 0 {
		// Root task is at the end of the chain
		root := parentChain[len(parentChain)-1]
		context.WriteString(fmt.Sprintf("ROOT TASK: %s\n", root.Title))
		if root.Description != "" {
			context.WriteString(fmt.Sprintf("ROOT GOAL: %s\n", root.Description))
		}
		context.WriteString("\n")

		// Show parent hierarchy if there are intermediate parents
		if len(parentChain) > 1 {
			context.WriteString("PARENT CHAIN:\n")
			// Walk from root down to immediate parent
			for i := len(parentChain) - 1; i >= 0; i-- {
				parent := parentChain[i]
				indent := strings.Repeat("  ", len(parentChain)-1-i)
				context.WriteString(fmt.Sprintf("%s└─ %s \"%s\"\n", indent, parent.ID, parent.Title))
			}
			context.WriteString("\n")
		}
	}

	// Find and list completed sibling results
	if task.ParentID != nil {
		var completedSiblings []*Task

		// Get siblings that completed before this task (by creation time)
		for _, t := range tasks {
			if t.ParentID != nil && *t.ParentID == *task.ParentID {
				if t.ID != task.ID && t.Status == "completed" {
					completedSiblings = append(completedSiblings, t)
				}
			}
		}

		if len(completedSiblings) > 0 {
			context.WriteString("PREVIOUS SIBLING RESULTS:\n")
			context.WriteString("Your siblings have already completed work. You can reference their findings:\n\n")
			for _, sibling := range completedSiblings {
				resultPath := fmt.Sprintf(".hearth/results/%s.md", sibling.ID)
				context.WriteString(fmt.Sprintf("- %s \"%s\" → Result: %s\n", sibling.ID, sibling.Title, resultPath))
			}
			context.WriteString("\nYou can read these files to avoid duplicating work and build on their findings.\n\n")
		}
	}

	if context.Len() > 0 {
		return context.String() + "---\n\n"
	}
	return ""
}

// buildParentChain walks up the task hierarchy and returns chain from immediate parent to root
func buildParentChain(task *Task, tasks map[string]*Task) []*Task {
	var chain []*Task

	current := task
	for current.ParentID != nil {
		parent := tasks[*current.ParentID]
		if parent == nil {
			break
		}
		chain = append(chain, parent)
		current = parent
	}

	return chain
}

// StoreTaskResult stores a task result to .hearth/results/<task-id>.md
func StoreTaskResult(workspaceDir, taskID, content string) (string, error) {
	resultsDir := filepath.Join(workspaceDir, ".hearth", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create results directory: %w", err)
	}

	resultPath := filepath.Join(resultsDir, fmt.Sprintf("%s.md", taskID))
	if err := os.WriteFile(resultPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write result file: %w", err)
	}

	return resultPath, nil
}
