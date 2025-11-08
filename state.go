package hearth

import "time"

// HearthState holds all tasks
type HearthState struct {
	Tasks map[string]*Task
}

// Task represents a task in the system
type Task struct {
	ID          string
	Title       string
	Description string
	ParentID    *string
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
}
