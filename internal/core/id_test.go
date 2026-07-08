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

func TestParseSessionIDRejectsInvalidInput(t *testing.T) {
	_, _, err := ParseSessionID("invalid")
	if err == nil {
		t.Fatal("expected invalid session id error")
	}
	if !IsCode(err, ErrInvalidSessionID) {
		t.Fatalf("expected %s, got %v", ErrInvalidSessionID, err)
	}
}
