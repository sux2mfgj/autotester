package envinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"perf-runner/ssh"
)

// EnvironmentInfo represents system environment information
type EnvironmentInfo struct {
	Hostname         string             `json:"hostname"`
	KernelVersion    string             `json:"kernel_version"`
	OSInfo           string             `json:"os_info"`
	Architecture     string             `json:"architecture"`
	CPUInfo          CPUInfo            `json:"cpu_info"`
	MemoryInfo       MemoryInfo         `json:"memory_info"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces"`
	SoftwareVersions SoftwareVersions   `json:"software_versions"`
	Timestamp        time.Time          `json:"timestamp"`
}


// Collector handles environment information collection
type Collector struct {
	sshClient *ssh.Client
	local     bool
}

// NewCollector creates a new environment information collector
func NewCollector(sshClient *ssh.Client) *Collector {
	return &Collector{
		sshClient: sshClient,
		local:     sshClient == nil,
	}
}

// NewLocalCollector creates a collector for local system information
func NewLocalCollector() *Collector {
	return &Collector{
		sshClient: nil,
		local:     true,
	}
}

// Collect gathers comprehensive environment information
func (c *Collector) Collect(ctx context.Context) (*EnvironmentInfo, error) {
	env := &EnvironmentInfo{
		Timestamp: time.Now(),
	}

	// Collect basic system information
	if err := c.collectSystemInfo(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to collect system info: %w", err)
	}

	// Collect network information
	if err := c.collectNetworkInfo(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to collect network info: %w", err)
	}

	// Collect software versions
	if err := c.collectSoftwareVersions(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to collect software versions: %w", err)
	}

	return env, nil
}

// collectSystemInfo gathers basic system information
func (c *Collector) collectSystemInfo(ctx context.Context, env *EnvironmentInfo) error {
	commands := map[string]*string{
		"hostname":       &env.Hostname,
		"uname -r":       &env.KernelVersion,
		"uname -a":       &env.OSInfo,
	}

	for cmd, target := range commands {
		result, err := c.executeCommand(ctx, cmd)
		if err != nil {
			return fmt.Errorf("failed to execute %s: %w", cmd, err)
		}
		*target = strings.TrimSpace(result)
	}

	// Set architecture
	if c.local {
		env.Architecture = runtime.GOARCH
	} else {
		arch, err := c.executeCommand(ctx, "uname -m")
		if err != nil {
			return fmt.Errorf("failed to get architecture: %w", err)
		}
		env.Architecture = strings.TrimSpace(arch)
	}

	// Collect CPU info
	if err := c.collectCPUInfo(ctx, env); err != nil {
		return fmt.Errorf("failed to collect CPU info: %w", err)
	}

	// Collect memory info
	if err := c.collectMemoryInfo(ctx, env); err != nil {
		return fmt.Errorf("failed to collect memory info: %w", err)
	}

	return nil
}

// collectCPUInfo gathers CPU information
func (c *Collector) collectCPUInfo(ctx context.Context, env *EnvironmentInfo) error {
	if c.local {
		env.CPUInfo.Cores = runtime.NumCPU()
		env.CPUInfo.Threads = runtime.NumCPU() // Simplified for local
	}

	// Get CPU model from /proc/cpuinfo
	cpuInfo, err := c.executeCommand(ctx, "grep 'model name' /proc/cpuinfo | head -1 | cut -d':' -f2")
	if err == nil {
		env.CPUInfo.Model = strings.TrimSpace(cpuInfo)
	}

	// Get CPU count
	coreCount, err := c.executeCommand(ctx, "nproc")
	if err == nil {
		env.CPUInfo.Cores = parseInt(strings.TrimSpace(coreCount))
	}

	return nil
}

// collectMemoryInfo gathers memory information
func (c *Collector) collectMemoryInfo(ctx context.Context, env *EnvironmentInfo) error {
	// Get memory info from /proc/meminfo
	memTotal, err := c.executeCommand(ctx, "grep MemTotal /proc/meminfo | awk '{print $2 \" \" $3}'")
	if err == nil {
		env.MemoryInfo.Total = strings.TrimSpace(memTotal)
	}

	memAvailable, err := c.executeCommand(ctx, "grep MemAvailable /proc/meminfo | awk '{print $2 \" \" $3}'")
	if err == nil {
		env.MemoryInfo.Available = strings.TrimSpace(memAvailable)
	}

	return nil
}

// collectNetworkInfo gathers network interface information
func (c *Collector) collectNetworkInfo(ctx context.Context, env *EnvironmentInfo) error {
	if c.local {
		return c.collectLocalNetworkInfo(env)
	}
	return c.collectRemoteNetworkInfo(ctx, env)
}

// collectLocalNetworkInfo gathers local network information
func (c *Collector) collectLocalNetworkInfo(env *EnvironmentInfo) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		netInterface := NetworkInterface{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
			MTU:        iface.MTU,
			IsUp:       iface.Flags&net.FlagUp != 0,
		}

		// Get IP addresses
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				netInterface.IPAddresses = append(netInterface.IPAddresses, addr.String())
			}
		}

		env.NetworkInterfaces = append(env.NetworkInterfaces, netInterface)
	}

	return nil
}

// collectRemoteNetworkInfo gathers remote network information
func (c *Collector) collectRemoteNetworkInfo(ctx context.Context, env *EnvironmentInfo) error {
	// Get interface list
	ifaceList, err := c.executeCommand(ctx, "ip link show | grep '^[0-9]' | cut -d':' -f2 | tr -d ' '")
	if err != nil {
		return fmt.Errorf("failed to get interface list: %w", err)
	}

	interfaces := strings.Split(strings.TrimSpace(ifaceList), "\n")
	for _, ifaceName := range interfaces {
		if ifaceName == "" {
			continue
		}

		netInterface := NetworkInterface{Name: ifaceName}

		// Get IP addresses
		ipOutput, err := c.executeCommand(ctx, fmt.Sprintf("ip addr show %s | grep 'inet ' | awk '{print $2}'", ifaceName))
		if err == nil {
			ips := strings.Split(strings.TrimSpace(ipOutput), "\n")
			for _, ip := range ips {
				if ip != "" {
					netInterface.IPAddresses = append(netInterface.IPAddresses, ip)
				}
			}
		}

		// Get MTU
		mtuOutput, err := c.executeCommand(ctx, fmt.Sprintf("cat /sys/class/net/%s/mtu", ifaceName))
		if err == nil {
			netInterface.MTU = parseInt(strings.TrimSpace(mtuOutput))
		}

		// Get MAC address
		macOutput, err := c.executeCommand(ctx, fmt.Sprintf("cat /sys/class/net/%s/address", ifaceName))
		if err == nil {
			netInterface.MACAddress = strings.TrimSpace(macOutput)
		}

		// Check if interface is up
		stateOutput, err := c.executeCommand(ctx, fmt.Sprintf("cat /sys/class/net/%s/operstate", ifaceName))
		if err == nil {
			netInterface.IsUp = strings.TrimSpace(stateOutput) == "up"
		}

		// Try to get speed (may not be available for all interfaces)
		speedOutput, err := c.executeCommand(ctx, fmt.Sprintf("cat /sys/class/net/%s/speed 2>/dev/null || echo 'unknown'", ifaceName))
		if err == nil && strings.TrimSpace(speedOutput) != "unknown" {
			netInterface.Speed = strings.TrimSpace(speedOutput) + " Mbps"
		}

		env.NetworkInterfaces = append(env.NetworkInterfaces, netInterface)
	}

	return nil
}

// collectSoftwareVersions gathers versions of relevant software
func (c *Collector) collectSoftwareVersions(ctx context.Context, env *EnvironmentInfo) error {
	// Check for ib_send_bw
	if version, err := c.executeCommand(ctx, "ib_send_bw --version 2>&1 | head -1"); err == nil {
		env.SoftwareVersions.IbSendBw = strings.TrimSpace(version)
	}

	// Check for iperf3
	if version, err := c.executeCommand(ctx, "iperf3 --version 2>&1 | head -1"); err == nil {
		env.SoftwareVersions.Iperf3 = strings.TrimSpace(version)
	}

	// Check for DPDK
	if version, err := c.executeCommand(ctx, "dpdk-testpmd --version 2>&1 | head -1"); err == nil {
		env.SoftwareVersions.DPDK = strings.TrimSpace(version)
	}

	// Check for socat
	if version, err := c.executeCommand(ctx, "socat -V 2>&1 | head -1"); err == nil {
		env.SoftwareVersions.Socat = strings.TrimSpace(version)
	}

	// Check SSH version
	if version, err := c.executeCommand(ctx, "ssh -V 2>&1"); err == nil {
		env.SoftwareVersions.SSHVersion = strings.TrimSpace(version)
	}

	return nil
}

// executeCommand executes a command either locally or remotely
func (c *Collector) executeCommand(ctx context.Context, command string) (string, error) {
	if c.local {
		// Execute command locally
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("local command execution failed: %w", err)
		}
		return string(output), nil
	}

	result, err := c.sshClient.ExecuteCommand(ctx, command)
	if err != nil {
		return "", err
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Error)
	}

	return result.Output, nil
}

// Helper function to parse integer strings
func parseInt(s string) int {
	// Simple integer parsing - could be enhanced
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// ToJSON converts environment info to JSON string
func (e *EnvironmentInfo) ToJSON() (string, error) {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}