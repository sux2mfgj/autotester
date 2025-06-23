package envinfo

import (
	"log"
	"sync"
)

// Global registry for auto-discovery
var (
	globalModuleFactories = make(map[string]func() Module)
	globalMutex           sync.RWMutex
)

// RegisterModule registers a module factory function for auto-discovery
// This is typically called from init() functions in module files
func RegisterModule(name string, factory func() Module) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalModuleFactories[name] = factory
}

// GetRegisteredModuleNames returns all auto-registered module names
func GetRegisteredModuleNames() []string {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	
	names := make([]string, 0, len(globalModuleFactories))
	for name := range globalModuleFactories {
		names = append(names, name)
	}
	return names
}

// RegisterDefaultModules registers all auto-discovered modules
func RegisterDefaultModules(registry *ModuleRegistry) error {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	
	// Register all auto-discovered modules
	for _, factory := range globalModuleFactories {
		module := factory()
		if err := registry.Register(module); err != nil {
			return err
		}
	}

	return nil
}

// GetDefaultRegistry creates a registry with all auto-discovered modules pre-registered
func GetDefaultRegistry(logger *log.Logger) (*ModuleRegistry, error) {
	registry := NewModuleRegistry(logger)
	
	if err := RegisterDefaultModules(registry); err != nil {
		return nil, err
	}
	
	return registry, nil
}