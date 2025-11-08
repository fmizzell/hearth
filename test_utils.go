package hearth

import "fmt"

// Test utilities - shared helpers for tests

func strPtr(s string) *string {
	return &s
}

// MockClaudeCaller is a test double that doesn't call real Claude
type MockClaudeCaller struct {
	CallCount int
	Responses map[string]string // taskID -> response
}

func (m *MockClaudeCaller) Call(prompt, workDir string) (string, error) {
	m.CallCount++
	// Just return a mock response
	return fmt.Sprintf("Mock Claude response (call #%d)", m.CallCount), nil
}
