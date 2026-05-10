package pipeline

import "github.com/koolay/java-profiler/contracts/profiling"

type ProfileBatchLimits struct {
	MaxSamplesPerWindow int
	MaxSamplesPerBatch  int
}

func DefaultProfileBatchLimits() ProfileBatchLimits {
	return ProfileBatchLimits{
		MaxSamplesPerWindow: 50_000,
		MaxSamplesPerBatch:  10_000,
	}
}

func BoundProfileSamples(samples []profiling.ProfileSample, limits ProfileBatchLimits) ([]profiling.ProfileSample, profiling.ProfileBatchMetadata) {
	metadata := profiling.ProfileBatchMetadata{
		WindowRawSampleCount:        len(samples),
		WindowAggregatedSampleCount: len(samples),
	}
	if limits.MaxSamplesPerWindow <= 0 || len(samples) <= limits.MaxSamplesPerWindow {
		return samples, metadata
	}

	bounded := make([]profiling.ProfileSample, limits.MaxSamplesPerWindow)
	copy(bounded, samples[:limits.MaxSamplesPerWindow])
	metadata.DroppedSampleCount = len(samples) - limits.MaxSamplesPerWindow
	metadata.DroppedStackCount = metadata.DroppedSampleCount
	metadata.Truncated = true
	return bounded, metadata
}

func BatchMetadataForPart(base profiling.ProfileBatchMetadata, partIndex, partCount, batchSampleCount int) profiling.ProfileBatchMetadata {
	base.PartIndex = partIndex
	base.PartCount = partCount
	base.BatchSampleCount = batchSampleCount
	return base
}
