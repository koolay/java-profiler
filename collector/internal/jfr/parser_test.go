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

func TestParserRejectsInvalidJFRMagic(t *testing.T) {
	_, err := (Parser{}).Parse(strings.NewReader("FLR\x00bad"))
	if err == nil {
		t.Fatalf("expected invalid JFR data to fail")
	}
}
