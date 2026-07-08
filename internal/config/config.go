package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type SourceConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Paths           []string `yaml:"paths"`
	UseDefaultPaths bool     `yaml:"use_default_paths"`
}

type Limits struct {
	DetailChars  int `yaml:"detail_chars"`
	ContextChars int `yaml:"context_chars"`
}

type Config struct {
	Sources map[string]SourceConfig `yaml:"sources"`
	Limits  Limits                  `yaml:"limits"`
}

func Default() Config {
	return Config{
		Sources: map[string]SourceConfig{
			"codex":  {Enabled: true, UseDefaultPaths: true},
			"claude": {Enabled: true, UseDefaultPaths: true},
			"cursor": {Enabled: true, UseDefaultPaths: true},
		},
		Limits: Limits{DetailChars: 50000, ContextChars: 20000},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		return Config{}, core.NewError(core.ErrInvalidConfig, err.Error())
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, core.NewError(core.ErrInvalidConfig, err.Error())
	}
	normalize(&cfg)
	return cfg, nil
}

func LoadOptional(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}
	return Load(path)
}

func normalize(cfg *Config) {
	defaults := Default()
	if cfg.Sources == nil {
		cfg.Sources = defaults.Sources
	}
	for name, def := range defaults.Sources {
		if _, ok := cfg.Sources[name]; !ok {
			cfg.Sources[name] = def
		}
	}
	if cfg.Limits.DetailChars <= 0 {
		cfg.Limits.DetailChars = defaults.Limits.DetailChars
	}
	if cfg.Limits.ContextChars <= 0 {
		cfg.Limits.ContextChars = defaults.Limits.ContextChars
	}
}

func expandHome(path string) string {
	if len(path) < 2 || path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
