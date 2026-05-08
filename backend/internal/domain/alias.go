package domain

import root "github.com/koolay/java-profiler/domain"

type (
	ProfileType        = root.ProfileType
	EnablementMode     = root.EnablementMode
	TargetDesiredState = root.TargetDesiredState
	StatusReason       = root.StatusReason
	BatchType          = root.BatchType
	TargetIdentity     = root.TargetIdentity
	TimeWindow         = root.TimeWindow
	RetentionPolicy    = root.RetentionPolicy
	Confidence         = root.Confidence
)

const (
	ProfileTypeCPU            = root.ProfileTypeCPU
	ProfileTypeAllocBytes     = root.ProfileTypeAllocBytes
	ProfileTypeAllocObjects   = root.ProfileTypeAllocObjects
	ProfileTypeLockContention = root.ProfileTypeLockContention
	ProfileTypeLockDelay      = root.ProfileTypeLockDelay

	EnablementDisabled   = root.EnablementDisabled
	EnablementContinuous = root.EnablementContinuous
	EnablementTemporary  = root.EnablementTemporary

	TargetDesiredStateDisabled    = root.TargetDesiredStateDisabled
	TargetDesiredStateEnabled     = root.TargetDesiredStateEnabled
	TargetDesiredStateTemporary   = root.TargetDesiredStateTemporary
	TargetDesiredStateUnsupported = root.TargetDesiredStateUnsupported

	StatusReasonNone               = root.StatusReasonNone
	StatusReasonDisabledByMetadata = root.StatusReasonDisabledByMetadata
	StatusReasonTemporaryExpired   = root.StatusReasonTemporaryExpired
	StatusReasonInvalidDuration    = root.StatusReasonInvalidDuration
	StatusReasonUnsupportedJVM     = root.StatusReasonUnsupportedJVM
	StatusReasonProfilerConflict   = root.StatusReasonProfilerConflict
	StatusReasonAttachFailed       = root.StatusReasonAttachFailed
	StatusReasonUploadRetryable    = root.StatusReasonUploadRetryable
	StatusReasonUploadDropped      = root.StatusReasonUploadDropped
	StatusReasonStorageRejected    = root.StatusReasonStorageRejected
	StatusReasonAccepted           = root.StatusReasonAccepted

	BatchTypeProfile        = root.BatchTypeProfile
	BatchTypeThreadSnapshot = root.BatchTypeThreadSnapshot
	BatchTypeTargetStatus   = root.BatchTypeTargetStatus
	BatchTypeCollectorBeat  = root.BatchTypeCollectorBeat
	BatchTypeIngestion      = root.BatchTypeIngestion
	BatchTypeRetention      = root.BatchTypeRetention
	BatchTypeArtifactIndex  = root.BatchTypeArtifactIndex

	ConfidenceExactThreadCPU  = root.ConfidenceExactThreadCPU
	ConfidenceSampledRUNNABLE = root.ConfidenceSampledRUNNABLE
	ConfidenceProfileOnly     = root.ConfidenceProfileOnly
)

var (
	AllProfileTypes = root.AllProfileTypes
)

func DefaultRetentionPolicy() RetentionPolicy { return root.DefaultRetentionPolicy() }
func StableProfileTypeNames() []string        { return root.StableProfileTypeNames() }
