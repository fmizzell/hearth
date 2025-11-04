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
