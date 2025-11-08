package hearth

import (
	"fmt"
	"strings"
	"time"

	"github.com/cumulusrpg/atmos"
	"github.com/fmizzell/hearth/prompts"
)

// ============================================================================
// REDUCERS - Build state from orchestration events
// ============================================================================

func reduceNextTaskSelected(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(HearthState)
	e := event.(*NextTaskSelected)

	if e.TaskID != "" {
		// Task was selected, mark as in-progress
		if task, exists := s.Tasks[e.TaskID]; exists {
			task.Status = "in-progress"
		}
	}

	return s
}

func reduceTaskExecuted(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(HearthState)
	// TaskExecuted event is recorded in log (result path available for context building)
	return s
}

// ============================================================================
// BEFORE HOOKS - Where the work happens
// ============================================================================

// TypedListenerFunc is a helper for creating listeners from functions
type TypedListenerFunc[T atmos.Event] func(*atmos.Engine, T)

func (f TypedListenerFunc[T]) HandleTyped(engine *atmos.Engine, event T) {
	f(engine, event)
}

// beforeNextTaskSelected finds the next eligible task using depth-first traversal
func beforeNextTaskSelected(engine *atmos.Engine, event *NextTaskSelected) {
	state := engine.GetState("hearth").(HearthState)

	// Convert map to slice for findNextTask function
	var tasks []*Task
	for _, task := range state.Tasks {
		tasks = append(tasks, task)
	}

	// Use existing depth-first logic
	nextTask := findNextTask(tasks)

	if nextTask == nil {
		// No tasks available - signal halt
		event.TaskID = ""
		event.Reason = "no eligible tasks"
		return
	}

	// Task found
	event.TaskID = nextTask.ID
	event.Reason = "depth-first-next"

	// Log task selection
	fmt.Printf("📋 Working on %s\n", nextTask.ID)
	fmt.Printf("   Title: %s\n", nextTask.Title)
	if nextTask.Description != "" {
		desc := nextTask.Description
		if len(desc) > 100 {
			desc = desc[:100] + "..."
		}
		fmt.Printf("   Description: %s\n", desc)
	}
	fmt.Println()
}

// beforeTaskExecuted handles task execution using real Claude caller
func beforeTaskExecuted(engine *atmos.Engine, event *TaskExecuted) {
	state := engine.GetState("hearth").(HearthState)

	task := state.Tasks[event.TaskID]
	if task == nil {
		// Task not found - this shouldn't happen if NextTaskSelected worked correctly
		return
	}

	// Get services from engine
	workspaceDir := engine.GetService("workspace_dir")
	claudeCaller := engine.GetService("claude_caller")

	if workspaceDir == nil || claudeCaller == nil {
		// Services not registered - can't execute
		// This happens in tests that don't register services
		event.ResultPath = fmt.Sprintf(".hearth/results/%s.md", event.TaskID)
		return
	}

	// Log execution start
	fmt.Println("🤖 Calling Claude...")
	fmt.Println()

	// Create Hearth instance from engine (for helper methods)
	h := &Hearth{engine: engine}

	// Execute task: build context, call Claude, store result
	resultPath, err := ExecuteTask(
		h,
		task,
		workspaceDir.(string),
		claudeCaller.(ClaudeCaller),
	)

	if err != nil {
		// TODO: Handle execution errors properly
		// For now, set empty result path to prevent crashes
		event.ResultPath = ""
		return
	}

	event.ResultPath = resultPath

	// Log execution completion
	fmt.Printf("✓ Task %s executed\n", event.TaskID)
	if resultPath != "" {
		fmt.Printf("   Result stored: %s\n", resultPath)
	}
	fmt.Println()
}

// beforeSummaryGenerated generates a summary by calling Claude with all child results
func beforeSummaryGenerated(engine *atmos.Engine, event *SummaryGenerated) {
	state := engine.GetState("hearth").(HearthState)

	parent := state.Tasks[event.ParentTaskID]
	if parent == nil {
		return
	}

	// Get services
	workspaceDir := engine.GetService("workspace_dir")
	claudeCaller := engine.GetService("claude_caller")

	if workspaceDir == nil || claudeCaller == nil {
		event.SummaryPath = "" // Can't generate summary without services
		return
	}

	// Build enriched prompt with child result references
	var childrenContext strings.Builder
	childrenContext.WriteString("\n\nYour subtasks have completed. Here are the results:\n\n")

	for _, t := range state.Tasks {
		if t.ParentID != nil && *t.ParentID == event.ParentTaskID && t.Status == "completed" {
			resultPath := fmt.Sprintf(".hearth/results/%s.md", t.ID)
			childrenContext.WriteString(fmt.Sprintf("- %s \"%s\" → Result: %s\n", t.ID, t.Title, resultPath))
		}
	}
	childrenContext.WriteString("\nPlease read these result files and synthesize them into a final answer for the original task.\n")

	prompt := parent.Description
	if prompt == "" {
		prompt = parent.Title
	}

	taskContext := fmt.Sprintf(`
ORIGINAL TASK: %s
TASK ID: %s

`, parent.Title, parent.ID)

	fullPrompt := taskContext + prompt + childrenContext.String() + "\n" + prompts.TaskSystemInstructions

	// Call Claude to generate summary
	response, err := claudeCaller.(ClaudeCaller).Call(fullPrompt, workspaceDir.(string))
	if err != nil {
		event.SummaryPath = ""
		return
	}

	// Store summary (overwrites original result)
	summaryPath, err := StoreTaskResult(workspaceDir.(string), event.ParentTaskID, response)
	if err != nil {
		event.SummaryPath = ""
		return
	}

	event.SummaryPath = summaryPath
}

// ============================================================================
// LISTENERS - Event chain orchestration
// ============================================================================

func onExecuteTasksRequested(engine *atmos.Engine, event *ExecuteTasksRequested) {
	// Trigger scheduler to find next task
	engine.Emit(&NextTaskSelected{Time: time.Now()})
}

func onNextTaskSelected(engine *atmos.Engine, event *NextTaskSelected) {
	if event.TaskID == "" {
		// No tasks available - halt execution
		return
	}

	// Task was selected, execute it
	engine.Emit(&TaskExecuted{
		TaskID: event.TaskID,
		Time:   time.Now(),
	})
}

func onTaskExecuted(engine *atmos.Engine, event *TaskExecuted) {
	state := engine.GetState("hearth").(HearthState)

	task := state.Tasks[event.TaskID]
	if task == nil {
		return
	}

	// Check if task has children
	hasChildren := false
	for _, t := range state.Tasks {
		if t.ParentID != nil && *t.ParentID == event.TaskID {
			hasChildren = true
			break
		}
	}

	if hasChildren {
		// Has children - don't complete yet, go depth-first into children
		engine.Emit(&NextTaskSelected{Time: time.Now()})
		return
	}

	// No children - complete the task
	engine.Emit(&TaskCompleted{
		TaskID: event.TaskID,
		Time:   time.Now(),
	})
}

// onTaskCompletedParent handles parent auto-completion
// This replaces the autoCompleteParent mutation in the reducer
func onTaskCompletedParent(engine *atmos.Engine, event *TaskCompleted) {
	state := engine.GetState("hearth").(HearthState)

	task := state.Tasks[event.TaskID]
	if task == nil {
		return
	}

	// Log task completion
	// Check if this task has children (spawned subtasks)
	hasChildren := false
	for _, t := range state.Tasks {
		if t.ParentID != nil && *t.ParentID == event.TaskID {
			hasChildren = true
			break
		}
	}

	if hasChildren {
		childCount := 0
		for _, t := range state.Tasks {
			if t.ParentID != nil && *t.ParentID == event.TaskID {
				childCount++
			}
		}
		fmt.Printf("✓ Task %s spawned %d subtasks (will auto-complete when subtasks finish)\n", event.TaskID, childCount)
		fmt.Println()
	} else {
		fmt.Printf("✓ Task %s completed\n", event.TaskID)
		fmt.Println()
	}

	// Check if parent should be completed
	if task.ParentID != nil {
		parent := state.Tasks[*task.ParentID]
		if parent != nil && parent.Status != "completed" {
			// Check if all siblings are complete
			allSiblingsDone := true
			for _, t := range state.Tasks {
				if t.ParentID != nil && *t.ParentID == *task.ParentID {
					if t.Status != "completed" {
						allSiblingsDone = false
						break
					}
				}
			}

			if allSiblingsDone {
				// All children done - request summary generation
				engine.Emit(&SummaryRequested{
					ParentTaskID: *task.ParentID,
					Time:         time.Now(),
				})
				// Summary generation will complete the parent
			}
		}
	}
}

// onTaskCompletedSchedule continues orchestration scheduling after task completion
// Only used during active orchestration (after TaskExecuted)
func onTaskCompletedSchedule(engine *atmos.Engine, event *TaskCompleted) {
	// Check if we're in active orchestration by looking at recent events
	events := engine.GetEvents()
	if len(events) < 2 {
		return // Not enough events to determine context
	}

	// Check if this completion came from orchestration (TaskExecuted before TaskCompleted)
	// This prevents auto-scheduling when tasks are manually completed
	lastEvent := events[len(events)-2] // event before this TaskCompleted
	if lastEvent.Type() == "task_executed" {
		executed := lastEvent.(*TaskExecuted)
		if executed.TaskID == event.TaskID {
			// This task was executed by orchestration - continue scheduling
			engine.Emit(&NextTaskSelected{Time: time.Now()})
		}
	}
}

// onSummaryRequested emits SummaryGenerated event (before hook does the work)
func onSummaryRequested(engine *atmos.Engine, event *SummaryRequested) {
	engine.Emit(&SummaryGenerated{
		ParentTaskID: event.ParentTaskID,
		Time:         time.Now(),
	})
}

// onSummaryGenerated completes the parent task after summary is generated
func onSummaryGenerated(engine *atmos.Engine, event *SummaryGenerated) {
	// Summary complete - now complete the parent task
	engine.Emit(&TaskCompleted{
		TaskID: event.ParentTaskID,
		Time:   time.Now(),
	})
}

// ============================================================================
// SETUP - Wire up orchestration in engine
// ============================================================================

func setupOrchestration(engine *atmos.Engine) {
	// Register reducers
	engine.When("next_task_selected").Updates("hearth", reduceNextTaskSelected)
	engine.When("task_executed").Updates("hearth", reduceTaskExecuted)

	// Register before hooks (where work happens)
	engine.When("next_task_selected").
		Before(atmos.NewTypedListener(TypedListenerFunc[*NextTaskSelected](beforeNextTaskSelected)))
	engine.When("task_executed").
		Before(atmos.NewTypedListener(TypedListenerFunc[*TaskExecuted](beforeTaskExecuted)))
	engine.When("summary_generated").
		Before(atmos.NewTypedListener(TypedListenerFunc[*SummaryGenerated](beforeSummaryGenerated)))

	// Register listeners (event chaining)
	engine.When("execute_tasks_requested").
		Then(atmos.NewTypedListener(TypedListenerFunc[*ExecuteTasksRequested](onExecuteTasksRequested)))
	engine.When("next_task_selected").
		Then(atmos.NewTypedListener(TypedListenerFunc[*NextTaskSelected](onNextTaskSelected)))
	engine.When("task_executed").
		Then(atmos.NewTypedListener(TypedListenerFunc[*TaskExecuted](onTaskExecuted)))

	// Parent auto-completion (always runs - replaces reducer mutation)
	engine.When("task_completed").
		Then(atmos.NewTypedListener(TypedListenerFunc[*TaskCompleted](onTaskCompletedParent)))

	// Orchestration scheduling continuation (only during active orchestration)
	engine.When("task_completed").
		Then(atmos.NewTypedListener(TypedListenerFunc[*TaskCompleted](onTaskCompletedSchedule)))

	// Summary generation chain
	engine.When("summary_requested").
		Then(atmos.NewTypedListener(TypedListenerFunc[*SummaryRequested](onSummaryRequested)))
	engine.When("summary_generated").
		Then(atmos.NewTypedListener(TypedListenerFunc[*SummaryGenerated](onSummaryGenerated)))

	// Register event factories for persistence
	engine.When("execute_tasks_requested", func() atmos.Event { return &ExecuteTasksRequested{} })
	engine.When("next_task_selected", func() atmos.Event { return &NextTaskSelected{} })
	engine.When("task_executed", func() atmos.Event { return &TaskExecuted{} })
	engine.When("summary_requested", func() atmos.Event { return &SummaryRequested{} })
	engine.When("summary_generated", func() atmos.Event { return &SummaryGenerated{} })
}
