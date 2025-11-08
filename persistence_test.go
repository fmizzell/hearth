package hearth

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestEventPersistence tests that events are saved and loaded correctly
func TestEventPersistence(t *testing.T) {
	// Create temporary workspace
	tmpDir := t.TempDir()

	// Step 1: Create hearth, add some tasks, save events
	h1, err := NewHearth(tmpDir)
	assert.NoError(t, err)

	// Use mock caller to avoid real Claude API calls
	mockCaller := &MockClaudeCaller{}
	h1.Engine().RegisterService("claude_caller", mockCaller)

	err = h1.Process(&TaskCreated{
		TaskID:      "T1",
		Title:       "Analyze codebase",
		Description: "Run analysis",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h1.Process(&TaskCreated{
		TaskID:      "T2",
		Title:       "Fix bug",
		Description: "Fix type assertions",
		ParentID:    strPtr("T1"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h1.Process(&TaskCompleted{
		TaskID: "T2",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository
	eventsFile := filepath.Join(tmpDir, ".hearth", "events.json")

	// Verify .hearth directory was created
	hearthDir := filepath.Join(tmpDir, ".hearth")
	assert.DirExists(t, hearthDir)

	// Verify events.json exists
	assert.FileExists(t, eventsFile)

	// Step 2: Load new hearth instance from same directory
	h2, err := NewHearth(tmpDir)
	assert.NoError(t, err)
	h2.Engine().RegisterService("claude_caller", mockCaller)

	// Step 3: Verify state matches (events were replayed)
	tasks := h2.GetTasks()
	assert.Equal(t, 2, len(tasks))

	// Verify T1 auto-completed (all children done)
	t1 := h2.GetTask("T1")
	assert.NotNil(t, t1)
	assert.Equal(t, "Analyze codebase", t1.Title)
	assert.Equal(t, "completed", t1.Status)

	// Verify T2 was completed
	t2 := h2.GetTask("T2")
	assert.NotNil(t, t2)
	assert.Equal(t, "Fix bug", t2.Title)
	assert.Equal(t, "completed", t2.Status)
	assert.NotNil(t, t2.ParentID)
	assert.Equal(t, "T1", *t2.ParentID)

	// Step 4: Add more events with h2, verify h3 sees them
	err = h2.Process(&TaskCreated{
		TaskID: "T3",
		Title:  "Add tests",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Load h3
	h3, err := NewHearth(tmpDir)
	assert.NoError(t, err)
	h3.Engine().RegisterService("claude_caller", mockCaller)

	tasks = h3.GetTasks()
	assert.Equal(t, 3, len(tasks))

	t3 := h3.GetTask("T3")
	assert.NotNil(t, t3)
	assert.Equal(t, "Add tests", t3.Title)
}

// TestEventPersistence_EmptyWorkspace tests creating a new workspace with no existing events
func TestEventPersistence_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Load hearth from empty workspace
	h, err := NewHearth(tmpDir)
	assert.NoError(t, err)
	h.Engine().RegisterService("claude_caller", &MockClaudeCaller{})

	// Should have no tasks
	tasks := h.GetTasks()
	assert.Equal(t, 0, len(tasks))

	// Should be able to add tasks
	err = h.Process(&TaskCreated{
		TaskID:      "T1",
		Title:       "First task",
		Description: "Description",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	tasks = h.GetTasks()
	assert.Equal(t, 1, len(tasks))
}

// TestEventPersistence_EventMerging tests that concurrent instances merge events correctly
func TestEventPersistence_EventMerging(t *testing.T) {
	tmpDir := t.TempDir()
	mockCaller := &MockClaudeCaller{}

	// Instance 1: Create and save task T1
	h1, err := NewHearth(tmpDir)
	assert.NoError(t, err)
	h1.Engine().RegisterService("claude_caller", mockCaller)

	err = h1.Process(&TaskCreated{
		TaskID:      "T1",
		Title:       "Task from instance 1",
		Description: "First instance",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Instance 2: Load existing, add T2, save
	h2, err := NewHearth(tmpDir)
	assert.NoError(t, err)
	h2.Engine().RegisterService("claude_caller", mockCaller)

	err = h2.Process(&TaskCreated{
		TaskID:      "T2",
		Title:       "Task from instance 2",
		Description: "Second instance",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Instance 3: Load and verify both tasks are present
	h3, err := NewHearth(tmpDir)
	assert.NoError(t, err)
	h3.Engine().RegisterService("claude_caller", mockCaller)

	tasks := h3.GetTasks()
	assert.Equal(t, 2, len(tasks))

	t1 := h3.GetTask("T1")
	assert.NotNil(t, t1)
	assert.Equal(t, "Task from instance 1", t1.Title)

	t2 := h3.GetTask("T2")
	assert.NotNil(t, t2)
	assert.Equal(t, "Task from instance 2", t2.Title)
}

// TestEventPersistence_ComplexHierarchy tests persistence of complex parent-child relationships
func TestEventPersistence_ComplexHierarchy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a complex hierarchy with multiple levels
	h1, err := NewHearth(tmpDir)
	assert.NoError(t, err)

	// Use mock caller to avoid real Claude API calls
	mockCaller := &MockClaudeCaller{}
	h1.Engine().RegisterService("claude_caller", mockCaller)

	// Create root task
	err = h1.Process(&TaskCreated{
		TaskID:      "ROOT",
		Title:       "Root epic",
		Description: "Top level",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create child tasks
	err = h1.Process(&TaskCreated{
		TaskID:      "CHILD1",
		Title:       "Child 1",
		Description: "First child",
		ParentID:    strPtr("ROOT"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h1.Process(&TaskCreated{
		TaskID:      "CHILD2",
		Title:       "Child 2",
		Description: "Second child",
		ParentID:    strPtr("ROOT"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create grandchild
	err = h1.Process(&TaskCreated{
		TaskID:      "GRANDCHILD1",
		Title:       "Grandchild",
		Description: "Child of CHILD1",
		ParentID:    strPtr("CHILD1"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Complete grandchild
	err = h1.Process(&TaskCompleted{
		TaskID: "GRANDCHILD1",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Complete CHILD2
	err = h1.Process(&TaskCompleted{
		TaskID: "CHILD2",
		Time:   time.Now(),
	})
	assert.NoError(t, err)

	// Events auto-persist via FileRepository

	// Load new instance and verify all relationships preserved
	h2, err := NewHearth(tmpDir)
	assert.NoError(t, err)
	h2.Engine().RegisterService("claude_caller", mockCaller)

	// Verify hierarchy
	root := h2.GetTask("ROOT")
	assert.NotNil(t, root)
	assert.Equal(t, "completed", root.Status) // Should auto-complete

	child1 := h2.GetTask("CHILD1")
	assert.NotNil(t, child1)
	assert.Equal(t, "completed", child1.Status) // Should auto-complete
	assert.NotNil(t, child1.ParentID)
	assert.Equal(t, "ROOT", *child1.ParentID)

	child2 := h2.GetTask("CHILD2")
	assert.NotNil(t, child2)
	assert.Equal(t, "completed", child2.Status)
	assert.NotNil(t, child2.ParentID)
	assert.Equal(t, "ROOT", *child2.ParentID)

	grandchild1 := h2.GetTask("GRANDCHILD1")
	assert.NotNil(t, grandchild1)
	assert.Equal(t, "completed", grandchild1.Status)
	assert.NotNil(t, grandchild1.ParentID)
	assert.Equal(t, "CHILD1", *grandchild1.ParentID)
}
