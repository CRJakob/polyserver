package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// ServerMetrics tracks real-time performance metrics
type ServerMetrics struct {
	// Connection metrics
	ActiveConnections int64
	TotalConnections  int64
	DisconnectedCount int64

	// Bandwidth metrics (bytes)
	BytesSent     int64
	BytesReceived int64

	// Packet metrics
	PacketsSent     int64
	PacketsReceived int64

	// Game state metrics
	ActiveSessions int64
	PlayersJoined  int64
	PlayersLeft    int64

	// Performance metrics
	CarUpdatesProcessed int64
	CarUpdatesFailed    int64
	PingsProcessed      int64

	// Timing metrics
	AverageLatency    float64
	PeakLatency       float64
	AverageFrameTime  float64
	StartTime         time.Time
	LastResetTime     time.Time

	// Lock for latency metrics
	latencyLock sync.Mutex
	latencies   []float64
}

var globalMetrics = &ServerMetrics{
	StartTime:     time.Now(),
	LastResetTime: time.Now(),
	latencies:     make([]float64, 0, 1000),
}

// GetMetrics returns the global metrics instance
func GetMetrics() *ServerMetrics {
	return globalMetrics
}

// RecordBytesOut records outgoing bytes
func RecordBytesOut(bytes int64) {
	atomic.AddInt64(&globalMetrics.BytesSent, bytes)
}

// RecordBytesIn records incoming bytes
func RecordBytesIn(bytes int64) {
	atomic.AddInt64(&globalMetrics.BytesReceived, bytes)
}

// RecordPacketOut records outgoing packet
func RecordPacketOut(bytes int64) {
	atomic.AddInt64(&globalMetrics.PacketsSent, 1)
	RecordBytesOut(bytes)
}

// RecordPacketIn records incoming packet
func RecordPacketIn(bytes int64) {
	atomic.AddInt64(&globalMetrics.PacketsReceived, 1)
	RecordBytesIn(bytes)
}

// RecordPlayerJoin increments join counter
func RecordPlayerJoin() {
	atomic.AddInt64(&globalMetrics.PlayersJoined, 1)
	atomic.AddInt64(&globalMetrics.ActiveConnections, 1)
	atomic.AddInt64(&globalMetrics.TotalConnections, 1)
}

// RecordPlayerLeft decrements active connections
func RecordPlayerLeft() {
	atomic.AddInt64(&globalMetrics.ActiveConnections, -1)
	atomic.AddInt64(&globalMetrics.PlayersLeft, 1)
}

// RecordLatency records a latency measurement (in ms)
func RecordLatency(latencyMs float64) {
	globalMetrics.latencyLock.Lock()
	defer globalMetrics.latencyLock.Unlock()

	globalMetrics.latencies = append(globalMetrics.latencies, latencyMs)
	if latencyMs > globalMetrics.PeakLatency {
		globalMetrics.PeakLatency = latencyMs
	}

	// Keep only last 1000 samples for memory efficiency
	if len(globalMetrics.latencies) > 1000 {
		globalMetrics.latencies = globalMetrics.latencies[1:]
	}

	// Calculate average
	sum := 0.0
	for _, l := range globalMetrics.latencies {
		sum += l
	}
	globalMetrics.AverageLatency = sum / float64(len(globalMetrics.latencies))
}

// RecordCarUpdate increments car update counter
func RecordCarUpdate() {
	atomic.AddInt64(&globalMetrics.CarUpdatesProcessed, 1)
}

// RecordCarUpdateFailed increments failed car update counter
func RecordCarUpdateFailed() {
	atomic.AddInt64(&globalMetrics.CarUpdatesFailed, 1)
}

// RecordPing increments ping counter
func RecordPing() {
	atomic.AddInt64(&globalMetrics.PingsProcessed, 1)
}

// Snapshot returns a point-in-time copy of metrics
type Snapshot struct {
	Uptime                 float64
	ActiveConnections      int64
	TotalConnections       int64
	BytesSent              int64
	BytesReceived          int64
	MbpsSent               float64
	MbpsReceived           float64
	PacketsSent            int64
	PacketsReceived        int64
	AveragePacketSizeOut   float64
	AveragePacketSizeIn    float64
	PlayersJoined          int64
	PlayersLeft            int64
	CarUpdatesProcessed    int64
	CarUpdatesFailed       int64
	CarUpdateFailureRate   float64
	PingsProcessed         int64
	AverageLatencyMs       float64
	PeakLatencyMs          float64
}

// Snapshot returns current metrics as a snapshot
func (m *ServerMetrics) Snapshot() Snapshot {
	uptime := time.Since(m.StartTime).Seconds()
	sent := atomic.LoadInt64(&m.BytesSent)
	received := atomic.LoadInt64(&m.BytesReceived)
	pktSent := atomic.LoadInt64(&m.PacketsSent)
	pktReceived := atomic.LoadInt64(&m.PacketsReceived)
	carUpdates := atomic.LoadInt64(&m.CarUpdatesProcessed)
	carFailed := atomic.LoadInt64(&m.CarUpdatesFailed)

	mbpsSent := 0.0
	mbpsReceived := 0.0
	avgPktOut := 0.0
	avgPktIn := 0.0
	failureRate := 0.0

	if uptime > 0 {
		mbpsSent = (float64(sent) * 8) / (uptime * 1_000_000)
		mbpsReceived = (float64(received) * 8) / (uptime * 1_000_000)
	}

	if pktSent > 0 {
		avgPktOut = float64(sent) / float64(pktSent)
	}

	if pktReceived > 0 {
		avgPktIn = float64(received) / float64(pktReceived)
	}

	if carUpdates > 0 {
		failureRate = float64(carFailed) / float64(carUpdates) * 100
	}

	return Snapshot{
		Uptime:               uptime,
		ActiveConnections:    atomic.LoadInt64(&m.ActiveConnections),
		TotalConnections:     atomic.LoadInt64(&m.TotalConnections),
		BytesSent:            sent,
		BytesReceived:        received,
		MbpsSent:             mbpsSent,
		MbpsReceived:         mbpsReceived,
		PacketsSent:          pktSent,
		PacketsReceived:      pktReceived,
		AveragePacketSizeOut: avgPktOut,
		AveragePacketSizeIn:  avgPktIn,
		PlayersJoined:        atomic.LoadInt64(&m.PlayersJoined),
		PlayersLeft:          atomic.LoadInt64(&m.PlayersLeft),
		CarUpdatesProcessed:  carUpdates,
		CarUpdatesFailed:     carFailed,
		CarUpdateFailureRate: failureRate,
		PingsProcessed:       atomic.LoadInt64(&m.PingsProcessed),
		AverageLatencyMs:     m.AverageLatency,
		PeakLatencyMs:        m.PeakLatency,
	}
}

// Reset clears accumulated metrics
func (m *ServerMetrics) Reset() {
	atomic.StoreInt64(&m.BytesSent, 0)
	atomic.StoreInt64(&m.BytesReceived, 0)
	atomic.StoreInt64(&m.PacketsSent, 0)
	atomic.StoreInt64(&m.PacketsReceived, 0)
	m.latencyLock.Lock()
	m.latencies = make([]float64, 0, 1000)
	m.PeakLatency = 0
	m.AverageLatency = 0
	m.latencyLock.Unlock()
	m.LastResetTime = time.Now()
}
