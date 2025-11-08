package hearth

import (
	"github.com/cumulusrpg/atmos"
)

// TaskCompletionValidator ensures a task can only be completed if it has no incomplete children
type TaskCompletionValidator struct{}

func (v *TaskCompletionValidator) ValidateTyped(engine *atmos.Engine, event *TaskCompleted) bool {
	state := engine.GetState("hearth").(HearthState)

	// Check if this task has any children
	hasIncompleteChildren := false
	for _, task := range state.Tasks {
		if task.ParentID != nil && *task.ParentID == event.TaskID {
			// Found a child - check if it's incomplete
			if task.Status != "completed" {
				hasIncompleteChildren = true
				break
			}
		}
	}

	// Reject completion if task has incomplete children
	// (parent tasks auto-complete when all children are done)
	return !hasIncompleteChildren
}

// reduceTaskCreated handles TaskCreated events
func reduceTaskCreated(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(HearthState)
	e := event.(*TaskCreated)

	s.Tasks[e.TaskID] = &Task{
		ID:          e.TaskID,
		Title:       e.Title,
		Description: e.Description,
		ParentID:    e.ParentID,
		Status:      "todo",
		CreatedAt:   e.Time,
	}

	return s
}

// reduceTaskStarted handles TaskStarted events
func reduceTaskStarted(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(HearthState)
	e := event.(*TaskStarted)

	if task, exists := s.Tasks[e.TaskID]; exists {
		task.Status = "in-progress"
	}

	return s
}

// reduceTaskCompleted handles TaskCompleted events
func reduceTaskCompleted(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(HearthState)
	e := event.(*TaskCompleted)

	if task, exists := s.Tasks[e.TaskID]; exists {
		task.Status = "completed"
		completedAt := e.Time
		task.CompletedAt = &completedAt

		// NOTE: Parent completion is now handled by onTaskCompletedOrchestration listener
		// which emits TaskCompleted events for parents instead of mutating state
	}

	return s
}

// ============================================================================
// ORCHESTRATION REDUCERS - Build state from orchestration events
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
