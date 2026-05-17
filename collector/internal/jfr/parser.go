package jfr

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	grafanaparser "github.com/grafana/jfr-parser/parser"
	"github.com/grafana/jfr-parser/parser/types"
	"github.com/grafana/jfr-parser/parser/types/def"
)

type Event struct {
	Type   string
	Value  uint64
	Frames []string
	Labels map[string]string
}

type Parser struct{}

func (Parser) Parse(r io.Reader) ([]Event, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if bytes.HasPrefix(data, []byte("FLR\x00")) {
		return parseAsyncProfilerJFR(data)
	}
	return parseFixtureEvents(bytes.NewReader(data))
}

func (Parser) ParseFile(path string) ([]Event, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return (Parser{}).Parse(bytes.NewReader(data))
}

func parseFixtureEvents(r io.Reader) ([]Event, error) {
	scanner := bufio.NewScanner(r)
	var events []Event
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, err
		}
		event := Event{Type: parts[0], Value: value, Frames: strings.Split(parts[2], ";")}
		if len(parts) > 3 {
			event.Labels = parseFixtureLabels(parts[3])
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}

func parseFixtureLabels(raw string) map[string]string {
	labels := map[string]string{}
	for _, pair := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		labels[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return labels
}

func parseAsyncProfilerJFR(data []byte) ([]Event, error) {
	p := grafanaparser.NewParser(data, grafanaparser.Options{SymbolProcessor: grafanaparser.ProcessSymbols})
	var events []Event
	for {
		typ, err := p.ParseEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse async-profiler jfr: %w", err)
		}
		switch typ {
		case p.TypeMap.T_EXECUTION_SAMPLE:
			events = append(events, Event{Type: "execution_sample", Value: 1, Frames: stackFrames(p, p.ExecutionSample.StackTrace)})
		case p.TypeMap.T_WALL_CLOCK_SAMPLE:
			frames := stackFrames(p, p.WallClockSample.StackTrace)
			value := uint64(p.WallClockSample.Samples)
			if value == 0 {
				value = 1
			}
			events = append(events, Event{Type: "wall_clock", Value: value, Frames: frames})
			if isJavaIOStack(frames) {
				events = append(events, Event{Type: "io_wait", Value: value, Frames: frames})
			}
			if isJVMGCStack(frames) {
				events = append(events, Event{Type: "gc_pause", Value: value * DefaultWallClockSampleValueNS, Frames: frames, Labels: map[string]string{"collector": "JVM", "action": "wall-clock GC activity", "cause": "GC thread sample"}})
			}
		case p.TypeMap.T_ALLOC_IN_NEW_TLAB:
			frames := stackFrames(p, p.ObjectAllocationInNewTLAB.StackTrace)
			events = append(events, Event{Type: "alloc_objects", Value: 1, Frames: frames}, Event{Type: "alloc_bytes", Value: p.ObjectAllocationInNewTLAB.TlabSize, Frames: frames})
		case p.TypeMap.T_ALLOC_OUTSIDE_TLAB:
			frames := stackFrames(p, p.ObjectAllocationOutsideTLAB.StackTrace)
			events = append(events, Event{Type: "alloc_objects", Value: 1, Frames: frames}, Event{Type: "alloc_bytes", Value: p.ObjectAllocationOutsideTLAB.AllocationSize, Frames: frames})
		case p.TypeMap.T_ALLOC_SAMPLE:
			frames := stackFrames(p, p.ObjectAllocationSample.StackTrace)
			events = append(events, Event{Type: "alloc_objects", Value: 1, Frames: frames}, Event{Type: "alloc_bytes", Value: p.ObjectAllocationSample.Weight, Frames: frames})
		case p.TypeMap.T_MONITOR_ENTER:
			frames := stackFrames(p, p.JavaMonitorEnter.StackTrace)
			events = append(events, Event{Type: "monitor_enter", Value: 1, Frames: frames}, Event{Type: "lock_delay", Value: p.JavaMonitorEnter.Duration, Frames: frames})
		case def.TypeID(0):
			continue
		}
	}
	return events, nil
}

func isJavaIOStack(frames []string) bool {
	for _, frame := range frames {
		normalized := strings.ReplaceAll(frame, "/", ".")
		if strings.Contains(normalized, "java.io.") ||
			strings.Contains(normalized, "java.net.") ||
			strings.Contains(normalized, "java.nio.") ||
			strings.Contains(normalized, "sun.nio.ch.") ||
			strings.Contains(normalized, "jdk.internal.net.http.") ||
			strings.Contains(normalized, "io.netty.channel.") ||
			strings.Contains(normalized, "okhttp3.") ||
			strings.Contains(normalized, "org.apache.http.") {
			return true
		}
	}
	return false
}

func isJVMGCStack(frames []string) bool {
	for _, frame := range frames {
		normalized := strings.ToLower(strings.ReplaceAll(frame, "/", "."))
		if strings.Contains(normalized, ".gc.") ||
			strings.Contains(normalized, "java.lang.system.gc") ||
			strings.Contains(normalized, "java.lang.runtime.gc") ||
			strings.Contains(normalized, "garbagecollect") ||
			strings.Contains(normalized, "g1") ||
			strings.Contains(normalized, "shenandoah") ||
			strings.Contains(normalized, "zgc") ||
			strings.Contains(normalized, "vm_gc") {
			return true
		}
	}
	return false
}

func stackFrames(p *grafanaparser.Parser, ref types.StackTraceRef) []string {
	stack := p.GetStacktrace(ref)
	if stack == nil {
		return nil
	}
	frames := make([]string, 0, len(stack.Frames))
	for _, frame := range stack.Frames {
		method := p.GetMethod(frame.Method)
		if method == nil {
			continue
		}
		methodName := p.GetSymbolString(method.Name)
		class := p.GetClass(method.Type)
		if class == nil {
			frames = append(frames, methodName)
			continue
		}
		className := p.GetSymbolString(class.Name)
		if frame.LineNumber > 0 {
			frames = append(frames, fmt.Sprintf("%s.%s:%d", className, methodName, frame.LineNumber))
		} else {
			frames = append(frames, className+"."+methodName)
		}
	}
	return frames
}
