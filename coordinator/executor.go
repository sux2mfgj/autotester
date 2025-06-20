package coordinator

import (
	"context"
	"fmt"
	"time"

	"tester/config"
	"tester/runner"
	"tester/ssh"
)

// TestExecutor handles execution of individual test scenarios
type TestExecutor struct {
	coordinator *Coordinator
}

// NewTestExecutor creates a new test executor
func NewTestExecutor(coord *Coordinator) *TestExecutor {
	return &TestExecutor{coordinator: coord}
}

// ExecuteTest runs a single test scenario
func (e *TestExecutor) ExecuteTest(ctx context.Context, test *config.TestScenario) (*TestResult, error) {
	startTime := time.Now()
	
	result := &TestResult{
		ScenarioName: test.Name,
		StartTime:    startTime,
	}
	
	// Get runner
	r, exists := e.coordinator.runners[e.coordinator.config.Runner]
	if !exists {
		return nil, fmt.Errorf("runner %s not found", e.coordinator.config.Runner)
	}
	
	// Get host configurations
	clientHost := e.coordinator.config.GetClientHost(test)
	serverHost := e.coordinator.config.GetServerHost(test)
	
	if clientHost == nil {
		return nil, fmt.Errorf("client host %s not found", test.Client)
	}
	if serverHost == nil {
		return nil, fmt.Errorf("server host %s not found", test.Server)
	}
	
	// Get SSH clients
	clientSSH := e.coordinator.sshClients[test.Client]
	serverSSH := e.coordinator.sshClients[test.Server]
	
	if clientSSH == nil {
		return nil, fmt.Errorf("SSH client for host %s not connected", test.Client)
	}
	if serverSSH == nil {
		return nil, fmt.Errorf("SSH client for host %s not connected", test.Server)
	}
	
	// Prepare runner configurations
	serverConfig := e.coordinator.config.MergeRunnerConfig(serverHost.Runner, test.Config)
	serverConfig.Role = "server"
	
	clientConfig := e.coordinator.config.MergeRunnerConfig(clientHost.Runner, test.Config)
	clientConfig.Role = "client"
	clientConfig.Host = serverHost.SSH.Host // Client connects to server (SSH host)
	
	// If no specific target host is configured, use the server's SSH host
	// This allows for separate SSH host and InfiniBand target IPs
	if clientConfig.TargetHost == "" {
		clientConfig.TargetHost = serverHost.SSH.Host
	}
	
	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, e.coordinator.config.Timeout)
	defer cancel()
	
	// Execute the test
	if err := e.executeClientServerTest(testCtx, r, clientSSH, serverSSH, clientConfig, serverConfig, result, test); err != nil {
		return nil, err
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = result.ClientResult != nil && result.ClientResult.Success && 
		(result.ServerResult == nil || result.ServerResult.Success) &&
		result.Error == ""
	
	return result, nil
}

// executeClientServerTest handles the coordination between client and server
func (e *TestExecutor) executeClientServerTest(
	ctx context.Context,
	r runner.Runner,
	clientSSH, serverSSH *ssh.Client,
	clientConfig, serverConfig *runner.Config,
	result *TestResult,
	test *config.TestScenario,
) error {
	// Build commands for display using runner's own method
	result.ServerCommand = r.BuildCommand(*serverConfig)
	result.ClientCommand = r.BuildCommand(*clientConfig)
	
	// Start server first
	e.coordinator.logger.Printf("  Starting server on %s", test.Server)
	serverDone := make(chan *runner.Result, 1)
	serverErr := make(chan error, 1)
	
	go func() {
		serverResult, err := e.runRemoteCommand(ctx, serverSSH, r, serverConfig)
		if err != nil {
			serverErr <- err
			return
		}
		serverDone <- serverResult
	}()
	
	// Wait a bit for server to start
	time.Sleep(2 * time.Second)
	
	// Start client
	e.coordinator.logger.Printf("  Starting client on %s", test.Client)
	clientResult, err := e.runRemoteCommand(ctx, clientSSH, r, clientConfig)
	if err != nil {
		return fmt.Errorf("client execution failed: %w", err)
	}
	
	result.ClientResult = clientResult
	
	// Wait for server to complete or timeout
	select {
	case serverResult := <-serverDone:
		result.ServerResult = serverResult
	case err := <-serverErr:
		result.Error = fmt.Sprintf("server execution failed: %v", err)
	case <-ctx.Done():
		result.Error = "test timed out"
	}
	
	return nil
}

// runRemoteCommand executes a runner command on a remote host via SSH
func (e *TestExecutor) runRemoteCommand(ctx context.Context, sshClient *ssh.Client, r runner.Runner, config *runner.Config) (*runner.Result, error) {
	// Validate configuration
	if err := r.Validate(*config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Build command for remote execution using runner's own method
	command := r.BuildCommand(*config)
	
	// Display command before execution
	e.coordinator.logger.Printf("  Executing command on %s: %s", config.Role, command)
	
	// Execute command via SSH
	sshResult, err := sshClient.ExecuteCommand(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("SSH command execution failed: %w", err)
	}
	
	// Convert SSH result to runner result
	runnerResult := &runner.Result{
		Success:   sshResult.ExitCode == 0,
		Output:    sshResult.Output,
		Error:     sshResult.Error,
		ExitCode:  sshResult.ExitCode,
		StartTime: time.Now(), // Approximate
		EndTime:   time.Now(), // Approximate
		Metrics:   make(map[string]interface{}),
	}
	
	// Parse metrics from command output
	if err := r.ParseMetrics(runnerResult); err != nil {
		e.coordinator.logger.Printf("  Warning: failed to parse metrics: %v", err)
		// Continue execution - metrics parsing failure shouldn't fail the test
	}
	
	return runnerResult, nil
}