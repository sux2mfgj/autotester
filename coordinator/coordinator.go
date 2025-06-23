package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"perf-runner/config"
	"perf-runner/runner"
	"perf-runner/ssh"
)

// Coordinator manages test execution across multiple hosts
type Coordinator struct {
	config    *config.TestConfig
	runners   map[string]runner.Runner
	sshClients map[string]*ssh.Client
	logger    *log.Logger
	mu        sync.RWMutex
	collectEnv bool
}

// NewCoordinator creates a new test coordinator
func NewCoordinator(cfg *config.TestConfig, logger *log.Logger) *Coordinator {
	if logger == nil {
		logger = log.Default()
	}
	
	return &Coordinator{
		config:     cfg,
		runners:    make(map[string]runner.Runner),
		sshClients: make(map[string]*ssh.Client),
		logger:     logger,
		collectEnv: false,
	}
}

// SetEnvironmentCollection enables or disables environment information collection
func (c *Coordinator) SetEnvironmentCollection(enabled bool) {
	c.collectEnv = enabled
}

// RegisterRunner registers a runner implementation
func (c *Coordinator) RegisterRunner(name string, r runner.Runner) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.runners[name] = r
}

// ConnectHosts establishes SSH connections to all configured hosts
func (c *Coordinator) ConnectHosts(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var wg sync.WaitGroup
	errCh := make(chan error, len(c.config.Hosts))
	
	for hostName, hostConfig := range c.config.Hosts {
		wg.Add(1)
		go func(name string, cfg *config.HostConfig) {
			defer wg.Done()
			
			client := ssh.NewClient(cfg.SSH)
			if err := client.Connect(ctx); err != nil {
				errCh <- fmt.Errorf("failed to connect to host %s: %w", name, err)
				return
			}
			
			c.sshClients[name] = client
			c.logger.Printf("Connected to host %s (%s)", name, cfg.SSH.Host)
		}(hostName, hostConfig)
	}
	
	wg.Wait()
	close(errCh)
	
	// Check for connection errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("connection errors: %v", errors)
	}
	
	return nil
}

// RunAllTests executes all configured test scenarios
func (c *Coordinator) RunAllTests(ctx context.Context) ([]*TestResult, error) {
	c.logger.Printf("Starting test execution for %d scenarios", len(c.config.Tests))
	
	var results []*TestResult
	for i, test := range c.config.Tests {
		c.logger.Printf("Running test %d/%d: %s", i+1, len(c.config.Tests), test.Name)
		
		repeat := test.Repeat
		if repeat <= 0 {
			repeat = 1
		}
		
		for j := 0; j < repeat; j++ {
			if repeat > 1 {
				c.logger.Printf("  Iteration %d/%d", j+1, repeat)
			}
			
			result, err := c.RunTest(ctx, &test)
			if err != nil {
				c.logger.Printf("Test %s failed: %v", test.Name, err)
				result = &TestResult{
					ScenarioName: test.Name,
					Success:      false,
					Error:        err.Error(),
					StartTime:    time.Now(),
					EndTime:      time.Now(),
				}
			}
			
			results = append(results, result)
			
			// Delay between iterations
			if j < repeat-1 && test.Delay > 0 {
				c.logger.Printf("  Waiting %v before next iteration", test.Delay)
				time.Sleep(test.Delay)
			}
		}
	}
	
	return results, nil
}

// RunTest executes a single test scenario
func (c *Coordinator) RunTest(ctx context.Context, test *config.TestScenario) (*TestResult, error) {
	executor := NewTestExecutor(c)
	return executor.ExecuteTest(ctx, test)
}


// Cleanup closes all SSH connections
func (c *Coordinator) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for hostName, client := range c.sshClients {
		if err := client.Close(); err != nil {
			c.logger.Printf("Error closing connection to host %s: %v", hostName, err)
		}
	}
	
	c.sshClients = make(map[string]*ssh.Client)
}