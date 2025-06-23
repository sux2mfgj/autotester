package runner

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Config represents the configuration for a test run
type Config struct {
	// Common fields
	Duration time.Duration            `yaml:"duration"`
	Args     map[string]interface{}   `yaml:"args"`
	Env      map[string]string        `yaml:"env"`
	
	// Role-specific arguments (takes precedence over Args when specified)
	ServerArgs map[string]interface{} `yaml:"server_args,omitempty"`
	ClientArgs map[string]interface{} `yaml:"client_args,omitempty"`
	
	// Role-specific environment variables (takes precedence over Env when specified)
	ServerEnv  map[string]string      `yaml:"server_env,omitempty"`
	ClientEnv  map[string]string      `yaml:"client_env,omitempty"`
	
	// Role-specific settings
	Role     string                   `yaml:"role"` // "client" or "server"
	
	// Network settings
	Host       string                 `yaml:"host"`        // SSH host or general host identifier
	TargetHost string                 `yaml:"target_host"` // Specific target IP for client connections
	Port       int                    `yaml:"port"`
}

// Result represents the result of a test execution
type Result struct {
	Success    bool                     `json:"success"`
	Output     string                   `json:"output"`
	Error      string                   `json:"error,omitempty"`
	ExitCode   int                      `json:"exit_code"`
	Duration   time.Duration            `json:"duration"`
	Metrics    map[string]interface{}   `json:"metrics,omitempty"`
	StartTime  time.Time                `json:"start_time"`
	EndTime    time.Time                `json:"end_time"`
}

// Runner interface defines the contract for test program runners
type Runner interface {
	// Validate checks if the configuration is valid for this runner
	Validate(config Config) error
	
	// Name returns the name of the runner
	Name() string
	
	// SupportsRole returns true if the runner supports the given role
	SupportsRole(role string) bool
	
	// BuildCommand constructs the command line for remote execution
	BuildCommand(config Config) string
	
	// ParseMetrics extracts performance metrics from command output
	ParseMetrics(result *Result) error
	
	// SetExecutablePath sets the custom executable path for this runner
	SetExecutablePath(path string)
}

// Registry holds all registered runners
type Registry struct {
	runners map[string]func() Runner
	mu      sync.RWMutex
}

var globalRegistry = &Registry{
	runners: make(map[string]func() Runner),
}

// Register adds a runner factory to the global registry
func Register(name string, factory func() Runner) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.runners[name] = factory
}

// Create creates a new runner instance by name
func Create(name string) (Runner, error) {
	return CreateWithPath(name, "")
}

// CreateWithPath creates a new runner instance by name with a custom binary path
func CreateWithPath(name string, binaryPath string) (Runner, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	
	factory, exists := globalRegistry.runners[name]
	if !exists {
		return nil, fmt.Errorf("runner %s not found", name)
	}
	
	// Create the runner with default path first
	runner := factory()
	
	// If a custom binary path is specified, update it
	if binaryPath != "" {
		runner.SetExecutablePath(binaryPath)
	}
	
	return runner, nil
}

// GetRegistered returns all registered runner names
func GetRegistered() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	
	names := make([]string, 0, len(globalRegistry.runners))
	for name := range globalRegistry.runners {
		names = append(names, name)
	}
	return names
}

// buildEnvPrefix creates a shell environment variable prefix from the config's effective Env map
// Returns a string like "VAR1=value1 VAR2=value2 " (with trailing space) or empty string if no env vars
func buildEnvPrefix(config Config) string {
	effectiveEnv := config.GetEffectiveEnv()
	
	if len(effectiveEnv) == 0 {
		return ""
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(effectiveEnv))
	for k := range effectiveEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var envParts []string
	for _, key := range keys {
		value := effectiveEnv[key]
		// Basic shell escaping for values containing spaces or special characters
		if strings.ContainsAny(value, " \t\n\"'$`\\") {
			value = fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "'\"'\"'"))
		}
		envParts = append(envParts, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(envParts, " ") + " "
}

// GetEffectiveArgs returns the effective arguments for the given role
// Role-specific args (ServerArgs/ClientArgs) take precedence over general Args
func (c *Config) GetEffectiveArgs() map[string]interface{} {
	// Start with general args
	effective := make(map[string]interface{})
	for k, v := range c.Args {
		effective[k] = v
	}
	
	// Override with role-specific args
	var roleArgs map[string]interface{}
	switch c.Role {
	case "server":
		roleArgs = c.ServerArgs
	case "client":
		roleArgs = c.ClientArgs
	}
	
	for k, v := range roleArgs {
		effective[k] = v
	}
	
	return effective
}

// GetEffectiveEnv returns the effective environment variables for the given role
// Role-specific env (ServerEnv/ClientEnv) take precedence over general Env
func (c *Config) GetEffectiveEnv() map[string]string {
	// Start with general env
	effective := make(map[string]string)
	for k, v := range c.Env {
		effective[k] = v
	}
	
	// Override with role-specific env
	var roleEnv map[string]string
	switch c.Role {
	case "server":
		roleEnv = c.ServerEnv
	case "client":
		roleEnv = c.ClientEnv
	}
	
	for k, v := range roleEnv {
		effective[k] = v
	}
	
	return effective
}
