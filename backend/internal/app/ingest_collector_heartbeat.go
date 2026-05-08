package app

import "time"

type CollectorHeartbeat struct {
	CollectorID    string    `json:"collector_id"`
	Node           string    `json:"node"`
	ObservedAt     time.Time `json:"observed_at"`
	DroppedBatches uint64    `json:"dropped_batches"`
}
