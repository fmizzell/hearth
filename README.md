# Hearth 🔥

**Autonomous task orchestration for Claude Code**

Hearth is a hierarchical task management system that enables AI agents to break down complex coding tasks into smaller, manageable subtasks and execute them autonomously with full context preservation across the entire task tree.

[![Go Report Card](https://goreportcard.com/badge/github.com/fmizzell/hearth)](https://goreportcard.com/report/github.com/fmizzell/hearth)
[![CI](https://github.com/fmizzell/hearth/workflows/CI/badge.svg)](https://github.com/fmizzell/hearth/actions)
[![codecov](https://codecov.io/gh/fmizzell/hearth/branch/main/graph/badge.svg)](https://codecov.io/gh/fmizzell/hearth)
[![Go Version](https://img.shields.io/github/go-mod/go-version/fmizzell/hearth)](https://github.com/fmizzell/hearth/blob/main/go.mod)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## What is Hearth?

Hearth solves the "telephone problem" in AI-driven development: when complex tasks are broken into subtasks, context is often lost. Hearth maintains full context throughout arbitrarily deep task hierarchies, enabling AI agents to:

- 📋 Break down complex work into manageable pieces
- 🔗 Share knowledge between sibling tasks
- 🎯 Stay aligned with root goals through deep hierarchies
- 📊 Automatically synthesize findings from subtasks
- 🔄 Execute tasks in correct depth-first order

## Key Features

### Event-Sourced Architecture
Built on [Atmos](https://github.com/cumulusrpg/atmos), providing complete audit trails and replay capability. All state changes flow through immutable events (`TaskCreated`, `TaskStarted`, `TaskCompleted`).

### Hierarchical Task Decomposition
Tasks can have parent-child relationships forming trees of arbitrary depth. Parent tasks auto-complete when all children finish, triggering automatic result synthesis.

### Context Injection
Child tasks automatically receive:
- **Root goal** - The ultimate objective from the top of the hierarchy
- **Parent chain** - Full lineage showing how this task fits in
- **Sibling results** - Findings from previously completed siblings

### Result Persistence
Every task stores its output to `.hearth/results/<task-id>.md`, creating a knowledge base that:
- Prevents duplicate work between siblings
- Enables progressive refinement
- Supports parent summarization
- Provides complete execution history

### Depth-First Execution
Tasks execute in proper hierarchical order - completing entire subtrees before moving to the next sibling, ensuring logical progression through complex workflows.

## Installation

**IMPORTANT:** Hearth must be in your PATH for autonomous task creation to work. When Claude Code creates subtasks, it runs `hearth add` commands which require the binary to be accessible.

### Option 1: Using Go Install (Recommended)

```bash
# Install directly to your GOPATH/bin (usually in PATH)
go install github.com/fmizzell/hearth/cmd/hearth@latest

# Verify it's accessible
hearth --help
```

### Option 2: From Source

```bash
# Clone the repository
git clone https://github.com/fmizzell/hearth.git
cd hearth

# Build the binary
go build -o bin/hearth ./cmd/hearth

# Add to PATH (required, not optional!)
# Add this to your ~/.bashrc, ~/.zshrc, or equivalent:
export PATH="$PATH:$HOME/path/to/hearth/bin"

# Or create a symlink to a directory already in PATH:
ln -s "$(pwd)/bin/hearth" ~/.local/bin/hearth

# Verify it's accessible from any directory
cd ~
hearth --help
```

## Quick Start

### 1. Create a Task

```bash
# Initialize with a built-in preset
hearth run --preset hello

# Or create a custom task
hearth add -t "Refactor authentication" \
           -d "Update login system to use OAuth2"
```

### 2. Run the Autonomous Loop

```bash
hearth run
```

Hearth will:
1. Get the next task
2. Send its description to Claude Code
3. Store Claude's response to `.hearth/results/<task-id>.md`
4. Mark the task complete
5. If Claude created subtasks, process them recursively
6. Generate parent summaries when subtrees complete
7. Repeat until all tasks are done

### 3. View Task Status

```bash
# List all tasks in execution order
hearth list

# Filter by status
hearth list --status todo
hearth list --status completed
```

## How It Works

### Task Lifecycle

```
┌─────────────────────────────────────────────────────────┐
│ 1. Task Created                                         │
│    - Store event to .hearth/events.json                │
│    - Status: "todo"                                     │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Task Execution                                       │
│    - Build context (root goal, parent chain, siblings) │
│    - Call Claude Code with enriched prompt             │
│    - Store response to .hearth/results/<task-id>.md    │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Task Completion                                      │
│    - If has children: Status stays "todo"              │
│    - If no children: Status → "completed"              │
│    - Check if parent needs summary                     │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Parent Summary (when all children complete)         │
│    - Re-run parent with child result references        │
│    - Overwrite parent result with synthesis            │
│    - Mark parent complete                              │
│    - Recursively trigger grandparent summary if needed │
└─────────────────────────────────────────────────────────┘
```

### Context Injection Example

When a child task executes, it sees:

```
ROOT TASK: Analyze codebase for security issues
ROOT GOAL: Identify vulnerabilities and create fix tasks

PREVIOUS SIBLING RESULTS:
Your siblings have already completed work. You can reference their findings:

- T-abc123 "Scan dependencies" → Result: .hearth/results/T-abc123.md
- T-def456 "Check input validation" → Result: .hearth/results/T-def456.md

You can read these files to avoid duplicating work and build on their findings.

---

CURRENT TASK: Review authentication code
...
```

## Real-World Example

Here's what happened when we ran `hearth run --preset code-quality`:

```
📋 Tasks:

✓ Code Quality Analysis
  ✓ Explore codebase structure
    ✓ Analyze file metrics
      ✓ Identify all Go files
      ✓ Run gocyclo
      ✓ Calculate metrics
      ✓ Generate metrics table
    ✓ Document dependencies
    ✓ Analyze test coverage
    ✓ Create analysis list
  ✓ Run static analysis
    ✓ Run gofmt
    ✓ Run go vet
    ✓ Run golangci-lint
    ✓ Additional tools
      ✓ Run staticcheck
      ✓ Run race detector
      ✓ Consolidate findings
    ✓ Consolidate all findings
  ✓ Identify quality issues
  ✓ Generate final report
```

**Result:** 20+ interconnected tasks, 4 levels deep, generated a comprehensive 586-line professional code quality report with zero context loss.

## Use Cases

### Code Quality Analysis
```bash
hearth run --preset code-quality
```
Systematically analyzes codebase, runs multiple tools, identifies issues, generates actionable report.

### Large Refactoring
Create a task to refactor a module. Claude breaks it into file-by-file subtasks, each with context about the overall refactoring goal.

### Multi-File Features
Implement features that span multiple files. Each file gets its own subtask, with access to what was done in other files.

### Research & Documentation
Gather information from multiple sources, then synthesize into a comprehensive document. Each research task stores findings, final task combines them all.

### Test Suite Generation
Create comprehensive tests for a module. Claude creates subtasks for different test categories, each aware of tests already written.

## Project Structure

```
hearth/
├── cmd/hearth/          # CLI application
│   ├── main.go         # Entry point
│   ├── run.go          # Task execution loop
│   ├── add.go          # Task creation
│   └── list.go         # Task display
├── prompts/            # Built-in task presets
│   ├── hello.txt
│   └── code-quality-analysis.txt
├── hearth.go           # Core task management
├── events.go           # Event definitions
├── reducers.go         # State management
├── file_repository.go  # Event persistence
└── .hearth/            # Runtime data (gitignored)
    ├── events.json     # Event log
    └── results/        # Task output files
```

## CLI Commands

### `hearth add`
Create a new task.

```bash
# Simple task
hearth add -t "Fix bug in login" -d "Add null check to user validation"

# With parent (creates subtask)
hearth add -t "Update tests" -p T-parent-id

# With dependency
hearth add -t "Deploy" -d "Deploy to production" --depends-on T-test-id
```

### `hearth run`
Execute tasks autonomously.

```bash
# Run all pending tasks
hearth run

# Start with a preset
hearth run --preset code-quality
hearth run --preset hello
```

### `hearth list`
View task status.

```bash
# All tasks in execution order
hearth list

# Filter by status
hearth list --status todo
hearth list --status completed
hearth list --status in-progress
```

### `hearth complete`
Manually mark a task complete.

```bash
hearth complete T-12345
```

## Configuration

Hearth uses your current directory as the workspace. The `.hearth/` directory stores all state:

```
.hearth/
├── events.json          # Event sourcing log
└── results/
    ├── T-abc123.md     # Task results
    └── T-def456.md
```

You can have multiple independent workspaces by running Hearth in different directories.

## Advanced Usage

### Creating Custom Presets

Add `.txt` files to `prompts/` directory:

```txt
# prompts/my-custom-preset.txt
Analyze the database schema and create migration tasks for normalization issues.

Break this into:
1. Analyzing current schema
2. Identifying normalization violations
3. Creating migration subtasks
4. Generating rollback scripts
```

Use it:
```bash
hearth run --preset my-custom-preset
```

### Workspace Flag

Run in a different directory:

```bash
hearth run --workspace /path/to/project
```

## Architecture

### Event Sourcing
All state changes are events stored in `.hearth/events.json`. State is reconstructed by replaying events through reducers. This provides:
- Complete audit trail
- Time-travel debugging
- Crash recovery
- Concurrent safety (with file locking)

### Depth-First Execution
The `GetNextTask()` algorithm traverses the task tree depth-first:
1. Find root tasks (no parent)
2. Sort by creation time
3. For each root, recursively search its subtree
4. Return first eligible leaf task

This ensures logical execution order where subtrees complete before siblings.

### Pure Functions
Core algorithms like `findNextTask(tasks)` are pure functions that take task slices and return results, making them easy to test and reason about.

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v -run TestFindNextTask_DepthFirst
```

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Write tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Commit with descriptive messages
6. Push to your fork
7. Open a Pull Request

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [Atmos](https://github.com/cumulusrpg/atmos) event sourcing framework
- Designed for [Claude Code](https://claude.com/claude-code) autonomous development
- Inspired by hierarchical task networks and event-driven architectures

---

**Made with 🔥 for autonomous AI development**
