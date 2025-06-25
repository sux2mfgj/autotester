
- [x] Enhancing Network Performance Measurement: Adding an Intermediate Node
We currently have a function to manage and execute performance measurements between two points. We'd like to extend this to enable performance measurement in a system where communication occurs between two endpoints with an intermediate point in between. The two endpoints will remain the same as before, but the intermediate point will run an application that forwards packets.

**COMPLETED**: Implemented 3-node topology support with the following features:
- Added `intermediate` field to test scenarios in YAML configuration
- Support for "intermediate" role in both ib_send_bw and iperf3 runners  
- 3-node execution flow: Server → Intermediate → Client
- Intermediate nodes run packet forwarding applications (socat for iperf3, custom forwarding for ib_send_bw)
- Comprehensive validation for 3-node configurations
- Example configurations provided for both InfiniBand and TCP/UDP scenarios
- Results include intermediate node metrics and commands
- Backward compatible with existing 2-node configurations
- [x] create a sample configuration that use a dpdk testpmd as intermediate node

**COMPLETED**: Created examples/example-dpdk-testpmd.yaml with DPDK testpmd configuration for packet forwarding including:
- DPDK-specific parameters (cores, memory channels, huge pages, ports)
- Multiple test scenarios (single/multi-stream, TCP/UDP)
- High-performance configuration options
- Baseline comparison without DPDK forwarder

- [x] `sudo` support at remote host as configurable

**COMPLETED**: Implemented configurable sudo support for remote hosts with the following features:
- Added `use_sudo` field to SSH configuration structure
- Commands are automatically prefixed with `sudo` when `use_sudo: true` is set
- Applied to both `ExecuteCommand` and `ExecuteCommandAsync` methods
- Per-host configuration allows mixed environments (some hosts with sudo, others without)
- Documentation added to CLAUDE.md with configuration examples and requirements
- Example configuration file created at `examples/example-sudo-support.yaml`
- Backward compatible with existing configurations (default is `use_sudo: false`)
