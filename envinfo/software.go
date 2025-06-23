package envinfo

import (
	"context"
	"strings"
)

// SoftwareVersions represents versions of relevant software
type SoftwareVersions struct {
	IbSendBw   string `json:"ib_send_bw,omitempty"`
	Iperf3     string `json:"iperf3,omitempty"`
	DPDK       string `json:"dpdk,omitempty"`
	Socat      string `json:"socat,omitempty"`
	SSHVersion string `json:"ssh_version,omitempty"`
	GCC        string `json:"gcc,omitempty"`
	Python     string `json:"python,omitempty"`
	Git        string `json:"git,omitempty"`
}

// SoftwareModule collects software version information
type SoftwareModule struct{}

// NewSoftwareModule creates a new software version module
func NewSoftwareModule() *SoftwareModule {
	return &SoftwareModule{}
}

// Name returns the module name
func (m *SoftwareModule) Name() string {
	return "software"
}

// Description returns the module description
func (m *SoftwareModule) Description() string {
	return "Collects software version information for common tools and libraries"
}

// IsAvailable checks if the module can run
func (m *SoftwareModule) IsAvailable(ctx context.Context, executor CommandExecutor) bool {
	// Software version collection should always be available
	return true
}

// Collect gathers software version information
func (m *SoftwareModule) Collect(ctx context.Context, executor CommandExecutor) (interface{}, error) {
	versions := &SoftwareVersions{}

	// Define software checks with their version commands
	softwareChecks := map[string]*string{
		"ib_send_bw --version 2>&1 | head -1":        &versions.IbSendBw,
		"iperf3 --version 2>&1 | head -1":            &versions.Iperf3,
		"dpdk-testpmd --version 2>&1 | head -1":      &versions.DPDK,
		"socat -V 2>&1 | head -1":                    &versions.Socat,
		"ssh -V 2>&1":                                &versions.SSHVersion,
		"gcc --version 2>&1 | head -1":               &versions.GCC,
		"python3 --version 2>&1":                     &versions.Python,
		"git --version 2>&1":                         &versions.Git,
	}

	// Execute each check
	for cmd, target := range softwareChecks {
		if version, err := executor.Execute(ctx, cmd); err == nil {
			*target = strings.TrimSpace(version)
		}
		// Note: We don't treat missing software as an error
	}

	return versions, nil
}

// Auto-register this module
func init() {
	RegisterModule("software", func() Module {
		return NewSoftwareModule()
	})
}