# iperf3 Runner Documentation

The `iperf3` runner executes TCP/UDP network bandwidth tests using the industry-standard iperf3 tool.

## Overview

`iperf3` is a widely-used network testing tool that measures TCP and UDP bandwidth between two hosts. It provides comprehensive network performance metrics and is simpler to use than specialized tools, making it ideal for general network testing scenarios.

## Prerequisites

- `iperf3` installed on target hosts
- Network connectivity between test hosts
- Appropriate firewall configuration for test ports
- SSH access to target hosts

## Network Configuration

### Basic Configuration

| Field | Type | Description |
|-------|------|-------------|
| `target_host` | string | Specific IP address for client to connect to (overrides SSH host) |
| `port` | int | Port number for the test (default: 5201) |

### Separate SSH and Test Networks

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

### Per-Test Target Override

You can override `target_host` for specific tests:

```yaml
tests:
  - name: "Test with specific network IP"
    client: "tcp_client"
    server: "tcp_server"
    config:
      target_host: "10.0.0.200"  # Override for this test
      duration: 30s
```

## Parameters

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

## Configuration Examples

### Basic TCP Test

```yaml
name: "Basic TCP Bandwidth Test"
runner: "iperf3"
timeout: 5m

hosts:
  tcp_server:
    ssh:
      host: "192.168.1.100"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "server"
    runner:
      port: 5201

  tcp_client:
    ssh:
      host: "192.168.1.101"
      user: "testuser"
      key_path: "~/.ssh/id_rsa"
    role: "client"
    runner:
      port: 5201

tests:
  - name: "Basic TCP Bandwidth"
    client: "tcp_client"
    server: "tcp_server"
    config:
      duration: 30s
      args:
        parallel_streams: 1
        window_size: "128K"
```

### Multi-Stream High-Performance Test

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

### UDP Bandwidth Test

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

### Reverse Direction (Download) Test

```yaml
tests:
  - name: "Download Speed Test"
    client: "tcp_client"
    server: "tcp_server"
    config:
      duration: 30s
      args:
        parallel_streams: 4
        window_size: "2M"
        reverse: true           # Server sends to client
```

### Rate-Limited Test

```yaml
tests:
  - name: "Rate Limited Test"
    client: "tcp_client"
    server: "tcp_server"
    config:
      duration: 30s
      args:
        parallel_streams: 1
        bitrate: "100M"         # Limit to 100 Mbps
        window_size: "1M"
```

### IPv6 Test

```yaml
tests:
  - name: "IPv6 TCP Test"
    client: "tcp_client"
    server: "tcp_server"
    config:
      duration: 30s
      args:
        parallel_streams: 2
        window_size: "1M"
        ipv6: true              # Force IPv6
```

## Output Metrics

The runner extracts the following metrics from iperf3 output:

| Metric | Description |
|--------|-------------|
| `bandwidth_bps` | Bandwidth in bits per second |
| `bandwidth_mbps` | Bandwidth in megabits per second |
| `bandwidth_gbps` | Bandwidth in gigabits per second |
| `retransmits` | TCP retransmission count |
| `parallel_streams` | Number of parallel streams used |
| `actual_duration` | Actual test duration |

### Example Output

```json
{
  "bandwidth_bps": 9876543210,
  "bandwidth_mbps": 9876.54,
  "bandwidth_gbps": 9.88,
  "retransmits": 0,
  "parallel_streams": 4,
  "actual_duration": 30.0
}
```

## Protocol Differences

### TCP vs UDP

#### TCP (Transmission Control Protocol)
- **Reliability**: Reliable, connection-oriented
- **Flow Control**: Built-in congestion control
- **Bitrate**: No limit needed (uses available bandwidth)
- **Use Case**: Most network testing scenarios
- **Metrics**: Bandwidth, retransmissions, congestion window

#### UDP (User Datagram Protocol)  
- **Reliability**: Unreliable, connectionless
- **Flow Control**: No built-in flow control
- **Bitrate**: Requires bitrate specification to prevent flooding
- **Use Case**: Latency-sensitive applications, multicast
- **Metrics**: Bandwidth, packet loss, jitter

### Direction Testing

#### Upload (Default)
- **Direction**: Client sends to server
- **Command**: Standard iperf3 test
- **Use Case**: Testing upload bandwidth

#### Download (Reverse)
- **Direction**: Server sends to client  
- **Command**: Use `reverse: true`
- **Use Case**: Testing download bandwidth

## Performance Tuning

### For High-Speed Networks (10G+)

```yaml
args:
  parallel_streams: 8
  window_size: "8M"
  buffer_length: "1M"
  omit_seconds: 10
```

**Rationale:**
- Multiple streams overcome single-stream limitations
- Large windows handle high bandwidth-delay products
- Large buffers reduce system call overhead
- Omitting slow start gives more accurate results

### For WAN Testing

```yaml
args:
  parallel_streams: 1
  window_size: "2M"
  omit_seconds: 5
  interval: 5
```

**Rationale:**
- Single stream avoids congestion issues
- Moderate window size for variable latency
- Skip initial ramp-up period
- Frequent reporting for analysis

### For UDP Testing

```yaml
args:
  protocol: "udp"
  bitrate: "1G"          # Start conservative
  parallel_streams: 1
```

**Rationale:**
- Conservative bitrate prevents packet loss
- Single stream for simplicity
- Gradually increase bitrate to find limits

### For Low-Latency Networks

```yaml
args:
  parallel_streams: 4
  window_size: "512K"
  buffer_length: "64K"
  interval: 1
```

## Troubleshooting

### Common Issues

#### 1. Connection Refused
**Symptoms**: `iperf3: error - unable to connect to server`
**Solutions**:
```bash
# Test server connectivity
telnet <server_ip> 5201

# Check if iperf3 server is running
ps aux | grep iperf3

# Manually start server for testing
iperf3 -s -p 5201
```

#### 2. Firewall Blocking
**Symptoms**: Connection timeouts
**Solutions**:
```bash
# Allow iperf3 port (example for ufw)
sudo ufw allow 5201

# Check firewall status
sudo ufw status

# For iptables
sudo iptables -A INPUT -p tcp --dport 5201 -j ACCEPT
```

#### 3. Low Performance
**Symptoms**: Bandwidth much lower than expected
**Solutions**:
```yaml
args:
  parallel_streams: 4     # Try multiple streams
  window_size: "2M"      # Increase window size
  buffer_length: "1M"    # Increase buffer size
```

#### 4. UDP Packet Loss
**Symptoms**: High packet loss in UDP tests
**Solutions**:
```yaml
args:
  protocol: "udp"
  bitrate: "100M"        # Reduce from 1G
  buffer_length: "256K"  # Increase buffer
```

#### 5. Variable Results
**Symptoms**: Inconsistent bandwidth measurements
**Solutions**:
- Run multiple test iterations
- Use longer test duration
- Check for background network traffic
- Verify CPU and memory resources

### Debug Commands

```bash
# Test basic connectivity
iperf3 -c <server_ip> -t 10

# Test with multiple streams
iperf3 -c <server_ip> -P 4 -t 30

# Test UDP with specific bitrate
iperf3 -c <server_ip> -u -b 100M -t 10

# Test reverse direction
iperf3 -c <server_ip> -R -t 10

# Test with larger window
iperf3 -c <server_ip> -w 2M -t 10

# JSON output for parsing
iperf3 -c <server_ip> -J -t 10
```

### Network Analysis

```bash
# Check network interface statistics
ip -s link

# Monitor network traffic
iftop -i eth0

# Check TCP settings
sysctl net.core.rmem_max
sysctl net.core.wmem_max

# Check network latency
ping <server_ip>

# Check routing
traceroute <server_ip>
```

## Best Practices

### Test Design
1. **Run multiple iterations** for statistical significance
2. **Use appropriate test duration** (30s minimum for TCP)
3. **Test both directions** (upload and download)
4. **Document network configuration** for reproducibility

### Performance Optimization
1. **Tune TCP window sizes** for your network
2. **Use multiple streams** for high-speed links
3. **Monitor system resources** during tests
4. **Avoid background traffic** during testing

### Result Interpretation
1. **Consider network overhead** in results
2. **Account for protocol differences** (TCP vs UDP)
3. **Factor in system limitations** (CPU, memory)
4. **Compare with theoretical maximums**

## Integration Examples

### Combining with InfiniBand Tests

```yaml
tests:
  - name: "InfiniBand Performance"
    runner: "ib_send_bw"
    # ... IB configuration
    
  - name: "TCP Performance"
    runner: "iperf3"
    # ... TCP configuration
    
  - name: "Performance Comparison"
    # Compare IB vs TCP results
```

### Automated Testing

```bash
# Run comprehensive network tests
./perf-runner -config network-suite.yaml -json > results.json

# Parse results for analysis
jq '.tests[] | {name: .name, bandwidth: .metrics.bandwidth_gbps}' results.json
```

### Monitoring Integration

The JSON output format makes it easy to integrate with monitoring systems:

```json
{
  "test_name": "High Performance TCP",
  "success": true,
  "metrics": {
    "bandwidth_gbps": 9.87,
    "bandwidth_mbps": 9870,
    "retransmits": 0,
    "parallel_streams": 4
  }
}
```

## Comparison with InfiniBand Tools

| Aspect | iperf3 | InfiniBand perftest |
|--------|--------|-------------------|
| **Protocol** | TCP/UDP | RDMA/IB verbs |
| **Network** | Any IP network | InfiniBand fabric |
| **Complexity** | Simple | Hardware-specific |
| **Output** | JSON/Text | Text only |
| **Use Case** | General networking | HPC/Storage |
| **Performance** | Up to network limit | RDMA performance |
| **Prerequisites** | Basic network | InfiniBand hardware |
| **Debugging** | Standard network tools | IB-specific tools |

## Advanced Configuration

### Custom Port Range

```yaml
hosts:
  server:
    runner:
      port: 5555        # Custom port
  client:
    runner:
      port: 5555
```

### Multiple Network Interfaces

```yaml
tests:
  - name: "Interface 1 Test"
    config:
      args:
        bind_address: "10.0.1.100"
        
  - name: "Interface 2 Test"
    config:
      args:
        bind_address: "10.0.2.100"
```

### Load Testing

```yaml
tests:
  - name: "Sustained Load Test"
    config:
      duration: 300s    # 5 minutes
      args:
        parallel_streams: 16
        interval: 30    # Report every 30 seconds
    repeat: 10          # Run 10 times
    delay: 60s          # 1 minute between runs
```