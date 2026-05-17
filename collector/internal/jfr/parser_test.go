package jfr

import (
	"strings"
	"testing"
)

func TestParserParsesFixtureEvents(t *testing.T) {
	events, err := (Parser{}).Parse(strings.NewReader("execution_sample|100|A.a;B.b\nalloc_bytes|42|C.c\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Value != 100 || len(events[0].Frames) != 2 {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestJavaIOStackClassifierUsesJavaOwnershipFrames(t *testing.T) {
	if !isJavaIOStack([]string{"com.example.Client.call", "sun/nio/ch/SocketDispatcher.read0"}) {
		t.Fatalf("expected Java socket stack to be classified as I/O wait")
	}
	if isJavaIOStack([]string{"com.example.CpuBurn.loop", "java.lang.Thread.run"}) {
		t.Fatalf("CPU-only stack should not be classified as I/O wait")
	}
}

func TestJVMGCStackClassifierUsesJVMOwnershipFrames(t *testing.T) {
	if !isJVMGCStack([]string{"jdk/internal/vm/G1CollectedHeap.doCollection"}) {
		t.Fatalf("expected JVM GC stack to be classified as GC evidence")
	}
	if !isJVMGCStack([]string{"java/lang/System.gc", "com/example/DemoHttpService.createGcPressure"}) {
		t.Fatalf("expected explicit System.gc stack to be classified as GC evidence")
	}
	if isJVMGCStack([]string{"com.example.Checkout.handle"}) {
		t.Fatalf("application stack should not be classified as GC evidence")
	}
}

func TestParserRejectsInvalidJFRMagic(t *testing.T) {
	_, err := (Parser{}).Parse(strings.NewReader("FLR\x00bad"))
	if err == nil {
		t.Fatalf("expected invalid JFR data to fail")
	}
}
