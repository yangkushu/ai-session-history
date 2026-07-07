package cli

import (
	"bytes"
	"testing"
)

func TestRunShowsHelpForNoArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage: ai-history")) {
		t.Fatalf("expected usage in stderr, got %q", stderr.String())
	}
}
