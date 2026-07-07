package readers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yangkushu/ai-session-history/internal/core"
)

func TestCursorStorageReaderReportsUnavailableWithoutStorage(t *testing.T) {
	reader := NewCursorStorageReader([]string{t.TempDir()})

	diagnostic := reader.Doctor()

	if diagnostic.Source != core.SourceCursor {
		t.Fatalf("unexpected source: %s", diagnostic.Source)
	}
	if diagnostic.Status != "unavailable" {
		t.Fatalf("unexpected status: %s", diagnostic.Status)
	}
	if diagnostic.Code != core.ErrSourceUnavailable {
		t.Fatalf("unexpected code: %s", diagnostic.Code)
	}
}

func TestCursorStorageReaderReportsUnsupportedStateDBUntilRealFixtureIsImplemented(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "workspaceStorage", "demo", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader := NewCursorStorageReader([]string{root})

	diagnostic := reader.Doctor()

	if diagnostic.Status != "unsupported_format" {
		t.Fatalf("unexpected status: %+v", diagnostic)
	}
	if diagnostic.Code != core.ErrUnsupportedFormat {
		t.Fatalf("unexpected code: %+v", diagnostic)
	}
}
