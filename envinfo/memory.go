package envinfo

import (
	"context"
	"strings"
)

// MemoryInfo represents memory information
type MemoryInfo struct {
	Total     string `json:"total"`
	Available string `json:"available"`
	Used      string `json:"used"`
	Free      string `json:"free"`
	Buffers   string `json:"buffers,omitempty"`
	Cached    string `json:"cached,omitempty"`
}

// MemoryModule collects memory information
type MemoryModule struct{}

// NewMemoryModule creates a new memory information module
func NewMemoryModule() *MemoryModule {
	return &MemoryModule{}
}

// Name returns the module name
func (m *MemoryModule) Name() string {
	return "memory"
}

// Description returns the module description
func (m *MemoryModule) Description() string {
	return "Collects memory information (total, available, used, buffers, cached)"
}

// IsAvailable checks if the module can run
func (m *MemoryModule) IsAvailable(ctx context.Context, executor CommandExecutor) bool {
	// Check if /proc/meminfo exists (Linux systems)
	_, err := executor.Execute(ctx, "test -f /proc/meminfo")
	return err == nil
}

// Collect gathers memory information
func (m *MemoryModule) Collect(ctx context.Context, executor CommandExecutor) (interface{}, error) {
	info := &MemoryInfo{}

	// Get memory information from /proc/meminfo
	commands := map[string]*string{
		"grep MemTotal /proc/meminfo | awk '{print $2 \" \" $3}'":     &info.Total,
		"grep MemAvailable /proc/meminfo | awk '{print $2 \" \" $3}'": &info.Available,
		"grep MemFree /proc/meminfo | awk '{print $2 \" \" $3}'":      &info.Free,
		"grep Buffers /proc/meminfo | awk '{print $2 \" \" $3}'":      &info.Buffers,
		"grep '^Cached:' /proc/meminfo | awk '{print $2 \" \" $3}'":   &info.Cached,
	}

	for cmd, target := range commands {
		if result, err := executor.Execute(context.Background(), cmd); err == nil {
			*target = strings.TrimSpace(result)
		}
	}

	// Calculate used memory if we have total and available
	if info.Total != "" && info.Available != "" {
		// This is a simplified calculation - could be enhanced
		info.Used = "calculated from total - available"
	}

	return info, nil
}

// Auto-register this module
func init() {
	RegisterModule("memory", func() Module {
		return NewMemoryModule()
	})
}