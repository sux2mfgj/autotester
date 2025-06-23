package runner

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Auto-register the testpmd runner
func init() {
	Register("testpmd", func() Runner {
		return NewTestpmdRunner("")
	})
}

// TestpmdRunner implements the Runner interface for DPDK testpmd
type TestpmdRunner struct {
	executablePath string
}

// NewTestpmdRunner creates a new testpmd runner
func NewTestpmdRunner(executablePath string) *TestpmdRunner {
	if executablePath == "" {
		executablePath = "dpdk-testpmd"
	}
	return &TestpmdRunner{
		executablePath: executablePath,
	}
}

// Name returns the name of the runner
func (r *TestpmdRunner) Name() string {
	return "testpmd"
}

// SetExecutablePath sets the custom executable path for this runner
func (r *TestpmdRunner) SetExecutablePath(path string) {
	r.executablePath = path
}

// SupportsRole returns true if the runner supports the given role
func (r *TestpmdRunner) SupportsRole(role string) bool {
	// testpmd is primarily designed for intermediate packet forwarding
	// but can also be used as client/server for packet generation/reception
	return role == "intermediate" || role == "client" || role == "server"
}

// Validate checks if the configuration is valid for testpmd
func (r *TestpmdRunner) Validate(config Config) error {
	if !r.SupportsRole(config.Role) {
		return fmt.Errorf("unsupported role: %s", config.Role)
	}

	effectiveArgs := config.GetEffectiveArgs()

	// Validate core count if specified
	if cores, exists := effectiveArgs["cores"]; exists {
		if coreCount, ok := cores.(int); ok {
			if coreCount <= 0 {
				return fmt.Errorf("cores must be greater than 0")
			}
		}
	}

	// Validate memory channels
	if memChannels, exists := effectiveArgs["memory_channels"]; exists {
		if channels, ok := memChannels.(int); ok {
			if channels <= 0 || channels > 8 {
				return fmt.Errorf("memory_channels must be between 1 and 8")
			}
		}
	}

	// Validate ports configuration
	if ports, exists := effectiveArgs["ports"]; exists {
		if portStr, ok := ports.(string); ok {
			// Basic validation for port specification format
			if !strings.Contains(portStr, ",") && !strings.Contains(portStr, "-") {
				// Single port specification
				if _, err := strconv.Atoi(portStr); err != nil {
					return fmt.Errorf("invalid port specification: %s", portStr)
				}
			}
		} else if portList, ok := ports.([]interface{}); ok {
			if len(portList) == 0 {
				return fmt.Errorf("ports list cannot be empty")
			}
		}
	}

	// For intermediate role, validate forwarding mode
	if config.Role == "intermediate" {
		if fwdMode, exists := effectiveArgs["forward_mode"]; exists {
			if mode, ok := fwdMode.(string); ok {
				validModes := []string{"io", "mac", "macswap", "flowgen", "rxonly", "txonly", "csum", "icmpecho", "ieee1588", "tm"}
				isValid := false
				for _, validMode := range validModes {
					if mode == validMode {
						isValid = true
						break
					}
				}
				if !isValid {
					return fmt.Errorf("invalid forward_mode: %s. Valid modes: %v", mode, validModes)
				}
			}
		}
	}

	return nil
}

// BuildCommand constructs the full command line for remote execution
func (r *TestpmdRunner) BuildCommand(config Config) string {
	// Build environment variable prefix
	envPrefix := buildEnvPrefix(config)
	
	cmd := r.executablePath
	
	// Get effective arguments
	effectiveArgs := config.GetEffectiveArgs()

	// EAL (Environment Abstraction Layer) arguments come first
	ealArgs := []string{}

	// Core list/mask
	if cores, exists := effectiveArgs["cores"]; exists {
		if coreList, ok := cores.(string); ok {
			ealArgs = append(ealArgs, fmt.Sprintf("-l %s", coreList))
		} else if coreCount, ok := cores.(int); ok {
			// Generate core list 0-N
			coreList := make([]string, coreCount)
			for i := 0; i < coreCount; i++ {
				coreList[i] = strconv.Itoa(i)
			}
			ealArgs = append(ealArgs, fmt.Sprintf("-l %s", strings.Join(coreList, ",")))
		}
	}

	// Memory channels
	if memChannels, exists := effectiveArgs["memory_channels"]; exists {
		if channels, ok := memChannels.(int); ok {
			ealArgs = append(ealArgs, fmt.Sprintf("-n %d", channels))
		}
	}

	// Huge page directory
	if hugepageDir, exists := effectiveArgs["hugepage_dir"]; exists {
		if dir, ok := hugepageDir.(string); ok {
			ealArgs = append(ealArgs, fmt.Sprintf("--huge-dir %s", dir))
		}
	}

	// File prefix for multi-process
	if filePrefix, exists := effectiveArgs["file_prefix"]; exists {
		if prefix, ok := filePrefix.(string); ok {
			ealArgs = append(ealArgs, fmt.Sprintf("--file-prefix %s", prefix))
		}
	}

	// PCI allowlist/blocklist
	if allowPci, exists := effectiveArgs["allow_pci"]; exists {
		if pciList, ok := allowPci.([]interface{}); ok {
			for _, pci := range pciList {
				if pciStr, ok := pci.(string); ok {
					ealArgs = append(ealArgs, fmt.Sprintf("-a %s", pciStr))
				}
			}
		} else if pciStr, ok := allowPci.(string); ok {
			ealArgs = append(ealArgs, fmt.Sprintf("-a %s", pciStr))
		}
	}

	if blockPci, exists := effectiveArgs["block_pci"]; exists {
		if pciList, ok := blockPci.([]interface{}); ok {
			for _, pci := range pciList {
				if pciStr, ok := pci.(string); ok {
					ealArgs = append(ealArgs, fmt.Sprintf("-b %s", pciStr))
				}
			}
		} else if pciStr, ok := blockPci.(string); ok {
			ealArgs = append(ealArgs, fmt.Sprintf("-b %s", pciStr))
		}
	}

	// VDEV (virtual devices)
	if vdevs, exists := effectiveArgs["vdev"]; exists {
		if vdevList, ok := vdevs.([]interface{}); ok {
			for _, vdev := range vdevList {
				if vdevStr, ok := vdev.(string); ok {
					ealArgs = append(ealArgs, fmt.Sprintf("--vdev %s", vdevStr))
				}
			}
		} else if vdevStr, ok := vdevs.(string); ok {
			ealArgs = append(ealArgs, fmt.Sprintf("--vdev %s", vdevStr))
		}
	}

	// Add EAL arguments to command
	if len(ealArgs) > 0 {
		cmd += " " + strings.Join(ealArgs, " ")
	}

	// Separator between EAL and application arguments
	cmd += " --"

	// Application-specific arguments
	appArgs := []string{}

	// Interactive mode (default for intermediate)
	if config.Role == "intermediate" {
		if interactive, exists := effectiveArgs["interactive"]; !exists || (exists && interactive.(bool)) {
			appArgs = append(appArgs, "-i")
		}
	}

	// Port configuration
	if ports, exists := effectiveArgs["ports"]; exists {
		if portStr, ok := ports.(string); ok {
			appArgs = append(appArgs, fmt.Sprintf("--portlist=%s", portStr))
		} else if portList, ok := ports.([]interface{}); ok {
			var portStrs []string
			for _, port := range portList {
				if portInt, ok := port.(int); ok {
					portStrs = append(portStrs, strconv.Itoa(portInt))
				} else if portStr, ok := port.(string); ok {
					portStrs = append(portStrs, portStr)
				}
			}
			if len(portStrs) > 0 {
				appArgs = append(appArgs, fmt.Sprintf("--portlist=%s", strings.Join(portStrs, ",")))
			}
		}
	}

	// Number of RX/TX queues
	if rxQueues, exists := effectiveArgs["rx_queues"]; exists {
		if queues, ok := rxQueues.(int); ok {
			appArgs = append(appArgs, fmt.Sprintf("--rxq=%d", queues))
		}
	}

	if txQueues, exists := effectiveArgs["tx_queues"]; exists {
		if queues, ok := txQueues.(int); ok {
			appArgs = append(appArgs, fmt.Sprintf("--txq=%d", queues))
		}
	}

	// Number of RX/TX descriptors
	if rxDesc, exists := effectiveArgs["rx_descriptors"]; exists {
		if desc, ok := rxDesc.(int); ok {
			appArgs = append(appArgs, fmt.Sprintf("--rxd=%d", desc))
		}
	}

	if txDesc, exists := effectiveArgs["tx_descriptors"]; exists {
		if desc, ok := txDesc.(int); ok {
			appArgs = append(appArgs, fmt.Sprintf("--txd=%d", desc))
		}
	}

	// Burst size
	if burstSize, exists := effectiveArgs["burst_size"]; exists {
		if burst, ok := burstSize.(int); ok {
			appArgs = append(appArgs, fmt.Sprintf("--burst=%d", burst))
		}
	}

	// Forwarding mode (for intermediate role)
	if fwdMode, exists := effectiveArgs["forward_mode"]; exists {
		if mode, ok := fwdMode.(string); ok {
			appArgs = append(appArgs, fmt.Sprintf("--forward-mode=%s", mode))
		}
	}

	// Auto-start forwarding
	if autoStart, exists := effectiveArgs["auto_start"]; exists {
		if start, ok := autoStart.(bool); ok && start {
			appArgs = append(appArgs, "--auto-start")
		}
	}

	// Stats period
	if statsPeriod, exists := effectiveArgs["stats_period"]; exists {
		if period, ok := statsPeriod.(int); ok {
			appArgs = append(appArgs, fmt.Sprintf("--stats-period=%d", period))
		}
	}

	// Coremask for forwarding
	if fwdCores, exists := effectiveArgs["forward_cores"]; exists {
		if cores, ok := fwdCores.(string); ok {
			appArgs = append(appArgs, fmt.Sprintf("--coremask=%s", cores))
		}
	}

	// Flow control
	if flowCtrl, exists := effectiveArgs["flow_control"]; exists {
		if fc, ok := flowCtrl.(string); ok {
			appArgs = append(appArgs, fmt.Sprintf("--flow-control=%s", fc))
		}
	}

	// Enable hardware stripping
	if hwVlan, exists := effectiveArgs["hw_vlan"]; exists {
		if enable, ok := hwVlan.(bool); ok && enable {
			appArgs = append(appArgs, "--enable-hw-vlan")
		}
	}

	// CRC stripping
	if crcStrip, exists := effectiveArgs["crc_strip"]; exists {
		if enable, ok := crcStrip.(bool); ok && enable {
			appArgs = append(appArgs, "--crc-strip")
		}
	}

	// Disable RSS
	if disableRss, exists := effectiveArgs["disable_rss"]; exists {
		if disable, ok := disableRss.(bool); ok && disable {
			appArgs = append(appArgs, "--disable-rss")
		}
	}

	// Add application arguments
	if len(appArgs) > 0 {
		cmd += " " + strings.Join(appArgs, " ")
	}

	return envPrefix + cmd
}

// ParseMetrics extracts performance metrics from testpmd output
func (r *TestpmdRunner) ParseMetrics(result *Result) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	if result.Metrics == nil {
		result.Metrics = make(map[string]interface{})
	}

	output := result.Output
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse port statistics
		if strings.Contains(line, "Statistics for port") {
			r.parsePortStats(line, lines, result)
		}

		// Parse throughput information
		if strings.Contains(line, "Throughput") || strings.Contains(line, "Mpps") {
			r.parseThroughputStats(line, result)
		}

		// Parse packet counts
		if strings.Contains(line, "RX-packets:") || strings.Contains(line, "TX-packets:") {
			r.parsePacketStats(line, result)
		}

		// Parse error statistics
		if strings.Contains(line, "RX-errors:") || strings.Contains(line, "TX-errors:") {
			r.parseErrorStats(line, result)
		}

		// Parse bytes statistics
		if strings.Contains(line, "RX-bytes:") || strings.Contains(line, "TX-bytes:") {
			r.parseByteStats(line, result)
		}
	}

	return nil
}

// parsePortStats parses port-specific statistics
func (r *TestpmdRunner) parsePortStats(line string, lines []string, result *Result) {
	// Extract port number
	portRegex := regexp.MustCompile(`Statistics for port (\d+)`)
	matches := portRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return
	}

	portNum := matches[1]
	portKey := fmt.Sprintf("port_%s", portNum)

	if result.Metrics[portKey] == nil {
		result.Metrics[portKey] = make(map[string]interface{})
	}
	portMetrics := result.Metrics[portKey].(map[string]interface{})
	portMetrics["port_id"] = portNum
}

// parseThroughputStats parses throughput-related statistics
func (r *TestpmdRunner) parseThroughputStats(line string, result *Result) {
	// Look for patterns like "12.345 Mpps" or "1.234 Gbps"
	throughputRegex := regexp.MustCompile(`(\d+\.?\d*)\s*(Mpps|Gpps|Kpps|pps|Gbps|Mbps|Kbps)`)
	matches := throughputRegex.FindStringSubmatch(line)
	
	if len(matches) >= 3 {
		if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
			unit := matches[2]
			switch unit {
			case "Mpps":
				result.Metrics["throughput_mpps"] = value
				result.Metrics["throughput_pps"] = value * 1e6
			case "Gpps":
				result.Metrics["throughput_gpps"] = value
				result.Metrics["throughput_pps"] = value * 1e9
			case "Kpps":
				result.Metrics["throughput_kpps"] = value
				result.Metrics["throughput_pps"] = value * 1e3
			case "pps":
				result.Metrics["throughput_pps"] = value
			case "Gbps":
				result.Metrics["throughput_gbps"] = value
				result.Metrics["throughput_bps"] = value * 1e9
			case "Mbps":
				result.Metrics["throughput_mbps"] = value
				result.Metrics["throughput_bps"] = value * 1e6
			case "Kbps":
				result.Metrics["throughput_kbps"] = value
				result.Metrics["throughput_bps"] = value * 1e3
			}
		}
	}
}

// parsePacketStats parses packet count statistics
func (r *TestpmdRunner) parsePacketStats(line string, result *Result) {
	// Parse RX-packets: 123456
	rxPacketsRegex := regexp.MustCompile(`RX-packets:\s*(\d+)`)
	if matches := rxPacketsRegex.FindStringSubmatch(line); len(matches) > 1 {
		if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			result.Metrics["rx_packets"] = count
		}
	}

	// Parse TX-packets: 123456
	txPacketsRegex := regexp.MustCompile(`TX-packets:\s*(\d+)`)
	if matches := txPacketsRegex.FindStringSubmatch(line); len(matches) > 1 {
		if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			result.Metrics["tx_packets"] = count
		}
	}
}

// parseErrorStats parses error count statistics
func (r *TestpmdRunner) parseErrorStats(line string, result *Result) {
	// Parse RX-errors: 123
	rxErrorsRegex := regexp.MustCompile(`RX-errors:\s*(\d+)`)
	if matches := rxErrorsRegex.FindStringSubmatch(line); len(matches) > 1 {
		if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			result.Metrics["rx_errors"] = count
		}
	}

	// Parse TX-errors: 123
	txErrorsRegex := regexp.MustCompile(`TX-errors:\s*(\d+)`)
	if matches := txErrorsRegex.FindStringSubmatch(line); len(matches) > 1 {
		if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			result.Metrics["tx_errors"] = count
		}
	}
}

// parseByteStats parses byte count statistics
func (r *TestpmdRunner) parseByteStats(line string, result *Result) {
	// Parse RX-bytes: 123456 (1.2 MB)
	rxBytesRegex := regexp.MustCompile(`RX-bytes:\s*(\d+)`)
	if matches := rxBytesRegex.FindStringSubmatch(line); len(matches) > 1 {
		if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			result.Metrics["rx_bytes"] = count
		}
	}

	// Parse TX-bytes: 123456 (1.2 MB)
	txBytesRegex := regexp.MustCompile(`TX-bytes:\s*(\d+)`)
	if matches := txBytesRegex.FindStringSubmatch(line); len(matches) > 1 {
		if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			result.Metrics["tx_bytes"] = count
		}
	}
}