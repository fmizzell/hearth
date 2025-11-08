package main

import (
	"fmt"
	"os"
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
	Run:   run,
}

func init() {
	runCmd.Flags().StringVar(&taskPreset, "preset", "", "Initialize with a preset task: hello, code-quality")
}

func run(cmd *cobra.Command, args []string) {
	// Get workspace directory (--workspace flag or current directory)
	workspaceDir, err := getWorkspaceDir()
	if err != nil {
		fatal("Failed to get workspace directory: %v", err)
	}

	// Verify workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		fatal("Workspace directory does not exist: %s", workspaceDir)
	}

	fmt.Println("🔥 Hearth - Autonomous Task Orchestration")
	fmt.Printf("📂 Workspace: %s\n", workspaceDir)
	fmt.Println()

	// Create hearth instance with persistence
	// Services (workspace dir + Claude caller) are automatically registered
	h, err := hearth.NewHearth(workspaceDir)
	if err != nil {
		fatal("Failed to create hearth: %v", err)
	}

	// If a preset is specified, create task
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

		err = createTask(workspaceDir, taskID, title, description, nil)
		if err != nil {
			fatal("Failed to create preset task: %v", err)
		}

		fmt.Printf("✓ Created task %s from preset '%s'\n", taskID, taskPreset)
		fmt.Println()
	}

	// Start autonomous orchestration
	fmt.Println("🤖 Starting autonomous task execution...")
	fmt.Println()

	err = h.Process(&hearth.ExecuteTasksRequested{Time: time.Now()})
	if err != nil {
		fatal("Failed to start orchestration: %v", err)
	}

	fmt.Println()
	fmt.Println("✅ All tasks completed!")
	fmt.Println("🎉 Hearth finished!")
}
