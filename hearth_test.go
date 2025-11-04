package hearth

import (
	"testing"
	"time"

	"github.com/cumulusrpg/atmos"
	"github.com/stretchr/testify/assert"
)

// TestHearthJourney is an end-to-end test showing how Hearth should work.
// Hearth has ONE method: Process(event) - everything else is queries.
// We'll comment out most of it and gradually uncomment as we implement each piece.
func TestHearthJourney(t *testing.T) {
	// Step 1: Create a new Hearth engine
	h := NewHearth("test-project")
	assert.NotNil(t, h)

	// // Step 2: Add tasks by processing TaskCreated events
	// err := h.Process(TaskCreated{
	// 	TaskID:      "T1",
	// 	Title:       "Implement login endpoint",
	// 	Description: "Create POST /login that accepts email/password",
	// 	Time:        time.Now(),
	// })
	// assert.NoError(t, err)
	//
	// err = h.Process(TaskCreated{
	// 	TaskID:      "T2",
	// 	Title:       "Add login tests",
	// 	Description: "Write tests for login endpoint",
	// 	DependsOn:   strPtr("T1"), // T2 depends on T1
	// 	Time:        time.Now(),
	// })
	// assert.NoError(t, err)
	//
	// err = h.Process(TaskCreated{
	// 	TaskID:      "T3",
	// 	Title:       "Write documentation",
	// 	Description: "Document the login API",
	// 	Time:        time.Now(),
	// })
	// assert.NoError(t, err)
	//
	// // Step 3: Query state
	// tasks := h.GetTasks()
	// assert.Equal(t, 3, len(tasks))
	// assert.Equal(t, "todo", tasks["T1"].Status)
	// assert.Equal(t, "todo", tasks["T2"].Status)
	// assert.Equal(t, "todo", tasks["T3"].Status)
	//
	// // Step 4: Get next task (should be T1, since T2 depends on it)
	// next := h.GetNextTask()
	// assert.NotNil(t, next)
	// assert.Equal(t, "T1", next.ID)
	// assert.Equal(t, "Implement login endpoint", next.Title)
	//
	// // Step 5: Start work on T1 by processing TaskStarted event
	// err = h.Process(TaskStarted{
	// 	TaskID: "T1",
	// 	Time:   time.Now(),
	// })
	// assert.NoError(t, err)
	//
	// task := h.GetTask("T1")
	// assert.Equal(t, "in-progress", task.Status)
	//
	// // Step 6: Add logs by processing TaskLogAdded events
	// err = h.Process(TaskLogAdded{
	// 	TaskID:  "T1",
	// 	Message: "Created the endpoint handler",
	// 	Time:    time.Now(),
	// })
	// assert.NoError(t, err)
	//
	// err = h.Process(TaskLogAdded{
	// 	TaskID:  "T1",
	// 	Message: "Added input validation",
	// 	Time:    time.Now(),
	// })
	// assert.NoError(t, err)
	//
	// logs := h.GetLogs("T1")
	// assert.Equal(t, 2, len(logs))
	// assert.Equal(t, "Created the endpoint handler", logs[0].Message)
	//
	// // Step 7: Complete T1 by processing TaskCompleted event
	// err = h.Process(TaskCompleted{
	// 	TaskID: "T1",
	// 	Time:   time.Now(),
	// })
	// assert.NoError(t, err)
	//
	// task = h.GetTask("T1")
	// assert.Equal(t, "completed", task.Status)
	// assert.NotNil(t, task.CompletedAt)
	//
	// // Step 8: Now T2 should be available (its dependency is done)
	// next = h.GetNextTask()
	// assert.NotNil(t, next)
	// assert.Equal(t, "T2", next.ID)
	//
	// // Step 9: Complete T2
	// h.Process(TaskStarted{TaskID: "T2", Time: time.Now()})
	// h.Process(TaskLogAdded{TaskID: "T2", Message: "Wrote test cases", Time: time.Now()})
	// h.Process(TaskCompleted{TaskID: "T2", Time: time.Now()})
	//
	// // Step 10: T3 has no dependencies, so it's available
	// next = h.GetNextTask()
	// assert.NotNil(t, next)
	// assert.Equal(t, "T3", next.ID)
	//
	// // Step 11: Complete T3
	// h.Process(TaskStarted{TaskID: "T3", Time: time.Now()})
	// h.Process(TaskCompleted{TaskID: "T3", Time: time.Now()})
	//
	// // Step 12: No more tasks
	// next = h.GetNextTask()
	// assert.Nil(t, next)
	//
	// // Step 13: Verify all tasks completed
	// tasks = h.GetTasks()
	// for _, task := range tasks {
	// 	assert.Equal(t, "completed", task.Status)
	// }
	//
	// // Step 14: Test persistence - save and reload
	// events := h.GetEvents()
	// assert.True(t, len(events) > 0)
	//
	// // Create new engine and replay events
	// h2 := NewHearth("test-project")
	// err = h2.SetEvents(events)
	// assert.NoError(t, err)
	//
	// // State should be identical
	// tasks2 := h2.GetTasks()
	// assert.Equal(t, len(tasks), len(tasks2))
	// for id, task := range tasks2 {
	// 	assert.Equal(t, tasks[id].ID, task.ID)
	// 	assert.Equal(t, tasks[id].Status, task.Status)
	// 	assert.Equal(t, tasks[id].Title, task.Title)
	// }
	//
	// // Step 15: Test iteration tracking
	// h3 := NewHearth("test-project-2")
	// h3.Process(TaskCreated{
	// 	TaskID:      "T1",
	// 	Title:       "Simple task",
	// 	Description: "Do something",
	// 	Time:        time.Now(),
	// })
	//
	// // Simulate an agent iteration
	// h3.Process(IterationStarted{
	// 	IterationID: 1,
	// 	TaskID:      "T1",
	// 	Prompt:      "Work on the task",
	// 	Time:        time.Now(),
	// })
	//
	// h3.Process(IterationCompleted{
	// 	IterationID: 1,
	// 	TaskID:      "T1",
	// 	Response:    "Task completed successfully",
	// 	Success:     true,
	// 	Time:        time.Now(),
	// })
	//
	// iterations := h3.GetIterations()
	// assert.Equal(t, 1, len(iterations))
	// assert.Equal(t, "T1", iterations[0].TaskID)
	// assert.True(t, iterations[0].Success)
}

// Helper function
func strPtr(s string) *string {
	return &s
}

// Event types (will define these properly later)
type TaskCreated struct {
	TaskID      string
	Title       string
	Description string
	DependsOn   *string
	Time        time.Time
}

func (e TaskCreated) Type() string         { return "task_created" }
func (e TaskCreated) Timestamp() time.Time { return e.Time }

type TaskStarted struct {
	TaskID string
	Time   time.Time
}

func (e TaskStarted) Type() string         { return "task_started" }
func (e TaskStarted) Timestamp() time.Time { return e.Time }

type TaskLogAdded struct {
	TaskID  string
	Message string
	Time    time.Time
}

func (e TaskLogAdded) Type() string         { return "task_log_added" }
func (e TaskLogAdded) Timestamp() time.Time { return e.Time }

type TaskCompleted struct {
	TaskID string
	Time   time.Time
}

func (e TaskCompleted) Type() string         { return "task_completed" }
func (e TaskCompleted) Timestamp() time.Time { return e.Time }

type IterationStarted struct {
	IterationID int
	TaskID      string
	Prompt      string
	Time        time.Time
}

func (e IterationStarted) Type() string         { return "iteration_started" }
func (e IterationStarted) Timestamp() time.Time { return e.Time }

type IterationCompleted struct {
	IterationID int
	TaskID      string
	Response    string
	Success     bool
	Time        time.Time
}

func (e IterationCompleted) Type() string         { return "iteration_completed" }
func (e IterationCompleted) Timestamp() time.Time { return e.Time }

// Hearth interface (will implement later)
type Hearth struct {
	engine *atmos.Engine
}

func NewHearth(projectID string) *Hearth {
	return &Hearth{}
}

func (h *Hearth) Process(event atmos.Event) error {
	return nil
}

func (h *Hearth) GetTask(id string) *Task {
	return nil
}

func (h *Hearth) GetNextTask() *Task {
	return nil
}

func (h *Hearth) GetTasks() map[string]*Task {
	return nil
}

func (h *Hearth) GetLogs(taskID string) []TaskLog {
	return nil
}

func (h *Hearth) GetEvents() []atmos.Event {
	return nil
}

func (h *Hearth) SetEvents(events []atmos.Event) error {
	return nil
}

func (h *Hearth) GetIterations() []Iteration {
	return nil
}

// State types (will define properly later)
type Task struct {
	ID          string
	Title       string
	Description string
	DependsOn   *string
	Status      string
	Logs        []TaskLog
	CreatedAt   time.Time
	CompletedAt *time.Time
}

type TaskLog struct {
	Message   string
	CreatedAt time.Time
}

type Iteration struct {
	ID      int
	TaskID  string
	Prompt  string
	Success bool
}
