package hearth

import (
	"fmt"
	"strings"

	"github.com/cumulusrpg/atmos"
	"github.com/fmizzell/hearth/prompts"
)

// ============================================================================
// BEFORE HOOKS - Where the work happens
// ============================================================================

// beforeNextTaskSelected finds the next eligible task using depth-first traversal
func beforeNextTaskSelected(engine *atmos.Engine, event *NextTaskSelected) {
	state := engine.GetState("hearth").(HearthState)

	// Convert map to slice for findNextTask function
	var tasks []*Task
	for _, task := range state.Tasks {
		tasks = append(tasks, task)
	}

	// Use existing depth-first logic
	nextTask := findNextTask(tasks)

	if nextTask == nil {
		// No tasks available - signal halt
		event.TaskID = ""
		event.Reason = "no eligible tasks"
		return
	}

	// Task found
	event.TaskID = nextTask.ID
	event.Reason = "depth-first-next"

	// Log task selection
	fmt.Printf("📋 Working on %s\n", nextTask.ID)
	fmt.Printf("   Title: %s\n", nextTask.Title)
	if nextTask.Description != "" {
		desc := nextTask.Description
		if len(desc) > 100 {
			desc = desc[:100] + "..."
		}
		fmt.Printf("   Description: %s\n", desc)
	}
	fmt.Println()
}

// beforeTaskExecuted handles task execution using real Claude caller
func beforeTaskExecuted(engine *atmos.Engine, event *TaskExecuted) {
	state := engine.GetState("hearth").(HearthState)

	task := state.Tasks[event.TaskID]
	if task == nil {
		// Task not found - this shouldn't happen if NextTaskSelected worked correctly
		return
	}

	// Get services from engine
	workspaceDir := engine.GetService("workspace_dir")
	claudeCaller := engine.GetService("claude_caller")

	if workspaceDir == nil || claudeCaller == nil {
		// Services not registered - can't execute
		// This happens in tests that don't register services
		event.ResultPath = fmt.Sprintf(".hearth/results/%s.md", event.TaskID)
		return
	}

	// Log execution start
	fmt.Println("🤖 Calling Claude...")
	fmt.Println()

	// Execute task: build context, call Claude, store result
	resultPath, err := ExecuteTask(
		event.TaskID,
		state.Tasks,
		workspaceDir.(string),
		claudeCaller.(ClaudeCaller),
	)

	if err != nil {
		// TODO: Handle execution errors properly
		// For now, set empty result path to prevent crashes
		event.ResultPath = ""
		return
	}

	event.ResultPath = resultPath

	// Log execution completion
	fmt.Printf("✓ Task %s executed\n", event.TaskID)
	if resultPath != "" {
		fmt.Printf("   Result stored: %s\n", resultPath)
	}
	fmt.Println()
}

// beforeSummaryGenerated generates a summary by calling Claude with all child results
func beforeSummaryGenerated(engine *atmos.Engine, event *SummaryGenerated) {
	state := engine.GetState("hearth").(HearthState)

	parent := state.Tasks[event.ParentTaskID]
	if parent == nil {
		return
	}

	// Get services
	workspaceDir := engine.GetService("workspace_dir")
	claudeCaller := engine.GetService("claude_caller")

	if workspaceDir == nil || claudeCaller == nil {
		event.SummaryPath = "" // Can't generate summary without services
		return
	}

	// Build enriched prompt with child result references
	var childrenContext strings.Builder
	childrenContext.WriteString("\n\nYour subtasks have completed. Here are the results:\n\n")

	for _, t := range state.Tasks {
		if t.ParentID != nil && *t.ParentID == event.ParentTaskID && t.Status == "completed" {
			resultPath := fmt.Sprintf(".hearth/results/%s.md", t.ID)
			childrenContext.WriteString(fmt.Sprintf("- %s \"%s\" → Result: %s\n", t.ID, t.Title, resultPath))
		}
	}
	childrenContext.WriteString("\nPlease read these result files and synthesize them into a final answer for the original task.\n")

	prompt := parent.Description
	if prompt == "" {
		prompt = parent.Title
	}

	taskContext := fmt.Sprintf(`
ORIGINAL TASK: %s
TASK ID: %s

`, parent.Title, parent.ID)

	fullPrompt := taskContext + prompt + childrenContext.String() + "\n" + prompts.TaskSystemInstructions

	// Call Claude to generate summary
	response, err := claudeCaller.(ClaudeCaller).Call(fullPrompt, workspaceDir.(string))
	if err != nil {
		event.SummaryPath = ""
		return
	}

	// Store summary (overwrites original result)
	summaryPath, err := StoreTaskResult(workspaceDir.(string), event.ParentTaskID, response)
	if err != nil {
		event.SummaryPath = ""
		return
	}

	event.SummaryPath = summaryPath
}
