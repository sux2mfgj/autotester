package envinfo

import (
	"context"
	"runtime"
	"strings"
	"time"
)

// SystemInfo represents basic system information
type SystemInfo struct {
	Hostname      string    `json:"hostname"`
	KernelVersion string    `json:"kernel_version"`
	OSInfo        string    `json:"os_info"`
	Architecture  string    `json:"architecture"`
	Timestamp     time.Time `json:"timestamp"`
}

// SystemModule collects basic system information
type SystemModule struct{}

// NewSystemModule creates a new system information module
func NewSystemModule() *SystemModule {
	return &SystemModule{}
}

// Name returns the module name
func (m *SystemModule) Name() string {
	return "system"
}

// Description returns the module description
func (m *SystemModule) Description() string {
	return "Collects basic system information (hostname, kernel, OS, architecture)"
}

// IsAvailable checks if the module can run
func (m *SystemModule) IsAvailable(ctx context.Context, executor CommandExecutor) bool {
	// System information should always be available
	return true
}

// Collect gathers system information
func (m *SystemModule) Collect(ctx context.Context, executor CommandExecutor) (interface{}, error) {
	info := &SystemInfo{
		Timestamp: time.Now(),
	}

	// Collect hostname
	if hostname, err := executor.Execute(ctx, "hostname"); err == nil {
		info.Hostname = strings.TrimSpace(hostname)
	}

	// Collect kernel version
	if kernel, err := executor.Execute(ctx, "uname -r"); err == nil {
		info.KernelVersion = strings.TrimSpace(kernel)
	}

	// Collect OS info
	if osInfo, err := executor.Execute(ctx, "uname -a"); err == nil {
		info.OSInfo = strings.TrimSpace(osInfo)
	}

	// Collect architecture
	if arch, err := executor.Execute(ctx, "uname -m"); err == nil {
		info.Architecture = strings.TrimSpace(arch)
	} else {
		// Fallback to Go runtime architecture
		info.Architecture = runtime.GOARCH
	}

	return info, nil
}

// Auto-register this module
func init() {
	RegisterModule("system", func() Module {
		return NewSystemModule()
	})
}