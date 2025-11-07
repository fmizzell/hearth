package hearth

import (
	"errors"
	"sort"
	"time"

	"github.com/cumulusrpg/atmos"
)

var ErrEventRejected = errors.New("event was rejected by validators")

// Hearth is the main engine wrapper
type Hearth struct {
	engine *atmos.Engine
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
	ParentID    *string
	DependsOn   *string
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// NewHearth creates a new Hearth instance with optional atmos options
func NewHearth(opts ...atmos.EngineOption) *Hearth {
	engine := atmos.NewEngine(opts...)

	// Register initial state
	engine.RegisterState("hearth", HearthState{
		Tasks: make(map[string]*Task),
	})

	// Register event handlers
	engine.When("task_created", func() atmos.Event { return &TaskCreated{} }).
		Updates("hearth", reduceTaskCreated)

	engine.When("task_started", func() atmos.Event { return &TaskStarted{} }).
		Updates("hearth", reduceTaskStarted)

	engine.When("task_completed", func() atmos.Event { return &TaskCompleted{} }).
		Requires(atmos.Valid(&TaskCompletionValidator{})).
		Updates("hearth", reduceTaskCompleted)

	return &Hearth{
		engine: engine,
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

// GetTask returns a specific task by ID
func (h *Hearth) GetTask(id string) *Task {
	state := h.engine.GetState("hearth").(HearthState)
	return state.Tasks[id]
}

// GetChildTasks returns all child tasks of a parent task
func (h *Hearth) GetChildTasks(parentID string) []*Task {
	state := h.engine.GetState("hearth").(HearthState)

	var children []*Task
	var childIDs []string

	// Collect child task IDs
	for id, task := range state.Tasks {
		if task.ParentID != nil && *task.ParentID == parentID {
			childIDs = append(childIDs, id)
		}
	}

	// Sort for deterministic order
	sort.Strings(childIDs)

	// Build result
	for _, id := range childIDs {
		children = append(children, state.Tasks[id])
	}

	return children
}

// GetNextTask returns the next task to work on (first todo task with satisfied dependencies)
func (h *Hearth) GetNextTask() *Task {
	state := h.engine.GetState("hearth").(HearthState)

	// Collect all tasks
	var tasks []*Task
	for _, task := range state.Tasks {
		tasks = append(tasks, task)
	}

	// Sort by creation time to process tasks in the order they were created
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	// Find first task with status "todo" that has no unsatisfied dependencies
	for _, task := range tasks {
		if task.Status != "todo" {
			continue
		}

		// Skip tasks that have children (parent/epic tasks - work on leaves only)
		hasChildren := false
		for _, otherTask := range state.Tasks {
			if otherTask.ParentID != nil && *otherTask.ParentID == task.ID {
				hasChildren = true
				break
			}
		}
		if hasChildren {
			continue
		}

		// Check if task has an upstream dependency
		if task.DependsOn != nil {
			// Check if dependency is completed
			dep := state.Tasks[*task.DependsOn]
			if dep == nil || dep.Status != "completed" {
				continue // Skip this task, dependency not satisfied
			}
		}

		// Task is ready to work on
		return task
	}

	return nil
}
