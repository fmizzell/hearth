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
	addDependsOn   string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new task",
	Long:  `Add a new task to the current workspace with optional parent/dependency relationships.`,
	Run:   addTask,
}

func init() {
	addCmd.Flags().StringVarP(&addTitle, "title", "t", "", "Task title (required)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Task description")
	addCmd.Flags().StringVarP(&addParent, "parent", "p", "", "Parent task ID (for hierarchical tasks)")
	addCmd.Flags().StringVarP(&addDependsOn, "depends-on", "D", "", "Task ID this task depends on")
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

	// Prepare optional parent/depends-on pointers
	var parentPtr, dependsOnPtr *string
	if addParent != "" {
		parentPtr = &addParent
	}
	if addDependsOn != "" {
		dependsOnPtr = &addDependsOn
	}

	// Create task using helper (loads, creates, saves)
	err = createTask(workspaceDir, taskID, addTitle, addDescription, parentPtr, dependsOnPtr)
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
	if addDependsOn != "" {
		fmt.Printf("  Depends on: %s\n", addDependsOn)
	}
}

func generateTaskID() string {
	// Generate short UUID-based ID
	return "T-" + uuid.New().String()[:8]
}
