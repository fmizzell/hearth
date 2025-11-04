package hearth

import "github.com/cumulusrpg/atmos"

// Hearth is the main engine wrapper
type Hearth struct {
	projectID string
	engine    *atmos.Engine
}

// NewHearth creates a new Hearth instance
func NewHearth(projectID string) *Hearth {
	return &Hearth{
		projectID: projectID,
	}
}
