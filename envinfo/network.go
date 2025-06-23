package envinfo

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// NetworkInterface represents network interface information
type NetworkInterface struct {
	Name         string   `json:"name"`
	IPAddresses  []string `json:"ip_addresses"`
	MACAddress   string   `json:"mac_address"`
	MTU          int      `json:"mtu"`
	IsUp         bool     `json:"is_up"`
	Speed        string   `json:"speed,omitempty"`
	Driver       string   `json:"driver,omitempty"`
}

// NetworkInfo represents all network information
type NetworkInfo struct {
	Interfaces []NetworkInterface `json:"interfaces"`
}

// NetworkModule collects network interface information
type NetworkModule struct{}

// NewNetworkModule creates a new network information module
func NewNetworkModule() *NetworkModule {
	return &NetworkModule{}
}

// Name returns the module name
func (m *NetworkModule) Name() string {
	return "network"
}

// Description returns the module description
func (m *NetworkModule) Description() string {
	return "Collects network interface information (IPs, MAC, MTU, status, speed)"
}

// IsAvailable checks if the module can run
func (m *NetworkModule) IsAvailable(ctx context.Context, executor CommandExecutor) bool {
	// Network information should always be available
	return true
}

// Collect gathers network information
func (m *NetworkModule) Collect(ctx context.Context, executor CommandExecutor) (interface{}, error) {
	info := &NetworkInfo{}

	// Try to collect using local Go net package first (more reliable)
	if localInterfaces, err := m.collectLocalInterfaces(); err == nil && len(localInterfaces) > 0 {
		info.Interfaces = localInterfaces
	} else {
		// Fallback to remote/command-based collection
		if remoteInterfaces, err := m.collectRemoteInterfaces(ctx, executor); err == nil {
			info.Interfaces = remoteInterfaces
		}
	}

	return info, nil
}

// collectLocalInterfaces uses Go's net package for local interface information
func (m *NetworkModule) collectLocalInterfaces() ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var result []NetworkInterface
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

		result = append(result, netInterface)
	}

	return result, nil
}

// collectRemoteInterfaces uses command execution for remote interface information
func (m *NetworkModule) collectRemoteInterfaces(ctx context.Context, executor CommandExecutor) ([]NetworkInterface, error) {
	// Get interface list
	ifaceList, err := executor.Execute(ctx, "ip link show | grep '^[0-9]' | cut -d':' -f2 | tr -d ' '")
	if err != nil {
		return nil, fmt.Errorf("failed to get interface list: %w", err)
	}

	var result []NetworkInterface
	interfaceNames := strings.Split(strings.TrimSpace(ifaceList), "\n")
	
	for _, ifaceName := range interfaceNames {
		if ifaceName == "" {
			continue
		}

		netInterface := NetworkInterface{Name: ifaceName}

		// Get IP addresses
		if ipOutput, err := executor.Execute(ctx, fmt.Sprintf("ip addr show %s | grep 'inet ' | awk '{print $2}'", ifaceName)); err == nil {
			ips := strings.Split(strings.TrimSpace(ipOutput), "\n")
			for _, ip := range ips {
				if ip != "" {
					netInterface.IPAddresses = append(netInterface.IPAddresses, ip)
				}
			}
		}

		// Get MTU
		if mtuOutput, err := executor.Execute(ctx, fmt.Sprintf("cat /sys/class/net/%s/mtu", ifaceName)); err == nil {
			if mtu, parseErr := strconv.Atoi(strings.TrimSpace(mtuOutput)); parseErr == nil {
				netInterface.MTU = mtu
			}
		}

		// Get MAC address
		if macOutput, err := executor.Execute(ctx, fmt.Sprintf("cat /sys/class/net/%s/address", ifaceName)); err == nil {
			netInterface.MACAddress = strings.TrimSpace(macOutput)
		}

		// Check if interface is up
		if stateOutput, err := executor.Execute(ctx, fmt.Sprintf("cat /sys/class/net/%s/operstate", ifaceName)); err == nil {
			netInterface.IsUp = strings.TrimSpace(stateOutput) == "up"
		}

		// Try to get speed (may not be available for all interfaces)
		if speedOutput, err := executor.Execute(ctx, fmt.Sprintf("cat /sys/class/net/%s/speed 2>/dev/null || echo 'unknown'", ifaceName)); err == nil && strings.TrimSpace(speedOutput) != "unknown" {
			netInterface.Speed = strings.TrimSpace(speedOutput) + " Mbps"
		}

		// Try to get driver
		if driverOutput, err := executor.Execute(ctx, fmt.Sprintf("readlink /sys/class/net/%s/device/driver 2>/dev/null | xargs basename", ifaceName)); err == nil {
			driver := strings.TrimSpace(driverOutput)
			if driver != "" && driver != "basename" {
				netInterface.Driver = driver
			}
		}

		result = append(result, netInterface)
	}

	return result, nil
}
// Auto-register this module
func init() {
	RegisterModule("network", func() Module {
		return NewNetworkModule()
	})
}
