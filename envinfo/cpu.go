package envinfo

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// CPUInfo represents CPU information
type CPUInfo struct {
	Model     string `json:"model"`
	Cores     int    `json:"cores"`
	Threads   int    `json:"threads"`
	Frequency string `json:"frequency,omitempty"`
}

// CPUModule collects CPU information
type CPUModule struct{}

// NewCPUModule creates a new CPU information module
func NewCPUModule() *CPUModule {
	return &CPUModule{}
}

// Name returns the module name
func (m *CPUModule) Name() string {
	return "cpu"
}

// Description returns the module description
func (m *CPUModule) Description() string {
	return "Collects CPU information (model, cores, threads, frequency)"
}

// IsAvailable checks if the module can run
func (m *CPUModule) IsAvailable(ctx context.Context, executor CommandExecutor) bool {
	// CPU information should always be available
	return true
}

// Collect gathers CPU information
func (m *CPUModule) Collect(ctx context.Context, executor CommandExecutor) (interface{}, error) {
	info := &CPUInfo{
		Cores:   runtime.NumCPU(),
		Threads: runtime.NumCPU(), // Default assumption
	}

	// Try to get CPU model from /proc/cpuinfo
	if cpuModel, err := executor.Execute(ctx, "grep 'model name' /proc/cpuinfo | head -1 | cut -d':' -f2"); err == nil {
		info.Model = strings.TrimSpace(cpuModel)
	}

	// Try to get more accurate core count
	if coreCount, err := executor.Execute(ctx, "nproc"); err == nil {
		if cores, parseErr := strconv.Atoi(strings.TrimSpace(coreCount)); parseErr == nil {
			info.Cores = cores
			info.Threads = cores // Update threads to match
		}
	}

	// Try to get physical core count vs logical processors
	if physicalCores, err := executor.Execute(ctx, "grep 'cpu cores' /proc/cpuinfo | head -1 | cut -d':' -f2"); err == nil {
		if physical, parseErr := strconv.Atoi(strings.TrimSpace(physicalCores)); parseErr == nil && physical > 0 {
			info.Cores = physical
			// Keep threads as the logical processor count from nproc
		}
	}

	// Try to get CPU frequency
	if freq, err := executor.Execute(ctx, "grep 'cpu MHz' /proc/cpuinfo | head -1 | cut -d':' -f2"); err == nil {
		freqStr := strings.TrimSpace(freq)
		if freqStr != "" {
			info.Frequency = fmt.Sprintf("%s MHz", freqStr)
		}
	}

	return info, nil
}

// Auto-register this module
func init() {
	RegisterModule("cpu", func() Module {
		return NewCPUModule()
	})
}