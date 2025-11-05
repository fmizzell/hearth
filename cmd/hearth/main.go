package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fmizzell/hearth"
	"github.com/fmizzell/hearth/prompts"
)

func main() {
	// Get target directory from command line
	if len(os.Args) < 2 {
		log.Fatal("Usage: hearth <target-directory>")
	}
	targetDir := os.Args[1]

	// Verify target directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		log.Fatalf("Target directory does not exist: %s", targetDir)
	}

	// Make target directory absolute
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	fmt.Println("🔥 Hearth - Code Quality Analysis")
	fmt.Printf("📂 Target: %s\n", absTargetDir)
	fmt.Println()

	// Create hearth instance
	h := hearth.NewHearth("code-quality")

	// Add code quality analysis task
	err = h.Process(hearth.TaskCreated{
		TaskID:      "T1",
		Title:       "Analyze codebase for quality issues",
		Description: fmt.Sprintf("Perform comprehensive code quality analysis on: %s", absTargetDir),
		Time:        time.Now(),
	})
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}

	// Main loop
	iteration := 0
	for {
		iteration++

		// Get next task
		task := h.GetNextTask()
		if task == nil {
			fmt.Println("✅ All tasks completed!")
			break
		}

		fmt.Printf("📋 Iteration %d: Working on %s\n", iteration, task.ID)
		fmt.Printf("   Title: %s\n", task.Title)
		fmt.Printf("   Description: %s\n", task.Description)
		fmt.Println()

		// Create output directory for code quality reports
		codeQualityDir := filepath.Join(absTargetDir, "code-quality")
		if err := os.MkdirAll(codeQualityDir, 0755); err != nil {
			log.Fatalf("Failed to create code-quality directory: %v", err)
		}
		fmt.Printf("📊 Output: %s\n", codeQualityDir)

		// Use prompt from prompts package
		prompt := prompts.CodeQualityAnalysis

		// Call Claude (working directly in target directory)
		fmt.Println("🤖 Starting analysis...")
		response, err := callClaude(prompt, absTargetDir)
		if err != nil {
			log.Fatalf("Failed to call Claude: %v", err)
		}

		fmt.Println()
		fmt.Println("💬 Claude's response:")
		fmt.Println(response)
		fmt.Println()

		// Mark task completed
		err = h.Process(hearth.TaskCompleted{
			TaskID: task.ID,
			Time:   time.Now(),
		})
		if err != nil {
			log.Fatalf("Failed to complete task: %v", err)
		}

		fmt.Printf("✓ Task %s completed\n", task.ID)
		fmt.Println()
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
