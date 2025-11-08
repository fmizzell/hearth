package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	workspaceFlag string
)

var rootCmd = &cobra.Command{
	Use:   "hearth",
	Short: "Hearth - Task orchestration for autonomous coding",
	Long:  `Hearth is a minimalist task orchestration system for autonomous coding with Claude Code.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&workspaceFlag, "workspace", "w", "", "Workspace directory (defaults to current directory)")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
}

func getWorkspaceDir() (string, error) {
	var dir string

	// Use --workspace flag if provided
	if workspaceFlag != "" {
		dir = workspaceFlag
	} else {
		// Default to current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		dir = cwd
	}

	// Make absolute
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absDir, nil
}

func fatal(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}
