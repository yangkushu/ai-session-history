package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigEnablesAllSources(t *testing.T) {
	cfg := Default()
	for _, source := range []string{"codex", "claude", "cursor"} {
		if !cfg.Sources[source].Enabled {
			t.Fatalf("expected %s enabled", source)
		}
		if !cfg.Sources[source].UseDefaultPaths {
			t.Fatalf("expected %s default paths enabled", source)
		}
	}
	if cfg.Limits.DetailChars != 50000 || cfg.Limits.ContextChars != 20000 {
		t.Fatalf("unexpected limits: %+v", cfg.Limits)
	}
}

func TestLoadConfigOverridesSourceAndLimits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(`
sources:
  cursor:
    enabled: false
    use_default_paths: false
    paths:
      - ~/cursor-fixture
limits:
  detail_chars: 100
  context_chars: 80
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Sources["cursor"].Enabled {
		t.Fatal("expected cursor disabled")
	}
	if cfg.Sources["cursor"].UseDefaultPaths {
		t.Fatal("expected default paths disabled")
	}
	if cfg.Limits.DetailChars != 100 || cfg.Limits.ContextChars != 80 {
		t.Fatalf("unexpected limits: %+v", cfg.Limits)
	}
}
