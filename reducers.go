package hearth

import (
	"time"

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
		DependsOn:   e.DependsOn,
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

		// Check if this task has a parent, and if so, check if all siblings are done
		if task.ParentID != nil {
			autoCompleteParent(s, *task.ParentID, e.Time)
		}
	}

	return s
}

// autoCompleteParent recursively completes parent tasks when all children are done
func autoCompleteParent(state HearthState, parentID string, completedTime time.Time) {
	// Check if all children of this parent are completed
	allChildrenDone := true
	for _, task := range state.Tasks {
		if task.ParentID != nil && *task.ParentID == parentID {
			if task.Status != "completed" {
				allChildrenDone = false
				break
			}
		}
	}

	// If all children done, complete the parent
	if allChildrenDone {
		if parent, exists := state.Tasks[parentID]; exists && parent.Status != "completed" {
			parent.Status = "completed"
			parent.CompletedAt = &completedTime

			// Recursively check the parent's parent
			if parent.ParentID != nil {
				autoCompleteParent(state, *parent.ParentID, completedTime)
			}
		}
	}
}
