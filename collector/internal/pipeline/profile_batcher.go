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
	BatchID     string                         `json:"batch_id"`
	CollectorID string                         `json:"collector_id"`
	ReceivedAt  time.Time                      `json:"received_at"`
	Samples     []profiling.ProfileSample      `json:"samples"`
	Metadata    profiling.ProfileBatchMetadata `json:"metadata"`
}

func BuildProfileBatch(batchID, collectorID string, samples []profiling.ProfileSample, metadata profiling.ProfileBatchMetadata) ([]byte, error) {
	payload := ProfileBatchPayload{BatchID: batchID, CollectorID: collectorID, ReceivedAt: time.Now().UTC(), Samples: samples, Metadata: metadata}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type ThreadSnapshotBatchPayload struct {
	BatchID     string                     `json:"BatchID"`
	CollectorID string                     `json:"CollectorID"`
	ReceivedAt  time.Time                  `json:"ReceivedAt"`
	Snapshots   []profiling.ThreadSnapshot `json:"Snapshots"`
	Deadlocks   []profiling.DeadlockEvent  `json:"Deadlocks"`
}

func BuildThreadSnapshotBatch(batchID, collectorID string, snapshots []profiling.ThreadSnapshot, deadlocks []profiling.DeadlockEvent) (Batch, error) {
	payload := ThreadSnapshotBatchPayload{BatchID: batchID, CollectorID: collectorID, ReceivedAt: time.Now().UTC(), Snapshots: snapshots, Deadlocks: deadlocks}
	data, err := json.Marshal(payload)
	if err != nil {
		return Batch{}, err
	}
	return Batch{ID: batchID, Type: "thread_snapshot", Bytes: len(data), CreatedAt: payload.ReceivedAt, Payload: data}, nil
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

func ThreadSnapshotURL(profileURL string) string {
	if strings.Contains(profileURL, "/profile-batches") {
		return strings.Replace(profileURL, "/profile-batches", "/thread-snapshot-batches", 1)
	}
	return strings.TrimRight(profileURL, "/") + "/thread-snapshot-batches"
}
