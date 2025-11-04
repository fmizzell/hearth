package main

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/fmizzell/hearth"
)

func main() {
	// Create hearth instance
	h := hearth.NewHearth("test-project")

	// Add a simple task
	err := h.Process(hearth.TaskCreated{
		TaskID:      "T1",
		Title:       "Write a hello world function",
		Description: "Create a simple Go function that returns 'Hello, World!'",
		Time:        time.Now(),
	})
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}

	fmt.Println("🔥 Hearth started!")
	fmt.Println()

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

		// Build prompt for Claude
		prompt := fmt.Sprintf(`You are working on a task:

Title: %s
Description: %s

Please complete this task. When done, respond with a summary of what you did.`, task.Title, task.Description)

		// Call Claude
		fmt.Println("🤖 Calling Claude...")
		response, err := callClaude(prompt)
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

// callClaude invokes the claude CLI and returns the response
func callClaude(prompt string) (string, error) {
	cmd := exec.Command("claude", prompt)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}
