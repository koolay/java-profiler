package pipeline

import (
	"testing"
	"time"
)

func TestLocalBufferDropsOldestWhenFull(t *testing.T) {
	buffer := NewLocalBuffer(10, 2)
	buffer.Push(Batch{ID: "a", Bytes: 6, CreatedAt: time.Unix(1, 0)})
	buffer.Push(Batch{ID: "b", Bytes: 6, CreatedAt: time.Unix(2, 0)})
	stats := buffer.Stats()
	if stats.DroppedBatches != 1 || stats.OldestDroppedAt.Unix() != 1 {
		t.Fatalf("unexpected drop stats: %+v", stats)
	}
	if stats.CurrentBatches != 1 || stats.CurrentBufferedBytes != 6 {
		t.Fatalf("unexpected current stats: %+v", stats)
	}
	next, ok := buffer.Pop()
	if !ok || next.ID != "b" {
		t.Fatalf("expected batch b, got %+v ok=%v", next, ok)
	}
	stats = buffer.Stats()
	if stats.CurrentBatches != 0 || stats.CurrentBufferedBytes != 0 {
		t.Fatalf("expected empty buffer stats, got %+v", stats)
	}
}
