package pipeline

import (
	"encoding/json"
	"testing"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
)

func TestBuildProfileBatchIncludesMetadata(t *testing.T) {
	payload, err := BuildProfileBatch(
		"batch-a",
		"collector-a",
		nil,
		profiling.ProfileBatchMetadata{
			WindowRawSampleCount:        100,
			WindowAggregatedSampleCount: 12,
			BatchSampleCount:            10,
			DroppedSampleCount:          2,
			DroppedStackCount:           1,
			Truncated:                   true,
			PartIndex:                   1,
			PartCount:                   2,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	var decoded ProfileBatchPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Metadata.WindowRawSampleCount != 100 {
		t.Fatalf("raw sample count = %d", decoded.Metadata.WindowRawSampleCount)
	}
	if !decoded.Metadata.Truncated {
		t.Fatalf("metadata should mark the batch as truncated")
	}
}
