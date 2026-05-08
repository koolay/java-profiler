package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

type JVMEligibility struct {
	HotSpotCompatible bool
	Conflict          bool
	Vendor            string
	Version           string
	Reason            string
}

type HotSpotDetector struct{}

func (HotSpotDetector) Detect(processRoot string) JVMEligibility {
	maps, err := os.ReadFile(filepath.Join(processRoot, "maps"))
	if err != nil {
		return JVMEligibility{Reason: "maps_unavailable"}
	}
	text := strings.ToLower(string(maps))
	eligibility := JVMEligibility{}
	if strings.Contains(text, "libjvm") && (strings.Contains(text, "hotspot") || strings.Contains(text, "server/libjvm") || strings.Contains(text, "jre/lib")) {
		eligibility.HotSpotCompatible = true
		eligibility.Vendor = "hotspot-compatible"
	}
	if strings.Contains(text, "libasyncprofiler") || strings.Contains(text, "async-profiler") {
		eligibility.Conflict = true
		eligibility.Reason = "profiler_conflict"
	}
	if !eligibility.HotSpotCompatible && eligibility.Reason == "" {
		eligibility.Reason = "unsupported_jvm"
	}
	return eligibility
}
