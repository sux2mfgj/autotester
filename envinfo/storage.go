package envinfo

import (
	"context"
	"strings"
)

// StorageInfo represents storage/disk information
type StorageInfo struct {
	Devices []StorageDevice `json:"devices"`
}

// StorageDevice represents information about a storage device
type StorageDevice struct {
	Device     string `json:"device"`
	Size       string `json:"size"`
	Used       string `json:"used"`
	Available  string `json:"available"`
	UsePercent string `json:"use_percent"`
	MountPoint string `json:"mount_point"`
	FileSystem string `json:"filesystem,omitempty"`
}

// StorageModule collects storage information
type StorageModule struct{}

// NewStorageModule creates a new storage information module
func NewStorageModule() *StorageModule {
	return &StorageModule{}
}

// Name returns the module name
func (m *StorageModule) Name() string {
	return "storage"
}

// Description returns the module description
func (m *StorageModule) Description() string {
	return "Collects storage/disk usage information for mounted filesystems"
}

// IsAvailable checks if the module can run
func (m *StorageModule) IsAvailable(ctx context.Context, executor CommandExecutor) bool {
	// Check if 'df' command is available (use a simple df call that works on most systems)
	_, err := executor.Execute(ctx, "df / | head -1")
	return err == nil
}

// Collect gathers storage information
func (m *StorageModule) Collect(ctx context.Context, executor CommandExecutor) (interface{}, error) {
	info := &StorageInfo{}
	
	// Get disk usage information
	output, err := executor.Execute(ctx, "df -h | grep -v tmpfs | grep -v udev | tail -n +2")
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) >= 6 {
			device := StorageDevice{
				Device:     fields[0],
				Size:       fields[1],
				Used:       fields[2],
				Available:  fields[3],
				UsePercent: fields[4],
				MountPoint: fields[5],
			}
			info.Devices = append(info.Devices, device)
		}
	}
	
	// Try to get filesystem types
	if fsOutput, err := executor.Execute(ctx, "df -T | grep -v tmpfs | grep -v udev | tail -n +2"); err == nil {
		fsLines := strings.Split(strings.TrimSpace(fsOutput), "\n")
		for i, line := range fsLines {
			if line == "" || i >= len(info.Devices) {
				continue
			}
			
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				info.Devices[i].FileSystem = fields[1]
			}
		}
	}
	
	return info, nil
}

// Auto-register this module - this is all you need to add a new module!
func init() {
	RegisterModule("storage", func() Module {
		return NewStorageModule()
	})
}