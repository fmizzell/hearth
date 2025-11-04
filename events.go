package hearth

import "time"

// TaskCreated event
type TaskCreated struct {
	TaskID      string
	Title       string
	Description string
	DependsOn   *string
	Time        time.Time
}

func (e TaskCreated) Type() string         { return "task_created" }
func (e TaskCreated) Timestamp() time.Time { return e.Time }

// TaskCompleted event
type TaskCompleted struct {
	TaskID string
	Time   time.Time
}

func (e TaskCompleted) Type() string         { return "task_completed" }
func (e TaskCompleted) Timestamp() time.Time { return e.Time }
