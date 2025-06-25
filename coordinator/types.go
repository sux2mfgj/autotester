package coordinator

import (
	"time"

	"perf-runner/envinfo"
	"perf-runner/runner"
)

// TestResult represents the result of a complete test scenario
type TestResult struct {
	ScenarioName         string           `json:"scenario_name"`
	Success              bool             `json:"success"`
	StartTime            time.Time        `json:"start_time"`
	EndTime              time.Time        `json:"end_time"`
	Duration             time.Duration    `json:"duration"`
	ClientResult         *runner.Result   `json:"client_result,omitempty"`
	ServerResult         *runner.Result   `json:"server_result,omitempty"`
	IntermediateResult   *runner.Result   `json:"intermediate_result,omitempty"`   // 3-node topology
	Intermediate1Result  *runner.Result   `json:"intermediate1_result,omitempty"`  // 4-node topology
	Intermediate2Result  *runner.Result   `json:"intermediate2_result,omitempty"`  // 4-node topology
	ClientCommand        string           `json:"client_command,omitempty"`
	ServerCommand        string           `json:"server_command,omitempty"`
	IntermediateCommand  string           `json:"intermediate_command,omitempty"`  // 3-node topology
	Intermediate1Command string           `json:"intermediate1_command,omitempty"` // 4-node topology
	Intermediate2Command string           `json:"intermediate2_command,omitempty"` // 4-node topology
	Error                string           `json:"error,omitempty"`
	EnvironmentInfo      *EnvironmentData `json:"environment_info,omitempty"`
}

// EnvironmentData contains environment information for all hosts in the test
type EnvironmentData struct {
	ClientEnv        *envinfo.EnvironmentInfo `json:"client,omitempty"`
	ServerEnv        *envinfo.EnvironmentInfo `json:"server,omitempty"`
	IntermediateEnv  *envinfo.EnvironmentInfo `json:"intermediate,omitempty"`  // 3-node topology
	Intermediate1Env *envinfo.EnvironmentInfo `json:"intermediate1,omitempty"` // 4-node topology
	Intermediate2Env *envinfo.EnvironmentInfo `json:"intermediate2,omitempty"` // 4-node topology
}