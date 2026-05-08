package pipeline

import (
	"sync"
	"time"
)

type Batch struct {
	ID        string
	Type      string
	Bytes     int
	CreatedAt time.Time
	Payload   []byte
}

type DropStats struct {
	DroppedBatches       uint64
	DroppedBytes         uint64
	OldestDroppedAt      time.Time
	CurrentBatches       int
	CurrentBufferedBytes int
}

type LocalBuffer struct {
	mu         sync.Mutex
	maxBytes   int
	maxBatches int
	batches    []Batch
	stats      DropStats
}

func NewLocalBuffer(maxBytes, maxBatches int) *LocalBuffer {
	if maxBytes <= 0 {
		maxBytes = 16 << 20
	}
	if maxBatches <= 0 {
		maxBatches = 128
	}
	return &LocalBuffer{maxBytes: maxBytes, maxBatches: maxBatches}
}

func (b *LocalBuffer) Push(batch Batch) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if batch.Bytes == 0 {
		batch.Bytes = len(batch.Payload)
	}
	b.batches = append(b.batches, batch)
	for b.bufferedBytes() > b.maxBytes || len(b.batches) > b.maxBatches {
		dropped := b.batches[0]
		b.batches = b.batches[1:]
		b.stats.DroppedBatches++
		b.stats.DroppedBytes += uint64(dropped.Bytes)
		if b.stats.OldestDroppedAt.IsZero() || dropped.CreatedAt.Before(b.stats.OldestDroppedAt) {
			b.stats.OldestDroppedAt = dropped.CreatedAt
		}
	}
}

func (b *LocalBuffer) Pop() (Batch, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.batches) == 0 {
		return Batch{}, false
	}
	next := b.batches[0]
	b.batches = b.batches[1:]
	return next, true
}

func (b *LocalBuffer) Stats() DropStats {
	b.mu.Lock()
	defer b.mu.Unlock()
	stats := b.stats
	stats.CurrentBatches = len(b.batches)
	stats.CurrentBufferedBytes = b.bufferedBytes()
	return stats
}

func (b *LocalBuffer) bufferedBytes() int {
	total := 0
	for _, batch := range b.batches {
		total += batch.Bytes
	}
	return total
}
