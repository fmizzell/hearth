package hearth

import (
	"fmt"
	"testing"
	"time"

	"github.com/cumulusrpg/atmos"
)

// ============================================================================
// TEST-ONLY EVENTS (not in production yet)
// ============================================================================

type OrcTaskCreated struct {
	TaskID      string
	Description string
	ParentID    *string
	Time        time.Time
}

func (e *OrcTaskCreated) Type() string { return "orc_task_created" }

type OrcTaskCompleted struct {
	TaskID string
	Time   time.Time
}

func (e *OrcTaskCompleted) Type() string { return "orc_task_completed" }

// ============================================================================
// STATE
// ============================================================================

type OrchestratedTask struct {
	ID          string
	Description string
	ParentID    *string
	Children    []string
	Status      string // "todo", "in-progress", "completed"
	ResultPath  string
}

type OrchestrationState struct {
	Tasks          map[string]*OrchestratedTask
	CompletedTasks map[string]bool
}

func NewOrchestrationState() *OrchestrationState {
	return &OrchestrationState{
		Tasks:          make(map[string]*OrchestratedTask),
		CompletedTasks: make(map[string]bool),
	}
}

// ============================================================================
// REDUCERS
// ============================================================================

func ReduceOrcTaskCreated(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(*OrchestrationState)
	e := event.(*OrcTaskCreated)

	s.Tasks[e.TaskID] = &OrchestratedTask{
		ID:          e.TaskID,
		Description: e.Description,
		ParentID:    e.ParentID,
		Children:    []string{},
		Status:      "todo",
	}

	// Add to parent's children
	if e.ParentID != nil {
		parent := s.Tasks[*e.ParentID]
		if parent != nil {
			parent.Children = append(parent.Children, e.TaskID)
		}
	}

	return s
}

func ReduceNextTaskSelected(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(*OrchestrationState)
	e := event.(*NextTaskSelected)

	if e.TaskID != "" {
		task := s.Tasks[e.TaskID]
		if task != nil {
			task.Status = "in-progress"
		}
	}

	return s
}

func ReduceTaskExecuted(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(*OrchestrationState)
	e := event.(*TaskExecuted)

	task := s.Tasks[e.TaskID]
	if task != nil {
		task.ResultPath = e.ResultPath
	}

	return s
}

func ReduceOrcTaskCompleted(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(*OrchestrationState)
	e := event.(*OrcTaskCompleted)

	task := s.Tasks[e.TaskID]
	if task != nil {
		task.Status = "completed"
	}

	s.CompletedTasks[e.TaskID] = true

	return s
}

// ============================================================================
// BEFORE HOOKS
// ============================================================================

func BeforeNextTaskSelected(engine *atmos.Engine, event *NextTaskSelected) {
	state := engine.GetState("orchestration").(*OrchestrationState)

	// Find next depth-first task
	nextTask := findNextDepthFirst(state)

	if nextTask == nil {
		event.TaskID = "" // Signal halt
		return
	}

	event.TaskID = nextTask.ID
}

func BeforeTaskExecuted(engine *atmos.Engine, event *TaskExecuted) {
	state := engine.GetState("orchestration").(*OrchestrationState)

	task := state.Tasks[event.TaskID]
	if task == nil {
		// TODO: how to error here?
		return
	}

	// TODO: gather context
	// TODO: build prompt
	// TODO: call Claude

	// Mock storing result
	resultPath := fmt.Sprintf(".hearth/results/%s.md", event.TaskID)

	event.ResultPath = resultPath
}

// ============================================================================
// LISTENERS
// ============================================================================

func OnExecuteTasksRequested(engine *atmos.Engine, event *ExecuteTasksRequested) {
	// Trigger scheduler
	engine.Emit(&NextTaskSelected{Time: time.Now()})
}

func OnNextTaskSelected(engine *atmos.Engine, event *NextTaskSelected) {
	if event.TaskID == "" {
		// Nothing to do, halt
		return
	}

	// Execute the selected task
	engine.Emit(&TaskExecuted{
		TaskID: event.TaskID,
		Time:   time.Now(),
	})
}

func OnTaskExecuted(engine *atmos.Engine, event *TaskExecuted) {
	state := engine.GetState("orchestration").(*OrchestrationState)

	task := state.Tasks[event.TaskID]
	if task == nil {
		return
	}

	// Check if task has children (Claude might have run `hearth add`)
	if len(task.Children) > 0 {
		// Don't complete, go depth-first into children
		engine.Emit(&NextTaskSelected{Time: time.Now()})
		return
	}

	// No children, complete the task
	engine.Emit(&OrcTaskCompleted{
		TaskID: event.TaskID,
		Time:   time.Now(),
	})
}

func OnOrcTaskCompleted(engine *atmos.Engine, event *OrcTaskCompleted) {
	state := engine.GetState("orchestration").(*OrchestrationState)

	task := state.Tasks[event.TaskID]

	// Check if parent needs completion
	if task.ParentID != nil {
		parent := state.Tasks[*task.ParentID]
		if parent != nil && allChildrenComplete(parent, state) {
			// All children done, complete the parent
			engine.Emit(&OrcTaskCompleted{
				TaskID: *task.ParentID,
				Time:   time.Now(),
			})
			// TODO: Generate summary before completing
			return
		}
	}

	// Continue scheduling next task
	engine.Emit(&NextTaskSelected{Time: time.Now()})
}

// ============================================================================
// HELPERS
// ============================================================================

func findNextDepthFirst(state *OrchestrationState) *OrchestratedTask {
	// Find any "todo" task
	for _, task := range state.Tasks {
		if task.Status == "todo" {
			return task
		}
	}
	return nil
}

func allChildrenComplete(parent *OrchestratedTask, state *OrchestrationState) bool {
	for _, childID := range parent.Children {
		if !state.CompletedTasks[childID] {
			return false
		}
	}
	return true
}

// ============================================================================
// TEST
// ============================================================================

func TestOrchestrationLoop(t *testing.T) {
	// Create engine
	engine := atmos.NewEngine()

	// Register state
	engine.RegisterState("orchestration", NewOrchestrationState())

	// Register reducers using fluent API
	engine.When("orc_task_created").Updates("orchestration", ReduceOrcTaskCreated)
	engine.When("next_task_selected").Updates("orchestration", ReduceNextTaskSelected)
	engine.When("task_executed").Updates("orchestration", ReduceTaskExecuted)
	engine.When("orc_task_completed").Updates("orchestration", ReduceOrcTaskCompleted)

	// Register before hooks
	engine.When("next_task_selected").
		Before(atmos.NewTypedListener(TypedListenerFunc[*NextTaskSelected](BeforeNextTaskSelected)))
	engine.When("task_executed").
		Before(atmos.NewTypedListener(TypedListenerFunc[*TaskExecuted](BeforeTaskExecuted)))

	// Register listeners
	engine.When("execute_tasks_requested").
		Then(atmos.NewTypedListener(TypedListenerFunc[*ExecuteTasksRequested](OnExecuteTasksRequested)))
	engine.When("next_task_selected").
		Then(atmos.NewTypedListener(TypedListenerFunc[*NextTaskSelected](OnNextTaskSelected)))
	engine.When("task_executed").
		Then(atmos.NewTypedListener(TypedListenerFunc[*TaskExecuted](OnTaskExecuted)))
	engine.When("orc_task_completed").
		Then(atmos.NewTypedListener(TypedListenerFunc[*OrcTaskCompleted](OnOrcTaskCompleted)))

	// Create a task
	success := engine.Emit(&OrcTaskCreated{
		TaskID:      "T-001",
		Description: "Test task",
		Time:        time.Now(),
	})
	if !success {
		t.Fatal("Failed to create task")
	}

	// Verify task was created
	state := engine.GetState("orchestration").(*OrchestrationState)
	if state.Tasks["T-001"] == nil {
		t.Fatal("Task was not created")
	}
	if state.Tasks["T-001"].Status != "todo" {
		t.Fatalf("Expected status 'todo', got '%s'", state.Tasks["T-001"].Status)
	}

	// Start execution
	success = engine.Emit(&ExecuteTasksRequested{Time: time.Now()})
	if !success {
		t.Fatal("Failed to start execution")
	}

	// Verify task was executed and completed
	state = engine.GetState("orchestration").(*OrchestrationState)
	if state.Tasks["T-001"].Status != "completed" {
		t.Fatalf("Expected status 'completed', got '%s'", state.Tasks["T-001"].Status)
	}
	if state.Tasks["T-001"].ResultPath == "" {
		t.Fatal("Expected result path to be set")
	}
}

func TestOrchestrationWithChildren(t *testing.T) {
	// Create engine
	engine := atmos.NewEngine()

	// Register state
	engine.RegisterState("orchestration", NewOrchestrationState())

	// Register everything using fluent API
	engine.When("orc_task_created").Updates("orchestration", ReduceOrcTaskCreated)
	engine.When("next_task_selected").Updates("orchestration", ReduceNextTaskSelected)
	engine.When("task_executed").Updates("orchestration", ReduceTaskExecuted)
	engine.When("orc_task_completed").Updates("orchestration", ReduceOrcTaskCompleted)

	engine.When("next_task_selected").
		Before(atmos.NewTypedListener(TypedListenerFunc[*NextTaskSelected](BeforeNextTaskSelected)))
	engine.When("task_executed").
		Before(atmos.NewTypedListener(TypedListenerFunc[*TaskExecuted](BeforeTaskExecuted)))

	engine.When("execute_tasks_requested").
		Then(atmos.NewTypedListener(TypedListenerFunc[*ExecuteTasksRequested](OnExecuteTasksRequested)))
	engine.When("next_task_selected").
		Then(atmos.NewTypedListener(TypedListenerFunc[*NextTaskSelected](OnNextTaskSelected)))
	engine.When("task_executed").
		Then(atmos.NewTypedListener(TypedListenerFunc[*TaskExecuted](OnTaskExecuted)))
	engine.When("orc_task_completed").
		Then(atmos.NewTypedListener(TypedListenerFunc[*OrcTaskCompleted](OnOrcTaskCompleted)))

	// Create parent task
	engine.Emit(&OrcTaskCreated{
		TaskID:      "T-001",
		Description: "Parent task",
		Time:        time.Now(),
	})

	// Create child task (simulating Claude running `hearth add`)
	parentID := "T-001"
	engine.Emit(&OrcTaskCreated{
		TaskID:      "T-001-1",
		Description: "Child task",
		ParentID:    &parentID,
		Time:        time.Now(),
	})

	// Start execution
	engine.Emit(&ExecuteTasksRequested{Time: time.Now()})

	// Verify both tasks completed
	state := engine.GetState("orchestration").(*OrchestrationState)
	if state.Tasks["T-001-1"].Status != "completed" {
		t.Fatalf("Expected child status 'completed', got '%s'", state.Tasks["T-001-1"].Status)
	}
	if state.Tasks["T-001"].Status != "completed" {
		t.Fatalf("Expected parent status 'completed', got '%s'", state.Tasks["T-001"].Status)
	}
}
