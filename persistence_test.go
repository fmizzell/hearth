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
	h1, err := NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

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

	// Save events to file
	eventsFile := filepath.Join(tmpDir, ".hearth", "events.json")
	err = h1.SaveToFile(eventsFile)
	assert.NoError(t, err)

	// Verify .hearth directory was created
	hearthDir := filepath.Join(tmpDir, ".hearth")
	assert.DirExists(t, hearthDir)

	// Verify events.json exists
	assert.FileExists(t, eventsFile)

	// Step 2: Load new hearth instance from same directory
	h2, err := NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

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

	// Save events
	err = h2.SaveToFile(eventsFile)
	assert.NoError(t, err)

	// Load h3
	h3, err := NewHearthWithPersistence(tmpDir)
	assert.NoError(t, err)

	tasks = h3.GetTasks()
	assert.Equal(t, 3, len(tasks))

	t3 := h3.GetTask("T3")
	assert.NotNil(t, t3)
	assert.Equal(t, "Add tests", t3.Title)
}
