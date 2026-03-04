# PolyServer Enhanced - Stress Testing & Monitoring Edition

An enhanced version of PolyServer with built-in stress testing, real-time monitoring, and bandwidth optimization improvements.

## What's New

### 1. **Metrics & Monitoring** 📊
Built-in comprehensive metrics tracking:
- Real-time bandwidth monitoring (Mbps in/out)
- Connection tracking (active, total, disconnected)
- Packet statistics and average packet sizes
- Game state metrics (car updates, ping latency)
- Player join/leave tracking
- Failure rate detection

### 2. **Performance Profiling** 🔍
- Integrated pprof on port 6060 for CPU/memory profiling
- Heap analysis for memory leak detection
- Goroutine monitoring
- CPU profile collection during tests

### 3. **Metrics Endpoints** 🔌
New control API endpoints:
- `GET /metrics` - Full metrics snapshot (JSON)
- `POST /metrics/reset` - Reset counters (useful for benchmarking phases)

### 4. **Bandwidth Optimizations**
- Metrics recording for outbound/inbound traffic
- Better visibility into compression effectiveness
- Tracking of packet sizes for protocol optimization

### 5. **Stress Test Suite** 🧪
Included bash scripts:
- `stress-test.sh` - Comprehensive multi-mode stress testing
- `monitor.sh` - Real-time terminal dashboard during tests

## Installation

### 1. Build the Enhanced Version

```bash
cd polyserver-enhanced
go build -o polyserver .
```

### 2. Install Test Dependencies

```bash
# Install wrk (load testing tool)
sudo apt install build-essential libssl-dev git -y
git clone https://github.com/wg/wrk.git
cd wrk
make
sudo cp wrk /usr/local/bin/

# Install jq (JSON processing)
sudo apt install jq -y
```

### 3. Update systemd Service

```bash
sudo tee /etc/systemd/system/polyserver.service > /dev/null << 'EOF'
[Unit]
Description=PolyServer Enhanced
After=network.target

[Service]
Type=simple
User=jakob
WorkingDirectory=/home/jakob/polyserver-enhanced
ExecStart=/home/jakob/polyserver-enhanced/polyserver -port 8091
Restart=always

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl restart polyserver
```

### 4. Make Test Scripts Executable

```bash
chmod +x stress-test.sh monitor.sh
```

## Usage

### Baseline Metrics (No Load)

Before stress testing, check the server baseline:

```bash
# Terminal 1: Monitor
./monitor.sh http://localhost:9090 2

# Terminal 2: Check metrics
curl http://localhost:9090/metrics | jq
```

### Quick Stress Test (10 min total)

```bash
# Terminal 1: Run test
./stress-test.sh http://localhost:9090 60 ramp

# Terminal 2: Watch metrics in real-time (in another terminal)
./monitor.sh http://localhost:9090 1
```

This runs a ramp-up test:
- 10 concurrent connections for 60s
- 50 concurrent connections for 60s
- 100, 200, 500, 1000 concurrent connections

With 15s cool-down between phases.

### Full Ramp-Up Test

```bash
./stress-test.sh http://localhost:9090 120 ramp
```

Tests load from 10 to 1000 concurrent connections over 2 hours.

### Sustained Load Test (Find Breaking Point)

```bash
# Hold 500 connections for 10 minutes
./stress-test.sh http://localhost:9090 600 sustained 600 500

# Or try 1000 connections if server handles 500 easily
./stress-test.sh http://localhost:9090 600 sustained 600 1000
```

### Profile CPU Usage During Load

```bash
# Terminal 1: Start profiling
./stress-test.sh http://localhost:9090 60 profile cpu
# Browser opens at http://localhost:8080

# Terminal 2: Run your stress test
./stress-test.sh http://localhost:9090 60 ramp
```

### Profile Memory During Load

```bash
# Terminal 1: Get heap profile
./stress-test.sh http://localhost:9090 60 profile heap

# Terminal 2: Run sustained test
./stress-test.sh http://localhost:9090 600 sustained 600 500

# Compare heap snapshots in the pprof UI
```

## Metrics Explained

### Bandwidth Metrics

```
out: 15.50 Mbps
in:  8.20 Mbps
```

Expected bandwidth per player in P2P relay:
- **Uplink**: ~0.5-1.5 Mbps per player (depends on game state update frequency)
- **Downlink**: Depends on player count (fan-out effect)

Formula: `Total_out = num_players × avg_player_rate × num_recipients`

### Connection Metrics

```
Active Connections: 42 / Total: 150
```

- **Active**: Currently connected players
- **Total**: Total players that joined (including disconnected)
- **Useful for**: Finding max sustainable concurrent connections

### Latency Metrics

```
Avg Latency: 45ms  Peak: 234ms
```

- **Avg Latency**: Round-trip time for ping packets
- **Peak**: Worst-case latency observed
- **Threshold**: >100ms may cause noticeable lag

### Packet Efficiency

```
Avg Packet Size: Out=256 bytes, In=128 bytes
```

- Smaller = better for bandwidth efficiency
- If growing over time = possible memory leak (reassembly buffers)

### Car Updates

```
Car Updates: 45000 (Failures: 12, Rate: 0.03%)
```

- Tracks game state update success rate
- Failure rate >1% = potential network congestion
- Monitor during ramp-up to find sweet spot

## What to Look For During Tests

### 1. **Linear Scaling** ✅
Bandwidth and latency scale linearly with connections up to a point.

```
10 conn  -> 1.5 Mbps out, 45ms latency
100 conn -> 15 Mbps out, 48ms latency (good)
1000 conn-> 150 Mbps out, 65ms latency (acceptable)
```

### 2. **Breaking Point** 🔴
Latency or failure rate spikes sharply:

```
500 conn  -> 45ms latency
600 conn  -> 150ms latency (BREAK!)
650 conn  -> 85% car update failure
```

This is your **maximum sustainable load**.

### 3. **Memory Leaks** 💾
Watch via pprof or `top`:

```bash
# In another terminal during sustained test
watch -n 1 'ps aux | grep polyserver | grep -v grep'
```

Memory should stabilize after initial growth. If it keeps climbing:
```
t=0min   -> 139MB
t=5min   -> 156MB
t=10min  -> 175MB (leak!)
```

### 4. **CPU Utilization**
Check Netdata during test:

```bash
# CPU should match load
10 conn   -> 5% CPU
100 conn  -> 15% CPU
1000 conn -> 80-90% CPU (near saturation)
```

If CPU spikes earlier than bandwidth, you're CPU-limited (tune code).

## Bandwidth Optimization Analysis

### Current Compression Effectiveness

The metrics endpoint shows compression impact:

```bash
curl http://localhost:9090/metrics | jq '.avg_packet_size_out'
# If high: compression could be improved
# If low: already efficient
```

### Further Optimizations (Future Work)

1. **Adaptive Compression**: Use faster compression (zstd) for real-time data
   - Current: zlib level 9 (slow but best ratio)
   - Recommended: zstd level 3 (3x faster, 85% ratio)

2. **Delta Encoding**: Only send car state changes
   - Current: Send full state per frame
   - Potential savings: 50-70%

3. **Quantization**: Reduce float precision for coordinates
   - Current: Full precision floats
   - Potential savings: 25% per state

4. **Message Batching**: Group multiple updates
   - Current: Already done (good!)
   - Potential: Tune batch size

## Comparing Before/After Optimization

1. **Baseline Test** (before optimization):
   ```bash
   ./stress-test.sh http://localhost:9090 120 ramp
   # Record max connections and bandwidth
   ```

2. **Apply optimization**

3. **Repeat Test**:
   ```bash
   ./stress-test.sh http://localhost:9090 120 ramp
   # Compare metrics
   ```

## Server Resource Limits

Check and increase if needed:

```bash
# Current file descriptor limit
ulimit -n

# Increase to 65536
ulimit -n 65536

# Make permanent (in /etc/security/limits.conf)
sudo bash -c 'echo "* soft nofile 65536" >> /etc/security/limits.conf'
sudo bash -c 'echo "* hard nofile 65536" >> /etc/security/limits.conf'
```

## Troubleshooting

### "Cannot reach server"
```bash
# Check if running
ps aux | grep polyserver | grep -v grep

# Check port
netstat -tlnp | grep 9090

# Start if stopped
cd ~/polyserver-enhanced
./polyserver -port 8091
```

### "Metrics show 0 values"
- Server might not be processing traffic
- Check invite code and verify players are joining
- Metrics only track actual traffic

### "Latency spikes at specific concurrency"
- That's your breaking point
- Test 10-20% below that number for safe operation
- Consider load balancing across multiple server instances

### "Memory keeps growing"
- Run heap profile: `./stress-test.sh http://localhost:9090 60 profile heap`
- Look for growing allocations in pprof UI
- Common culprit: uncleared player map or pending messages

## Performance Expectations (Your Hardware)

Your server: Ryzen 7 7800X3D, 32GB RAM, RTX 5070

Expected sustainable load:
- **Single instance**: 100-300 concurrent players
- **Bandwidth ceiling**: ~500 Mbps (home server ethernet limit)
- **Memory**: ~200-400 MB per 100 players
- **CPU**: Scales with player count and state update frequency

## Next Steps

1. Run baseline: `./stress-test.sh http://localhost:9090 30 baseline`
2. Run ramp test: `./stress-test.sh http://localhost:9090 60 ramp`
3. Identify max safe concurrent connections
4. Run sustained test at 70% of that for 30+ minutes
5. Check for memory leaks: `./stress-test.sh http://localhost:9090 60 profile heap`
6. Optimize based on findings

## Files Modified

- **metrics/metrics.go** - New metrics tracking module
- **game/carupdate.go** - Added bandwidth recording
- **game/main.go** - Added player join/leave metrics
- **game/player.go** - Added packet size recording
- **server.go** - Added metrics endpoints and pprof
- **stress-test.sh** - New test harness
- **monitor.sh** - New real-time monitoring

## License

Same as original PolyServer
