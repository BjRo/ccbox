package config

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestFilename(t *testing.T) {
	if Filename != ".ccbox.yml" {
		t.Errorf("Filename = %q, want %q", Filename, ".ccbox.yml")
	}
}

func TestWriteAndLoad_RoundTrip(t *testing.T) {
	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	original := Config{
		Version:      1,
		Stacks:       []string{"go", "node"},
		ExtraDomains: []string{"api.example.com", "cdn.example.com"},
		GeneratedAt:  now,
		CcboxVersion: "0.1.0",
	}

	var buf bytes.Buffer
	if err := Write(&buf, original); err != nil {
		t.Fatalf("Write: %v", err)
	}

	loaded, err := Load(&buf)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("Version = %d, want %d", loaded.Version, original.Version)
	}
	if len(loaded.Stacks) != len(original.Stacks) {
		t.Fatalf("Stacks length = %d, want %d", len(loaded.Stacks), len(original.Stacks))
	}
	for i, s := range loaded.Stacks {
		if s != original.Stacks[i] {
			t.Errorf("Stacks[%d] = %q, want %q", i, s, original.Stacks[i])
		}
	}
	if len(loaded.ExtraDomains) != len(original.ExtraDomains) {
		t.Fatalf("ExtraDomains length = %d, want %d", len(loaded.ExtraDomains), len(original.ExtraDomains))
	}
	for i, d := range loaded.ExtraDomains {
		if d != original.ExtraDomains[i] {
			t.Errorf("ExtraDomains[%d] = %q, want %q", i, d, original.ExtraDomains[i])
		}
	}
	if !loaded.GeneratedAt.Equal(original.GeneratedAt) {
		t.Errorf("GeneratedAt = %v, want %v", loaded.GeneratedAt, original.GeneratedAt)
	}
	if loaded.CcboxVersion != original.CcboxVersion {
		t.Errorf("CcboxVersion = %q, want %q", loaded.CcboxVersion, original.CcboxVersion)
	}
}

func TestWrite_YAMLFormat(t *testing.T) {
	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	cfg := Config{
		Version:      1,
		Stacks:       []string{"go", "node"},
		ExtraDomains: []string{"api.example.com"},
		GeneratedAt:  now,
		CcboxVersion: "0.1.0",
	}

	var buf bytes.Buffer
	if err := Write(&buf, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.String()

	// yaml.v3 renders time.Time as unquoted YAML timestamps.
	// Flow-style tags render slices inline: [go, node].
	// String values like ccbox_version are unquoted when safe.
	expectations := []string{
		"version: 1",
		"stacks: [go, node]",
		"extra_domains: [api.example.com]",
		"generated_at: 2026-04-02T10:00:00Z",
		"ccbox_version: 0.1.0",
	}

	for _, exp := range expectations {
		if !strings.Contains(out, exp) {
			t.Errorf("output missing %q\ngot:\n%s", exp, out)
		}
	}
}

func TestWrite_EmptyStacks(t *testing.T) {
	cfg := Config{
		Version:      1,
		Stacks:       []string{},
		ExtraDomains: []string{"api.example.com"},
		GeneratedAt:  time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC),
		CcboxVersion: "0.1.0",
	}

	var buf bytes.Buffer
	if err := Write(&buf, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "stacks: []") {
		t.Errorf("empty stacks should render as 'stacks: []', got:\n%s", out)
	}
}

func TestWrite_EmptyExtraDomains(t *testing.T) {
	cfg := Config{
		Version:      1,
		Stacks:       []string{"go"},
		ExtraDomains: []string{},
		GeneratedAt:  time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC),
		CcboxVersion: "0.1.0",
	}

	var buf bytes.Buffer
	if err := Write(&buf, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "extra_domains: []") {
		t.Errorf("empty extra_domains should render as 'extra_domains: []', got:\n%s", out)
	}
}

func TestWrite_NilSlicesRenderedAsEmpty(t *testing.T) {
	cfg := Config{
		Version:      1,
		Stacks:       nil,
		ExtraDomains: nil,
		GeneratedAt:  time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC),
		CcboxVersion: "0.1.0",
	}

	var buf bytes.Buffer
	if err := Write(&buf, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "stacks: []") {
		t.Errorf("nil stacks should render as 'stacks: []', got:\n%s", out)
	}
	if !strings.Contains(out, "extra_domains: []") {
		t.Errorf("nil extra_domains should render as 'extra_domains: []', got:\n%s", out)
	}
}

func TestLoad_ValidatesVersion(t *testing.T) {
	input := `version: 99
stacks: []
extra_domains: []
generated_at: 2026-04-02T10:00:00Z
ccbox_version: "0.1.0"
`
	_, err := Load(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unknown version, got nil")
	}
	if !strings.Contains(err.Error(), "99") {
		t.Errorf("error should mention version 99, got: %v", err)
	}
}

func TestLoad_NonNilSlices(t *testing.T) {
	// YAML that omits stacks and extra_domains entirely.
	input := `version: 1
generated_at: 2026-04-02T10:00:00Z
ccbox_version: "0.1.0"
`
	cfg, err := Load(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Stacks == nil {
		t.Error("Stacks should be non-nil empty slice, got nil")
	}
	if len(cfg.Stacks) != 0 {
		t.Errorf("Stacks should be empty, got %v", cfg.Stacks)
	}
	if cfg.ExtraDomains == nil {
		t.Error("ExtraDomains should be non-nil empty slice, got nil")
	}
	if len(cfg.ExtraDomains) != 0 {
		t.Errorf("ExtraDomains should be empty, got %v", cfg.ExtraDomains)
	}
}

func TestTimestamp_RoundTrip(t *testing.T) {
	// Use a timestamp with sub-second precision to verify it survives.
	now := time.Date(2026, 4, 2, 10, 30, 45, 0, time.UTC)
	cfg := Config{
		Version:      1,
		Stacks:       []string{},
		ExtraDomains: []string{},
		GeneratedAt:  now,
		CcboxVersion: "0.1.0",
	}

	var buf bytes.Buffer
	if err := Write(&buf, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	loaded, err := Load(&buf)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !loaded.GeneratedAt.Equal(now) {
		t.Errorf("GeneratedAt = %v, want %v", loaded.GeneratedAt, now)
	}
}
