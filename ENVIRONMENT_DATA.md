# Environment Data Collection - Modular Architecture

## Overview

The environment data collection system now uses a **modular, plugin-based architecture** that makes it easy to add new types of environment information collection. Each module is independent and can be enabled/disabled as needed.

## Architecture

- **Module Interface**: All modules implement the `Module` interface
- **Command Executor**: Abstraction for running commands (local or remote via SSH)
- **Module Registry**: Auto-discovery and management of available modules
- **Availability Checking**: Modules can check if they're compatible with the target system

## Built-in Modules

The system includes these core modules by default:

### 1. System Module (`system`)
- **Hostname**: System hostname
- **Kernel Version**: `uname -r` output  
- **OS Info**: Full `uname -a` output
- **Architecture**: System architecture (arm64, x86_64, etc.)
- **Timestamp**: When the data was collected
- **Availability**: Always available

### 2. CPU Module (`cpu`)
- **Model**: CPU model name from `/proc/cpuinfo`
- **Cores**: Number of physical CPU cores
- **Threads**: Number of logical processors 
- **Frequency**: CPU frequency in MHz
- **Availability**: Always available

### 3. Memory Module (`memory`)
- **Total**: Total memory from `/proc/meminfo`
- **Available**: Available memory from `/proc/meminfo`
- **Used**: Used memory (calculated)
- **Free**: Free memory
- **Buffers**: Buffer memory
- **Cached**: Cached memory
- **Availability**: Linux systems only (requires `/proc/meminfo`)

### 4. Network Module (`network`)
For each network interface:
- **Name**: Interface name (eth0, ib0, lo0, etc.)
- **IP Addresses**: All assigned IP addresses with CIDR
- **MAC Address**: Hardware address
- **MTU**: Maximum Transmission Unit
- **Status**: Whether interface is up/down
- **Speed**: Link speed in Mbps (when available)
- **Driver**: Network driver (when available)
- **Availability**: Always available

### 5. Software Module (`software`)
- **ib_send_bw**: InfiniBand performance tool version
- **iperf3**: Network performance tool version
- **DPDK**: Data Plane Development Kit version
- **socat**: Socket concatenation tool version
- **SSH**: SSH client version
- **GCC**: GCC compiler version
- **Python**: Python interpreter version
- **Git**: Git version control version
- **Availability**: Always available (gracefully handles missing software)

## How to Add New Environment Modules

The system uses **automatic module discovery** - simply add a new `.go` file under `envinfo/` and it will be automatically registered and available! No manual registration needed.

### 1. Create a New Module File

Create a new file in the `envinfo` package (e.g., `envinfo/storage.go`):

```go
package envinfo

import (
    "context"
    "strings"
)

// StorageInfo represents storage/disk information
type StorageInfo struct {
    Devices []StorageDevice `json:"devices"`
}

type StorageDevice struct {
    Device     string `json:"device"`
    Size       string `json:"size"`
    Used       string `json:"used"`
    Available  string `json:"available"`
    UsePercent string `json:"use_percent"`
    MountPoint string `json:"mount_point"`
}

// StorageModule collects storage information
type StorageModule struct{}

// NewStorageModule creates a new storage module
func NewStorageModule() *StorageModule {
    return &StorageModule{}
}

// Name returns the module name
func (m *StorageModule) Name() string {
    return "storage"
}

// Description returns what this module collects
func (m *StorageModule) Description() string {
    return "Collects storage/disk usage information"
}

// IsAvailable checks if the module can run on this system
func (m *StorageModule) IsAvailable(ctx context.Context, executor CommandExecutor) bool {
    // Check if 'df' command is available
    _, err := executor.Execute(ctx, "df --version")
    return err == nil
}

// Collect gathers storage information
func (m *StorageModule) Collect(ctx context.Context, executor CommandExecutor) (interface{}, error) {
    info := &StorageInfo{}
    
    // Get disk usage information
    output, err := executor.Execute(ctx, "df -h | grep -v tmpfs | tail -n +2")
    if err != nil {
        return nil, err
    }
    
    lines := strings.Split(strings.TrimSpace(output), "\n")
    for _, line := range lines {
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
    
    return info, nil
}

// Auto-register this module - this is all you need!
func init() {
    RegisterModule("storage", func() Module {
        return NewStorageModule()
    })
}
```

### 2. That's It! ✨

The module is **automatically discovered and registered** when you build the project. No manual registration needed!

### 3. Use the New Module

The module is immediately available:

```go
// Create collector with all default modules (including new ones)
collector, err := envinfo.NewLocalModularCollector(logger)
if err != nil {
    return err
}

// Collect from all available modules
envInfo, err := collector.CollectModular(ctx)
if err != nil {
    return err
}

// Access data from your specific module
if storageData, exists := envInfo.GetModuleData("storage"); exists {
    if storage, ok := storageData.(*envinfo.StorageInfo); ok {
        fmt.Printf("Found %d storage devices\n", len(storage.Devices))
    }
}
```

### 4. Extend Existing Modules

You can also add fields to existing modules:

```go
// Add to existing SoftwareVersions struct in software.go
type SoftwareVersions struct {
    // ... existing fields ...
    NewTool     string `json:"new_tool,omitempty"`
    CustomApp   string `json:"custom_app,omitempty"`
}

// Add to the collection logic in software.go
func (m *SoftwareModule) Collect(ctx context.Context, executor CommandExecutor) (interface{}, error) {
    versions := &SoftwareVersions{}

    // ... existing checks ...
    
    // Add new software version checks
    if version, err := executor.Execute(ctx, "newtool --version 2>&1 | head -1"); err == nil {
        versions.NewTool = strings.TrimSpace(version)
    }
    
    if version, err := executor.Execute(ctx, "customapp -v 2>&1"); err == nil {
        versions.CustomApp = strings.TrimSpace(version)
    }

    return versions, nil
}
```

## Key Benefits of the Auto-Discovery Architecture

### 1. **Zero-Configuration Extensibility**
- **Just add a file** - no manual registration needed
- **Automatic discovery** - modules register themselves via `init()` functions
- **No code changes required** - existing code automatically sees new modules
- Each module is completely independent and self-contained

### 2. **Availability Checking**
- Modules can check if they're compatible with the target system
- Graceful handling of missing dependencies or unavailable features
- Linux-specific modules (like memory) automatically skip on other platforms

### 3. **Flexibility**
- Enable/disable specific modules as needed
- Selective data collection for performance or security reasons
- Custom module combinations for different environments

### 4. **Developer-Friendly**
- **Drop-in modules** - just add a file and it works
- Each module is isolated and easier to test  
- Clear separation of concerns
- No complex interdependencies between collection types
- **Perfect for teams** - developers can add modules independently

## Usage Examples

### Basic Usage
```go
// Use all available modules
collector, _ := envinfo.NewLocalModularCollector(logger)
envInfo, _ := collector.CollectModular(ctx)

// Access specific module data
if systemData, exists := envInfo.GetModuleData("system"); exists {
    system := systemData.(*envinfo.SystemInfo)
    fmt.Printf("Running on %s\n", system.Hostname)
}
```

### Selective Module Usage
```go
// Use only specific modules
collector, _ := envinfo.NewLocalModularCollector(logger)
collector.SetEnabledModules([]string{"system", "network"})
envInfo, _ := collector.CollectModular(ctx)

fmt.Printf("Enabled modules: %v\n", envInfo.GetModuleNames())
```

### Check Available Modules
```go
// See all auto-registered modules (before creating collector)
fmt.Printf("Auto-registered: %v\n", envinfo.GetRegisteredModuleNames())

collector, _ := envinfo.NewLocalModularCollector(logger)
available := collector.GetAvailableModules(ctx)
all := collector.ListAllModules()

fmt.Printf("Available on this system: %v\n", available)
fmt.Printf("All registered modules: %v\n", all)
```

## Real-World Example: Storage Module

The system includes a working example in `envinfo/storage.go` that was added using the auto-discovery system:

```bash
# The storage module collects:
# - Disk usage for all mounted filesystems  
# - Device names, sizes, used/available space
# - Mount points and filesystem types
# - Automatically works on Linux, macOS, etc.

# Just by adding the file, it became available:
$ ./perf-runner -config examples/example-with-env.yaml -json | jq '.modules.storage'
```

## Common Extensions Examples

### Add InfiniBand Interface Information
```go
// Add to NetworkInterface struct
type NetworkInterface struct {
    // ... existing fields ...
    IBInfo *InfiniBandInfo `json:"ib_info,omitempty"`
}

type InfiniBandInfo struct {
    LID    string `json:"lid,omitempty"`
    Rate   string `json:"rate,omitempty"`
    State  string `json:"state,omitempty"`
}

// In collectRemoteNetworkInfo(), add IB detection:
if strings.HasPrefix(ifaceName, "ib") {
    ibInfo := &InfiniBandInfo{}
    
    if lid, err := c.executeCommand(ctx, fmt.Sprintf("cat /sys/class/infiniband/*/ports/*/lid")); err == nil {
        ibInfo.LID = strings.TrimSpace(lid)
    }
    
    if rate, err := c.executeCommand(ctx, fmt.Sprintf("cat /sys/class/infiniband/*/ports/*/rate")); err == nil {
        ibInfo.Rate = strings.TrimSpace(rate)
    }
    
    netInterface.IBInfo = ibInfo
}
```

### Add System Load Information
```go
// Add to EnvironmentInfo
type EnvironmentInfo struct {
    // ... existing fields ...
    SystemLoad LoadInfo `json:"system_load"`
}

type LoadInfo struct {
    Load1min  float64 `json:"load_1min"`
    Load5min  float64 `json:"load_5min"`
    Load15min float64 `json:"load_15min"`
}

// In collectSystemInfo():
if loadAvg, err := c.executeCommand(ctx, "cat /proc/loadavg | awk '{print $1,$2,$3}'"); err == nil {
    parts := strings.Fields(strings.TrimSpace(loadAvg))
    if len(parts) >= 3 {
        env.SystemLoad.Load1min, _ = strconv.ParseFloat(parts[0], 64)
        env.SystemLoad.Load5min, _ = strconv.ParseFloat(parts[1], 64)
        env.SystemLoad.Load15min, _ = strconv.ParseFloat(parts[2], 64)
    }
}
```

### Add Storage Information
```go
// Add to EnvironmentInfo
type EnvironmentInfo struct {
    // ... existing fields ...
    StorageInfo []StorageDevice `json:"storage_info"`
}

type StorageDevice struct {
    Device     string `json:"device"`
    Size       string `json:"size"`
    Used       string `json:"used"`
    Available  string `json:"available"`
    UsePercent string `json:"use_percent"`
    MountPoint string `json:"mount_point"`
}

// Create new collection method
func (c *Collector) collectStorageInfo(ctx context.Context, env *EnvironmentInfo) error {
    output, err := c.executeCommand(ctx, "df -h | grep -v tmpfs | grep -v udev | tail -n +2")
    if err != nil {
        return err
    }
    
    lines := strings.Split(strings.TrimSpace(output), "\n")
    for _, line := range lines {
        fields := strings.Fields(line)
        if len(fields) >= 6 {
            storage := StorageDevice{
                Device:     fields[0],
                Size:       fields[1],
                Used:       fields[2],
                Available:  fields[3],
                UsePercent: fields[4],
                MountPoint: fields[5],
            }
            env.StorageInfo = append(env.StorageInfo, storage)
        }
    }
    
    return nil
}
```

## Testing New Data Collection

1. **Build and test locally:**
   ```bash
   go build -o perf-runner
   ```

2. **Create test configuration:**
   ```yaml
   name: "Environment Test"
   runner: "iperf3"
   collect_env: true
   # ... rest of config
   ```

3. **Test with existing functionality:**
   ```bash
   ./perf-runner -json -config test-config.yaml
   ```

4. **Verify new fields appear in JSON output**

The environment collection system is designed to be easily extensible while gracefully handling failures (missing commands, unavailable data, etc.).