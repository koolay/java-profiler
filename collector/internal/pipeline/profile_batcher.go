package pipeline

import (
	"encoding/json"
	"time"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
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
