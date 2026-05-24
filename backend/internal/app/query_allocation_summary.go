package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/backend/internal/metrics"
	"github.com/koolay/java-profiler/domain"
)

const (
	DefaultAllocationPathLimit      = 50
	MaxAllocationPathLimit          = 200
	DefaultAllocationSelfFrameLimit = 50
	MaxAllocationSelfFrameLimit     = 200
	DefaultAllocationNodeLimit      = 500
	MaxAllocationNodeLimit          = 2000
	MaxAllocationSummaryRange       = 7 * 24 * time.Hour
	MaxAllocationNamespaceOnlyRange = 30 * time.Minute
)

var ErrInvalidAllocationSummaryQuery = errors.New("invalid allocation summary query")

type AllocationSummaryValidationCode string

const (
	AllocationSummaryValidationNamespaceRequired          AllocationSummaryValidationCode = "namespace_required"
	AllocationSummaryValidationInvalidProfileType         AllocationSummaryValidationCode = "invalid_profile_type"
	AllocationSummaryValidationInvalidJVM                 AllocationSummaryValidationCode = "invalid_jvm"
	AllocationSummaryValidationInvalidTimeRange           AllocationSummaryValidationCode = "invalid_time_range"
	AllocationSummaryValidationRangeExceeded              AllocationSummaryValidationCode = "range_exceeded"
	AllocationSummaryValidationNamespaceOnlyRangeExceeded AllocationSummaryValidationCode = "namespace_only_range_exceeded"
)

type AllocationSummaryValidationError struct {
	Code            AllocationSummaryValidationCode
	Field           string
	Message         string
	SuggestedAction string
}

func (e AllocationSummaryValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ErrInvalidAllocationSummaryQuery.Error(), e.Message)
}

func (e AllocationSummaryValidationError) Unwrap() error {
	return ErrInvalidAllocationSummaryQuery
}

type AllocationSummaryQuery struct {
	Namespace      string
	Service        string
	Pod            string
	Container      string
	JVM            string
	ProfileType    domain.ProfileType
	Start          time.Time
	End            time.Time
	PathLimit      int
	SelfFrameLimit int
	NodeLimit      int
}

type AllocationSummary struct {
	SchemaVersion  int                          `json:"schema_version"`
	RequestedScope AllocationSummaryScope       `json:"requested_scope"`
	EffectiveScope AllocationSummaryScope       `json:"effective_scope"`
	Coverage       AllocationSummaryCoverage    `json:"coverage"`
	TopPaths       []AllocationTopPath          `json:"top_paths"`
	TopSelfFrames  []AllocationTopSelfFrame     `json:"top_self_frames"`
	Insights       []AllocationInsight          `json:"insights"`
	Limitations    []AllocationLimitation       `json:"limitations"`
	Semantics      domain.ProfileValueSemantics `json:"semantics"`
}

type AllocationSummaryScope struct {
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
	JVM       string `json:"jvm"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

type AllocationSummaryCoverage struct {
	HasData                bool      `json:"has_data"`
	EmptyState             string    `json:"empty_state,omitempty"`
	ProfileType            string    `json:"profile_type"`
	TotalValue             uint64    `json:"total_value"`
	ValueUnit              string    `json:"value_unit"`
	ScannedSamples         int       `json:"scanned_samples"`
	ReturnedPaths          int       `json:"returned_paths"`
	ReturnedSelfFrames     int       `json:"returned_self_frames"`
	OmittedPathsLowerBound int       `json:"omitted_paths_lower_bound"`
	OmittedNodesLowerBound int       `json:"omitted_nodes_lower_bound"`
	Partial                bool      `json:"partial"`
	PartialReasons         []string  `json:"partial_reasons,omitempty"`
	NewestProfileEnd       time.Time `json:"newest_profile_end,omitempty"`
}

type AllocationTopPath struct {
	Rank        int      `json:"rank"`
	LeafFrame   string   `json:"leaf_frame"`
	TotalValue  uint64   `json:"total_value"`
	SelfValue   uint64   `json:"self_value"`
	Percent     float64  `json:"percent"`
	Category    string   `json:"category"`
	SampleCount int      `json:"sample_count"`
	Path        []string `json:"path"`
}

type AllocationTopSelfFrame struct {
	Rank      int     `json:"rank"`
	Frame     string  `json:"frame"`
	SelfValue uint64  `json:"self_value"`
	Percent   float64 `json:"percent"`
	Category  string  `json:"category"`
}

type AllocationInsight struct {
	Severity      string `json:"severity"`
	Category      string `json:"category"`
	MessageCode   string `json:"message_code"`
	EvidenceFrame string `json:"evidence_frame"`
	EvidenceValue uint64 `json:"evidence_value"`
}

type AllocationLimitation struct {
	Code        string `json:"code"`
	MessageCode string `json:"message_code"`
}

func QueryAllocationSummary(ctx context.Context, repo ProfileQueryStore, statuses TargetStatusQueryStore, ingestion IngestionQueryStore, q AllocationSummaryQuery, exporter *metrics.Exporter) (AllocationSummary, error) {
	normalized, err := normalizeAllocationSummaryQuery(q)
	if err != nil {
		return AllocationSummary{}, err
	}
	window := domain.TimeWindow{StartedAt: normalized.Start, EndsAt: normalized.End}
	result := AllocationSummary{
		SchemaVersion:  1,
		RequestedScope: allocationScope(q, true),
		EffectiveScope: allocationScope(normalized, false),
		Semantics:      normalized.ProfileType.Semantics(window),
		TopPaths:       []AllocationTopPath{},
		TopSelfFrames:  []AllocationTopSelfFrame{},
		Insights:       []AllocationInsight{},
		Limitations:    []AllocationLimitation{},
	}
	result.Coverage.ProfileType = normalized.ProfileType.String()
	result.Coverage.ValueUnit = result.Semantics.ValueUnit

	fetchStarted := time.Now()
	coverageRows, err := repo.QueryProfileTargetSummary(ctx, clickhouse.ProfileQuery{
		Namespace:   normalized.Namespace,
		Service:     normalized.Service,
		Pod:         normalized.Pod,
		Container:   normalized.Container,
		ProcessID:   allocationProcessID(normalized),
		ProfileType: normalized.ProfileType,
		Start:       normalized.Start,
		End:         normalized.End,
		Limit:       5000,
	})
	if err != nil {
		return AllocationSummary{}, err
	}
	samples, err := repo.QueryTopStackSamples(ctx, clickhouse.ProfileQuery{
		Namespace:   normalized.Namespace,
		Service:     normalized.Service,
		Pod:         normalized.Pod,
		Container:   normalized.Container,
		ProcessID:   allocationProcessID(normalized),
		ProfileType: normalized.ProfileType,
		Start:       normalized.Start,
		End:         normalized.End,
		Limit:       maxInt(normalized.PathLimit, normalized.SelfFrameLimit, normalized.NodeLimit) + 1,
	})
	if err != nil {
		return AllocationSummary{}, err
	}
	recordMetric(exporter, "java_profiler_query_allocation_summary_fetch_seconds_total", time.Since(fetchStarted).Seconds())

	coverageTotal, coverageSamples, newestProfileEnd := allocationCoverage(coverageRows)
	if coverageTotal == 0 || len(samples) == 0 {
		result.Coverage.EmptyState = classifyAllocationEmptyState(ctx, statuses, ingestion, normalized)
		return result, nil
	}

	buildStarted := time.Now()
	result = buildAllocationSummary(result, samples, normalized, coverageTotal, coverageSamples, newestProfileEnd)
	if coverageTotal > 0 && (len(result.TopPaths) == 0 || len(result.TopSelfFrames) == 0) {
		result.Coverage.HasData = false
		result.Coverage.EmptyState = "no_stack_frames"
		result.Coverage.Partial = true
		result.Coverage.PartialReasons = appendReason(result.Coverage.PartialReasons, "no_stack_frames")
		result.Limitations = append(result.Limitations, AllocationLimitation{Code: "no_stack_frames", MessageCode: "allocation.empty.no_stack_frames"})
	}
	recordMetric(exporter, "java_profiler_query_allocation_summary_build_seconds_total", time.Since(buildStarted).Seconds())
	recordMetric(exporter, "java_profiler_query_allocation_summary_samples_total", float64(result.Coverage.ScannedSamples))
	recordMetric(exporter, "java_profiler_query_allocation_summary_paths_total", float64(len(result.TopPaths)))
	if result.Coverage.Partial {
		recordMetric(exporter, "java_profiler_query_allocation_summary_partial_total", 1)
	}
	return result, nil
}

func normalizeAllocationSummaryQuery(q AllocationSummaryQuery) (AllocationSummaryQuery, error) {
	q.Namespace = normalizeScopeValue(q.Namespace)
	q.Service = normalizeScopeValue(q.Service)
	q.Pod = normalizeScopeValue(q.Pod)
	q.Container = normalizeScopeValue(q.Container)
	q.JVM = normalizeScopeValue(q.JVM)
	if q.Namespace == "" {
		return q, AllocationSummaryValidationError{
			Code:            AllocationSummaryValidationNamespaceRequired,
			Field:           "namespace",
			Message:         "Allocation summary requires a namespace.",
			SuggestedAction: "Select a namespace before opening allocation Top Table evidence.",
		}
	}
	if q.ProfileType != domain.ProfileTypeAllocBytes && q.ProfileType != domain.ProfileTypeAllocObjects {
		return q, AllocationSummaryValidationError{
			Code:            AllocationSummaryValidationInvalidProfileType,
			Field:           "profile_type",
			Message:         "Allocation summary supports java_allocation_bytes or java_allocation_objects.",
			SuggestedAction: "Use an allocation profile type for allocation Top Table evidence.",
		}
	}
	if q.JVM != "" {
		processID, err := strconv.Atoi(q.JVM)
		if err != nil || processID <= 0 {
			return q, AllocationSummaryValidationError{
				Code:            AllocationSummaryValidationInvalidJVM,
				Field:           "jvm",
				Message:         "Allocation summary JVM scope must be a positive process id.",
				SuggestedAction: "Select a JVM from live suggestions or clear the JVM filter.",
			}
		}
	}
	if q.Start.IsZero() || q.End.IsZero() || !q.Start.Before(q.End) {
		return q, AllocationSummaryValidationError{
			Code:            AllocationSummaryValidationInvalidTimeRange,
			Field:           "time_range",
			Message:         "Allocation summary requires a valid start and end time range.",
			SuggestedAction: "Choose a valid time range before opening allocation Top Table evidence.",
		}
	}
	if q.End.Sub(q.Start) > MaxAllocationSummaryRange {
		return q, AllocationSummaryValidationError{
			Code:            AllocationSummaryValidationRangeExceeded,
			Field:           "time_range",
			Message:         "Allocation summary time range exceeds the retention window.",
			SuggestedAction: "Choose a range within the 7-day profile retention window.",
		}
	}
	if q.Service == "" && q.Pod == "" && q.End.Sub(q.Start) > MaxAllocationNamespaceOnlyRange {
		return q, AllocationSummaryValidationError{
			Code:            AllocationSummaryValidationNamespaceOnlyRangeExceeded,
			Field:           "time_range",
			Message:         "Namespace-only allocation summary is limited to short windows.",
			SuggestedAction: "Select a service or Pod, or shorten the time range to 30 minutes or less.",
		}
	}
	q.PathLimit = boundedQueryLimit(q.PathLimit, DefaultAllocationPathLimit, MaxAllocationPathLimit)
	q.SelfFrameLimit = boundedQueryLimit(q.SelfFrameLimit, DefaultAllocationSelfFrameLimit, MaxAllocationSelfFrameLimit)
	q.NodeLimit = boundedQueryLimit(q.NodeLimit, DefaultAllocationNodeLimit, MaxAllocationNodeLimit)
	return q, nil
}

func buildAllocationSummary(result AllocationSummary, samples []clickhouse.TopStackSample, q AllocationSummaryQuery, coverageTotal uint64, coverageSamples int, newestProfileEnd time.Time) AllocationSummary {
	type pathAgg struct {
		frames []string
		total  uint64
		count  int
	}
	byPath := map[string]*pathAgg{}
	bySelf := map[string]uint64{}
	for _, sample := range samples {
		if sample.Value == 0 || len(sample.Frames) == 0 {
			continue
		}
		if sample.EndedAt.After(newestProfileEnd) {
			newestProfileEnd = sample.EndedAt
		}
		frames := capFrames(sample.Frames, 128)
		key := strings.Join(frames, "\x00")
		current := byPath[key]
		if current == nil {
			current = &pathAgg{frames: frames}
			byPath[key] = current
		}
		current.total += sample.Value
		current.count += maxInt(sample.SampleCount, 1)
		leaf := frames[len(frames)-1]
		bySelf[leaf] += sample.Value
	}
	paths := make([]*pathAgg, 0, len(byPath))
	for _, item := range byPath {
		paths = append(paths, item)
	}
	sort.Slice(paths, func(i, j int) bool {
		if paths[i].total != paths[j].total {
			return paths[i].total > paths[j].total
		}
		return strings.Join(paths[i].frames, "\x00") < strings.Join(paths[j].frames, "\x00")
	})
	if len(paths) > q.PathLimit {
		result.Coverage.Partial = true
		result.Coverage.PartialReasons = appendReason(result.Coverage.PartialReasons, "path_limit")
		result.Coverage.OmittedPathsLowerBound = len(paths) - q.PathLimit
		paths = paths[:q.PathLimit]
	}
	nodeCount := 1
	for i, item := range paths {
		nextNodeCount := nodeCount + len(item.frames)
		if nextNodeCount > q.NodeLimit {
			result.Coverage.Partial = true
			result.Coverage.PartialReasons = appendReason(result.Coverage.PartialReasons, "node_limit")
			result.Coverage.OmittedNodesLowerBound += nextNodeCount - q.NodeLimit
			result.Coverage.OmittedPathsLowerBound += len(paths) - i
			break
		}
		nodeCount = nextNodeCount
		leaf := item.frames[len(item.frames)-1]
		result.TopPaths = append(result.TopPaths, AllocationTopPath{
			Rank:        i + 1,
			LeafFrame:   leaf,
			TotalValue:  item.total,
			SelfValue:   bySelf[leaf],
			Percent:     allocationPercent(item.total, coverageTotal),
			Category:    CategorizeAllocationFrames(item.frames),
			SampleCount: item.count,
			Path:        item.frames,
		})
	}

	type selfAgg struct {
		frame string
		value uint64
	}
	selfRows := make([]selfAgg, 0, len(bySelf))
	for frame, value := range bySelf {
		selfRows = append(selfRows, selfAgg{frame: frame, value: value})
	}
	sort.Slice(selfRows, func(i, j int) bool {
		if selfRows[i].value != selfRows[j].value {
			return selfRows[i].value > selfRows[j].value
		}
		return selfRows[i].frame < selfRows[j].frame
	})
	if len(selfRows) > q.SelfFrameLimit {
		result.Coverage.Partial = true
		result.Coverage.PartialReasons = appendReason(result.Coverage.PartialReasons, "self_frame_limit")
		selfRows = selfRows[:q.SelfFrameLimit]
	}
	for i, item := range selfRows {
		result.TopSelfFrames = append(result.TopSelfFrames, AllocationTopSelfFrame{
			Rank:      i + 1,
			Frame:     item.frame,
			SelfValue: item.value,
			Percent:   allocationPercent(item.value, coverageTotal),
			Category:  CategorizeAllocationFrames([]string{item.frame}),
		})
	}

	result.Coverage.HasData = coverageTotal > 0
	result.Coverage.TotalValue = coverageTotal
	result.Coverage.ScannedSamples = coverageSamples
	result.Coverage.ReturnedPaths = len(result.TopPaths)
	result.Coverage.ReturnedSelfFrames = len(result.TopSelfFrames)
	result.Coverage.NewestProfileEnd = newestProfileEnd
	result.Insights = allocationInsights(result.TopPaths)
	for _, reason := range result.Coverage.PartialReasons {
		result.Limitations = append(result.Limitations, AllocationLimitation{Code: "partial_result", MessageCode: "allocation.partial." + reason})
	}
	return result
}

func allocationCoverage(rows []clickhouse.ProfileTargetSummary) (uint64, int, time.Time) {
	var total uint64
	var samples int
	var newest time.Time
	for _, row := range rows {
		total += row.TotalValue
		samples += row.SampleCount
		if row.NewestProfileEnd.After(newest) {
			newest = row.NewestProfileEnd
		}
	}
	return total, samples, newest
}

func allocationProcessID(q AllocationSummaryQuery) int {
	if q.JVM == "" {
		return 0
	}
	processID, _ := strconv.Atoi(q.JVM)
	return processID
}

func CategorizeAllocationFrames(frames []string) string {
	joined := strings.ToLower(strings.Join(frames, " "))
	switch {
	case strings.Contains(joined, "stringbuilder") || strings.Contains(joined, "abstractstringbuilder") || strings.Contains(joined, "string.<init>") || strings.Contains(joined, "stringconcatfactory") || strings.Contains(joined, "formatter"):
		return "string_construction"
	case strings.Contains(joined, "arrays.copyof") || strings.Contains(joined, "arrays.copyofrange") || strings.Contains(joined, "system.arraycopy"):
		return "array_copy"
	case strings.Contains(joined, "hashmap.resize") || strings.Contains(joined, "arraylist.grow"):
		return "collection_growth"
	case strings.Contains(joined, "threadlocal"):
		return "thread_local_cleanup"
	case strings.Contains(joined, "db.query") || strings.Contains(joined, "executesqlbuilder") || strings.Contains(joined, "multiquerybuilder") || strings.Contains(joined, "basedb") || strings.Contains(joined, "repository") || strings.Contains(joined, "businessdatareader"):
		return "database_query_building"
	case strings.Contains(joined, "uribuilder") || strings.Contains(joined, "urlcreator") || strings.Contains(joined, "dbconfig"):
		return "url_or_config_building"
	case strings.Contains(joined, "json") || strings.Contains(joined, "jackson") || strings.Contains(joined, "gson") || strings.Contains(joined, "protobuf"):
		return "serialization_or_json"
	case strings.Contains(joined, ".so") || strings.Contains(joined, "[vdso]"):
		return "native_or_runtime"
	default:
		return "application_other"
	}
}

func classifyAllocationEmptyState(ctx context.Context, statuses TargetStatusQueryStore, ingestion IngestionQueryStore, q AllocationSummaryQuery) string {
	if statuses != nil {
		rows, err := statuses.LatestByService(ctx, clickhouse.TargetStatusQuery{Namespace: q.Namespace, Service: q.Service, Start: q.Start, End: q.End, Limit: MaxTargetStatusLimit})
		if err != nil {
			return "query_error"
		}
		matched := false
		for _, status := range rows {
			if q.Pod != "" && status.Target.Pod != q.Pod {
				continue
			}
			matched = true
			switch status.Reason {
			case domain.StatusReasonUnsupportedJVM:
				return "unsupported_runtime"
			case domain.StatusReasonDisabledByMetadata, domain.StatusReasonTemporaryExpired:
				return "profiling_disabled"
			case domain.StatusReasonUploadDropped, domain.StatusReasonUploadRetryable, domain.StatusReasonStorageRejected:
				return "ingestion_gap"
			}
		}
		if !matched {
			return "no_matching_target"
		}
	}
	if ingestion != nil {
		health, err := QueryIngestionHealth(ctx, ingestion, nil)
		if err != nil {
			return "query_error"
		}
		if health.Totals.DroppedSamples > 0 || health.Totals.DroppedStacks > 0 || health.Totals.Retryable > 0 || health.Totals.Rejected > 0 {
			return "ingestion_gap"
		}
	}
	return "no_samples_in_range"
}

func allocationScope(q AllocationSummaryQuery, requested bool) AllocationSummaryScope {
	format := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	value := func(v string) string {
		if requested && strings.TrimSpace(v) == "" {
			return "all"
		}
		return v
	}
	return AllocationSummaryScope{
		Namespace: q.Namespace,
		Service:   value(q.Service),
		Pod:       value(q.Pod),
		Container: value(q.Container),
		JVM:       value(q.JVM),
		Start:     format(q.Start),
		End:       format(q.End),
	}
}

func allocationInsights(paths []AllocationTopPath) []AllocationInsight {
	seen := map[string]struct{}{}
	var insights []AllocationInsight
	for _, path := range paths {
		if path.Category == "application_other" {
			continue
		}
		if _, ok := seen[path.Category]; ok {
			continue
		}
		seen[path.Category] = struct{}{}
		insights = append(insights, AllocationInsight{
			Severity:      "info",
			Category:      path.Category,
			MessageCode:   "allocation." + path.Category + ".dominant",
			EvidenceFrame: path.LeafFrame,
			EvidenceValue: path.TotalValue,
		})
		if len(insights) >= 3 {
			break
		}
	}
	return insights
}

func normalizeScopeValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "all") {
		return ""
	}
	return value
}

func capFrames(frames []string, limit int) []string {
	if len(frames) <= limit {
		return append([]string(nil), frames...)
	}
	return append([]string(nil), frames[len(frames)-limit:]...)
}

func allocationPercent(value, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) * 100 / float64(total)
}

func appendReason(reasons []string, reason string) []string {
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func maxInt(values ...int) int {
	max := 0
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}
