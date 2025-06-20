package coordinator

import (
	"time"

	"tester/runner"
)

// TestResult represents the result of a complete test scenario
type TestResult struct {
	ScenarioName string           `json:"scenario_name"`
	Success      bool             `json:"success"`
	StartTime    time.Time        `json:"start_time"`
	EndTime      time.Time        `json:"end_time"`
	Duration     time.Duration    `json:"duration"`
	ClientResult *runner.Result   `json:"client_result,omitempty"`
	ServerResult *runner.Result   `json:"server_result,omitempty"`
	Error        string           `json:"error,omitempty"`
}