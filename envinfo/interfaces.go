package envinfo

import (
	"context"
)

// Module represents a pluggable environment information collector
type Module interface {
	// Name returns the unique name of this module
	Name() string
	
	// Description returns a human-readable description of what this module collects
	Description() string
	
	// Collect gathers environment information and returns structured data
	Collect(ctx context.Context, executor CommandExecutor) (interface{}, error)
	
	// IsAvailable checks if this module can run on the current system
	IsAvailable(ctx context.Context, executor CommandExecutor) bool
}

// CommandExecutor abstracts command execution for modules
type CommandExecutor interface {
	Execute(ctx context.Context, command string) (string, error)
}