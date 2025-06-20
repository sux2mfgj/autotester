package config

import "fmt"

// Validator handles configuration validation
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateConfig validates the entire configuration
func (v *Validator) ValidateConfig(c *TestConfig) error {
	if c.Name == "" {
		return fmt.Errorf("test name is required")
	}
	
	if c.Runner == "" {
		return fmt.Errorf("runner is required")
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
	
	if host.Role != "" && host.Role != "client" && host.Role != "server" {
		return fmt.Errorf("host %s: invalid role %s, must be 'client' or 'server'", name, host.Role)
	}
	
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
	
	// Validate that client and server are different hosts
	if test.Client == test.Server {
		return fmt.Errorf("test %s: client and server cannot be the same host", test.Name)
	}
	
	if test.Repeat < 0 {
		return fmt.Errorf("test %s: repeat count cannot be negative", test.Name)
	}
	
	return nil
}