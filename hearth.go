package hearth

import (
	"errors"
	"time"

	"github.com/cumulusrpg/atmos"
)

var ErrEventRejected = errors.New("event was rejected by validators")

// Hearth is the main engine wrapper
type Hearth struct {
	projectID string
	engine    *atmos.Engine
}

// HearthState holds all tasks
type HearthState struct {
	Tasks map[string]*Task
}

// Task represents a task in the system
type Task struct {
	ID          string
	Title       string
	Description string
	DependsOn   *string
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// NewHearth creates a new Hearth instance
func NewHearth(projectID string) *Hearth {
	engine := atmos.NewEngine()

	// Register initial state
	engine.RegisterState("hearth", HearthState{
		Tasks: make(map[string]*Task),
	})

	// Register event handlers
	engine.When("task_created", func() atmos.Event { return &TaskCreated{} }).
		Updates("hearth", reduceTaskCreated)

	return &Hearth{
		projectID: projectID,
		engine:    engine,
	}
}

// Process is the single method to consume and process events
func (h *Hearth) Process(event atmos.Event) error {
	success := h.engine.Emit(event)
	if !success {
		return ErrEventRejected
	}
	return nil
}

// GetTasks returns all tasks
func (h *Hearth) GetTasks() map[string]*Task {
	state := h.engine.GetState("hearth").(HearthState)
	return state.Tasks
}
