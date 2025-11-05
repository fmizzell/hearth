package hearth

import (
	"time"

	"github.com/cumulusrpg/atmos"
)

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
