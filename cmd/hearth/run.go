package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fmizzell/hearth"
	"github.com/fmizzell/hearth/prompts"
	"github.com/spf13/cobra"
)

var (
	taskPreset string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run task loop - executes tasks by sending their description to Claude",
	Long:  `Run task loop. Gets next task and sends its description as the prompt to Claude. Uses current directory or --workspace flag.`,
	Run:   runAnalysis,
}

func init() {
	runCmd.Flags().StringVar(&taskPreset, "preset", "", "Initialize with a preset task: hello, code-quality")
}

func runAnalysis(cmd *cobra.Command, args []string) {
	// Get workspace directory (--workspace flag or current directory)
	workspaceDir, err := getWorkspaceDir()
	if err != nil {
		fatal("Failed to get workspace directory: %v", err)
	}

	// Verify workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		fatal("Workspace directory does not exist: %s", workspaceDir)
	}

	fmt.Println("🔥 Hearth - Task Orchestration")
	fmt.Printf("📂 Workspace: %s\n", workspaceDir)
	fmt.Println()

	// Create hearth instance with persistence
	h, err := hearth.NewHearthWithPersistence(workspaceDir)
	if err != nil {
		fatal("Failed to create hearth: %v", err)
	}

	// If a preset is specified, create task using helper (loads, creates, saves)
	if taskPreset != "" {
		var title, description string

		switch taskPreset {
		case "hello":
			title = "Hello World Test"
			description = prompts.Hello
		case "code-quality":
			title = "Code Quality Analysis"
			description = prompts.CodeQualityAnalysis
		default:
			fatal("Unknown preset: %s (use 'hello' or 'code-quality')", taskPreset)
		}

		// Generate unique task ID
		taskID := fmt.Sprintf("T-%d", time.Now().Unix())

		err = createTask(workspaceDir, taskID, title, description, nil, nil)
		if err != nil {
			fatal("Failed to create preset task: %v", err)
		}

		fmt.Printf("✓ Created task %s from preset '%s'\n\n", taskID, taskPreset)
	}

	// Create event channel for execution feedback
	events := make(chan *ExecutionEvent, 10)

	// Start goroutine to listen to events and display them
	go func() {
		for event := range events {
			switch event.Type {
			case "task_started":
				fmt.Printf("📋 Iteration %d: Working on %s\n", event.Iteration, event.TaskID)
				fmt.Printf("   Title: %s\n", event.TaskTitle)
				if event.TaskDescription != "" {
					fmt.Printf("   Description: %s\n", event.TaskDescription)
				}
				fmt.Println()
				fmt.Println("🤖 Starting task...")

			case "claude_response":
				fmt.Println()
				fmt.Println("💬 Claude's response:")
				fmt.Println(event.Response)
				fmt.Println()

			case "task_completed":
				fmt.Printf("✓ Task %s completed\n", event.TaskID)
				fmt.Println()

			case "task_spawned_children":
				fmt.Printf("✓ Task %s spawned %d subtasks (will auto-complete when subtasks finish)\n", event.TaskID, event.ChildCount)
				fmt.Println()

			case "all_done":
				fmt.Println("✅ All tasks completed!")
			}
		}
	}()

	// Run task execution loop
	err = executeTasks(h, workspaceDir, callClaude, events)
	if err != nil {
		fatal("Task execution failed: %v", err)
	}

	fmt.Println("🎉 Hearth finished!")
}

// ClaudeCallerFunc is the signature for calling Claude (real or mocked)
type ClaudeCallerFunc func(prompt, workDir string) (string, error)

// ExecutionEvent represents events during task execution
type ExecutionEvent struct {
	Type            string // "task_started", "claude_response", "task_completed", "task_spawned_children", "all_done"
	Iteration       int
	TaskID          string
	TaskTitle       string
	TaskDescription string
	Response        string
	ChildCount      int
}

// executeTasks runs the task loop: gets next task, calls Claude, stores result, marks complete
// Loops until GetNextTask() returns nil
// Sends execution events through the optional events channel
func executeTasks(h *hearth.Hearth, workspaceDir string, claudeCaller ClaudeCallerFunc, events chan<- *ExecutionEvent) error {
	defer func() {
		if events != nil {
			close(events)
		}
	}()

	iteration := 0
	for {
		// Get next task (FileRepository automatically reads latest state from disk)
		task := h.GetNextTask()
		if task == nil {
			// All tasks completed
			if events != nil {
				events <- &ExecutionEvent{Type: "all_done"}
			}
			break
		}

		iteration++

		// Notify task started
		if events != nil {
			events <- &ExecutionEvent{
				Type:            "task_started",
				Iteration:       iteration,
				TaskID:          task.ID,
				TaskTitle:       task.Title,
				TaskDescription: task.Description,
			}
		}

		// Use task description as the prompt to Claude
		prompt := task.Description
		if prompt == "" {
			prompt = task.Title // Fallback to title if no description
		}

		// Build full prompt with task context and instructions
		taskContext := fmt.Sprintf(`
CURRENT TASK: %s
CURRENT TASK ID: %s

IMPORTANT: Before starting work, assess if this task should be broken into subtasks.
If this task involves multiple steps or can be parallelized, you MUST create subtasks first.

`, task.Title, task.ID)

		fullPrompt := taskContext + prompt + "\n" + prompts.TaskSystemInstructions

		// Call Claude with the task description as the prompt
		response, err := claudeCaller(fullPrompt, workspaceDir)
		if err != nil {
			return fmt.Errorf("failed to call Claude: %w", err)
		}

		// Notify Claude responded
		if events != nil {
			events <- &ExecutionEvent{
				Type:     "claude_response",
				TaskID:   task.ID,
				Response: response,
			}
		}

		// Store result to .hearth/results/<task-id>.md
		resultsDir := filepath.Join(workspaceDir, ".hearth", "results")
		if err := os.MkdirAll(resultsDir, 0755); err != nil {
			return fmt.Errorf("failed to create results directory: %w", err)
		}

		resultPath := filepath.Join(resultsDir, fmt.Sprintf("%s.md", task.ID))
		if err := os.WriteFile(resultPath, []byte(response), 0644); err != nil {
			return fmt.Errorf("failed to write result file: %w", err)
		}

		// Try to mark task completed (FileRepository auto-persists and auto-reloads)
		err = h.Process(&hearth.TaskCompleted{
			TaskID: task.ID,
			Time:   time.Now(),
		})
		if err == hearth.ErrEventRejected {
			// Task has children - it will auto-complete when children finish
			children := h.GetChildTasks(task.ID)
			if events != nil {
				events <- &ExecutionEvent{
					Type:       "task_spawned_children",
					TaskID:     task.ID,
					ChildCount: len(children),
				}
			}
		} else if err != nil {
			return fmt.Errorf("failed to complete task: %w", err)
		} else {
			// Task completed successfully
			if events != nil {
				events <- &ExecutionEvent{
					Type:   "task_completed",
					TaskID: task.ID,
				}
			}

			// Check if parent needs summary now that this task is complete
			if err := checkAndGenerateParentSummary(h, task, workspaceDir, claudeCaller, events); err != nil {
				return fmt.Errorf("failed to generate parent summary: %w", err)
			}
		}
	}

	return nil
}

// checkAndGenerateParentSummary checks if a task's parent should now generate a summary
// (all children complete) and if so, re-runs the parent with enriched context
func checkAndGenerateParentSummary(h *hearth.Hearth, childTask *hearth.Task, workspaceDir string, claudeCaller ClaudeCallerFunc, events chan<- *ExecutionEvent) error {
	// Does this task have a parent?
	if childTask.ParentID == nil {
		return nil
	}

	parent := h.GetTask(*childTask.ParentID)
	if parent == nil {
		return nil
	}

	// Are all siblings complete?
	children := h.GetChildTasks(parent.ID)
	for _, child := range children {
		if child.Status != "completed" {
			return nil // Still waiting on siblings
		}
	}

	// All children complete! Generate parent summary
	if events != nil {
		events <- &ExecutionEvent{
			Type:   "parent_summary_started",
			TaskID: parent.ID,
		}
	}

	// Build enriched prompt with child result references
	childrenContext := "\n\nYour subtasks have completed. Here are the results:\n\n"
	for _, child := range children {
		resultPath := fmt.Sprintf(".hearth/results/%s.md", child.ID)
		childrenContext += fmt.Sprintf("- %s \"%s\" → Result: %s\n", child.ID, child.Title, resultPath)
	}
	childrenContext += "\nPlease read these result files and synthesize them into a final answer for the original task.\n"

	prompt := parent.Description
	if prompt == "" {
		prompt = parent.Title
	}

	taskContext := fmt.Sprintf(`
ORIGINAL TASK: %s
TASK ID: %s

`, parent.Title, parent.ID)

	fullPrompt := taskContext + prompt + childrenContext + "\n" + prompts.TaskSystemInstructions

	// Call Claude to generate summary
	response, err := claudeCaller(fullPrompt, workspaceDir)
	if err != nil {
		return fmt.Errorf("failed to generate parent summary: %w", err)
	}

	if events != nil {
		events <- &ExecutionEvent{
			Type:     "claude_response",
			TaskID:   parent.ID,
			Response: response,
		}
	}

	// Overwrite parent's result file with summary
	resultsDir := filepath.Join(workspaceDir, ".hearth", "results")
	resultPath := filepath.Join(resultsDir, fmt.Sprintf("%s.md", parent.ID))
	if err := os.WriteFile(resultPath, []byte(response), 0644); err != nil {
		return fmt.Errorf("failed to write parent summary: %w", err)
	}

	// Note: Parent will be auto-completed by the system when all children are done
	// Recursively check if grandparent needs summary
	return checkAndGenerateParentSummary(h, parent, workspaceDir, claudeCaller, events)
}

// callClaude invokes the claude CLI in a temporary workspace and returns the response
func callClaude(prompt, workDir string) (string, error) {
	cmd := exec.Command("claude",
		"--print",                        // Non-interactive output
		"--dangerously-skip-permissions", // Skip permission prompts (safe: sandboxed to workDir)
		prompt,
	)

	// Set Claude's working directory to the temp workspace
	cmd.Dir = workDir

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}
