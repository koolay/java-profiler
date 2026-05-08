package pipeline

import (
	"testing"
	"time"
)

func TestUploadSchedulerAddsDelay(t *testing.T) {
	start := time.Unix(100, 0)
	next := NewUploadScheduler(time.Minute, 0).Next(start)
	if next.Sub(start) != time.Minute {
		t.Fatalf("expected 1m delay, got %s", next.Sub(start))
	}
}
