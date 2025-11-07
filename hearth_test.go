package hearth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestHearthJourney shows the MINIMAL loop needed to work with Claude.
// We'll add features (logs, dependencies, etc) only when we actually need them.
func TestHearthJourney(t *testing.T) {
	// Step 1: Create hearth
	h := NewHearth()
	assert.NotNil(t, h)

	// Step 2: Add tasks with dependencies
	err := h.Process(&TaskCreated{
		TaskID:      "T1",
		Title:       "Implement login endpoint",
		Description: "Create POST /login that accepts email/password",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&TaskCreated{
		TaskID:      "T2",
		Title:       "Add login tests",
		Description: "Write tests for login endpoint",
		DependsOn:   strPtr("T1"), // T2 depends on T1
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&TaskCreated{
		TaskID:      "T3",
		Title:       "Write documentation",
		Description: "Document the login API",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Step 3: Get next task (should be T1, since T2 depends on it, T3 is independent)
	task := h.GetNextTask()
	assert.NotNil(t, task)
	assert.Equal(t, "T1", task.ID)
	assert.Equal(t, "todo", task.Status)

	// Step 4: Complete T1
	err = h.Process(&TaskCompleted{
		TaskID: "T1",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	task = h.GetTask("T1")
	assert.Equal(t, "completed", task.Status)
	assert.NotNil(t, task.CompletedAt)

	// Step 5: Now T2 should be available (its dependency is done)
	next := h.GetNextTask()
	assert.NotNil(t, next)
	assert.Equal(t, "T2", next.ID)

	// Step 6: Complete T2
	err = h.Process(&TaskCompleted{TaskID: "T2", Time: time.Now()})
	assert.NoError(t, err)

	// Step 7: T3 has no dependencies, should be available
	next = h.GetNextTask()
	assert.NotNil(t, next)
	assert.Equal(t, "T3", next.ID)

	// Step 8: Complete T3
	err = h.Process(&TaskCompleted{TaskID: "T3", Time: time.Now()})
	assert.NoError(t, err)

	// Step 9: No more tasks
	next = h.GetNextTask()
	assert.Nil(t, next)

	// Step 10: Test hierarchical tasks (parent/child relationships)
	// Create root EPIC task
	err = h.Process(&TaskCreated{
		TaskID:      "T4",
		Title:       "Improve code quality",
		Description: "Root epic for code improvements",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create child: Analysis (child of T4)
	err = h.Process(&TaskCreated{
		TaskID:      "T5",
		Title:       "Analyze codebase",
		Description: "Run analysis tools and identify issues",
		ParentID:    strPtr("T4"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create child: Apply fixes (child of T4, depends on T5)
	err = h.Process(&TaskCreated{
		TaskID:      "T6",
		Title:       "Apply priority fixes",
		Description: "Fix high priority issues",
		ParentID:    strPtr("T4"),
		DependsOn:   strPtr("T5"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create nested children (children of T6 - parallel fix tasks)
	err = h.Process(&TaskCreated{
		TaskID:      "T7",
		Title:       "Fix type assertions",
		Description: "Add safety checks",
		ParentID:    strPtr("T6"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&TaskCreated{
		TaskID:      "T8",
		Title:       "Fix task ordering",
		Description: "Add deterministic ordering",
		ParentID:    strPtr("T6"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Verify parent has children
	children := h.GetChildTasks("T4")
	assert.Equal(t, 2, len(children))
	assert.Equal(t, "T5", children[0].ID)
	assert.Equal(t, "T6", children[1].ID)

	// Verify nested parent has children
	nestedChildren := h.GetChildTasks("T6")
	assert.Equal(t, 2, len(nestedChildren))
	assert.Equal(t, "T7", nestedChildren[0].ID)
	assert.Equal(t, "T8", nestedChildren[1].ID)

	// Verify child knows its parent
	task = h.GetTask("T5")
	assert.NotNil(t, task.ParentID)
	assert.Equal(t, "T4", *task.ParentID)

	// T5 should be next (T6 depends on it, and T6 has children so won't be returned)
	next = h.GetNextTask()
	assert.Equal(t, "T5", next.ID)

	// Complete T5
	_ = h.Process(&TaskCompleted{TaskID: "T5", Time: time.Now()})

	// Now T7 and T8 are available (T6 is skipped because it has children)
	next = h.GetNextTask()
	assert.Equal(t, "T7", next.ID) // First by sort order

	// Complete T7
	_ = h.Process(&TaskCompleted{TaskID: "T7", Time: time.Now()})

	// T8 is next
	next = h.GetNextTask()
	assert.Equal(t, "T8", next.ID)

	// Complete T8
	_ = h.Process(&TaskCompleted{TaskID: "T8", Time: time.Now()})

	// T6 should auto-complete (all children done)
	task = h.GetTask("T6")
	assert.Equal(t, "completed", task.Status)

	// T4 should also auto-complete (all children done)
	task = h.GetTask("T4")
	assert.Equal(t, "completed", task.Status)

	// All done
	next = h.GetNextTask()
	assert.Nil(t, next)
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

// TestTaskShouldNotCompleteIfItHasChildren tests that a task should not be manually completed
// if it has children - it should only auto-complete when all children are done
func TestTaskShouldNotCompleteIfItHasChildren(t *testing.T) {
	h := NewHearth()

	// Create a task
	err := h.Process(&TaskCreated{
		TaskID:      "PARENT",
		Title:       "Main task",
		Description: "Do the work",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Simulate what happens in run loop: task creates children during execution
	err = h.Process(&TaskCreated{
		TaskID:   "CHILD1",
		Title:    "Subtask 1",
		ParentID: strPtr("PARENT"),
		Time:     time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&TaskCreated{
		TaskID:   "CHILD2",
		Title:    "Subtask 2",
		ParentID: strPtr("PARENT"),
		Time:     time.Now(),
	})
	assert.NoError(t, err)

	// Now try to complete the parent - the validator should reject this
	err = h.Process(&TaskCompleted{
		TaskID: "PARENT",
		Time:   time.Now(),
	})

	// The event should be rejected by the validator
	assert.Error(t, err, "Should not be able to complete parent with incomplete children")
	assert.Equal(t, ErrEventRejected, err)

	// Parent should still be todo
	parent := h.GetTask("PARENT")
	assert.Equal(t, "todo", parent.Status, "Parent should remain todo when it has incomplete children")
}

// Helper function
func strPtr(s string) *string {
	return &s
}

// TestFindNextTask_DepthFirst tests the depth-first task selection algorithm
func TestFindNextTask_DepthFirst(t *testing.T) {
	baseTime := time.Now()

	root := &Task{ID: "T-root", Status: "todo", CreatedAt: baseTime}
	child1 := &Task{ID: "T-child1", Status: "todo", ParentID: strPtr("T-root"), CreatedAt: baseTime.Add(1 * time.Second)}
	child2 := &Task{ID: "T-child2", Status: "todo", ParentID: strPtr("T-root"), CreatedAt: baseTime.Add(2 * time.Second)}
	child3 := &Task{ID: "T-child3", Status: "todo", ParentID: strPtr("T-root"), CreatedAt: baseTime.Add(3 * time.Second)}
	gc1 := &Task{ID: "T-gc1", Status: "todo", ParentID: strPtr("T-child1"), CreatedAt: baseTime.Add(4 * time.Second)}
	gc2 := &Task{ID: "T-gc2", Status: "todo", ParentID: strPtr("T-child1"), CreatedAt: baseTime.Add(5 * time.Second)}

	tasks := []*Task{root, child1, child2, child3, gc1, gc2}

	// Should get T-gc1 first (depth-first into child1's subtree)
	next := findNextTask(tasks)
	assert.Equal(t, "T-gc1", next.ID)

	// Mark gc1 complete
	gc1.Status = "completed"

	// Should get T-gc2 (continue in child1's subtree)
	next = findNextTask(tasks)
	assert.Equal(t, "T-gc2", next.ID)

	// Mark gc2 complete
	gc2.Status = "completed"

	// Should get T-child2 (child1's subtree done, move to next sibling)
	next = findNextTask(tasks)
	assert.Equal(t, "T-child2", next.ID)

	// Mark child2 complete
	child2.Status = "completed"

	// Should get T-child3
	next = findNextTask(tasks)
	assert.Equal(t, "T-child3", next.ID)
}

