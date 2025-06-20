package coordinator

import (
	"fmt"

	"tester/runner"
)

// CommandBuilder constructs command lines for remote execution
type CommandBuilder struct{}

// NewCommandBuilder creates a new command builder
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{}
}

// BuildCommand constructs the command line for remote execution
func (b *CommandBuilder) BuildCommand(r runner.Runner, config *runner.Config) string {
	switch r.Name() {
	case "ib_send_bw":
		return b.buildIbSendBwCommand(config)
	default:
		return ""
	}
}


// buildIbSendBwCommand builds the ib_send_bw command line
func (b *CommandBuilder) buildIbSendBwCommand(config *runner.Config) string {
	cmd := "ib_send_bw"
	
	// Server mode doesn't need a host argument, client does
	if config.Role == "client" {
		// Use TargetHost if specified, otherwise fall back to Host
		targetHost := config.TargetHost
		if targetHost == "" {
			targetHost = config.Host
		}
		cmd += fmt.Sprintf(" %s", targetHost)
	}
	
	// Port (if specified)
	if config.Port > 0 {
		cmd += fmt.Sprintf(" -p %d", config.Port)
	}
	
	// Duration (if specified) - ib_send_bw uses -D flag
	if config.Duration > 0 {
		cmd += fmt.Sprintf(" -D %d", int(config.Duration.Seconds()))
	}
	
	// Additional arguments from config
	for key, value := range config.Args {
		switch key {
		case "size":
			cmd += fmt.Sprintf(" -s %v", value)
		case "iterations":
			cmd += fmt.Sprintf(" -n %v", value)
		case "tx_depth":
			cmd += fmt.Sprintf(" -t %v", value)
		case "rx_depth":
			cmd += fmt.Sprintf(" -r %v", value)
		case "mtu":
			cmd += fmt.Sprintf(" -m %v", value)
		case "qp":
			cmd += fmt.Sprintf(" -q %v", value)
		case "connection":
			cmd += fmt.Sprintf(" -c %v", value)
		case "inline":
			cmd += fmt.Sprintf(" -I %v", value)
		case "gid_index":
			cmd += fmt.Sprintf(" -x %v", value)
		case "sl":
			cmd += fmt.Sprintf(" -S %v", value)
		case "cpu_freq":
			cmd += fmt.Sprintf(" -F %v", value)
		case "use_event":
			if useEvent, ok := value.(bool); ok && useEvent {
				cmd += " -e"
			}
		case "bidirectional":
			if bidir, ok := value.(bool); ok && bidir {
				cmd += " -b"
			}
		case "report_cycles":
			if cycles, ok := value.(bool); ok && cycles {
				cmd += " -C"
			}
		case "report_histogram":
			if hist, ok := value.(bool); ok && hist {
				cmd += " -H"
			}
		case "odp":
			if odp, ok := value.(bool); ok && odp {
				cmd += " -o"
			}
		case "report_gbits":
			if gbits, ok := value.(bool); ok && gbits {
				cmd += " -R"
			}
		}
	}
	
	return cmd
}