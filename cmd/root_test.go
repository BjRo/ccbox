package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand_Version(t *testing.T) {
	cmd := newRootCmd(nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	expected := "agentbox version dev"
	if !contains(output, expected) {
		t.Errorf("expected output to contain %q, got %q", expected, output)
	}
}

func TestRootCommand_Help(t *testing.T) {
	cmd := newRootCmd(nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, "agentbox") {
		t.Errorf("expected help output to contain %q, got %q", "agentbox", output)
	}
	if !contains(output, "init") {
		t.Errorf("expected help output to contain %q, got %q", "init", output)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
