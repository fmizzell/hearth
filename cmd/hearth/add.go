package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	addTitle       string
	addDescription string
	addParent      string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new task",
	Long:  `Add a new task to the current workspace with optional parent for hierarchical tasks.`,
	Run:   addTask,
}

func init() {
	addCmd.Flags().StringVarP(&addTitle, "title", "t", "", "Task title (required)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Task description")
	addCmd.Flags().StringVarP(&addParent, "parent", "p", "", "Parent task ID (for hierarchical tasks)")
	if err := addCmd.MarkFlagRequired("title"); err != nil {
		panic(fmt.Sprintf("Failed to mark title flag as required: %v", err))
	}
}

func addTask(cmd *cobra.Command, args []string) {
	workspaceDir, err := getWorkspaceDir()
	if err != nil {
		fatal("Failed to get workspace directory: %v", err)
	}

	// Generate task ID
	taskID := generateTaskID()

	// Prepare optional parent pointer
	var parentPtr *string
	if addParent != "" {
		parentPtr = &addParent
	}

	// Create task using helper (loads, creates, saves)
	err = createTask(workspaceDir, taskID, addTitle, addDescription, parentPtr)
	if err != nil {
		fatal("%v", err)
	}

	// Print confirmation
	fmt.Printf("✓ Task created: %s\n", taskID)
	fmt.Printf("  Title: %s\n", addTitle)
	if addDescription != "" {
		fmt.Printf("  Description: %s\n", addDescription)
	}
	if addParent != "" {
		fmt.Printf("  Parent: %s\n", addParent)
	}
}

func generateTaskID() string {
	// Generate short UUID-based ID
	return "T-" + uuid.New().String()[:8]
}
