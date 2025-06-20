# Runner Parameters Documentation

This document describes the configuration parameters for each supported test runner.

## ib_send_bw Runner

The `ib_send_bw` runner executes InfiniBand send bandwidth tests using the perftest suite.

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
| `ib_dev` | `--ib-dev` | `--ib-dev mlx5_0` |
| `gid_index` | `--gid-index` | `--gid-index 3` |
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