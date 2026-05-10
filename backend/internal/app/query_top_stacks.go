package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

type TopStackRow struct {
	Symbol       string `json:"symbol"`
	Location     string `json:"location"`
	ProfileType  string `json:"profile_type"`
	Self         uint64 `json:"self"`
	Total        uint64 `json:"total"`
	SelfPercent  string `json:"self_percent"`
	TotalPercent string `json:"total_percent"`
}

func QueryTopStacks(ctx context.Context, repo ProfileQueryStore, q clickhouse.ProfileQuery) ([]TopStackRow, error) {
	samples, err := repo.QuerySamples(ctx, q)
	if err != nil {
		return nil, err
	}
	return rankTopStacks(samples), nil
}

func rankTopStacks(samples []clickhouse.ProfileSample) []TopStackRow {
	type contribution struct {
		location    string
		profileType domain.ProfileType
		self        uint64
		total       uint64
	}

	var totalSamples uint64
	byLocation := make(map[string]contribution)
	for _, sample := range samples {
		if sample.Value == 0 || len(sample.Frames) == 0 {
			continue
		}
		totalSamples += sample.Value
		included := topTableFrames(sample.Frames)
		if len(included) == 0 {
			continue
		}

		leaf := sample.Frames[len(sample.Frames)-1]
		leafLocation := frameLocation(leaf)
		for _, frame := range included {
			location := frameLocation(frame)
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
		rows = append(rows, TopStackRow{
			Symbol:       frameSymbol(location),
			Location:     contribution.location,
			ProfileType:  string(contribution.profileType),
			Self:         contribution.self,
			Total:        contribution.total,
			SelfPercent:  percent(contribution.self, totalSamples),
			TotalPercent: percent(contribution.total, totalSamples),
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
	return rows
}

func topTableFrames(frames []string) []string {
	javaFrames := make([]string, 0, len(frames))
	for _, frame := range frames {
		if isApplicationJavaFrame(frame) {
			javaFrames = append(javaFrames, frame)
		}
	}
	if len(javaFrames) > 0 {
		return javaFrames
	}
	return frames
}

func frameSymbol(frame string) string {
	normalized := frameLocation(frame)
	if lineIndex := strings.LastIndex(normalized, ":"); lineIndex > -1 {
		if isDigits(normalized[lineIndex+1:]) {
			normalized = normalized[:lineIndex]
		}
	}
	parts := strings.Split(normalized, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return normalized
}

func frameLocation(frame string) string {
	return strings.ReplaceAll(frame, "/", ".")
}

func isApplicationJavaFrame(frame string) bool {
	symbol := frameSymbol(frame)
	if symbol == "" || strings.Contains(frame, "$$Lambda") {
		return false
	}
	location := frameLocation(frame)
	locationWithoutLine := frameWithoutLine(location)
	methodSeparator := strings.LastIndex(locationWithoutLine, ".")
	if methodSeparator < 0 {
		return false
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
		return false
	}
	if strings.ContainsAny(method, " \t") || strings.ContainsAny(simpleClass, " \t") {
		return false
	}
	if first := simpleClass[0]; first < 'A' || first > 'Z' {
		return false
	}
	excludedPrefixes := []string{"java.", "javax.", "jdk.", "sun.", "com.sun.", "org.graalvm.", "lib"}
	for _, prefix := range excludedPrefixes {
		if strings.HasPrefix(normalizedClass, prefix) {
			return false
		}
	}
	excludedFragments := []string{".so", "[vdso]", "pthread", "clock_gettime", "adapter", "stubroutine", "vtablestub", "itable stub"}
	for _, fragment := range excludedFragments {
		if strings.Contains(normalized, fragment) {
			return false
		}
	}
	excludedMethods := map[string]struct{}{
		"read": {}, "write": {}, "open": {}, "close": {}, "poll": {}, "select": {}, "epoll": {}, "recv": {}, "send": {}, "accept": {}, "connect": {}, "syscall": {}, "nanosleep": {},
	}
	_, excluded := excludedMethods[normalizedMethod]
	return !excluded
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
