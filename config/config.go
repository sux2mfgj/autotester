package config

import (
	"fmt"
	"os"
	"time"

	"perf-runner/runner"
	"perf-runner/ssh"

	"gopkg.in/yaml.v3"
)

// TestConfig represents the overall test configuration
type TestConfig struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description,omitempty"`
	
	// Runner configuration - supports both single runner and per-role runners
	Runner      string              `yaml:"runner,omitempty"`      // Single runner for all roles (legacy)
	Runners     *RoleRunners        `yaml:"runners,omitempty"`     // Per-role runner specification
	
	Timeout     time.Duration       `yaml:"timeout"`
	
	// Environment information collection
	CollectEnv  bool                `yaml:"collect_env,omitempty"`
	
	// Binary path configurations
	BinaryPaths map[string]string   `yaml:"binary_paths,omitempty"`
	
	// Host configurations
	Hosts       map[string]*HostConfig `yaml:"hosts"`
	
	// Test scenarios
	Tests       []TestScenario         `yaml:"tests"`
}

// RoleRunners allows specifying different runners per role
type RoleRunners struct {
	Client       string `yaml:"client"`                // Runner for client role
	Server       string `yaml:"server"`                // Runner for server role  
	Intermediate string `yaml:"intermediate,omitempty"` // Runner for intermediate role
}

// HostConfig represents configuration for a single host
type HostConfig struct {
	SSH      *ssh.Config       `yaml:"ssh"`
	Role     string            `yaml:"role"` // "client" or "server"
	Runner   *runner.Config    `yaml:"runner"`
}

// TestScenario represents a single test scenario
type TestScenario struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description,omitempty"`
	Client       string            `yaml:"client"` // Host name for client
	Server       string            `yaml:"server"` // Host name for server
	Intermediate string            `yaml:"intermediate,omitempty"` // Host name for intermediate node (optional, 3-node)
	Intermediate1 string           `yaml:"intermediate1,omitempty"` // Host name for first intermediate node (4-node)
	Intermediate2 string           `yaml:"intermediate2,omitempty"` // Host name for second intermediate node (4-node)
	Config       *runner.Config    `yaml:"config"`
	
	// Test-specific settings
	Repeat      int               `yaml:"repeat,omitempty"`
	Delay       time.Duration     `yaml:"delay,omitempty"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*TestConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}
	
	var config TestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}
	
	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}
	
	// Validate configuration
	validator := NewValidator()
	if err := validator.ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &config, nil
}

// GetRunnerForRole returns the appropriate runner name for the given role
func (c *TestConfig) GetRunnerForRole(role string) string {
	// If per-role runners are specified, use those
	if c.Runners != nil {
		switch role {
		case "client":
			if c.Runners.Client != "" {
				return c.Runners.Client
			}
		case "server":
			if c.Runners.Server != "" {
				return c.Runners.Server
			}
		case "intermediate":
			if c.Runners.Intermediate != "" {
				return c.Runners.Intermediate
			}
		}
	}
	
	// Fall back to the single runner configuration
	return c.Runner
}

// HasMixedRunners returns true if different runners are specified for different roles
func (c *TestConfig) HasMixedRunners() bool {
	if c.Runners == nil {
		return false
	}
	
	runners := []string{c.Runners.Client, c.Runners.Server, c.Runners.Intermediate}
	unique := make(map[string]bool)
	
	for _, runner := range runners {
		if runner != "" {
			unique[runner] = true
		}
	}
	
	return len(unique) > 1
}

// GetClientHost returns the client host configuration for a test
func (c *TestConfig) GetClientHost(test *TestScenario) *HostConfig {
	return c.Hosts[test.Client]
}

// GetServerHost returns the server host configuration for a test
func (c *TestConfig) GetServerHost(test *TestScenario) *HostConfig {
	return c.Hosts[test.Server]
}

// GetIntermediateHost returns the intermediate host configuration for a test
func (c *TestConfig) GetIntermediateHost(test *TestScenario) *HostConfig {
	if test.Intermediate == "" {
		return nil
	}
	return c.Hosts[test.Intermediate]
}

// HasIntermediateNode returns true if the test scenario includes an intermediate node
func (c *TestConfig) HasIntermediateNode(test *TestScenario) bool {
	return test.Intermediate != ""
}

// HasFourNodeTopology returns true if the test scenario includes two intermediate nodes
func (c *TestConfig) HasFourNodeTopology(test *TestScenario) bool {
	return test.Intermediate1 != "" && test.Intermediate2 != ""
}

// GetTopologyType returns the topology type for a test scenario
func (c *TestConfig) GetTopologyType(test *TestScenario) string {
	if c.HasFourNodeTopology(test) {
		return "4-node"
	} else if c.HasIntermediateNode(test) {
		return "3-node"
	}
	return "2-node"
}

// GetIntermediate1Host returns the first intermediate host configuration for a test
func (c *TestConfig) GetIntermediate1Host(test *TestScenario) *HostConfig {
	if test.Intermediate1 == "" {
		return nil
	}
	return c.Hosts[test.Intermediate1]
}

// GetIntermediate2Host returns the second intermediate host configuration for a test
func (c *TestConfig) GetIntermediate2Host(test *TestScenario) *HostConfig {
	if test.Intermediate2 == "" {
		return nil
	}
	return c.Hosts[test.Intermediate2]
}

// MergeRunnerConfig merges test-specific runner config with host-specific config
func (c *TestConfig) MergeRunnerConfig(hostConfig *runner.Config, testConfig *runner.Config) *runner.Config {
	if hostConfig == nil && testConfig == nil {
		return &runner.Config{
			Args:       make(map[string]interface{}),
			Env:        make(map[string]string),
			ServerArgs: make(map[string]interface{}),
			ClientArgs: make(map[string]interface{}),
			ServerEnv:  make(map[string]string),
			ClientEnv:  make(map[string]string),
		}
	}
	
	if hostConfig == nil {
		result := *testConfig // Copy
		if result.Args == nil {
			result.Args = make(map[string]interface{})
		}
		if result.Env == nil {
			result.Env = make(map[string]string)
		}
		if result.ServerArgs == nil {
			result.ServerArgs = make(map[string]interface{})
		}
		if result.ClientArgs == nil {
			result.ClientArgs = make(map[string]interface{})
		}
		if result.ServerEnv == nil {
			result.ServerEnv = make(map[string]string)
		}
		if result.ClientEnv == nil {
			result.ClientEnv = make(map[string]string)
		}
		return &result
	}
	
	if testConfig == nil {
		result := *hostConfig // Copy
		if result.Args == nil {
			result.Args = make(map[string]interface{})
		}
		if result.Env == nil {
			result.Env = make(map[string]string)
		}
		if result.ServerArgs == nil {
			result.ServerArgs = make(map[string]interface{})
		}
		if result.ClientArgs == nil {
			result.ClientArgs = make(map[string]interface{})
		}
		if result.ServerEnv == nil {
			result.ServerEnv = make(map[string]string)
		}
		if result.ClientEnv == nil {
			result.ClientEnv = make(map[string]string)
		}
		return &result
	}
	
	// Create a merged configuration
	merged := &runner.Config{
		Duration:   hostConfig.Duration,
		Args:       make(map[string]interface{}),
		Env:        make(map[string]string),
		ServerArgs: make(map[string]interface{}),
		ClientArgs: make(map[string]interface{}),
		ServerEnv:  make(map[string]string),
		ClientEnv:  make(map[string]string),
		Role:       hostConfig.Role,
		Host:       hostConfig.Host,
		TargetHost: hostConfig.TargetHost,
		Port:       hostConfig.Port,
	}
	
	// Copy host config
	for k, v := range hostConfig.Args {
		merged.Args[k] = v
	}
	for k, v := range hostConfig.Env {
		merged.Env[k] = v
	}
	for k, v := range hostConfig.ServerArgs {
		merged.ServerArgs[k] = v
	}
	for k, v := range hostConfig.ClientArgs {
		merged.ClientArgs[k] = v
	}
	for k, v := range hostConfig.ServerEnv {
		merged.ServerEnv[k] = v
	}
	for k, v := range hostConfig.ClientEnv {
		merged.ClientEnv[k] = v
	}
	
	// Override with test config
	if testConfig.Duration > 0 {
		merged.Duration = testConfig.Duration
	}
	if testConfig.Host != "" {
		merged.Host = testConfig.Host
	}
	if testConfig.TargetHost != "" {
		merged.TargetHost = testConfig.TargetHost
	}
	if testConfig.Port > 0 {
		merged.Port = testConfig.Port
	}
	if testConfig.Role != "" {
		merged.Role = testConfig.Role
	}
	
	for k, v := range testConfig.Args {
		merged.Args[k] = v
	}
	for k, v := range testConfig.Env {
		merged.Env[k] = v
	}
	for k, v := range testConfig.ServerArgs {
		merged.ServerArgs[k] = v
	}
	for k, v := range testConfig.ClientArgs {
		merged.ClientArgs[k] = v
	}
	for k, v := range testConfig.ServerEnv {
		merged.ServerEnv[k] = v
	}
	for k, v := range testConfig.ClientEnv {
		merged.ClientEnv[k] = v
	}
	
	return merged
}

// GetBinaryPath returns the binary path for a specific runner, or empty string if not configured
func (c *TestConfig) GetBinaryPath(runnerName string) string {
	if c.BinaryPaths == nil {
		return ""
	}
	return c.BinaryPaths[runnerName]
}

// SaveConfig saves configuration to a YAML file
func (c *TestConfig) SaveConfig(filename string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filename, err)
	}
	
	return nil
}