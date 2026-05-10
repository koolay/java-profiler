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
	bySymbol := make(map[string]contribution)
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
		for _, frame := range included {
			symbol := frameSymbol(frame)
			if symbol == "" {
				continue
			}
			current := bySymbol[symbol]
			current.location = bestLocation(current.location, frame)
			current.profileType = sample.ProfileType
			current.total += sample.Value
			if frame == leaf {
				current.self += sample.Value
			}
			bySymbol[symbol] = current
		}
	}

	rows := make([]TopStackRow, 0, len(bySymbol))
	for symbol, contribution := range bySymbol {
		rows = append(rows, TopStackRow{
			Symbol:       symbol,
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
		return rows[i].Symbol < rows[j].Symbol
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
	normalized := strings.ReplaceAll(frame, "/", ".")
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

func isApplicationJavaFrame(frame string) bool {
	symbol := frameSymbol(frame)
	if symbol == "" || strings.Contains(frame, "$$Lambda") {
		return false
	}
	normalized := strings.ToLower(strings.ReplaceAll(frame, "/", "."))
	methodSeparator := strings.LastIndex(frameWithoutLine(normalized), ".")
	if methodSeparator < 0 {
		return false
	}
	method := frameWithoutLine(normalized)[methodSeparator+1:]
	className := frameWithoutLine(normalized)[:methodSeparator]
	simpleClass := className
	if dot := strings.LastIndex(simpleClass, "."); dot > -1 {
		simpleClass = simpleClass[dot+1:]
	}
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
		if strings.HasPrefix(className, prefix) {
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
	_, excluded := excludedMethods[method]
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

func bestLocation(current, candidate string) string {
	if current == "" || candidate < current {
		return candidate
	}
	return current
}

func percent(value, total uint64) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", (float64(value)/float64(total))*100)
}
