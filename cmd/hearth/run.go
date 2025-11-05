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

	// Main loop
	iteration := 0
	for {
		iteration++

		// Reload hearth to pick up any new tasks created by Claude in previous iteration
		h, err = hearth.NewHearthWithPersistence(workspaceDir)
		if err != nil {
			fatal("Failed to reload hearth: %v", err)
		}

		// Get next task
		task := h.GetNextTask()
		if task == nil {
			fmt.Println("✅ All tasks completed!")
			break
		}

		fmt.Printf("📋 Iteration %d: Working on %s\n", iteration, task.ID)
		fmt.Printf("   Title: %s\n", task.Title)
		if task.Description != "" {
			fmt.Printf("   Description: %s\n", task.Description)
		}
		fmt.Println()

		// Use task description as the prompt to Claude
		prompt := task.Description
		if prompt == "" {
			prompt = task.Title // Fallback to title if no description
		}

		// Append task system instructions to every prompt
		// This gives Claude the ability to plan and break down work into subtasks
		fullPrompt := prompt + "\n" + prompts.TaskSystemInstructions

		// Call Claude with the task description as the prompt
		fmt.Println("🤖 Starting task...")
		response, err := callClaude(fullPrompt, workspaceDir)
		if err != nil {
			fatal("Failed to call Claude: %v", err)
		}

		fmt.Println()
		fmt.Println("💬 Claude's response:")
		fmt.Println(response)
		fmt.Println()

		// Mark task completed
		err = h.Process(&hearth.TaskCompleted{
			TaskID: task.ID,
			Time:   time.Now(),
		})
		if err != nil {
			fatal("Failed to complete task: %v", err)
		}

		fmt.Printf("✓ Task %s completed\n", task.ID)
		fmt.Println()

		// Save events after each task completion
		eventsFile := filepath.Join(workspaceDir, ".hearth", "events.json")
		err = h.SaveToFile(eventsFile)
		if err != nil {
			fatal("Failed to save events: %v", err)
		}
	}

	// Final save
	eventsFile := filepath.Join(workspaceDir, ".hearth", "events.json")
	err = h.SaveToFile(eventsFile)
	if err != nil {
		fatal("Failed to save events: %v", err)
	}

	fmt.Println("🎉 Hearth finished!")
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
