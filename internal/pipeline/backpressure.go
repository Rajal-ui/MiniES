package pipeline

type BackpressureConfig struct {
	MaxQueueSize      int
	HighWaterMark     float64
	RetryAfterSeconds int
}

func DefaultBackpressureConfig() *BackpressureConfig {
	return &BackpressureConfig{
		MaxQueueSize:      10000,
		HighWaterMark:     0.8,
		RetryAfterSeconds: 5,
	}
}

func (bp *BackpressureConfig) IsOverloaded(currentSize int) bool {
	return float64(currentSize)/float64(bp.MaxQueueSize) >= bp.HighWaterMark
}

func (bp *BackpressureConfig) RetryAfter(currentSize int) int {
	if !bp.IsOverloaded(currentSize) {
		return 0
	}
	return bp.RetryAfterSeconds
}
