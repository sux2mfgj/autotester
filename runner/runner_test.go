package runner

import (
	"fmt"
	"testing"
	"time"
)

func TestRegistry_Register(t *testing.T) {
	// Save original state
	originalRunners := make(map[string]func() Runner)
	globalRegistry.mu.Lock()
	for k, v := range globalRegistry.runners {
		originalRunners[k] = v
	}
	globalRegistry.mu.Unlock()

	// Clean registry for test
	globalRegistry.mu.Lock()
	globalRegistry.runners = make(map[string]func() Runner)
	globalRegistry.mu.Unlock()

	// Restore original state after test
	defer func() {
		globalRegistry.mu.Lock()
		globalRegistry.runners = originalRunners
		globalRegistry.mu.Unlock()
	}()

	// Test registration
	testRunner := &TestRunner{name: "test_runner"}
	Register("test_runner", func() Runner {
		return testRunner
	})

	// Verify registration
	registered := GetRegistered()
	found := false
	for _, name := range registered {
		if name == "test_runner" {
			found = true
			break
		}
	}

	if !found {
		t.Error("test_runner should be registered")
	}

	// Test creation
	instance, err := Create("test_runner")
	if err != nil {
		t.Fatalf("Failed to create test_runner: %v", err)
	}

	if instance.Name() != "test_runner" {
		t.Errorf("Expected name 'test_runner', got %q", instance.Name())
	}
}

func TestRegistry_Create(t *testing.T) {
	tests := []struct {
		name          string
		runnerName    string
		shouldSucceed bool
	}{
		{
			name:          "existing runner",
			runnerName:    "ib_send_bw", // This should be auto-registered
			shouldSucceed: true,
		},
		{
			name:          "non-existent runner",
			runnerName:    "non_existent_runner",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := Create(tt.runnerName)

			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if instance == nil {
					t.Error("Expected runner instance but got nil")
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if instance != nil {
					t.Error("Expected nil instance but got runner")
				}
			}
		})
	}
}

func TestRegistry_GetRegistered(t *testing.T) {
	registered := GetRegistered()

	// Should at least contain ib_send_bw (auto-registered)
	found := false
	for _, name := range registered {
		if name == "ib_send_bw" {
			found = true
			break
		}
	}

	if !found {
		t.Error("ib_send_bw should be auto-registered")
	}

	// Should return slice, not nil
	if registered == nil {
		t.Error("GetRegistered should not return nil")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to registry
	done := make(chan bool, 3)

	// Goroutine 1: Register runners
	go func() {
		for i := 0; i < 10; i++ {
			Register("concurrent_test_1", func() Runner {
				return &TestRunner{name: "concurrent_test_1"}
			})
		}
		done <- true
	}()

	// Goroutine 2: Get registered runners
	go func() {
		for i := 0; i < 10; i++ {
			GetRegistered()
		}
		done <- true
	}()

	// Goroutine 3: Create runners
	go func() {
		for i := 0; i < 10; i++ {
			Create("ib_send_bw")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should still work after concurrent access
	registered := GetRegistered()
	if len(registered) == 0 {
		t.Error("Registry should not be empty after concurrent access")
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "valid config with all fields",
			config: Config{
				Duration:   30 * time.Second,
				Args:       map[string]interface{}{"size": 65536},
				Env:        map[string]string{"TEST": "value"},
				Role:       "client",
				Host:       "192.168.1.100",
				TargetHost: "10.0.0.100",
				Port:       18515,
			},
			valid: true,
		},
		{
			name: "config with zero values",
			config: Config{
				Duration:   0,
				Args:       nil,
				Env:        nil,
				Role:       "",
				Host:       "",
				TargetHost: "",
				Port:       0,
			},
			valid: true, // Config struct itself should be valid
		},
		{
			name: "config with empty maps",
			config: Config{
				Args: make(map[string]interface{}),
				Env:  make(map[string]string),
				Role: "server",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that Config struct can be created and used
			// The actual validation logic is runner-specific
			if tt.config.Role != "" {
				// Basic sanity check
				if len(tt.config.Role) == 0 {
					t.Error("Role should not be empty after assignment")
				}
			}

			if tt.config.Args != nil {
				// Test Args map functionality
				tt.config.Args["test_key"] = "test_value"
				if val, exists := tt.config.Args["test_key"]; !exists || val != "test_value" {
					t.Error("Args map should be functional")
				}
			}

			if tt.config.Env != nil {
				// Test Env map functionality
				tt.config.Env["TEST_ENV"] = "test_value"
				if val, exists := tt.config.Env["TEST_ENV"]; !exists || val != "test_value" {
					t.Error("Env map should be functional")
				}
			}
		})
	}
}

func TestResult_Structure(t *testing.T) {
	result := &Result{
		Success:   true,
		Output:    "test output",
		Error:     "test error",
		ExitCode:  0,
		Duration:  time.Second,
		Metrics:   make(map[string]interface{}),
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Second),
	}

	// Test all fields are accessible
	if !result.Success {
		t.Error("Success field should be accessible")
	}

	if result.Output != "test output" {
		t.Error("Output field should be accessible")
	}

	if result.Error != "test error" {
		t.Error("Error field should be accessible")
	}

	if result.ExitCode != 0 {
		t.Error("ExitCode field should be accessible")
	}

	if result.Duration != time.Second {
		t.Error("Duration field should be accessible")
	}

	if result.Metrics == nil {
		t.Error("Metrics field should be accessible")
	}

	// Test Metrics map functionality
	result.Metrics["test_metric"] = 123.45
	if val, exists := result.Metrics["test_metric"]; !exists || val != 123.45 {
		t.Error("Metrics map should be functional")
	}

	// Test time fields
	if result.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	if result.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}

	if result.EndTime.Before(result.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}

// TestRunner is a mock runner for testing
type TestRunner struct {
	name string
}

func (r *TestRunner) Validate(config Config) error {
	return nil
}

func (r *TestRunner) Name() string {
	return r.name
}

func (r *TestRunner) SupportsRole(role string) bool {
	return role == "client" || role == "server"
}

func (r *TestRunner) BuildCommand(config Config) string {
	return "test_command"
}

func (r *TestRunner) ParseMetrics(result *Result) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}
	if result.Metrics == nil {
		result.Metrics = make(map[string]interface{})
	}
	result.Metrics["test_metric"] = "test_value"
	return nil
}

func (r *TestRunner) SetExecutablePath(path string) {
	// TestRunner doesn't need to do anything with the path
}

func TestRunner_Interface(t *testing.T) {
	// Test that TestRunner implements Runner interface
	var runner Runner = &TestRunner{name: "test"}

	// Test all interface methods
	if err := runner.Validate(Config{}); err != nil {
		t.Errorf("Validate should not return error: %v", err)
	}

	if name := runner.Name(); name != "test" {
		t.Errorf("Expected name 'test', got %q", name)
	}

	if !runner.SupportsRole("client") {
		t.Error("Should support client role")
	}

	if !runner.SupportsRole("server") {
		t.Error("Should support server role")
	}

	if !runner.SupportsRole("invalid") {
		// This is expected for TestRunner
	}

	if cmd := runner.BuildCommand(Config{}); cmd != "test_command" {
		t.Errorf("Expected 'test_command', got %q", cmd)
	}

	result := &Result{Metrics: make(map[string]interface{})}
	if err := runner.ParseMetrics(result); err != nil {
		t.Errorf("ParseMetrics should not return error: %v", err)
	}
	if val, exists := result.Metrics["test_metric"]; !exists || val != "test_value" {
		t.Error("ParseMetrics should set test_metric")
	}
}

func TestIbSendBwRunner_AutoRegistration(t *testing.T) {
	// Test that ib_send_bw is automatically registered on import
	registered := GetRegistered()
	
	found := false
	for _, name := range registered {
		if name == "ib_send_bw" {
			found = true
			break
		}
	}

	if !found {
		t.Error("ib_send_bw should be auto-registered via init()")
	}

	// Test that we can create it
	instance, err := Create("ib_send_bw")
	if err != nil {
		t.Fatalf("Should be able to create ib_send_bw: %v", err)
	}

	// Test that it's the correct type
	if instance.Name() != "ib_send_bw" {
		t.Errorf("Expected name 'ib_send_bw', got %q", instance.Name())
	}

	// Test that it implements all interface methods
	var runner Runner = instance
	_ = runner.Validate(Config{Role: "server"})
	_ = runner.SupportsRole("client")
	_ = runner.BuildCommand(Config{Role: "server"})
	result := &Result{Metrics: make(map[string]interface{})}
	if err := runner.ParseMetrics(result); err != nil {
		t.Errorf("ParseMetrics should not return error: %v", err)
	}
}