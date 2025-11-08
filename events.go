package hearth

import "time"

// TaskCreated event
type TaskCreated struct {
	TaskID      string
	Title       string
	Description string
	ParentID    *string
	Time        time.Time
}

func (e *TaskCreated) Type() string         { return "task_created" }
func (e *TaskCreated) Timestamp() time.Time { return e.Time }

// TaskStarted event
type TaskStarted struct {
	TaskID string
	Time   time.Time
}

func (e *TaskStarted) Type() string         { return "task_started" }
func (e *TaskStarted) Timestamp() time.Time { return e.Time }

// TaskCompleted event
type TaskCompleted struct {
	TaskID string
	Time   time.Time
}

func (e *TaskCompleted) Type() string         { return "task_completed" }
func (e *TaskCompleted) Timestamp() time.Time { return e.Time }

// ============================================================================
// ORCHESTRATION EVENTS - Event-driven task execution
// ============================================================================

// ExecuteTasksRequested triggers the orchestration loop
type ExecuteTasksRequested struct {
	Time time.Time
}

func (e *ExecuteTasksRequested) Type() string         { return "execute_tasks_requested" }
func (e *ExecuteTasksRequested) Timestamp() time.Time { return e.Time }

// NextTaskSelected represents scheduler picking next task
type NextTaskSelected struct {
	TaskID string // empty if no tasks available
	Reason string // why this task was selected
	Time   time.Time
}

func (e *NextTaskSelected) Type() string         { return "next_task_selected" }
func (e *NextTaskSelected) Timestamp() time.Time { return e.Time }

// TaskExecuted represents task execution completion
type TaskExecuted struct {
	TaskID     string
	ResultPath string // path to result file
	Time       time.Time
}

func (e *TaskExecuted) Type() string         { return "task_executed" }
func (e *TaskExecuted) Timestamp() time.Time { return e.Time }

// SummaryRequested is emitted when a parent task needs a summary (all children complete)
type SummaryRequested struct {
	ParentTaskID string
	Time         time.Time
}

func (e *SummaryRequested) Type() string         { return "summary_requested" }
func (e *SummaryRequested) Timestamp() time.Time { return e.Time }

// SummaryGenerated is emitted when a summary has been generated for a parent
type SummaryGenerated struct {
	ParentTaskID string
	SummaryPath  string // enriched by before hook
	Time         time.Time
}

func (e *SummaryGenerated) Type() string         { return "summary_generated" }
func (e *SummaryGenerated) Timestamp() time.Time { return e.Time }
