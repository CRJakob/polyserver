# PolyServer Metrics Dashboard Guide

## Overview

The web dashboard now includes a comprehensive real-time metrics section that displays:
- **Bandwidth** usage (Mbps and total bytes)
- **Active connections** and player join/leave stats
- **Server uptime**
- **Latency** metrics (average and peak)
- **Game state health** (car update stats)
- **Network packet** statistics

## Accessing the Dashboard

1. **Local access** (development):
   ```
   http://localhost:8091
   ```

2. **Remote access** (if port forwarded):
   ```
   http://your-server-ip:8091
   ```

## Metrics Explained

### 🌐 Bandwidth Section

**Outbound (⬆️)**
- Current upload rate in Mbps
- Total bytes sent since server start (or reset)
- Increases when sending player updates, track data, game state

**Inbound (⬇️)**
- Current download rate in Mbps
- Total bytes received since server start (or reset)
- Increases when players send car updates, inputs, actions

### 👥 Connections Section

**Active (🔗)**
- Number of players currently connected
- Should match player count in the scoreboard
- Decreases when players disconnect

**Joined (➕) / Left (➖)**
- Total cumulative count of player join/leave events
- `Joined - Left = Active` (should equal active connections)
- Useful for tracking session churn

### ⏱️ Uptime Section

**Server uptime** formatted as:
- Hours:Minutes:Seconds (e.g., "2h 34m 12s")
- Useful for knowing if server crashed/restarted
- Resets when server process stops

### ⚡ Performance Section

**Avg Latency**
- Average round-trip time (RTT) for ping packets in milliseconds
- Indicates how responsive the server is
- Lower is better (target: <50ms for good experience)

**Peak Latency**
- Worst-case latency observed since start (or reset)
- Shows if there were any spikes
- If consistently high, indicates congestion or slow network

### 🎮 Game State Section

**Car Updates**
- Total number of car state messages processed
- Increases as players drive and positions update
- Use to gauge game activity level

**Health**
- Percentage of successful car updates
- `100% - failure_rate = Health`
- <99.9% might indicate packet loss or congestion

### 📦 Packets Section

**Sent / Received**
- Total packet count since start
- One packet can contain multiple updates (batching)
- Useful for protocol efficiency analysis

**Avg Size**
- Average size of outbound packets in bytes
- Smaller is better (indicates good compression)
- Growing over time might indicate memory issue

## Real-Time Updates

The metrics dashboard updates **every 1 second** automatically. You'll see:
- Live bandwidth changes as players connect/disconnect
- Latency spikes when network is congested
- Car update rates as game activity changes

## Using Metrics to Monitor

### Daily Check
Before players connect, check:
1. **Uptime** - Did server crash overnight?
2. **Active connections** - Should be 0 (or expected)
3. **Health** - Should be 100%

### During Gameplay
Monitor:
1. **Bandwidth** - How much are your players using?
2. **Latency** - Is it stable or spiking?
3. **Car updates** - Is game activity normal?

### Performance Analysis
Use metrics to identify issues:

| Symptom | Likely Cause |
|---------|-------------|
| Latency spikes | Network congestion or GC pause |
| High outbound bandwidth | Too many players or frequent updates |
| Low car health | Packet loss or player network issues |
| Stagnant car updates | No players driving or bot stuck |

### Before/After Testing
1. **Reset metrics** before starting test
2. **Run load test**
3. **Check metrics** to analyze impact
4. **Compare** with previous runs

## Resetting Metrics

Click the **"Reset Metrics"** button to:
- Clear all counters (bytes, packets, uptime)
- Start fresh for a new measurement period
- Keep current session running

Useful for:
- Before running benchmarks
- After deploying code changes
- Isolating test periods
- Generating clean reports

## Interpreting Trends

### Bandwidth Should Scale Linearly
```
10 players  → 5 Mbps outbound
20 players  → 10 Mbps outbound (2x)
50 players  → 25 Mbps outbound (5x)
```

If not linear, you've found a bottleneck.

### Latency Should Remain Stable
```
Good:     30ms avg, 45ms peak (stable)
Warning:  50ms avg, 150ms peak (spiking)
Bad:      100ms+ avg (consistently high)
```

### Car Health Should Stay >99.9%
```
Good:     100% - 99.95% health
Warning:  99.9% - 99.5% health
Critical: <99.5% health (packet loss)
```

## API Access

If you want to read metrics programmatically:

```bash
# Get current metrics as JSON
curl http://localhost:9090/metrics | jq

# Reset metrics
curl -X POST http://localhost:9090/metrics/reset
```

## Troubleshooting

### Metrics showing all zeros
- Server might not be running
- Check that polyserver process is active
- Ensure port 9090 is accessible

### Bandwidth shows incoming but not outgoing
- Players might be connected but not sending updates
- Check game is running and not paused
- Verify car updates are being processed

### Latency very high but players not reporting lag
- Might be local network issue (check with Netdata)
- Could be one slow player affecting average
- Try resetting metrics and reconnecting players

### Bandwidth fluctuating wildly
- Normal during player joins/disconnects
- Wait a few seconds for it to stabilize
- Check if track data is being sent (happens on join)

## Next: Optimization

Once you understand baseline metrics:
1. Run a normal play session
2. Record the numbers
3. Make a code optimization
4. Reset metrics
5. Run same session
6. Compare results

The dashboard makes it easy to see the impact of changes!

## Tips

1. **Full screen the metrics** - Makes trends easier to spot
2. **Watch during peak hours** - See real usage patterns
3. **Keep baseline screenshot** - Compare before/after optimization
4. **Monitor after deploys** - Catch regressions immediately
5. **Use history** - Browser back button to see logs folder

Enjoy your metrics! 📊
