package render

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/firewall"
	"github.com/bjro/agentbox/internal/stack"
)

func TestMiseConfig_SingleStack(t *testing.T) {
	t.Parallel()
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	EnsureNode(&cfg)

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `go = "latest"`) {
		t.Error(`output missing go = "latest"`)
	}
	if !strings.Contains(out, `node = "lts"`) {
		t.Error(`output missing node = "lts"`)
	}
}

func TestMiseConfig_MultiStack(t *testing.T) {
	t.Parallel()
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	EnsureNode(&cfg)

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()
	for _, want := range []string{`go = "latest"`, `node = "lts"`, `python = "latest"`} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestMiseConfig_AllStacks(t *testing.T) {
	t.Parallel()
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	EnsureNode(&cfg)

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()

	// Structural: every runtime in cfg.Runtimes appears in output.
	for _, rt := range cfg.Runtimes {
		expected := fmt.Sprintf(`%s = "%s"`, rt.Tool, rt.Version)
		if !strings.Contains(out, expected) {
			t.Errorf("output missing runtime entry %q", expected)
		}
	}

	// Node appears exactly once.
	count := strings.Count(out, `node = "`)
	if count != 1 {
		t.Errorf("node appears %d times, want exactly 1", count)
	}
}

func TestMiseConfig_DirectConfig_CustomVersions(t *testing.T) {
	t.Parallel()
	cfg := GenerationConfig{
		Runtimes: []stack.Runtime{
			{Tool: "go", Version: "1.22.0"},
			{Tool: "node", Version: "20"},
		},
	}

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `go = "1.22.0"`) {
		t.Error(`output missing go = "1.22.0"`)
	}
	if !strings.Contains(out, `node = "20"`) {
		t.Error(`output missing node = "20"`)
	}
}

func TestMiseConfig_DirectConfig_EmptyRuntimes(t *testing.T) {
	t.Parallel()
	cfg := GenerationConfig{
		Runtimes: []stack.Runtime{},
	}

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[tools]") {
		t.Error("output missing [tools] header")
	}
	for _, artifact := range []string{"<no value>", "<nil>"} {
		if strings.Contains(out, artifact) {
			t.Errorf("output contains artifact %q", artifact)
		}
	}
}

func TestMiseConfig_NoTemplateArtifacts(t *testing.T) {
	t.Parallel()
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	EnsureNode(&cfg)

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()
	for _, artifact := range []string{"<no value>", "<nil>", "{{", "}}"} {
		if strings.Contains(out, artifact) {
			t.Errorf("output contains template artifact %q", artifact)
		}
	}
}

func TestMiseConfig_Deterministic(t *testing.T) {
	t.Parallel()
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	EnsureNode(&cfg)

	var buf1, buf2 bytes.Buffer
	if err := MiseConfig(&buf1, cfg); err != nil {
		t.Fatalf("MiseConfig (first): %v", err)
	}
	if err := MiseConfig(&buf2, cfg); err != nil {
		t.Fatalf("MiseConfig (second): %v", err)
	}

	if buf1.String() != buf2.String() {
		t.Error("MiseConfig output is not deterministic; two renders differ")
	}
}

func TestMiseConfig_TOMLFormat(t *testing.T) {
	t.Parallel()
	cfg := GenerationConfig{
		Runtimes: []stack.Runtime{
			{Tool: "go", Version: "latest"},
			{Tool: "node", Version: "lts"},
		},
	}

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()
	lines := strings.Split(out, "\n")

	// First non-empty line should be [tools].
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "[tools]" {
		t.Errorf("first line = %q, want %q", lines[0], "[tools]")
	}

	// Non-blank, non-header lines should match tool = "version" format.
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.Contains(trimmed, ` = "`) || !strings.HasSuffix(trimmed, `"`) {
			t.Errorf("line does not match TOML format: %q", trimmed)
		}
	}
}

func TestMiseConfig_TrailingNewline(t *testing.T) {
	t.Parallel()
	cfg := GenerationConfig{
		Runtimes: []stack.Runtime{
			{Tool: "go", Version: "latest"},
			{Tool: "node", Version: "lts"},
		},
	}

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Error("output does not end with a trailing newline")
	}
	if strings.HasSuffix(out, "\n\n") {
		t.Error("output ends with double trailing newline")
	}
}

func TestMiseConfig_NoTrailingWhitespace(t *testing.T) {
	t.Parallel()
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	EnsureNode(&cfg)

	var buf bytes.Buffer
	if err := MiseConfig(&buf, cfg); err != nil {
		t.Fatalf("MiseConfig: %v", err)
	}

	out := buf.String()
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("line %d has trailing whitespace: %q", i+1, line)
		}
	}

	// Ensure Domains field is present (prevent nil-pointer in Merge output).
	_ = firewall.MergedDomains{}
}
