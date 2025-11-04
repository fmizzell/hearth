# Hearth Design (Atmos-based)

## Overview

Hearth is a minimalist task loop for semi-autonomous coding with Claude Code, built on top of [Atmos](https://github.com/cumulusrpg/atmos) event sourcing framework.

## Why Atmos?

- **Complete audit trail**: Every agent action is an immutable event
- **Time travel**: Replay any session to see what happened
- **Testability**: Pure functions, no mocks needed
- **Simplicity**: State derived from events automatically

## Events

### Task Events
```go
type TaskCreated struct {
    TaskID      string
    Title       string
    Description string
    DependsOn   *string // Optional upstream task ID
    Time        time.Time
}

type TaskStarted struct {
    TaskID string
    Time   time.Time
}

type TaskLogAdded struct {
    TaskID  string
    Message string
    Time    time.Time
}

type TaskCompleted struct {
    TaskID string
    Time   time.Time
}
```

### Agent Iteration Events
```go
type IterationStarted struct {
    IterationID int
    TaskID      string
    Prompt      string
    Time        time.Time
}

type IterationCompleted struct {
    IterationID int
    TaskID      string
    Response    string
    Success     bool
    Time        time.Time
}
```

## State

```go
type HearthState struct {
    Tasks           map[string]*Task
    CurrentTask     *string
    IterationCount  int
    StartTime       time.Time
}

type Task struct {
    ID          string
    Title       string
    Description string
    DependsOn   *string
    Status      string // "todo", "in-progress", "completed"
    Logs        []string
    CreatedAt   time.Time
    CompletedAt *time.Time
}
```

## The Loop

```
1. Emit IterationStarted
2. Get next task from state (or stop if none)
3. Build prompt with task details + INSTRUCTIONS.md
4. Call `claude` CLI with prompt
5. Emit IterationCompleted
6. Check if task status changed to completed
7. Repeat
```

## Persistence

Events are serialized to `~/.hearth/<project>/events.json`:

```go
// Save after each iteration
events := engine.GetEvents()
jsonData, _ := engine.MarshalEvents(events)
os.WriteFile(eventsPath, jsonData, 0644)

// Load on startup
jsonData, _ := os.ReadFile(eventsPath)
events, _ := engine.UnmarshalEvents(jsonData)
engine.SetEvents(events)
```

## Commands

### `hearth init <project>`
Creates `~/.hearth/<project>/` directory and empty events log

### `hearth task add <title> [--depends-on T1]`
Emits TaskCreated event

### `hearth task list`
Shows all tasks from current state

### `hearth task log <task-id> <message>`
Emits TaskLogAdded event

### `hearth run <project>`
Main loop - keeps iterating until no more tasks

### `hearth replay <project>`
Rebuilds state from events (for debugging)

## File Structure

```
hearth/
├── cmd/hearth/
│   └── main.go          # CLI entry point
├── events.go            # Event definitions
├── state.go             # State definitions & reducers
├── engine.go            # Atmos engine setup
├── loop.go              # Main iteration loop
├── claude.go            # Claude CLI integration
└── INSTRUCTIONS.md      # Instructions given to Claude
```

## Example Usage

```bash
# Initialize project
hearth init my-app

# Add some tasks
hearth task add "Implement login endpoint"
hearth task add "Add login tests" --depends-on T1

# Run the loop
hearth run my-app

# Later: replay to see what happened
hearth replay my-app
```

## Benefits Over Original Design

1. **Full auditability**: Every action is logged as an event
2. **Debugging**: Replay any session to see what Claude did
3. **No SQL**: State is just Go structs derived from events
4. **Simpler code**: Atmos handles state management
5. **Already tested**: Atmos framework is battle-tested
