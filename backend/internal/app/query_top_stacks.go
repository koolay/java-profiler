package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/backend/internal/metrics"
	"github.com/koolay/java-profiler/domain"
)

type TopStackRow struct {
	Symbol       string                       `json:"symbol"`
	Location     string                       `json:"location"`
	ProfileType  string                       `json:"profile_type"`
	Self         uint64                       `json:"self"`
	Total        uint64                       `json:"total"`
	SelfDisplay  string                       `json:"self_display"`
	TotalDisplay string                       `json:"total_display"`
	SelfPercent  string                       `json:"self_percent"`
	TotalPercent string                       `json:"total_percent"`
	Semantics    domain.ProfileValueSemantics `json:"semantics"`
}

var topStackExcludedMethods = map[string]struct{}{
	"read": {}, "write": {}, "open": {}, "close": {}, "poll": {}, "select": {}, "epoll": {}, "recv": {}, "send": {}, "accept": {}, "connect": {}, "syscall": {}, "nanosleep": {},
}

var topStackExcludedPrefixes = []string{"java.", "javax.", "jdk.", "sun.", "com.sun.", "org.graalvm.", "lib"}

var topStackExcludedFragments = []string{".so", "[vdso]", "pthread", "clock_gettime", "adapter", "stubroutine", "vtablestub", "itable stub"}

func QueryTopStacks(ctx context.Context, repo ProfileQueryStore, q clickhouse.ProfileQuery, exporter *metrics.Exporter) ([]TopStackRow, error) {
	fetchStarted := time.Now()
	samples, err := repo.QueryTopStackSamples(ctx, q)
	if err != nil {
		return nil, err
	}
	recordMetric(exporter, "java_profiler_query_top_stacks_fetch_seconds_total", time.Since(fetchStarted).Seconds())
	rankStarted := time.Now()
	rows, stats := buildTopStacks(samples, domain.TimeWindow{StartedAt: q.Start, EndsAt: q.End})
	recordMetric(exporter, "java_profiler_query_top_stacks_rank_seconds_total", time.Since(rankStarted).Seconds())
	recordMetric(exporter, "java_profiler_query_top_stacks_samples_total", float64(stats.samples))
	recordMetric(exporter, "java_profiler_query_top_stacks_frames_total", float64(stats.frames))
	recordMetric(exporter, "java_profiler_query_top_stacks_rows_total", float64(len(rows)))
	return rows, nil
}

func rankTopStacks(samples []clickhouse.TopStackSample) []TopStackRow {
	rows, _ := buildTopStacks(samples, domain.TimeWindow{})
	return rows
}

type topStackStats struct {
	samples int
	frames  int
}

func buildTopStacks(samples []clickhouse.TopStackSample, window domain.TimeWindow) ([]TopStackRow, topStackStats) {
	type contribution struct {
		location    string
		profileType domain.ProfileType
		self        uint64
		total       uint64
	}

	classifier := newTopStackFrameClassifier()
	var totalSamples uint64
	stats := topStackStats{samples: len(samples)}
	byLocation := make(map[string]contribution)
	for _, sample := range samples {
		if sample.Value == 0 || len(sample.Frames) == 0 {
			continue
		}
		stats.frames += len(sample.Frames)
		totalSamples += sample.Value
		leafLocation := classifier.location(sample.Frames[len(sample.Frames)-1])
		for _, frame := range sample.Frames {
			location, ok := classifier.classify(frame)
			if !ok {
				continue
			}
			if location == "" {
				continue
			}
			current := byLocation[location]
			current.location = location
			current.profileType = sample.ProfileType
			current.total += sample.Value
			if location == leafLocation {
				current.self += sample.Value
			}
			byLocation[location] = current
		}
	}

	rows := make([]TopStackRow, 0, len(byLocation))
	for location, contribution := range byLocation {
		semantics := contribution.profileType.Semantics(window)
		rows = append(rows, TopStackRow{
			Symbol:       classifier.symbol(location),
			Location:     contribution.location,
			ProfileType:  string(contribution.profileType),
			Self:         contribution.self,
			Total:        contribution.total,
			SelfDisplay:  domain.FormatProfileValue(contribution.profileType, contribution.self, window),
			TotalDisplay: domain.FormatProfileValue(contribution.profileType, contribution.total, window),
			SelfPercent:  percent(contribution.self, totalSamples),
			TotalPercent: percent(contribution.total, totalSamples),
			Semantics:    semantics,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Total != rows[j].Total {
			return rows[i].Total > rows[j].Total
		}
		if rows[i].Self != rows[j].Self {
			return rows[i].Self > rows[j].Self
		}
		if rows[i].Symbol != rows[j].Symbol {
			return rows[i].Symbol < rows[j].Symbol
		}
		return rows[i].Location < rows[j].Location
	})
	return rows, stats
}

type topStackFrameClassifier struct {
	locationCache map[string]string
	symbolCache   map[string]string
	javaCache     map[string]bool
}

func newTopStackFrameClassifier() *topStackFrameClassifier {
	return &topStackFrameClassifier{
		locationCache: map[string]string{},
		symbolCache:   map[string]string{},
		javaCache:     map[string]bool{},
	}
}

func (c *topStackFrameClassifier) location(frame string) string {
	if cached, ok := c.locationCache[frame]; ok {
		return cached
	}
	normalized := strings.ReplaceAll(frame, "/", ".")
	c.locationCache[frame] = normalized
	return normalized
}

func (c *topStackFrameClassifier) symbol(frame string) string {
	if cached, ok := c.symbolCache[frame]; ok {
		return cached
	}
	normalized := c.location(frame)
	if lineIndex := strings.LastIndex(normalized, ":"); lineIndex > -1 {
		if isDigits(normalized[lineIndex+1:]) {
			normalized = normalized[:lineIndex]
		}
	}
	parts := strings.Split(normalized, ".")
	if len(parts) >= 2 {
		normalized = strings.Join(parts[len(parts)-2:], ".")
	}
	c.symbolCache[frame] = normalized
	return normalized
}

func (c *topStackFrameClassifier) classify(frame string) (string, bool) {
	if cached, ok := c.javaCache[frame]; ok {
		if !cached {
			return "", false
		}
		return c.locationCache[frame], true
	}
	symbol := c.symbol(frame)
	if symbol == "" || strings.Contains(frame, "$$Lambda") {
		c.javaCache[frame] = false
		return "", false
	}
	location := c.location(frame)
	locationWithoutLine := frameWithoutLine(location)
	methodSeparator := strings.LastIndex(locationWithoutLine, ".")
	if methodSeparator < 0 {
		c.javaCache[frame] = false
		return "", false
	}
	method := locationWithoutLine[methodSeparator+1:]
	className := locationWithoutLine[:methodSeparator]
	simpleClass := className
	if dot := strings.LastIndex(simpleClass, "."); dot > -1 {
		simpleClass = simpleClass[dot+1:]
	}
	normalized := strings.ToLower(location)
	normalizedClass := strings.ToLower(className)
	normalizedMethod := strings.ToLower(method)
	if simpleClass == "" || method == "" {
		c.javaCache[frame] = false
		return "", false
	}
	if strings.ContainsAny(method, " \t") || strings.ContainsAny(simpleClass, " \t") {
		c.javaCache[frame] = false
		return "", false
	}
	if first := simpleClass[0]; first < 'A' || first > 'Z' {
		c.javaCache[frame] = false
		return "", false
	}
	for _, prefix := range topStackExcludedPrefixes {
		if strings.HasPrefix(normalizedClass, prefix) {
			c.javaCache[frame] = false
			return "", false
		}
	}
	for _, fragment := range topStackExcludedFragments {
		if strings.Contains(normalized, fragment) {
			c.javaCache[frame] = false
			return "", false
		}
	}
	_, excluded := topStackExcludedMethods[normalizedMethod]
	c.javaCache[frame] = !excluded
	if excluded {
		return "", false
	}
	return location, true
}

func frameWithoutLine(frame string) string {
	if lineIndex := strings.LastIndex(frame, ":"); lineIndex > -1 && isDigits(frame[lineIndex+1:]) {
		return frame[:lineIndex]
	}
	return frame
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func percent(value, total uint64) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", (float64(value)/float64(total))*100)
}

func recordMetric(exporter *metrics.Exporter, name string, value float64) {
	if exporter == nil {
		return
	}
	exporter.Add(name, value)
}
