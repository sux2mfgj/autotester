# ib_send_bw Runner Documentation

The `ib_send_bw` runner executes InfiniBand send bandwidth tests using the perftest suite.

## Overview

`ib_send_bw` is part of the perftest suite and measures InfiniBand send bandwidth between two hosts. It supports various InfiniBand connection types (RC, UC, UD) and provides detailed performance metrics including bandwidth and message rates.

## Prerequisites

- InfiniBand hardware and drivers properly configured
- `perftest` suite installed on target hosts
- `ib_send_bw` executable available in PATH
- Active InfiniBand fabric between test hosts

## Network Configuration

### Basic Configuration

| Field | Type | Description |
|-------|------|-------------|
| `target_host` | string | Specific IP address for client to connect to (overrides SSH host) |
| `port` | int | Port number for the test (default: 18515) |

### Separate SSH and InfiniBand Networks

In many HPC environments, SSH management traffic and InfiniBand traffic use different networks:

```yaml
hosts:
  ib_server:
    ssh:
      host: "192.168.1.100"    # Management network for SSH
    runner:
      port: 18515
      # Server doesn't need target_host (listens on all interfaces)
      
  ib_client:
    ssh:
      host: "192.168.1.101"    # Management network for SSH  
    runner:
      port: 18515
      target_host: "10.0.0.100"  # InfiniBand network IP for testing
```

### Per-Test Target Override

You can override `target_host` for specific tests:

```yaml
tests:
  - name: "Test with specific IB IP"
    client: "ib_client"
    server: "ib_server"
    config:
      target_host: "10.0.0.200"  # Override for this test
      duration: 30s
```

## Parameters

### ib_send_bw Arguments

Supported ib_send_bw arguments through `config.args`:

| Argument | Type | Description |
|----------|------|-------------|
| `size` | int/string | Message size in bytes (e.g., 65536) |
| `iterations` | int | Number of iterations to run |
| `tx_depth` | int | Send queue depth |
| `rx_depth` | int | Receive queue depth |
| `mtu` | int | MTU size (e.g., 4096) |
| `qp` | int | Number of Queue Pairs |
| `connection` | string | Connection type (RC/UC/UD) |
| `inline` | int | Inline message size |
| `ib_dev` | string | InfiniBand device name (e.g., "mlx5_0") |
| `gid_index` | int | GID index to use |
| `sl` | int | Service level |
| `cpu_freq` | float | CPU frequency for cycle calculations |
| `use_event` | bool | Use event completion |
| `bidirectional` | bool | Bidirectional test |
| `report_cycles` | bool | Report CPU cycles |
| `report_histogram` | bool | Report latency histogram |
| `odp` | bool | Use On Demand Paging |
| `report_gbits` | bool | Report in Gb/sec instead of MB/sec |

### Command Line Mapping

The runner maps configuration parameters to ib_send_bw command line flags:

| Config Parameter | Command Flag | Example |
|------------------|--------------|---------|
| `size` | `-s` | `-s 65536` |
| `iterations` | `-n` | `-n 1000` |
| `tx_depth` | `-t` | `-t 128` |
| `rx_depth` | `-r` | `-r 128` |
| `mtu` | `-m` | `-m 4096` |
| `qp` | `-q` | `-q 4` |
| `connection` | `-c` | `-c RC` |
| `inline` | `-I` | `-I 64` |
| `ib_dev` | `-d` | `-d mlx5_0` |
| `gid_index` | `-x` | `-x 3` |
| `sl` | `-S` | `-S 0` |
| `cpu_freq` | `-F` | `-F 2.40` |
| `use_event` | `-e` | `-e` |
| `bidirectional` | `-b` | `-b` |
| `report_cycles` | `-C` | `-C` |
| `report_histogram` | `-H` | `-H` |
| `odp` | `-o` | `-o` |
| `report_gbits` | `-R` | `-R` |

## Configuration Examples

### Basic InfiniBand Test

```yaml
name: "Basic InfiniBand Send Bandwidth Test"
runner: "ib_send_bw"
timeout: 5m

hosts:
  ib_server:
    ssh:
      host: "192.168.1.100"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "server"
    runner:
      port: 18515
      args:
        ib_dev: "mlx5_0"
        gid_index: 3

  ib_client:
    ssh:
      host: "192.168.1.101"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "client"
    runner:
      port: 18515
      target_host: "10.0.0.100"
      args:
        ib_dev: "mlx5_0"
        gid_index: 3

tests:
  - name: "Basic IB Send BW Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 30s
      args:
        size: 65536
        iterations: 1000
        connection: "RC"
```

### High-Performance Configuration

```yaml
tests:
  - name: "High Performance IB Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 120s
      args:
        size: 1048576        # 1MB messages
        iterations: 5000
        tx_depth: 256
        rx_depth: 256
        connection: "RC"
        mtu: 4096
        qp: 4
        bidirectional: true
        report_gbits: true
        use_event: true
```

### Different Connection Types

```yaml
tests:
  - name: "RC Connection Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 30s
      args:
        size: 65536
        connection: "RC"    # Reliable Connection
        
  - name: "UC Connection Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 30s
      args:
        size: 65536
        connection: "UC"    # Unreliable Connection
        
  - name: "UD Connection Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 30s
      args:
        size: 4096         # UD typically uses smaller messages
        connection: "UD"    # Unreliable Datagram
```

## Output Metrics

The runner extracts the following metrics from ib_send_bw output:

| Metric | Description |
|--------|-------------|
| `bandwidth_mbps` | Bandwidth in MB/sec |
| `bandwidth_gbps` | Bandwidth in Gb/sec |
| `bandwidth_bps` | Bandwidth in bits per second |
| `bandwidth_readable` | Human-readable bandwidth string |
| `message_rate_pps` | Message rate in packets per second |
| `message_rate_mpps` | Message rate in millions of packets per second |
| `bytes` | Message size in bytes |
| `iterations` | Number of iterations completed |
| `connection_type` | Connection type used |
| `mtu` | MTU size |
| `message_size` | Message size |
| `num_qps` | Number of queue pairs |

### Example Output

```json
{
  "bandwidth_mbps": 12345.67,
  "bandwidth_gbps": 98.77,
  "bandwidth_bps": 98765432100,
  "message_rate_mpps": 0.18,
  "bytes": 65536,
  "iterations": 1000,
  "connection_type": "RC"
}
```

## Connection Types

### RC (Reliable Connection)
- **Description**: Provides reliable, connection-oriented service
- **Use Case**: Most common for bandwidth testing
- **Features**: Guaranteed delivery, flow control
- **Message Sizes**: Any size up to MTU

### UC (Unreliable Connection)  
- **Description**: Provides unreliable, connection-oriented service
- **Use Case**: Lower overhead than RC
- **Features**: No acknowledgments, higher performance potential
- **Message Sizes**: Any size up to MTU

### UD (Unreliable Datagram)
- **Description**: Provides unreliable, connectionless service
- **Use Case**: Multicast and broadcast scenarios
- **Features**: Lowest overhead, no connection state
- **Message Sizes**: Limited by MTU minus headers

## Performance Tuning

### For Maximum Bandwidth

```yaml
args:
  size: 1048576          # Large messages (1MB)
  tx_depth: 256          # Deep send queue
  rx_depth: 256          # Deep receive queue
  qp: 4                  # Multiple queue pairs
  connection: "RC"       # Reliable connection
  mtu: 4096             # Maximum MTU
  use_event: true       # Event-driven completion
  bidirectional: true   # Both directions
```

### For Low Latency

```yaml
args:
  size: 4096            # Smaller messages
  tx_depth: 1           # Shallow queues
  rx_depth: 1
  connection: "RC"
  inline: 64            # Inline small messages
  use_event: false      # Polling completion
```

### For Different Workloads

```yaml
# High throughput
args:
  size: 65536
  tx_depth: 128
  rx_depth: 128
  qp: 8
  connection: "RC"

# Mixed workload  
args:
  size: 16384
  tx_depth: 64
  rx_depth: 64
  qp: 2
  connection: "RC"
```

## Troubleshooting

### Common Issues

1. **Device not found**: Ensure `ib_dev` matches your InfiniBand device name
   ```bash
   # List available devices
   ibstat
   ```

2. **GID index error**: Verify the GID index exists for your device
   ```bash
   # List GIDs for device
   ibstat mlx5_0
   ```

3. **Port errors**: Check that the specified port is available and not in use

4. **Connection failures**: Verify InfiniBand connectivity between hosts
   ```bash
   # Test basic IB connectivity
   ibping -S  # on server
   ibping -c 10 <server_ip>  # on client
   ```

5. **Low performance**: Check for:
   - Proper MTU configuration
   - Queue depth settings
   - CPU affinity
   - NUMA topology

### Debug Commands

```bash
# Check IB device status
ibstat

# List available GIDs
show_gids

# Test basic connectivity
ibping -S    # Server
ibping <ip>  # Client

# Check IB port status
ibstatus

# Manual ib_send_bw test
ib_send_bw -d mlx5_0 -x 3    # Server
ib_send_bw -d mlx5_0 -x 3 <server_ip>  # Client
```

### Performance Analysis

1. **Check CPU utilization** during tests
2. **Monitor memory bandwidth** usage  
3. **Verify interrupt distribution** across cores
4. **Check for packet drops** in network stack
5. **Analyze queue pair utilization**

### Hardware Considerations

- **InfiniBand Generation**: EDR (100G), HDR (200G), NDR (400G)
- **Cable Quality**: Ensure proper cable specifications
- **Switch Configuration**: Check switch port settings
- **Firmware Versions**: Keep HCA firmware updated
- **PCIe Bandwidth**: Verify sufficient PCIe lanes and speed

## Best Practices

1. **Use consistent hardware** across test hosts
2. **Disable CPU frequency scaling** during tests
3. **Set proper CPU affinity** for test processes
4. **Use dedicated InfiniBand networks** for testing
5. **Monitor system resources** during tests
6. **Run multiple iterations** for statistical significance
7. **Document hardware configuration** for reproducibility

## Integration with Other Tools

The ib_send_bw runner works alongside other tools in the framework:

- **iperf3**: For TCP/UDP comparison testing
- **Future IB tools**: ib_read_bw, ib_write_bw, ib_send_lat
- **SSH coordination**: Seamless client-server orchestration
- **JSON output**: Integration with monitoring systems