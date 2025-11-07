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

// GetNextTask returns the next task to work on using depth-first traversal
func (h *Hearth) GetNextTask() *Task {
	state := h.engine.GetState("hearth").(HearthState)

	// Collect all tasks into a slice
	var tasks []*Task
	for _, task := range state.Tasks {
		tasks = append(tasks, task)
	}

	return findNextTask(tasks)
}

// findNextTask implements depth-first task selection
// Pure function that takes a slice of tasks and returns the next eligible task
func findNextTask(tasks []*Task) *Task {
	// Build a map for quick lookups
	taskMap := make(map[string]*Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	// Find root tasks (no parent)
	var roots []*Task
	for _, task := range tasks {
		if task.ParentID == nil {
			roots = append(roots, task)
		}
	}

	// Sort roots by creation time
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].CreatedAt.Before(roots[j].CreatedAt)
	})

	// Depth-first search through each root
	for _, root := range roots {
		if task := findNextInSubtree(root, taskMap); task != nil {
			return task
		}
	}

	return nil
}

// findNextInSubtree recursively searches a subtree for the next eligible task
func findNextInSubtree(parent *Task, taskMap map[string]*Task) *Task {
	// Find children of this parent
	var children []*Task
	for _, task := range taskMap {
		if task.ParentID != nil && *task.ParentID == parent.ID {
			children = append(children, task)
		}
	}

	// Sort children by creation time
	sort.Slice(children, func(i, j int) bool {
		return children[i].CreatedAt.Before(children[j].CreatedAt)
	})

	if len(children) == 0 {
		// No children - this is a leaf
		if parent.Status != "todo" {
			return nil
		}

		// Check dependencies
		if parent.DependsOn != nil {
			dep := taskMap[*parent.DependsOn]
			if dep == nil || dep.Status != "completed" {
				return nil
			}
		}

		return parent
	}

	// Has children - depth-first through them
	for _, child := range children {
		if task := findNextInSubtree(child, taskMap); task != nil {
			return task
		}
	}

	return nil
}
