package app

const (
	DefaultThreadDiagnosisLimit = 1000
	MaxThreadDiagnosisLimit     = 1000
	DefaultDeadlockLimit        = 500
	MaxDeadlockLimit            = 500
	DefaultTargetStatusLimit    = 500
	MaxTargetStatusLimit        = 500
)

func boundedQueryLimit(requested, fallback, maximum int) int {
	if requested <= 0 {
		return fallback
	}
	if requested > maximum {
		return maximum
	}
	return requested
}
