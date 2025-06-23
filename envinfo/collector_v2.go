package envinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"perf-runner/ssh"
)

// ModularEnvironmentInfo represents the new modular environment data structure
type ModularEnvironmentInfo struct {
	CollectionTime time.Time              `json:"collection_time"`
	Modules        map[string]interface{} `json:"modules"`
	HostInfo       HostInfo               `json:"host_info"`
}

// HostInfo represents basic information about the host where data was collected
type HostInfo struct {
	IsLocal    bool   `json:"is_local"`
	SSHHost    string `json:"ssh_host,omitempty"`
	SSHUser    string `json:"ssh_user,omitempty"`
}

// ModularCollector provides a modular approach to environment data collection
type ModularCollector struct {
	registry     *ModuleRegistry
	executor     CommandExecutor
	hostInfo     HostInfo
	logger       *log.Logger
	enabledModules []string
}

// NewModularCollector creates a new modular environment collector
func NewModularCollector(registry *ModuleRegistry, executor CommandExecutor, logger *log.Logger) *ModularCollector {
	if logger == nil {
		logger = log.Default()
	}

	var hostInfo HostInfo
	
	// Determine host info based on executor type
	switch executor.(type) {
	case *LocalExecutor:
		hostInfo.IsLocal = true
	case *RemoteExecutor:
		hostInfo.IsLocal = false
		// Note: SSH host/user info would need to be passed in if needed
	default:
		hostInfo.IsLocal = false // Default assumption
	}

	return &ModularCollector{
		registry:       registry,
		executor:       executor,
		hostInfo:       hostInfo,
		logger:         logger,
		enabledModules: []string{}, // Empty means all available modules
	}
}

// NewLocalModularCollector creates a collector for local system using default modules
func NewLocalModularCollector(logger *log.Logger) (*ModularCollector, error) {
	registry, err := GetDefaultRegistry(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create default registry: %w", err)
	}

	executor := NewLocalExecutor()
	return NewModularCollector(registry, executor, logger), nil
}

// NewRemoteModularCollector creates a collector for remote system using default modules
func NewRemoteModularCollector(sshClient *ssh.Client, logger *log.Logger) (*ModularCollector, error) {
	registry, err := GetDefaultRegistry(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create default registry: %w", err)
	}

	executor := NewRemoteExecutor(sshClient)
	
	collector := NewModularCollector(registry, executor, logger)
	collector.hostInfo.SSHHost = sshClient.Config().Host
	collector.hostInfo.SSHUser = sshClient.Config().User
	
	return collector, nil
}

// SetEnabledModules sets which modules should be run (empty slice means all available)
func (c *ModularCollector) SetEnabledModules(modules []string) {
	c.enabledModules = modules
}

// GetAvailableModules returns modules that can run on the target system
func (c *ModularCollector) GetAvailableModules(ctx context.Context) []string {
	return c.registry.GetAvailableModules(ctx, c.executor)
}

// ListAllModules returns all registered modules (whether available or not)
func (c *ModularCollector) ListAllModules() []string {
	return c.registry.ListModules()
}

// CollectModular gathers environment information using the modular approach
func (c *ModularCollector) CollectModular(ctx context.Context) (*ModularEnvironmentInfo, error) {
	c.logger.Printf("Starting modular environment collection...")
	
	// Collect data from modules
	moduleData, err := c.registry.CollectFromModules(ctx, c.executor, c.enabledModules)
	if err != nil {
		return nil, fmt.Errorf("failed to collect module data: %w", err)
	}

	info := &ModularEnvironmentInfo{
		CollectionTime: time.Now(),
		Modules:        moduleData,
		HostInfo:       c.hostInfo,
	}

	c.logger.Printf("Completed modular environment collection with %d modules", len(moduleData))
	return info, nil
}

// ToJSON converts the modular environment info to JSON
func (info *ModularEnvironmentInfo) ToJSON() (string, error) {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetModuleData returns data from a specific module
func (info *ModularEnvironmentInfo) GetModuleData(moduleName string) (interface{}, bool) {
	data, exists := info.Modules[moduleName]
	return data, exists
}

// GetModuleNames returns the names of all modules that collected data
func (info *ModularEnvironmentInfo) GetModuleNames() []string {
	names := make([]string, 0, len(info.Modules))
	for name := range info.Modules {
		names = append(names, name)
	}
	return names
}