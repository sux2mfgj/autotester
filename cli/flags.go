package cli

import (
	"flag"
	"time"
)

const (
	defaultConfigFile = "config.yaml"
	defaultTimeout    = 10 * time.Minute
)

// Flags represents command line flags
type Flags struct {
	ConfigFile *string
	Timeout    *time.Duration
	Verbose    *bool
	JSONOutput *bool
	Version    *bool
}

// NewFlags creates and parses command line flags
func NewFlags() *Flags {
	flags := &Flags{
		ConfigFile: flag.String("config", defaultConfigFile, "Path to configuration file"),
		Timeout:    flag.Duration("timeout", defaultTimeout, "Global timeout for all tests"),
		Verbose:    flag.Bool("verbose", false, "Enable verbose logging"),
		JSONOutput: flag.Bool("json", false, "Output results in JSON format"),
		Version:    flag.Bool("version", false, "Show version information"),
	}
	
	flag.Parse()
	return flags
}