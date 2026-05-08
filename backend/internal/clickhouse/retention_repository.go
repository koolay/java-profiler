package clickhouse

import (
	"context"
	"time"
)

type RetentionTableStatus struct {
	Table        string
	OldestRowAt  time.Time
	TTLLag       time.Duration
	Bytes        uint64
	Parts        uint64
	RetentionTTL time.Duration
}

type RetentionRepository struct {
	statuses []RetentionTableStatus
}

func NewRetentionRepository(statuses []RetentionTableStatus) *RetentionRepository {
	return &RetentionRepository{statuses: statuses}
}

func (r *RetentionRepository) Status(_ context.Context) ([]RetentionTableStatus, error) {
	out := make([]RetentionTableStatus, len(r.statuses))
	copy(out, r.statuses)
	return out, nil
}
