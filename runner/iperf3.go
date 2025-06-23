package runner

import (
	"fmt"
	"strconv"
	"strings"
)

// Auto-register the iperf3 runner
func init() {
	Register("iperf3", func() Runner {
		return NewIperf3Runner("")
	})
}

// Iperf3Runner implements the Runner interface for iperf3
type Iperf3Runner struct {
	executablePath string
}

// NewIperf3Runner creates a new iperf3 runner
func NewIperf3Runner(executablePath string) *Iperf3Runner {
	if executablePath == "" {
		executablePath = "iperf3"
	}
	return &Iperf3Runner{
		executablePath: executablePath,
	}
}

// Name returns the name of the runner
func (r *Iperf3Runner) Name() string {
	return "iperf3"
}

// SetExecutablePath sets the custom executable path for this runner
func (r *Iperf3Runner) SetExecutablePath(path string) {
	r.executablePath = path
}

// SupportsRole returns true if the runner supports the given role
func (r *Iperf3Runner) SupportsRole(role string) bool {
	return role == "client" || role == "server" || role == "intermediate"
}

// Validate checks if the configuration is valid for iperf3
func (r *Iperf3Runner) Validate(config Config) error {
	if !r.SupportsRole(config.Role) {
		return fmt.Errorf("unsupported role: %s", config.Role)
	}
	
	// For iperf3, client needs a target host but server doesn't
	if config.Role == "client" {
		if config.TargetHost == "" && config.Host == "" {
			return fmt.Errorf("target_host or host is required for client role")
		}
	}
	
	// For intermediate nodes, target host is required for forwarding
	if config.Role == "intermediate" {
		if config.TargetHost == "" && config.Host == "" {
			return fmt.Errorf("target_host or host is required for intermediate role")
		}
	}
	
	// Validate parallel streams if specified (use effective args)
	effectiveArgs := config.GetEffectiveArgs()
	if parallelStreams, exists := effectiveArgs["parallel_streams"]; exists {
		if streams, ok := parallelStreams.(int); ok {
			if streams <= 0 {
				return fmt.Errorf("parallel_streams must be greater than 0")
			}
		}
	}
	
	// Validate port if specified
	if config.Port < 0 || config.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	
	return nil
}

// BuildCommand constructs the full command line for remote execution
func (r *Iperf3Runner) BuildCommand(config Config) string {
	// Build environment variable prefix
	envPrefix := buildEnvPrefix(config)
	
	cmd := r.executablePath
	
	// Set role (server, client, or intermediate)
	if config.Role == "server" {
		cmd += " -s"
	} else if config.Role == "client" {
		// Client mode - determine target host
		targetHost := config.TargetHost
		if targetHost == "" {
			targetHost = config.Host
		}
		cmd += fmt.Sprintf(" -c %s", targetHost)
	} else if config.Role == "intermediate" {
		// Intermediate mode - run a proxy/relay
		// For iperf3, this would typically be a custom proxy tool or socat
		// We'll use a conceptual approach where the tool runs in relay mode
		cmd = "socat" // Use socat as a TCP/UDP relay tool
		
		targetHost := config.TargetHost
		if targetHost == "" {
			targetHost = config.Host
		}
		
		// Listen on the configured port and forward to target
		listenPort := config.Port
		if listenPort <= 0 {
			listenPort = 5201 // Default iperf3 port
		}
		
		targetPort := listenPort // Forward to same port on target
		cmd += fmt.Sprintf(" TCP-LISTEN:%d,fork TCP:%s:%d", listenPort, targetHost, targetPort)
		
		// Return early for socat command
		return envPrefix + cmd
	}
	
	// Port (if specified)
	if config.Port > 0 {
		cmd += fmt.Sprintf(" -p %d", config.Port)
	}
	
	// Duration (if specified)
	if config.Duration > 0 {
		cmd += fmt.Sprintf(" -t %d", int(config.Duration.Seconds()))
	}
	
	// Always request JSON output for easier parsing
	cmd += " -J"
	
	// Additional arguments from config (use effective args based on role)
	effectiveArgs := config.GetEffectiveArgs()
	for key, value := range effectiveArgs {
		switch key {
		case "parallel_streams":
			if streams, ok := value.(int); ok && streams > 0 {
				cmd += fmt.Sprintf(" -P %d", streams)
			}
		case "window_size":
			if window, ok := value.(string); ok && window != "" {
				cmd += fmt.Sprintf(" -w %s", window)
			}
		case "reverse":
			if reverse, ok := value.(bool); ok && reverse {
				cmd += " -R"
			}
		case "bitrate":
			if bitrate, ok := value.(string); ok && bitrate != "" {
				cmd += fmt.Sprintf(" -b %s", bitrate)
			}
		case "interval":
			if interval, ok := value.(int); ok && interval > 0 {
				cmd += fmt.Sprintf(" -i %d", interval)
			}
		case "protocol":
			if protocol, ok := value.(string); ok && strings.ToLower(protocol) == "udp" {
				cmd += " -u"
			}
		case "ipv6":
			if ipv6, ok := value.(bool); ok && ipv6 {
				cmd += " -6"
			}
		case "ipv4":
			if ipv4, ok := value.(bool); ok && ipv4 {
				cmd += " -4"
			}
		case "bind_address":
			if bindAddr, ok := value.(string); ok && bindAddr != "" {
				cmd += fmt.Sprintf(" -B %s", bindAddr)
			}
		case "omit_seconds":
			if omit, ok := value.(int); ok && omit >= 0 {
				cmd += fmt.Sprintf(" -O %d", omit)
			}
		case "buffer_length":
			if buffer, ok := value.(string); ok && buffer != "" {
				cmd += fmt.Sprintf(" -l %s", buffer)
			}
		case "verbose":
			if verbose, ok := value.(bool); ok && verbose {
				cmd += " -V"
			}
		}
	}

	return envPrefix + cmd
}

// ParseMetrics extracts performance metrics from iperf3 JSON output
func (r *Iperf3Runner) ParseMetrics(result *Result) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}
	
	if result.Metrics == nil {
		result.Metrics = make(map[string]interface{})
	}
	
	output := result.Output
	
	// iperf3 with -J flag outputs JSON, but we also need to handle text fallback
	if strings.Contains(output, `"start"`) && strings.Contains(output, `"end"`) {
		// JSON output detected - parse key metrics
		r.parseJSONMetrics(result, output)
	} else {
		// Text output fallback - parse basic metrics
		r.parseTextMetrics(result, output)
	}
	
	return nil
}

// parseJSONMetrics extracts metrics from iperf3 JSON output
func (r *Iperf3Runner) parseJSONMetrics(result *Result, output string) {
	// Extract bits per second from sum_received or sum_sent
	// First try sum_received (for client output), then sum_sent
	if bps := r.extractNumericValue(output, `"bits_per_second"`); bps > 0 {
		result.Metrics["bandwidth_bps"] = bps
		result.Metrics["bandwidth_mbps"] = bps / 1e6
		result.Metrics["bandwidth_gbps"] = bps / 1e9
	}
	
	// Extract retransmits if present
	if strings.Contains(output, `"retransmits"`) {
		if retrans := r.extractNumericValue(output, `"retransmits"`); retrans >= 0 {
			result.Metrics["retransmits"] = int(retrans)
		}
	}
	
	// Extract parallel streams
	if strings.Contains(output, `"streams"`) {
		if streams := r.extractNumericValue(output, `"streams"`); streams > 0 {
			result.Metrics["parallel_streams"] = int(streams)
		}
	}
	
	// Extract actual test duration
	if strings.Contains(output, `"duration"`) {
		if duration := r.extractNumericValue(output, `"duration"`); duration > 0 {
			result.Metrics["actual_duration"] = duration
		}
	}
}

// parseTextMetrics extracts basic metrics from iperf3 text output
func (r *Iperf3Runner) parseTextMetrics(result *Result, output string) {
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		// Look for bandwidth lines (typically contain "Mbits/sec" or "Gbits/sec")
		if strings.Contains(line, "Mbits/sec") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Mbits/sec" && i > 0 {
					if bw, err := strconv.ParseFloat(fields[i-1], 64); err == nil {
						result.Metrics["bandwidth_mbps"] = bw
						result.Metrics["bandwidth_bps"] = bw * 1e6
						result.Metrics["bandwidth_gbps"] = bw / 1e3
					}
					break
				}
			}
		} else if strings.Contains(line, "Gbits/sec") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Gbits/sec" && i > 0 {
					if bw, err := strconv.ParseFloat(fields[i-1], 64); err == nil {
						result.Metrics["bandwidth_gbps"] = bw
						result.Metrics["bandwidth_bps"] = bw * 1e9
						result.Metrics["bandwidth_mbps"] = bw * 1e3
					}
					break
				}
			}
		}
		
		// Look for retransmits - typical format: "934 Mbits/sec   15   85.3 KBytes"
		if (strings.Contains(line, "Mbits/sec") || strings.Contains(line, "Gbits/sec")) && 
		   strings.Contains(line, "sec") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if (field == "Mbits/sec" || field == "Gbits/sec") && i+1 < len(fields) {
					// Next field after bandwidth unit might be retransmits
					if retrans, err := strconv.Atoi(fields[i+1]); err == nil && retrans >= 0 {
						result.Metrics["retransmits"] = retrans
						break
					}
				}
			}
		}
	}
}

// extractNumericValue extracts a numeric value associated with a JSON key
func (r *Iperf3Runner) extractNumericValue(text, key string) float64 {
	// Simple JSON value extraction - look for "key": value pattern
	keyPattern := `"` + strings.Trim(key, `"`) + `"`
	keyIndex := strings.Index(text, keyPattern)
	if keyIndex == -1 {
		return -1
	}
	
	// Find the colon after the key
	colonIndex := strings.Index(text[keyIndex:], ":")
	if colonIndex == -1 {
		return -1
	}
	
	// Extract the value part
	valueStart := keyIndex + colonIndex + 1
	valueText := strings.TrimSpace(text[valueStart:])
	
	// Find the end of the value (comma, closing brace, or newline)
	var valueEnd int
	for i, char := range valueText {
		if char == ',' || char == '}' || char == '\n' || char == '\r' {
			valueEnd = i
			break
		}
	}
	
	if valueEnd == 0 {
		valueEnd = len(valueText)
	}
	
	valueStr := strings.TrimSpace(valueText[:valueEnd])
	
	// Parse the numeric value
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	
	return -1
}