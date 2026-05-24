package app

import (
	"testing"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

func TestTopStacksSeparatesSelfAndTotal(t *testing.T) {
	samples := []clickhouse.TopStackSample{
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
	if handle.Semantics.ValueUnit != "nanoseconds" || handle.SelfDisplay != "0 ns" || burn.SelfDisplay != "8 ns" {
		t.Fatalf("unexpected semantics/display: handle=%+v burn=%+v", handle, burn)
	}
}

func TestTopStacksKeepsJavaRowsWhenRuntimeFramesExist(t *testing.T) {
	samples := []clickhouse.TopStackSample{
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

func TestTopStacksKeepsSameSymbolFromDifferentPackagesDistinct(t *testing.T) {
	samples := []clickhouse.TopStackSample{
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "com.foo.CheckoutService.priceCart:10"}, Value: 6},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "org.acme.CheckoutService.priceCart:42"}, Value: 4},
	}

	rows := rankTopStacks(samples)
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2 distinct package rows: %#v", len(rows), rows)
	}
	foo := findTopStackRowByLocation(rows, "com.foo.CheckoutService.priceCart:10")
	acme := findTopStackRowByLocation(rows, "org.acme.CheckoutService.priceCart:42")

	if foo.Symbol != "CheckoutService.priceCart" || acme.Symbol != "CheckoutService.priceCart" {
		t.Fatalf("unexpected display symbols: foo=%q acme=%q", foo.Symbol, acme.Symbol)
	}
	if foo.Total != 6 || foo.Self != 6 {
		t.Fatalf("foo total/self = %d/%d, want 6/6", foo.Total, foo.Self)
	}
	if acme.Total != 4 || acme.Self != 4 {
		t.Fatalf("acme total/self = %d/%d, want 4/4", acme.Total, acme.Self)
	}
}

func TestTopStacksDoesNotPromoteRuntimeOnlyFrames(t *testing.T) {
	samples := []clickhouse.TopStackSample{
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "libc.so.6.pthread_cond_timedwait", "libjvm.so.PlatformMonitor::wait", "java.lang.Thread.run:1583"}, Value: 21},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "so.6", "[vdso]", "6.clock_gettime"}, Value: 9},
	}

	rows := rankTopStacks(samples)
	if len(rows) != 0 {
		t.Fatalf("runtime/native-only samples should not produce top table rows: %#v", rows)
	}
}

func TestTopStacksClassifiesRepresentativeProfileFrames(t *testing.T) {
	samples := []clickhouse.TopStackSample{
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "libasyncProfiler.so.StackWalker::walkVM"}, Value: 10},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "libc-2.17.so.__clock_gettime"}, Value: 10},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "pthread_cond_timedwait"}, Value: 10},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "java.lang.Thread.run:1583"}, Value: 10},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "jdk.internal.reflect.NativeMethodAccessorImpl.invoke0"}, Value: 10},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "com.acme.orders.CheckoutService.priceCart:42"}, Value: 20},
	}

	rows := rankTopStacks(samples)
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want only the application Java row: %#v", len(rows), rows)
	}
	if rows[0].Location != "com.acme.orders.CheckoutService.priceCart:42" {
		t.Fatalf("unexpected application row: %#v", rows[0])
	}
}

func TestTopStacksKeepsApplicationAdapterClasses(t *testing.T) {
	samples := []clickhouse.TopStackSample{
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "com.acme.payments.PaymentAdapter.apply:42"}, Value: 11},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "I2C adapter"}, Value: 5},
	}

	rows := rankTopStacks(samples)
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want only the application adapter row: %#v", len(rows), rows)
	}
	if rows[0].Location != "com.acme.payments.PaymentAdapter.apply:42" {
		t.Fatalf("unexpected adapter row: %#v", rows[0])
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

func findTopStackRowByLocation(rows []TopStackRow, location string) TopStackRow {
	for _, row := range rows {
		if row.Location == location {
			return row
		}
	}
	return TopStackRow{}
}
