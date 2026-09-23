package core

import "testing"

func TestSessionIDRoundTrip(t *testing.T) {
	id := MakeSessionID(SourceCodex, "abc")
	if id != "codex:abc" {
		t.Fatalf("unexpected id: %s", id)
	}

	source, native, err := ParseSessionID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != SourceCodex || native != "abc" {
		t.Fatalf("unexpected parsed id: %s %s", source, native)
	}
}

func TestPiSessionIDIsValid(t *testing.T) {
	id := MakeSessionID(SourcePi, "019f6aaf-29f9-7023-a67f-32ba88094b8e")
	source, native, err := ParseSessionID(id)
	if err != nil || source != SourcePi || native != "019f6aaf-29f9-7023-a67f-32ba88094b8e" {
		t.Fatalf("Pi session ID round trip failed: source=%q native=%q err=%v", source, native, err)
	}
}

func TestParseSessionIDRejectsInvalidInput(t *testing.T) {
	_, _, err := ParseSessionID("invalid")
	if err == nil {
		t.Fatal("expected invalid session id error")
	}
	if !IsCode(err, ErrInvalidSessionID) {
		t.Fatalf("expected %s, got %v", ErrInvalidSessionID, err)
	}
}
