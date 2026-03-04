# PolyServer Enhanced - Quick Start

## 30-Second Setup

```bash
# 1. Navigate to enhanced version
cd ~/polyserver-enhanced

# 2. Build
go build -o polyserver .

# 3. Make scripts executable
chmod +x stress-test.sh monitor.sh

# 4. Update systemd (if not done yet)
sudo systemctl stop polyserver
# Edit ExecStart in /etc/systemd/system/polyserver.service
# Change from: ExecStart=/home/jakob/polyserver-go/polyserver ...
# To:         ExecStart=/home/jakob/polyserver-enhanced/polyserver ...
sudo systemctl daemon-reload
sudo systemctl start polyserver
```

## Quick Test (5 minutes)

Run this to see if everything works:

```bash
# Terminal 1: Monitor in real-time
./monitor.sh http://localhost:9090

# Terminal 2: Run quick test
./stress-test.sh http://localhost:9090 30 baseline
```

You should see:
- Monitor shows: connections, bandwidth, latency
- Test shows: requests/sec, response times, errors (if any)

## One-Command Full Test

```bash
# This runs everything and saves results
./stress-test.sh http://localhost:9090 60 ramp
```

Takes about **10 minutes**, tests 10, 50, 100, 200, 500, 1000 concurrent connections.

## Key Metrics to Track

| Metric | Good | Warning | Critical |
|--------|------|---------|----------|
| **Avg Latency** | <50ms | 50-100ms | >100ms |
| **Failure Rate** | <0.1% | 0.1-1% | >1% |
| **Active Connections** | <100 | 100-500 | >500* |
| **Memory** | Stable | Slow growth | Linear growth |

*Depends on your needs

## Real-Time Monitoring with Netdata

While running stress tests, also open Netdata:

```bash
# In browser: http://localhost:19999
# Watch: CPU, RAM, Network I/O, TCP connections
```

Compare with your stress test metrics:
- Netdata shows **system-wide** usage
- `./monitor.sh` shows **polyserver-specific** metrics
- Both should scale together

## What to Run Next

1. **Baseline** (5 min):
   ```bash
   ./stress-test.sh http://localhost:9090 30 baseline
   ```
   Establishes your current performance baseline.

2. **Ramp-Up** (10 min):
   ```bash
   ./stress-test.sh http://localhost:9090 60 ramp
   ```
   Finds where performance degrades.

3. **Sustained** (30 min):
   ```bash
   ./stress-test.sh http://localhost:9090 600 sustained 600 200
   ```
   Tests stability under constant heavy load.
   (300 concurrent, 10 minutes)

4. **Profiling** (during load):
   ```bash
   # Terminal 1: Profile memory
   ./stress-test.sh http://localhost:9090 60 profile heap
   
   # Terminal 2: Run sustained test
   ./stress-test.sh http://localhost:9090 600 sustained 600 300
   ```

## Understanding Output

### Monitor Output Example
```
Active Connections: 245 / Total: 256
Out: 45.23 Mbps  ████████████░░░░░░░░░░░  (1.2GB)
In:  28.54 Mbps  ████████░░░░░░░░░░░░░░░░  (756MB)
Car Updates: 450000 (Failures: 2, Rate: 0.0%)
Avg Latency: 42ms  Peak: 145ms
```

- **Connections**: How many players are active
- **Bandwidth**: Current I/O (should scale with connections)
- **Updates**: Game state sync health
- **Latency**: Ping (increases under load, should be <100ms)

### Test Output Example
```
Running 60s test @ http://localhost:9090/status
  8 threads and 200 connections
Requests/sec:   18542
Latency avg:    10.65ms
Latency max:    234.23ms
Requests/sec Distribution:
  50%     6ms
  75%    12ms
  90%    24ms
  99%    78ms
```

## Optimizing from Results

### If latency is high (>100ms):
1. Reduce concurrent connections
2. Check CPU usage (might be bottleneck)
3. Check Netdata for context switches (CPU thrashing)

### If failure rate is high (>1%):
1. You're at the breaking point - reduce load
2. Check network saturation (Netdata)
3. Look for packet loss

### If memory is growing:
1. Run heap profile: `./stress-test.sh http://localhost:9090 60 profile heap`
2. Look for leaks in pprof
3. Check `/players` endpoint - should show N active, not accumulate

## Files in Enhanced Version

```
polyserver-enhanced/
├── metrics/
│   └── metrics.go           [NEW] Metrics tracking module
├── stress-test.sh           [NEW] Load testing harness
├── monitor.sh               [NEW] Real-time monitoring
├── STRESS_TESTING_GUIDE.md  [NEW] Detailed guide
├── QUICK_START.md           [NEW] This file
├── game/
│   ├── main.go              [MODIFIED] Added metrics
│   ├── player.go            [MODIFIED] Added metrics
│   └── carupdate.go         [MODIFIED] Added metrics
├── server.go                [MODIFIED] Added endpoints
└── [other files unchanged]
```

## Cleanup After Testing

Results are saved in `.log` files:
```bash
# Remove test logs
rm *.log

# Reset metrics
curl -X POST http://localhost:9090/metrics/reset
```

## Next: Actual Game Testing

Once you know your performance limits, test with real players:

1. Start with expected player count
2. Monitor in real-time: `./monitor.sh`
3. Watch Netdata: `http://localhost:19999`
4. Verify no latency spikes or packet loss
5. Leave running for 1+ hour to check stability

## Help

- Full guide: `STRESS_TESTING_GUIDE.md`
- Metrics endpoint: `curl http://localhost:9090/metrics | jq`
- Profiling: `go tool pprof http://localhost:6060/debug/pprof/heap`

Good luck! 🚀
