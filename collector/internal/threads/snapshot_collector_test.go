package threads

import (
	"testing"
	"time"
)

func TestScheduleExpires(t *testing.T) {
	schedule := Schedule{Interval: time.Second, ExpiresAt: time.Unix(10, 0)}
	if !schedule.Active(time.Unix(9, 0)) {
		t.Fatalf("schedule should be active before expiry")
	}
	if schedule.Active(time.Unix(11, 0)) {
		t.Fatalf("schedule should stop after expiry")
	}
}
