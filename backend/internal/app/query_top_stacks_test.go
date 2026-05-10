package app

import (
	"testing"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

func TestTopStacksSeparatesSelfAndTotal(t *testing.T) {
	samples := []clickhouse.ProfileSample{
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "Demo.handleWork:93", "Demo.burnCpu:188"}, Value: 8},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "Demo.handleWork:93", "Demo.writeJson:232"}, Value: 2},
	}

	rows := rankTopStacks(samples)
	handle := findTopStackRow(rows, "Demo.handleWork")
	burn := findTopStackRow(rows, "Demo.burnCpu")

	if handle.Total != 10 || handle.Self != 0 {
		t.Fatalf("handleWork total/self = %d/%d, want 10/0", handle.Total, handle.Self)
	}
	if burn.Total != 8 || burn.Self != 8 {
		t.Fatalf("burnCpu total/self = %d/%d, want 8/8", burn.Total, burn.Self)
	}
	if handle.TotalPercent != "100.0%" || burn.SelfPercent != "80.0%" {
		t.Fatalf("unexpected percents: handle total=%s burn self=%s", handle.TotalPercent, burn.SelfPercent)
	}
}

func TestTopStacksKeepsJavaRowsWhenRuntimeFramesExist(t *testing.T) {
	samples := []clickhouse.ProfileSample{
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "java.lang.Thread.run:1583", "Demo.handleWork:93", "Demo.burnCpu:188", "libjvm.so"}, Value: 12},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "java.lang.Thread.run:1583", "Demo.handleWork:93", "Demo.writeJson:232"}, Value: 5},
	}

	rows := rankTopStacks(samples)
	handle := findTopStackRow(rows, "Demo.handleWork")
	burn := findTopStackRow(rows, "Demo.burnCpu")

	if handle.Symbol == "" || burn.Symbol == "" {
		t.Fatalf("expected Java rows to be present, got %#v", rows)
	}
	if handle.Total != 17 || handle.Self != 0 {
		t.Fatalf("handleWork total/self = %d/%d, want 17/0", handle.Total, handle.Self)
	}
	if burn.Total != 12 || burn.Self != 0 {
		t.Fatalf("burnCpu total/self = %d/%d, want 12/0 because native leaf owns self", burn.Total, burn.Self)
	}
	if len(rows) < 2 || rows[0].Symbol != "Demo.handleWork" {
		t.Fatalf("expected Java total row to rank first when runtime/native frames exist, got %#v", rows)
	}
}

func findTopStackRow(rows []TopStackRow, symbol string) TopStackRow {
	for _, row := range rows {
		if row.Symbol == symbol {
			return row
		}
	}
	return TopStackRow{}
}
