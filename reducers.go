package hearth

import "github.com/cumulusrpg/atmos"

// reduceTaskCreated handles TaskCreated events
func reduceTaskCreated(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(HearthState)
	e := event.(TaskCreated)

	s.Tasks[e.TaskID] = &Task{
		ID:          e.TaskID,
		Title:       e.Title,
		Description: e.Description,
		DependsOn:   e.DependsOn,
		Status:      "todo",
		CreatedAt:   e.Time,
	}

	return s
}

// reduceTaskCompleted handles TaskCompleted events
func reduceTaskCompleted(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(HearthState)
	e := event.(TaskCompleted)

	if task, exists := s.Tasks[e.TaskID]; exists {
		task.Status = "completed"
		completedAt := e.Time
		task.CompletedAt = &completedAt
	}

	return s
}
