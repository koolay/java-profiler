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
		events = append(events, Event{Type: parts[0], Value: value, Frames: strings.Split(parts[2], ";")})
	}
	return events, scanner.Err()
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
