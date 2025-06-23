package coordinator

import (
	"context"
	"fmt"
	"time"

	"perf-runner/config"
	"perf-runner/envinfo"
	"perf-runner/runner"
	"perf-runner/ssh"
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
	
	// Get role-specific runners
	clientRunnerName := e.coordinator.config.GetRunnerForRole("client")
	serverRunnerName := e.coordinator.config.GetRunnerForRole("server")
	
	clientRunner, exists := e.coordinator.runners[clientRunnerName]
	if !exists {
		return nil, fmt.Errorf("client runner %s not found", clientRunnerName)
	}
	
	serverRunner, exists := e.coordinator.runners[serverRunnerName]
	if !exists {
		return nil, fmt.Errorf("server runner %s not found", serverRunnerName)
	}
	
	var intermediateRunner runner.Runner
	if e.coordinator.config.HasIntermediateNode(test) {
		intermediateRunnerName := e.coordinator.config.GetRunnerForRole("intermediate")
		var exists bool
		intermediateRunner, exists = e.coordinator.runners[intermediateRunnerName]
		if !exists {
			return nil, fmt.Errorf("intermediate runner %s not found", intermediateRunnerName)
		}
	}
	
	// Get host configurations
	clientHost := e.coordinator.config.GetClientHost(test)
	serverHost := e.coordinator.config.GetServerHost(test)
	intermediateHost := e.coordinator.config.GetIntermediateHost(test)
	
	if clientHost == nil {
		return nil, fmt.Errorf("client host %s not found", test.Client)
	}
	if serverHost == nil {
		return nil, fmt.Errorf("server host %s not found", test.Server)
	}
	
	// Get SSH clients
	clientSSH := e.coordinator.sshClients[test.Client]
	serverSSH := e.coordinator.sshClients[test.Server]
	var intermediateSSH *ssh.Client
	
	if clientSSH == nil {
		return nil, fmt.Errorf("SSH client for host %s not connected", test.Client)
	}
	if serverSSH == nil {
		return nil, fmt.Errorf("SSH client for host %s not connected", test.Server)
	}
	
	// Check intermediate node if specified
	if e.coordinator.config.HasIntermediateNode(test) {
		if intermediateHost == nil {
			return nil, fmt.Errorf("intermediate host %s not found", test.Intermediate)
		}
		intermediateSSH = e.coordinator.sshClients[test.Intermediate]
		if intermediateSSH == nil {
			return nil, fmt.Errorf("SSH client for intermediate host %s not connected", test.Intermediate)
		}
	}
	
	// Prepare runner configurations
	serverConfig := e.coordinator.config.MergeRunnerConfig(serverHost.Runner, test.Config)
	serverConfig.Role = "server"
	
	clientConfig := e.coordinator.config.MergeRunnerConfig(clientHost.Runner, test.Config)
	clientConfig.Role = "client"
	
	var intermediateConfig *runner.Config
	
	// Configure connection topology based on intermediate node presence
	if e.coordinator.config.HasIntermediateNode(test) {
		// 3-node topology: Client → Intermediate → Server
		intermediateConfig = e.coordinator.config.MergeRunnerConfig(intermediateHost.Runner, test.Config)
		intermediateConfig.Role = "intermediate"
		
		// Intermediate connects to server
		intermediateConfig.Host = serverHost.SSH.Host
		if intermediateConfig.TargetHost == "" {
			intermediateConfig.TargetHost = serverHost.SSH.Host
		}
		
		// Client connects to intermediate
		clientConfig.Host = intermediateHost.SSH.Host
		if clientConfig.TargetHost == "" {
			clientConfig.TargetHost = intermediateHost.SSH.Host
		}
	} else {
		// 2-node topology: Client → Server (original behavior)
		clientConfig.Host = serverHost.SSH.Host
		if clientConfig.TargetHost == "" {
			clientConfig.TargetHost = serverHost.SSH.Host
		}
	}
	
	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, e.coordinator.config.Timeout)
	defer cancel()
	
	// Execute the test based on topology
	if e.coordinator.config.HasIntermediateNode(test) {
		// 3-node topology
		if err := e.executeThreeNodeTest(testCtx, clientRunner, intermediateRunner, serverRunner, clientSSH, intermediateSSH, serverSSH, clientConfig, intermediateConfig, serverConfig, result, test); err != nil {
			return nil, err
		}
	} else {
		// 2-node topology (original)
		if err := e.executeClientServerTest(testCtx, clientRunner, serverRunner, clientSSH, serverSSH, clientConfig, serverConfig, result, test); err != nil {
			return nil, err
		}
	}
	
	// Collect environment information if requested
	if e.coordinator.collectEnv {
		if err := e.collectEnvironmentInfo(testCtx, result, test, clientSSH, serverSSH, intermediateSSH); err != nil {
			e.coordinator.logger.Printf("Warning: failed to collect environment info: %v", err)
		}
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = result.ClientResult != nil && result.ClientResult.Success && 
		(result.ServerResult == nil || result.ServerResult.Success) &&
		(result.IntermediateResult == nil || result.IntermediateResult.Success) &&
		result.Error == ""
	
	return result, nil
}

// executeClientServerTest handles the coordination between client and server
func (e *TestExecutor) executeClientServerTest(
	ctx context.Context,
	clientRunner, serverRunner runner.Runner,
	clientSSH, serverSSH *ssh.Client,
	clientConfig, serverConfig *runner.Config,
	result *TestResult,
	test *config.TestScenario,
) error {
	// Build commands for display using role-specific runners
	result.ServerCommand = serverRunner.BuildCommand(*serverConfig)
	result.ClientCommand = clientRunner.BuildCommand(*clientConfig)
	
	// Start server first
	e.coordinator.logger.Printf("  Starting server on %s", test.Server)
	serverDone := make(chan *runner.Result, 1)
	serverErr := make(chan error, 1)
	
	go func() {
		serverResult, err := e.runRemoteCommand(ctx, serverSSH, serverRunner, serverConfig)
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
	clientResult, err := e.runRemoteCommand(ctx, clientSSH, clientRunner, clientConfig)
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

// executeThreeNodeTest handles the coordination between client, intermediate, and server
func (e *TestExecutor) executeThreeNodeTest(
	ctx context.Context,
	clientRunner, intermediateRunner, serverRunner runner.Runner,
	clientSSH, intermediateSSH, serverSSH *ssh.Client,
	clientConfig, intermediateConfig, serverConfig *runner.Config,
	result *TestResult,
	test *config.TestScenario,
) error {
	// Build commands for display using role-specific runners
	result.ServerCommand = serverRunner.BuildCommand(*serverConfig)
	result.ClientCommand = clientRunner.BuildCommand(*clientConfig)
	result.IntermediateCommand = intermediateRunner.BuildCommand(*intermediateConfig)
	
	// Start server first
	e.coordinator.logger.Printf("  Starting server on %s", test.Server)
	serverDone := make(chan *runner.Result, 1)
	serverErr := make(chan error, 1)
	
	go func() {
		serverResult, err := e.runRemoteCommand(ctx, serverSSH, serverRunner, serverConfig)
		if err != nil {
			serverErr <- err
			return
		}
		serverDone <- serverResult
	}()
	
	// Wait for server to start
	time.Sleep(2 * time.Second)
	
	// Start intermediate node
	e.coordinator.logger.Printf("  Starting intermediate node on %s", test.Intermediate)
	intermediateDone := make(chan *runner.Result, 1)
	intermediateErr := make(chan error, 1)
	
	go func() {
		intermediateResult, err := e.runRemoteCommand(ctx, intermediateSSH, intermediateRunner, intermediateConfig)
		if err != nil {
			intermediateErr <- err
			return
		}
		intermediateDone <- intermediateResult
	}()
	
	// Wait for intermediate to establish connection to server
	time.Sleep(2 * time.Second)
	
	// Start client (connects to intermediate)
	e.coordinator.logger.Printf("  Starting client on %s", test.Client)
	clientResult, err := e.runRemoteCommand(ctx, clientSSH, clientRunner, clientConfig)
	if err != nil {
		return fmt.Errorf("client execution failed: %w", err)
	}
	
	result.ClientResult = clientResult
	
	// Wait for intermediate and server to complete or timeout
	select {
	case serverResult := <-serverDone:
		result.ServerResult = serverResult
	case err := <-serverErr:
		result.Error = fmt.Sprintf("server execution failed: %v", err)
	case <-ctx.Done():
		result.Error = "test timed out"
	}
	
	// Collect intermediate result
	select {
	case intermediateResult := <-intermediateDone:
		result.IntermediateResult = intermediateResult
	case err := <-intermediateErr:
		if result.Error == "" {
			result.Error = fmt.Sprintf("intermediate execution failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		// Give intermediate a bit more time to clean up
		e.coordinator.logger.Printf("  Warning: intermediate node did not complete within timeout")
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

// collectEnvironmentInfo gathers environment information from all hosts
func (e *TestExecutor) collectEnvironmentInfo(ctx context.Context, result *TestResult, test *config.TestScenario, clientSSH, serverSSH, intermediateSSH *ssh.Client) error {
	e.coordinator.logger.Printf("  Collecting environment information...")
	
	result.EnvironmentInfo = &EnvironmentData{}
	
	// Collect client environment
	if clientSSH != nil {
		collector := envinfo.NewCollector(clientSSH)
		if envInfo, err := collector.Collect(ctx); err != nil {
			e.coordinator.logger.Printf("  Warning: failed to collect client environment: %v", err)
		} else {
			result.EnvironmentInfo.ClientEnv = envInfo
			e.coordinator.logger.Printf("  Collected client environment from %s", test.Client)
		}
	}
	
	// Collect server environment
	if serverSSH != nil {
		collector := envinfo.NewCollector(serverSSH)
		if envInfo, err := collector.Collect(ctx); err != nil {
			e.coordinator.logger.Printf("  Warning: failed to collect server environment: %v", err)
		} else {
			result.EnvironmentInfo.ServerEnv = envInfo
			e.coordinator.logger.Printf("  Collected server environment from %s", test.Server)
		}
	}
	
	// Collect intermediate environment if applicable
	if intermediateSSH != nil {
		collector := envinfo.NewCollector(intermediateSSH)
		if envInfo, err := collector.Collect(ctx); err != nil {
			e.coordinator.logger.Printf("  Warning: failed to collect intermediate environment: %v", err)
		} else {
			result.EnvironmentInfo.IntermediateEnv = envInfo
			e.coordinator.logger.Printf("  Collected intermediate environment from %s", test.Intermediate)
		}
	}
	
	return nil
}