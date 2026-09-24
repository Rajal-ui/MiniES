package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	ingestCount     atomic.Int64
	errorCount      atomic.Int64
	queryCount      atomic.Int64
	queryLatencySum atomic.Int64
	queryLatencyN   atomic.Int64
	mu              sync.RWMutex
	startTime       time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{startTime: time.Now()}
}

func (m *Metrics) RecordIngest() {
	m.ingestCount.Add(1)
}

func (m *Metrics) RecordError() {
	m.errorCount.Add(1)
}

func (m *Metrics) RecordQuery(latencyMs int64) {
	m.queryCount.Add(1)
	m.queryLatencySum.Add(latencyMs)
	m.queryLatencyN.Add(1)
}

func (m *Metrics) IngestCount() int64 { return m.ingestCount.Load() }
func (m *Metrics) ErrorCount() int64  { return m.errorCount.Load() }
func (m *Metrics) QueryCount() int64  { return m.queryCount.Load() }

func (m *Metrics) AvgLatency() int64 {
	n := m.queryLatencyN.Load()
	if n == 0 {
		return 0
	}
	return m.queryLatencySum.Load() / n
}

func (m *Metrics) Uptime() time.Duration { return time.Since(m.startTime) }

func (m *Metrics) Snapshot() map[string]any {
	return map[string]any{
		"ingest_count":   m.IngestCount(),
		"error_count":    m.ErrorCount(),
		"query_count":    m.QueryCount(),
		"avg_latency_ms": m.AvgLatency(),
		"uptime_seconds": int(m.Uptime().Seconds()),
	}
}
