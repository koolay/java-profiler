package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Exporter struct {
	mu       sync.RWMutex
	counters map[string]float64
	gauges   map[string]float64
}

func NewExporter() *Exporter {
	return &Exporter{
		counters: map[string]float64{},
		gauges:   map[string]float64{},
	}
}

func (e *Exporter) Inc(name string) {
	e.Add(name, 1)
}

func (e *Exporter) Add(name string, value float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.counters[sanitize(name)] += value
}

func (e *Exporter) Set(name string, value float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.gauges[sanitize(name)] = value
}

func (e *Exporter) Snapshot() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var lines []string
	for name, value := range e.counters {
		lines = append(lines, fmt.Sprintf("%s %g", name, value))
	}
	for name, value := range e.gauges {
		lines = append(lines, fmt.Sprintf("%s %g", name, value))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

func sanitize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	if name == "" {
		return "java_profiler_unknown"
	}
	return name
}
