// Package config handles reading and writing .ccbox.yml configuration files.
package config

import (
	"fmt"
	"io"
	"time"

	"gopkg.in/yaml.v3"
)

// Filename is the name of the ccbox configuration file.
const Filename = ".ccbox.yml"

// Config represents the contents of a .ccbox.yml file.
type Config struct {
	Version      int       `yaml:"version"`
	Stacks       []string  `yaml:"stacks,flow"`
	ExtraDomains []string  `yaml:"extra_domains,flow"`
	GeneratedAt  time.Time `yaml:"generated_at"`
	CcboxVersion string    `yaml:"ccbox_version"`
}

// Write serializes cfg as YAML to w.
func Write(w io.Writer, cfg Config) error {
	// Ensure non-nil slices so YAML renders [] instead of null.
	if cfg.Stacks == nil {
		cfg.Stacks = []string{}
	}
	if cfg.ExtraDomains == nil {
		cfg.ExtraDomains = []string{}
	}

	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return enc.Close()
}

// Load reads a Config from r and validates it.
func Load(r io.Reader) (Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	if cfg.Version != 1 {
		return Config{}, fmt.Errorf("unsupported config version: %d", cfg.Version)
	}

	// Ensure non-nil empty slices for consistent behavior.
	if cfg.Stacks == nil {
		cfg.Stacks = []string{}
	}
	if cfg.ExtraDomains == nil {
		cfg.ExtraDomains = []string{}
	}

	return cfg, nil
}
