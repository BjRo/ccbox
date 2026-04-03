package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bjro/ccbox/internal/stack"
)

func TestDevContainer_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if err := DevContainer(&buf, cfg); err != nil {
		t.Fatalf("DevContainer: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n\nOutput:\n%s", err, buf.String())
	}
}

func TestDevContainer_FixedStructure(t *testing.T) {
	var buf bytes.Buffer
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if err := DevContainer(&buf, cfg); err != nil {
		t.Fatalf("DevContainer: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// build.dockerfile
	build, ok := parsed["build"].(map[string]any)
	if !ok {
		t.Fatal("missing or invalid 'build' field")
	}
	if df, _ := build["dockerfile"].(string); df != "Dockerfile" {
		t.Errorf("build.dockerfile = %q, want %q", df, "Dockerfile")
	}

	// remoteUser
	if ru, _ := parsed["remoteUser"].(string); ru != "node" {
		t.Errorf("remoteUser = %q, want %q", ru, "node")
	}

	// customizations.vscode.extensions
	customizations, ok := parsed["customizations"].(map[string]any)
	if !ok {
		t.Fatal("missing or invalid 'customizations' field")
	}
	vscode, ok := customizations["vscode"].(map[string]any)
	if !ok {
		t.Fatal("missing or invalid 'customizations.vscode' field")
	}
	extensions, ok := vscode["extensions"].([]any)
	if !ok {
		t.Fatal("missing or invalid 'customizations.vscode.extensions' field")
	}
	foundClaude := false
	for _, ext := range extensions {
		if ext == "anthropic.claude-code" {
			foundClaude = true
		}
	}
	if !foundClaude {
		t.Error("customizations.vscode.extensions missing 'anthropic.claude-code'")
	}

	// capAdd
	capAdd, ok := parsed["capAdd"].([]any)
	if !ok {
		t.Fatal("missing or invalid 'capAdd' field")
	}
	capAddStrs := make(map[string]bool)
	for _, c := range capAdd {
		if s, ok := c.(string); ok {
			capAddStrs[s] = true
		}
	}
	if !capAddStrs["NET_ADMIN"] {
		t.Error("capAdd missing 'NET_ADMIN'")
	}
	if !capAddStrs["NET_RAW"] {
		t.Error("capAdd missing 'NET_RAW'")
	}

	// securityOpt
	securityOpt, ok := parsed["securityOpt"].([]any)
	if !ok {
		t.Fatal("missing or invalid 'securityOpt' field")
	}
	foundSeccomp := false
	for _, s := range securityOpt {
		if s == "seccomp=unconfined" {
			foundSeccomp = true
		}
	}
	if !foundSeccomp {
		t.Error("securityOpt missing 'seccomp=unconfined'")
	}

	// workspaceFolder
	if wf, _ := parsed["workspaceFolder"].(string); wf != "/workspace" {
		t.Errorf("workspaceFolder = %q, want %q", wf, "/workspace")
	}

	// mounts (4 entries)
	mounts, ok := parsed["mounts"].([]any)
	if !ok {
		t.Fatal("missing or invalid 'mounts' field")
	}
	if len(mounts) != 4 {
		t.Errorf("mounts count = %d, want 4", len(mounts))
	}

	// postStartCommand
	psc, _ := parsed["postStartCommand"].(string)
	if !strings.Contains(psc, "sync-claude-settings.sh") {
		t.Error("postStartCommand missing 'sync-claude-settings.sh'")
	}
	if !strings.Contains(psc, "init-firewall.sh") {
		t.Error("postStartCommand missing 'init-firewall.sh'")
	}
}

func TestDevContainer_EmptyConfig(t *testing.T) {
	var buf bytes.Buffer
	cfg := GenerationConfig{}

	if err := DevContainer(&buf, cfg); err != nil {
		t.Fatalf("DevContainer: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON with empty config: %v\n\nOutput:\n%s", err, buf.String())
	}

	// Same structure should be present regardless of config content.
	if _, ok := parsed["build"]; !ok {
		t.Error("missing 'build' field with empty config")
	}
	if _, ok := parsed["remoteUser"]; !ok {
		t.Error("missing 'remoteUser' field with empty config")
	}
	if _, ok := parsed["capAdd"]; !ok {
		t.Error("missing 'capAdd' field with empty config")
	}
}

func TestDevContainer_MountsContent(t *testing.T) {
	var buf bytes.Buffer
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if err := DevContainer(&buf, cfg); err != nil {
		t.Fatalf("DevContainer: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	mounts, ok := parsed["mounts"].([]any)
	if !ok {
		t.Fatal("missing or invalid 'mounts' field")
	}

	mountStrs := make([]string, len(mounts))
	for i, m := range mounts {
		s, ok := m.(string)
		if !ok {
			t.Fatalf("mounts[%d] is not a string", i)
		}
		mountStrs[i] = s
	}

	// Join all mount strings for substring checking.
	allMounts := strings.Join(mountStrs, "\n")

	checks := []string{
		"ccbox-bash-history",
		"ccbox-claude-config",
		".config/gh",
		".gitconfig",
		"${localEnv:HOME}",
	}
	for _, want := range checks {
		if !strings.Contains(allMounts, want) {
			t.Errorf("mounts missing expected substring %q", want)
		}
	}

	// Verify bind mounts use ${localEnv:HOME}.
	for _, m := range mountStrs {
		if strings.Contains(m, "type=bind") && !strings.Contains(m, "${localEnv:HOME}") {
			t.Errorf("bind mount missing ${localEnv:HOME}: %s", m)
		}
	}
}

func TestDevContainer_IsStatic(t *testing.T) {
	// The devcontainer.json template has no Go template actions (it is fully
	// static). Rendering with different configs must produce byte-identical
	// output. This proves the template is truly stack-agnostic, as required
	// by the JSON template testing rules.
	cfgGo, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge (Go): %v", err)
	}

	cfgMulti, err := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge (Go+Node+Python): %v", err)
	}

	var bufGo, bufMulti bytes.Buffer
	if err := DevContainer(&bufGo, cfgGo); err != nil {
		t.Fatalf("DevContainer (Go): %v", err)
	}
	if err := DevContainer(&bufMulti, cfgMulti); err != nil {
		t.Fatalf("DevContainer (Go+Node+Python): %v", err)
	}

	if !bytes.Equal(bufGo.Bytes(), bufMulti.Bytes()) {
		t.Errorf("devcontainer.json differs between Go-only and Go+Node+Python configs; template should be fully static\n--- Go-only ---\n%s\n--- Go+Node+Python ---\n%s",
			bufGo.String(), bufMulti.String())
	}
}

func TestDevContainer_Deterministic(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	var buf1, buf2 bytes.Buffer
	if err := DevContainer(&buf1, cfg); err != nil {
		t.Fatalf("DevContainer (first): %v", err)
	}
	if err := DevContainer(&buf2, cfg); err != nil {
		t.Fatalf("DevContainer (second): %v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Errorf("outputs differ between two renders:\n--- first ---\n%s\n--- second ---\n%s", buf1.String(), buf2.String())
	}
}
