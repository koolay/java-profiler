package domain

import (
	"sort"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	root "github.com/koolay/java-profiler/domain"
)

type BusyThread struct {
	ThreadID          int64
	ThreadName        string
	CPUTimeNS         uint64
	RunnableSnapshots int
	Confidence        root.Confidence
	Stacks            []string
}

type SlowThread struct {
	ThreadID   int64
	ThreadName string
	State      string
	Lock       string
	Stacks     []string
}

func BuildBusyThreads(snapshots []clickhouse.ThreadSnapshot) []BusyThread {
	var out []BusyThread
	for _, snapshot := range snapshots {
		confidence := root.ConfidenceSampledRUNNABLE
		var cpu uint64
		if snapshot.CPUTimeNS != nil {
			confidence = root.ConfidenceExactThreadCPU
			cpu = *snapshot.CPUTimeNS
		}
		runnable := 0
		if snapshot.State == "RUNNABLE" {
			runnable = 1
		}
		if runnable == 0 && cpu == 0 {
			continue
		}
		out = append(out, BusyThread{ThreadID: snapshot.ThreadID, ThreadName: snapshot.ThreadName, CPUTimeNS: cpu, RunnableSnapshots: runnable, Confidence: confidence, Stacks: snapshot.StackFrames})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPUTimeNS > out[j].CPUTimeNS })
	return out
}

func BuildSlowThreads(snapshots []clickhouse.ThreadSnapshot) []SlowThread {
	var out []SlowThread
	for _, snapshot := range snapshots {
		switch snapshot.State {
		case "BLOCKED", "WAITING", "TIMED_WAITING":
			out = append(out, SlowThread{ThreadID: snapshot.ThreadID, ThreadName: snapshot.ThreadName, State: snapshot.State, Lock: snapshot.BlockedLock, Stacks: snapshot.StackFrames})
		}
	}
	return out
}
