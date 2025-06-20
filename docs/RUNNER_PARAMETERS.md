# Runner Parameters Documentation

This document provides an overview of the configuration parameters for supported test runners. For comprehensive documentation including examples, troubleshooting, and best practices, see the individual runner guides:

- **[ib_send_bw Runner](runners/ib_send_bw.md)** - Complete InfiniBand send bandwidth testing guide
- **[iperf3 Runner](runners/iperf3.md)** - Complete TCP/UDP network testing guide

## Quick Reference

### ib_send_bw Runner

The `ib_send_bw` runner executes InfiniBand send bandwidth tests using the perftest suite.

> **📖 For complete documentation, see [ib_send_bw Runner Guide](runners/ib_send_bw.md)**

### Network Configuration

| Field | Type | Description |
|-------|------|-------------|
| `target_host` | string | Specific IP address for client to connect to (overrides SSH host) |
| `port` | int | Port number for the test |

**Separate SSH and InfiniBand Networks:**

In many HPC environments, SSH management traffic and InfiniBand traffic use different networks. You can configure different IPs:

```yaml
hosts:
  ib_server:
    ssh:
      host: "192.168.1.100"    # Management network for SSH
    runner:
      # Server doesn't need target_host (listens on all interfaces)
      
  ib_client:
    ssh:
      host: "192.168.1.101"    # Management network for SSH  
    runner:
      target_host: "10.0.0.100"  # InfiniBand network IP for testing
```

You can also override `target_host` per test:

```yaml
tests:
  - name: "Test with specific IB IP"
    client: "ib_client"
    server: "ib_server"
    config:
      target_host: "10.0.0.200"  # Override for this test
```

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

### Configuration Examples

#### Basic Configuration

```yaml
hosts:
  ib_server:
    ssh:
      host: "192.168.1.100"
    role: "server"
    runner:
      port: 18515
      args:
        size: 65536
        iterations: 1000
        ib_dev: "mlx5_0"
        gid_index: 3

  ib_client:
    ssh:
      host: "192.168.1.101"
    role: "client"
    runner:
      port: 18515
      target_host: "10.0.0.100"  # InfiniBand network
      args:
        size: 65536
        iterations: 1000
        ib_dev: "mlx5_0"
        gid_index: 3
```

#### Advanced Configuration

```yaml
tests:
  - name: "High Performance Test"
    client: "ib_client"
    server: "ib_server"
    config:
      duration: 60s
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

#### Connection Types

- **RC (Reliable Connection)**: Provides reliable, connection-oriented service
- **UC (Unreliable Connection)**: Provides unreliable, connection-oriented service  
- **UD (Unreliable Datagram)**: Provides unreliable, connectionless service

#### Message Sizes

Common message sizes for testing:
- `4096` (4KB) - Small messages
- `65536` (64KB) - Medium messages
- `1048576` (1MB) - Large messages
- `16777216` (16MB) - Very large messages

#### Performance Tuning

For optimal performance:

```yaml
args:
  size: 65536
  iterations: 10000
  tx_depth: 128
  rx_depth: 128
  mtu: 4096
  connection: "RC"
  ib_dev: "mlx5_0"
  gid_index: 3
  use_event: true
  report_gbits: true
```

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

### Output Metrics

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

### Troubleshooting

#### Common Issues

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

#### Debug Commands

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
```

### iperf3 Runner

The `iperf3` runner executes TCP/UDP network bandwidth tests using the industry-standard iperf3 tool.

> **📖 For complete documentation, see [iperf3 Runner Guide](runners/iperf3.md)**

### Network Configuration

| Field | Type | Description |
|-------|------|-------------|
| `target_host` | string | Specific IP address for client to connect to (overrides SSH host) |
| `port` | int | Port number for the test (default: 5201) |

**Separate SSH and Test Networks:**

Similar to InfiniBand testing, you can use different networks for SSH management and actual testing:

```yaml
hosts:
  tcp_server:
    ssh:
      host: "192.168.1.100"    # Management network for SSH
    runner:
      port: 5201
      
  tcp_client:
    ssh:
      host: "192.168.1.101"    # Management network for SSH  
    runner:
      port: 5201
      target_host: "10.0.0.100"  # Test network IP for iperf3
```

### iperf3 Arguments

Supported iperf3 arguments through `config.args`:

| Argument | Type | Description |
|----------|------|-------------|
| `parallel_streams` | int | Number of parallel streams (-P flag) |
| `window_size` | string | TCP window size (e.g., "2M", "128K") |
| `reverse` | bool | Measure server-to-client bandwidth |
| `bitrate` | string | Target bitrate limit (e.g., "1G", "100M") |
| `interval` | int | Measurement interval in seconds |
| `protocol` | string | Protocol type ("tcp" or "udp") |
| `ipv6` | bool | Force IPv6 usage |
| `ipv4` | bool | Force IPv4 usage |
| `bind_address` | string | Bind to specific local address |
| `omit_seconds` | int | Omit initial seconds (TCP slow start) |
| `buffer_length` | string | Buffer size (e.g., "128K", "1M") |
| `verbose` | bool | Enable verbose output |

### Configuration Examples

#### Basic TCP Test

```yaml
hosts:
  tcp_server:
    ssh:
      host: "192.168.1.100"
    role: "server"
    runner:
      port: 5201

  tcp_client:
    ssh:
      host: "192.168.1.101"
    role: "client"
    runner:
      port: 5201
      args:
        parallel_streams: 1
        window_size: "128K"
```

#### High-Performance Multi-Stream Test

```yaml
tests:
  - name: "High Performance TCP"
    client: "tcp_client"
    server: "tcp_server"
    config:
      duration: 60s
      args:
        parallel_streams: 8       # 8 parallel TCP streams
        window_size: "4M"        # Large TCP window
        buffer_length: "1M"      # Large buffer
        omit_seconds: 5          # Skip TCP slow start
        interval: 10             # Report every 10 seconds
```

#### UDP Bandwidth Test

```yaml
tests:
  - name: "UDP Bandwidth Test"
    client: "tcp_client"
    server: "tcp_server"
    config:
      duration: 30s
      args:
        protocol: "udp"
        bitrate: "1G"           # UDP requires bitrate limit
        parallel_streams: 1
```

### Protocol Differences

#### TCP vs UDP
- **TCP**: Reliable, connection-oriented, no bitrate limit needed
- **UDP**: Unreliable, connectionless, requires bitrate specification

#### Direction Testing
- **Upload (default)**: Client sends to server
- **Download**: Use `reverse: true` for server-to-client testing

### Command Line Mapping

The runner maps configuration parameters to iperf3 command line flags:

| Config Parameter | Command Flag | Example |
|------------------|--------------|---------|
| `parallel_streams` | `-P` | `-P 4` |
| `window_size` | `-w` | `-w 2M` |
| `reverse` | `-R` | `-R` |
| `bitrate` | `-b` | `-b 1G` |
| `interval` | `-i` | `-i 5` |
| `protocol: "udp"` | `-u` | `-u` |
| `ipv6` | `-6` | `-6` |
| `ipv4` | `-4` | `-4` |
| `bind_address` | `-B` | `-B 10.0.0.1` |
| `omit_seconds` | `-O` | `-O 5` |
| `buffer_length` | `-l` | `-l 128K` |
| `verbose` | `-V` | `-V` |

### Output Metrics

The runner extracts the following metrics from iperf3 output:

| Metric | Description |
|--------|-------------|
| `bandwidth_bps` | Bandwidth in bits per second |
| `bandwidth_mbps` | Bandwidth in megabits per second |
| `bandwidth_gbps` | Bandwidth in gigabits per second |
| `retransmits` | TCP retransmission count |
| `parallel_streams` | Number of parallel streams used |
| `actual_duration` | Actual test duration |

### Performance Tuning

#### For High-Speed Networks (10G+)
```yaml
args:
  parallel_streams: 8
  window_size: "8M"
  buffer_length: "1M"
  omit_seconds: 10
```

#### For WAN Testing
```yaml
args:
  parallel_streams: 1
  window_size: "2M"
  omit_seconds: 5
  interval: 5
```

#### For UDP Testing
```yaml
args:
  protocol: "udp"
  bitrate: "1G"          # Start conservative
  parallel_streams: 1
```

### Troubleshooting

#### Common Issues

1. **Connection refused**: Ensure iperf3 server is running and port is open
   ```bash
   # Test server connectivity
   telnet <server_ip> 5201
   ```

2. **Firewall blocking**: Check firewall rules for iperf3 port
   ```bash
   # Allow iperf3 port (example for ufw)
   sudo ufw allow 5201
   ```

3. **Low performance**: Try multiple streams and larger windows
   ```yaml
   args:
     parallel_streams: 4
     window_size: "2M"
   ```

4. **UDP packet loss**: Reduce bitrate for UDP tests
   ```yaml
   args:
     protocol: "udp"
     bitrate: "100M"  # Reduce from 1G
   ```

#### Debug Commands

```bash
# Test basic connectivity
iperf3 -c <server_ip> -t 10

# Test with multiple streams
iperf3 -c <server_ip> -P 4 -t 30

# Test UDP with specific bitrate
iperf3 -c <server_ip> -u -b 100M -t 10

# Test reverse direction
iperf3 -c <server_ip> -R -t 10
```

### Comparison with InfiniBand Tools

| Aspect | iperf3 | InfiniBand perftest |
|--------|--------|-------------------|
| **Protocol** | TCP/UDP | RDMA/IB verbs |
| **Network** | Any IP network | InfiniBand fabric |
| **Complexity** | Simple | Hardware-specific |
| **Output** | JSON/Text | Text only |
| **Use Case** | General networking | HPC/Storage |
| **Performance** | Up to network limit | RDMA performance |