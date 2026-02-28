package metrics

import "time"

type RequestSample struct {
	Path      string
	Method    string
	Status    int
	Latency   time.Duration
	Timestamp time.Time
}
