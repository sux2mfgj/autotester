package envinfo

import (
	"context"
	"fmt"
	"log"
)


// ModuleRegistry manages available environment collection modules
type ModuleRegistry struct {
	modules map[string]Module
	logger  *log.Logger
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry(logger *log.Logger) *ModuleRegistry {
	if logger == nil {
		logger = log.Default()
	}
	
	return &ModuleRegistry{
		modules: make(map[string]Module),
		logger:  logger,
	}
}

// Register adds a module to the registry
func (r *ModuleRegistry) Register(module Module) error {
	name := module.Name()
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}
	
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %s already registered", name)
	}
	
	r.modules[name] = module
	r.logger.Printf("Registered environment module: %s - %s", name, module.Description())
	
	return nil
}

// GetModule returns a module by name
func (r *ModuleRegistry) GetModule(name string) (Module, bool) {
	module, exists := r.modules[name]
	return module, exists
}

// ListModules returns all registered module names
func (r *ModuleRegistry) ListModules() []string {
	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	return names
}

// CollectFromModules runs all available modules and collects their data
func (r *ModuleRegistry) CollectFromModules(ctx context.Context, executor CommandExecutor, enabledModules []string) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	
	// If no specific modules are enabled, use all available modules
	if len(enabledModules) == 0 {
		enabledModules = r.ListModules()
	}
	
	for _, moduleName := range enabledModules {
		module, exists := r.modules[moduleName]
		if !exists {
			r.logger.Printf("Warning: requested module %s not found", moduleName)
			continue
		}
		
		// Check if module is available on this system
		if !module.IsAvailable(ctx, executor) {
			r.logger.Printf("Skipping module %s: not available on this system", moduleName)
			continue
		}
		
		// Collect data from module
		data, err := module.Collect(ctx, executor)
		if err != nil {
			r.logger.Printf("Warning: module %s failed to collect data: %v", moduleName, err)
			continue
		}
		
		results[moduleName] = data
		r.logger.Printf("Collected data from module: %s", moduleName)
	}
	
	return results, nil
}

// GetAvailableModules returns modules that are available on the current system
func (r *ModuleRegistry) GetAvailableModules(ctx context.Context, executor CommandExecutor) []string {
	available := make([]string, 0)
	
	for name, module := range r.modules {
		if module.IsAvailable(ctx, executor) {
			available = append(available, name)
		}
	}
	
	return available
}