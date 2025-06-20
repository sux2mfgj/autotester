package runner

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IbSendBwRunner implements the Runner interface for ib_send_bw
type IbSendBwRunner struct {
	executablePath string
}

// NewIbSendBwRunner creates a new ib_send_bw runner
func NewIbSendBwRunner(executablePath string) *IbSendBwRunner {
	if executablePath == "" {
		executablePath = "ib_send_bw"
	}
	return &IbSendBwRunner{
		executablePath: executablePath,
	}
}

// Name returns the name of the runner
func (r *IbSendBwRunner) Name() string {
	return "ib_send_bw"
}

// SupportsRole returns true if the runner supports the given role
func (r *IbSendBwRunner) SupportsRole(role string) bool {
	return role == "client" || role == "server"
}

// Validate checks if the configuration is valid for ib_send_bw
func (r *IbSendBwRunner) Validate(config Config) error {
	if !r.SupportsRole(config.Role) {
		return fmt.Errorf("unsupported role: %s", config.Role)
	}
	
	// For ib_send_bw, client needs a target host but server doesn't
	if config.Role == "client" {
		if config.TargetHost == "" && config.Host == "" {
			return fmt.Errorf("target_host or host is required for client role")
		}
	}
	
	return nil
}

// Run executes ib_send_bw with the given configuration
func (r *IbSendBwRunner) Run(ctx context.Context, config Config) (*Result, error) {
	if err := r.Validate(config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	args := r.buildArgs(config)
	
	startTime := time.Now()
	cmd := exec.CommandContext(ctx, r.executablePath, args...)
	
	// Set environment variables if specified
	if len(config.Env) > 0 {
		env := make([]string, 0, len(config.Env))
		for k, v := range config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}
	
	output, err := cmd.CombinedOutput()
	endTime := time.Now()
	
	result := &Result{
		Success:   err == nil,
		Output:    string(output),
		Duration:  endTime.Sub(startTime),
		StartTime: startTime,
		EndTime:   endTime,
		Metrics:   make(map[string]interface{}),
	}
	
	if err != nil {
		result.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	}
	
	// Parse metrics from output
	if result.Success {
		r.parseMetrics(result)
	}
	
	return result, nil
}

// buildArgs constructs the command line arguments for ib_send_bw
func (r *IbSendBwRunner) buildArgs(config Config) []string {
	args := make([]string, 0)
	
	// Server mode doesn't need a host argument, client does
	if config.Role == "client" {
		// Use TargetHost if specified, otherwise fall back to Host
		targetHost := config.TargetHost
		if targetHost == "" {
			targetHost = config.Host
		}
		args = append(args, targetHost)
	}
	
	// Port (if specified)
	if config.Port > 0 {
		args = append(args, "-p", strconv.Itoa(config.Port))
	}
	
	// Duration (if specified) - ib_send_bw uses -D flag
	if config.Duration > 0 {
		args = append(args, "-D", strconv.Itoa(int(config.Duration.Seconds())))
	}
	
	// Additional arguments from config
	for key, value := range config.Args {
		switch key {
		case "size":
			// Message size in bytes
			if size, ok := value.(int); ok {
				args = append(args, "-s", strconv.Itoa(size))
			} else if sizeStr, ok := value.(string); ok {
				args = append(args, "-s", sizeStr)
			}
		case "iterations":
			// Number of iterations
			if iter, ok := value.(int); ok {
				args = append(args, "-n", strconv.Itoa(iter))
			}
		case "tx_depth":
			// Send queue depth
			if depth, ok := value.(int); ok {
				args = append(args, "-t", strconv.Itoa(depth))
			}
		case "rx_depth":
			// Receive queue depth  
			if depth, ok := value.(int); ok {
				args = append(args, "-r", strconv.Itoa(depth))
			}
		case "mtu":
			// MTU size
			if mtu, ok := value.(int); ok {
				args = append(args, "-m", strconv.Itoa(mtu))
			}
		case "qp":
			// Number of QPs
			if qp, ok := value.(int); ok {
				args = append(args, "-q", strconv.Itoa(qp))
			}
		case "connection":
			// Connection type (RC/UC/UD)
			if conn, ok := value.(string); ok {
				args = append(args, "-c", conn)
			}
		case "inline":
			// Inline size
			if inline, ok := value.(int); ok {
				args = append(args, "-I", strconv.Itoa(inline))
			}
		case "use_event":
			// Use event completion
			if useEvent, ok := value.(bool); ok && useEvent {
				args = append(args, "-e")
			}
		case "bidirectional":
			// Bidirectional test
			if bidir, ok := value.(bool); ok && bidir {
				args = append(args, "-b")
			}
		case "report_cycles":
			// Report CPU cycles
			if cycles, ok := value.(bool); ok && cycles {
				args = append(args, "-C")
			}
		case "report_histogram":
			// Report latency histogram
			if hist, ok := value.(bool); ok && hist {
				args = append(args, "-H")
			}
		case "cpu_freq":
			// CPU frequency for cycles calculation
			if freq, ok := value.(float64); ok {
				args = append(args, "-F", fmt.Sprintf("%.2f", freq))
			}
		case "ib_dev":
			// InfiniBand device
			if dev, ok := value.(string); ok {
				args = append(args, "-d", dev)
			}
		case "gid_index":
			// GID index
			if gid, ok := value.(int); ok {
				args = append(args, "-x", strconv.Itoa(gid))
			}
		case "sl":
			// Service level
			if sl, ok := value.(int); ok {
				args = append(args, "-S", strconv.Itoa(sl))
			}
		case "odp":
			// Use On Demand Paging
			if odp, ok := value.(bool); ok && odp {
				args = append(args, "-o")
			}
		case "report_gbits":
			// Report in Gb/sec instead of MB/sec
			if gbits, ok := value.(bool); ok && gbits {
				args = append(args, "-R")
			}
		}
	}
	
	return args
}

// parseMetrics extracts performance metrics from ib_send_bw output
func (r *IbSendBwRunner) parseMetrics(result *Result) {
	output := result.Output
	lines := strings.Split(output, "\n")
	
	// Look for the results table
	for i, line := range lines {
		// ib_send_bw typically outputs a table with headers like:
		// #bytes     #iterations    BW peak[MB/sec]    BW average[MB/sec]   MsgRate[Mpps]
		if strings.Contains(line, "#bytes") && strings.Contains(line, "BW") {
			// Parse the data line (usually the next line)
			if i+1 < len(lines) {
				dataLine := strings.TrimSpace(lines[i+1])
				if dataLine != "" && !strings.HasPrefix(dataLine, "#") {
					r.parseResultLine(dataLine, result)
				}
			}
			break
		}
		
		// Also look for single result lines (some versions output differently)
		if strings.Contains(line, "MB/sec") || strings.Contains(line, "Gb/sec") {
			r.parseResultLine(line, result)
		}
	}
	
	// Parse additional information
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Parse connection information
		if strings.Contains(line, "Connection type:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				result.Metrics["connection_type"] = strings.TrimSpace(parts[1])
			}
		}
		
		// Parse MTU
		if strings.Contains(line, "MTU:") {
			mtuRegex := regexp.MustCompile(`MTU:\s*(\d+)`)
			if matches := mtuRegex.FindStringSubmatch(line); len(matches) > 1 {
				if mtu, err := strconv.Atoi(matches[1]); err == nil {
					result.Metrics["mtu"] = mtu
				}
			}
		}
		
		// Parse message size
		if strings.Contains(line, "Message size:") {
			sizeRegex := regexp.MustCompile(`Message size:\s*(\d+)`)
			if matches := sizeRegex.FindStringSubmatch(line); len(matches) > 1 {
				if size, err := strconv.Atoi(matches[1]); err == nil {
					result.Metrics["message_size"] = size
				}
			}
		}
		
		// Parse QP information
		if strings.Contains(line, "Number of qps:") {
			qpRegex := regexp.MustCompile(`Number of qps:\s*(\d+)`)
			if matches := qpRegex.FindStringSubmatch(line); len(matches) > 1 {
				if qps, err := strconv.Atoi(matches[1]); err == nil {
					result.Metrics["num_qps"] = qps
				}
			}
		}
	}
}

// parseResultLine parses a result line containing bandwidth measurements
func (r *IbSendBwRunner) parseResultLine(line string, result *Result) {
	// Split by whitespace
	fields := strings.Fields(line)
	
	if len(fields) >= 4 {
		// Try to parse numerical fields
		for i, field := range fields {
			if value, err := strconv.ParseFloat(field, 64); err == nil {
				switch i {
				case 0:
					// Usually bytes
					if value > 0 {
						result.Metrics["bytes"] = int64(value)
					}
				case 1:
					// Usually iterations
					if value > 0 {
						result.Metrics["iterations"] = int64(value)
					}
				case 2:
					// Usually BW peak
					if value > 0 {
						result.Metrics["bandwidth_peak_mbps"] = value
						// Convert to bits per second
						result.Metrics["bandwidth_peak_bps"] = value * 1e6 * 8
					}
				case 3:
					// Usually BW average
					if value > 0 {
						result.Metrics["bandwidth_average_mbps"] = value
						result.Metrics["bandwidth_average_bps"] = value * 1e6 * 8
					}
				case 4:
					// Usually message rate
					if value > 0 {
						result.Metrics["message_rate_mpps"] = value
						result.Metrics["message_rate_pps"] = value * 1e6
					}
				}
			}
		}
	}
	
	// Parse bandwidth with units
	bwRegex := regexp.MustCompile(`(\d+\.?\d*)\s*(MB/sec|Gb/sec|GB/sec)`)
	if matches := bwRegex.FindStringSubmatch(line); len(matches) >= 3 {
		if bw, err := strconv.ParseFloat(matches[1], 64); err == nil {
			unit := matches[2]
			switch unit {
			case "MB/sec":
				result.Metrics["bandwidth_mbps"] = bw
				result.Metrics["bandwidth_bps"] = bw * 1e6 * 8
			case "GB/sec":
				result.Metrics["bandwidth_gbps"] = bw
				result.Metrics["bandwidth_bps"] = bw * 1e9 * 8
			case "Gb/sec":
				result.Metrics["bandwidth_gbps"] = bw
				result.Metrics["bandwidth_bps"] = bw * 1e9
			}
			result.Metrics["bandwidth_readable"] = matches[0]
		}
	}
	
	// Parse message rate
	rateRegex := regexp.MustCompile(`(\d+\.?\d*)\s*(Mpps|Kpps|pps)`)
	if matches := rateRegex.FindStringSubmatch(line); len(matches) >= 3 {
		if rate, err := strconv.ParseFloat(matches[1], 64); err == nil {
			unit := matches[2]
			switch unit {
			case "Mpps":
				result.Metrics["message_rate_mpps"] = rate
				result.Metrics["message_rate_pps"] = rate * 1e6
			case "Kpps":
				result.Metrics["message_rate_kpps"] = rate
				result.Metrics["message_rate_pps"] = rate * 1e3
			case "pps":
				result.Metrics["message_rate_pps"] = rate
			}
		}
	}
}