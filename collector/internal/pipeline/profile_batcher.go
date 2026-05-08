package pipeline

import (
	"encoding/json"
	"strings"
	"time"

	collectorStatus "github.com/koolay/java-profiler/collector/internal/status"
	profiling "github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

type ProfileBatchPayload struct {
	BatchID     string
	CollectorID string
	ReceivedAt  time.Time
	Samples     []profiling.ProfileSample
}

func BuildProfileBatch(batchID, collectorID string, samples []profiling.ProfileSample) (Batch, error) {
	payload := ProfileBatchPayload{BatchID: batchID, CollectorID: collectorID, ReceivedAt: time.Now().UTC(), Samples: samples}
	data, err := json.Marshal(payload)
	if err != nil {
		return Batch{}, err
	}
	return Batch{ID: batchID, Type: "profile", Bytes: len(data), CreatedAt: payload.ReceivedAt, Payload: data}, nil
}

type TargetStatusBatchPayload struct {
	BatchID     string                `json:"BatchID"`
	CollectorID string                `json:"CollectorID"`
	ReceivedAt  time.Time             `json:"ReceivedAt"`
	Statuses    []TargetStatusPayload `json:"Statuses"`
}

type TargetStatusPayload struct {
	BatchID      string                    `json:"batch_id"`
	Target       domain.TargetIdentity     `json:"target"`
	StatusAt     time.Time                 `json:"status_at"`
	DesiredState domain.TargetDesiredState `json:"desired_state"`
	Reason       domain.StatusReason       `json:"reason"`
	Message      string                    `json:"message"`
}

func BuildTargetStatusBatch(batchID, collectorID string, statuses []collectorStatus.TargetStatus) (Batch, error) {
	payloadStatuses := make([]TargetStatusPayload, 0, len(statuses))
	for _, status := range statuses {
		payloadStatuses = append(payloadStatuses, TargetStatusPayload{
			BatchID:      batchID,
			Target:       status.Target,
			StatusAt:     status.StatusAt,
			DesiredState: status.State,
			Reason:       status.Reason,
			Message:      status.Message,
		})
	}
	payload := TargetStatusBatchPayload{BatchID: batchID, CollectorID: collectorID, ReceivedAt: time.Now().UTC(), Statuses: payloadStatuses}
	data, err := json.Marshal(payload)
	if err != nil {
		return Batch{}, err
	}
	return Batch{ID: batchID, Type: "target_status", Bytes: len(data), CreatedAt: payload.ReceivedAt, Payload: data}, nil
}

func TargetStatusURL(profileURL string) string {
	if strings.Contains(profileURL, "/profile-batches") {
		return strings.Replace(profileURL, "/profile-batches", "/target-status-batches", 1)
	}
	return strings.TrimRight(profileURL, "/") + "/target-status-batches"
}
