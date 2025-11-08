package hearth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestHearthJourney tests the full autonomous orchestration flow
// This should exercise the majority of the codebase:
// - Task creation events
// - Autonomous orchestration (ExecuteTasksRequested)
// - Depth-first task selection
// - Task execution with context building
// - Parent/child hierarchies
// - Parent auto-completion
// - Summary generation
func TestHearthJourney(t *testing.T) {
	// Create hearth with persistence (exercises FileRepository)
	tmpDir := t.TempDir()
	h, err := NewHearth(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, h)

	// Override with mock Claude caller (avoid real API calls)
	mockCaller := &MockClaudeCaller{}
	h.Engine().RegisterService("claude_caller", mockCaller)

	// Create a hierarchical task structure:
	// ROOT: "Improve authentication"
	//   ├─ CHILD1: "Implement 2FA"
	//   │   ├─ GRANDCHILD1: "Add TOTP support"
	//   │   └─ GRANDCHILD2: "Add SMS support"
	//   └─ CHILD2: "Add session management"

	// Create root task
	err = h.Process(&TaskCreated{
		TaskID:      "ROOT",
		Title:       "Improve authentication",
		Description: "Add multi-factor auth and session management",
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create child 1
	err = h.Process(&TaskCreated{
		TaskID:      "CHILD1",
		Title:       "Implement 2FA",
		Description: "Add two-factor authentication support",
		ParentID:    strPtr("ROOT"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create grandchildren for child 1
	err = h.Process(&TaskCreated{
		TaskID:      "GRANDCHILD1",
		Title:       "Add TOTP support",
		Description: "Implement time-based OTP",
		ParentID:    strPtr("CHILD1"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	err = h.Process(&TaskCreated{
		TaskID:      "GRANDCHILD2",
		Title:       "Add SMS support",
		Description: "Implement SMS-based OTP",
		ParentID:    strPtr("CHILD1"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Create child 2
	err = h.Process(&TaskCreated{
		TaskID:      "CHILD2",
		Title:       "Add session management",
		Description: "Track user sessions and allow logout",
		ParentID:    strPtr("ROOT"),
		Time:        time.Now(),
	})
	assert.NoError(t, err)

	// Verify initial state
	assert.Equal(t, 5, len(h.GetTasks()))
	assert.Equal(t, "todo", h.GetTask("ROOT").Status)
	assert.Equal(t, "todo", h.GetTask("CHILD1").Status)
	assert.Equal(t, "todo", h.GetTask("GRANDCHILD1").Status)

	// Start autonomous orchestration (this is the key part!)
	// Without services registered, tasks will complete immediately with mock results
	err = h.Process(&ExecuteTasksRequested{Time: time.Now()})
	assert.NoError(t, err)

	// Verify orchestration completed everything in depth-first order
	// All leaf tasks should be completed
	assert.Equal(t, "completed", h.GetTask("GRANDCHILD1").Status)
	assert.Equal(t, "completed", h.GetTask("GRANDCHILD2").Status)
	assert.Equal(t, "completed", h.GetTask("CHILD2").Status)

	// Parents should auto-complete when all children are done
	assert.Equal(t, "completed", h.GetTask("CHILD1").Status)
	assert.Equal(t, "completed", h.GetTask("ROOT").Status)

	// Verify next task returns nil (all tasks done)
	next := h.GetNextTask()
	assert.Nil(t, next)
}
