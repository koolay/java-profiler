package app

import "github.com/koolay/java-profiler/domain"

type ServiceSummary struct {
	Namespace         string               `json:"namespace"`
	Service           string               `json:"service"`
	AvailableProfiles []domain.ProfileType `json:"available_profiles"`
	StatusCounts      map[string]int       `json:"status_counts"`
}
