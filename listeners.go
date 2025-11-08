package hearth

import (
	"fmt"
	"time"

	"github.com/cumulusrpg/atmos"
)

// ============================================================================
// HELPER - Convert functions to typed listeners
// ============================================================================

// TypedListenerFunc converts a plain function into a typed listener
// TODO: This should be built into Atmos so we can pass functions directly
type TypedListenerFunc[T atmos.Event] func(*atmos.Engine, T)

func (f TypedListenerFunc[T]) HandleTyped(engine *atmos.Engine, event T) {
	f(engine, event)
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

	// Continue orchestration - find next task
	engine.Emit(&NextTaskSelected{Time: time.Now()})
}
