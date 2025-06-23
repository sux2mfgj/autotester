package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Validator handles configuration validation
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateConfig validates the entire configuration
func (v *Validator) ValidateConfig(c *TestConfig) error {
	if c == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	if c.Name == "" {
		return fmt.Errorf("test name is required")
	}
	
	// Validate runner configuration
	if err := v.validateRunners(c); err != nil {
		return err
	}
	
	if len(c.Hosts) == 0 {
		return fmt.Errorf("at least one host must be configured")
	}
	
	if len(c.Tests) == 0 {
		return fmt.Errorf("at least one test scenario must be defined")
	}
	
	// Validate hosts
	for name, host := range c.Hosts {
		if err := v.validateHost(name, host); err != nil {
			return err
		}
	}
	
	// Validate test scenarios
	for i, test := range c.Tests {
		if err := v.validateTestScenario(c, i, &test); err != nil {
			return err
		}
	}
	
	// Validate binary paths
	if err := v.validateBinaryPaths(c); err != nil {
		return err
	}
	
	return nil
}

// validateHost validates a single host configuration
func (v *Validator) validateHost(name string, host *HostConfig) error {
	if host == nil {
		return fmt.Errorf("host %s: configuration is nil", name)
	}
	
	if host.SSH == nil {
		return fmt.Errorf("host %s: SSH configuration is required", name)
	}
	
	if host.SSH.Host == "" {
		return fmt.Errorf("host %s: SSH host is required", name)
	}
	
	if host.SSH.User == "" {
		return fmt.Errorf("host %s: SSH user is required", name)
	}
	
	if host.SSH.KeyPath == "" && host.SSH.Password == "" {
		return fmt.Errorf("host %s: either SSH key path or password is required", name)
	}
	
	if host.Role != "" && host.Role != "client" && host.Role != "server" && host.Role != "intermediate" {
		return fmt.Errorf("host %s: invalid role %s, must be 'client', 'server', or 'intermediate'", name, host.Role)
	}
	
	return nil
}

// validateRunners validates the runner configuration
func (v *Validator) validateRunners(c *TestConfig) error {
	// Must have either single runner or per-role runners
	if c.Runner == "" && c.Runners == nil {
		return fmt.Errorf("either 'runner' or 'runners' configuration is required")
	}
	
	// If both are specified, warn but allow (per-role takes precedence)
	if c.Runner != "" && c.Runners != nil {
		// This is valid - per-role runners will take precedence
	}
	
	// If using per-role runners, validate that at least one is specified
	if c.Runners != nil {
		if c.Runners.Client == "" && c.Runners.Server == "" && c.Runners.Intermediate == "" {
			return fmt.Errorf("when using 'runners', at least one of client, server, or intermediate must be specified")
		}
	}
	
	// TODO: Validate that specified runners are actually registered
	// This would require importing the runner package, which might create circular dependencies
	
	return nil
}

// validateTestScenario validates a single test scenario
func (v *Validator) validateTestScenario(c *TestConfig, index int, test *TestScenario) error {
	if test.Name == "" {
		return fmt.Errorf("test %d: name is required", index)
	}
	
	if test.Client == "" {
		return fmt.Errorf("test %s: client host is required", test.Name)
	}
	
	if test.Server == "" {
		return fmt.Errorf("test %s: server host is required", test.Name)
	}
	
	// Check if referenced hosts exist
	if _, exists := c.Hosts[test.Client]; !exists {
		return fmt.Errorf("test %s: client host %s not found in hosts configuration", test.Name, test.Client)
	}
	
	if _, exists := c.Hosts[test.Server]; !exists {
		return fmt.Errorf("test %s: server host %s not found in hosts configuration", test.Name, test.Server)
	}
	
	// Check intermediate host if specified
	if test.Intermediate != "" {
		if _, exists := c.Hosts[test.Intermediate]; !exists {
			return fmt.Errorf("test %s: intermediate host %s not found in hosts configuration", test.Name, test.Intermediate)
		}
		
		// Validate that intermediate is different from client and server
		if test.Intermediate == test.Client {
			return fmt.Errorf("test %s: intermediate and client cannot be the same host", test.Name)
		}
		if test.Intermediate == test.Server {
			return fmt.Errorf("test %s: intermediate and server cannot be the same host", test.Name)
		}
	}
	
	// Validate that client and server are different hosts
	if test.Client == test.Server {
		return fmt.Errorf("test %s: client and server cannot be the same host", test.Name)
	}
	
	if test.Repeat < 0 {
		return fmt.Errorf("test %s: repeat count cannot be negative", test.Name)
	}
	
	return nil
}

// validateBinaryPaths validates binary path configurations
func (v *Validator) validateBinaryPaths(c *TestConfig) error {
	if c.BinaryPaths == nil {
		return nil // Binary paths are optional
	}
	
	for runnerName, binaryPath := range c.BinaryPaths {
		if binaryPath == "" {
			return fmt.Errorf("binary_paths.%s: path cannot be empty", runnerName)
		}
		
		// Check if the path is absolute or check if it exists in PATH
		if filepath.IsAbs(binaryPath) {
			// For absolute paths, check if the file exists and is executable
			if err := v.validateAbsoluteBinaryPath(runnerName, binaryPath); err != nil {
				return err
			}
		} else {
			// For relative paths, we'll trust that they exist in PATH
			// (checking PATH during config validation might be too strict)
		}
	}
	
	return nil
}

// validateAbsoluteBinaryPath validates an absolute binary path
func (v *Validator) validateAbsoluteBinaryPath(runnerName, binaryPath string) error {
	// Check if file exists
	info, err := os.Stat(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("binary_paths.%s: file does not exist: %s", runnerName, binaryPath)
		}
		return fmt.Errorf("binary_paths.%s: cannot access file %s: %v", runnerName, binaryPath, err)
	}
	
	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("binary_paths.%s: %s is not a regular file", runnerName, binaryPath)
	}
	
	// Check if it's executable (on Unix-like systems)
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("binary_paths.%s: %s is not executable", runnerName, binaryPath)
	}
	
	return nil
}