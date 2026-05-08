package pipeline

import (
	"math/rand"
	"time"
)

type UploadScheduler struct {
	Interval time.Duration
	Jitter   time.Duration
	rand     *rand.Rand
}

func NewUploadScheduler(interval, jitter time.Duration) UploadScheduler {
	return UploadScheduler{Interval: interval, Jitter: jitter, rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (s UploadScheduler) Next(after time.Time) time.Time {
	delay := s.Interval
	if delay <= 0 {
		delay = time.Minute
	}
	if s.Jitter > 0 && s.rand != nil {
		delay += time.Duration(s.rand.Int63n(int64(s.Jitter)))
	}
	return after.Add(delay)
}
