package dockerfile

import (
	"errors"
	"testing"
)

func TestSplitAtCustomStage_HappyPath(t *testing.T) {
	t.Parallel()

	content := "FROM debian:bookworm-slim AS agentbox\nRUN apt-get update\nWORKDIR /workspace\n\nFROM agentbox AS custom\n# User stuff\n"
	agentbox, user, err := SplitAtCustomStage(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agentbox+user != content {
		t.Error("round-trip failed: agentboxPart + userPart != original")
	}

	if agentbox != "FROM debian:bookworm-slim AS agentbox\nRUN apt-get update\nWORKDIR /workspace\n\n" {
		t.Errorf("agentbox part = %q", agentbox)
	}
	if user != "FROM agentbox AS custom\n# User stuff\n" {
		t.Errorf("user part = %q", user)
	}
}

func TestSplitAtCustomStage_UserContentPreserved(t *testing.T) {
	t.Parallel()

	content := "FROM debian:bookworm-slim AS agentbox\nWORKDIR /workspace\n\nFROM agentbox AS custom\nRUN echo hello\nRUN apt-get install -y vim\nCOPY myfile /tmp/myfile\n"
	_, user, err := SplitAtCustomStage(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "FROM agentbox AS custom\nRUN echo hello\nRUN apt-get install -y vim\nCOPY myfile /tmp/myfile\n"
	if user != expected {
		t.Errorf("user part = %q, want %q", user, expected)
	}
}

func TestSplitAtCustomStage_CaseInsensitiveKeywords(t *testing.T) {
	t.Parallel()

	for _, line := range []string{
		"from agentbox as custom",
		"FROM agentbox AS custom",
		"From agentbox As custom",
	} {
		t.Run(line, func(t *testing.T) {
			content := "FROM debian:bookworm-slim AS agentbox\n\n" + line + "\n"
			_, user, err := SplitAtCustomStage(content)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", line, err)
			}
			if user != line+"\n" {
				t.Errorf("user part = %q, want %q", user, line+"\n")
			}
		})
	}
}

func TestSplitAtCustomStage_LeadingWhitespace(t *testing.T) {
	t.Parallel()

	content := "FROM debian:bookworm-slim AS agentbox\n  FROM agentbox AS custom\n"
	_, user, err := SplitAtCustomStage(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != "  FROM agentbox AS custom\n" {
		t.Errorf("user part = %q", user)
	}
}

func TestSplitAtCustomStage_NoCustomStage(t *testing.T) {
	t.Parallel()

	content := "FROM debian:bookworm-slim\nRUN apt-get update\n"
	_, _, err := SplitAtCustomStage(content)
	if !errors.Is(err, ErrNoCustomStage) {
		t.Errorf("expected ErrNoCustomStage, got: %v", err)
	}
}

func TestSplitAtCustomStage_EmptyInput(t *testing.T) {
	t.Parallel()

	_, _, err := SplitAtCustomStage("")
	if !errors.Is(err, ErrNoCustomStage) {
		t.Errorf("expected ErrNoCustomStage for empty input, got: %v", err)
	}
}

func TestSplitAtCustomStage_CustomStageFirstLine(t *testing.T) {
	t.Parallel()

	content := "FROM agentbox AS custom\nRUN echo hello\n"
	agentbox, user, err := SplitAtCustomStage(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agentbox != "" {
		t.Errorf("agentbox part should be empty, got %q", agentbox)
	}
	if user != content {
		t.Errorf("user part = %q, want %q", user, content)
	}
}

func TestSplitAtCustomStage_MultipleFROMLines(t *testing.T) {
	t.Parallel()

	content := "FROM debian:bookworm-slim AS agentbox\nFROM node:20 AS builder\nRUN npm build\n\nFROM agentbox AS custom\nRUN echo hello\n"
	agentbox, user, err := SplitAtCustomStage(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agentbox+user != content {
		t.Error("round-trip failed")
	}
	if user != "FROM agentbox AS custom\nRUN echo hello\n" {
		t.Errorf("user part = %q", user)
	}
}

func TestSplitAtCustomStage_RoundTrip(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"FROM debian:bookworm-slim AS agentbox\nWORKDIR /workspace\n\nFROM agentbox AS custom\n",
		"FROM agentbox AS custom\n",
		"line1\nline2\nFROM agentbox AS custom\nline4\nline5\n",
		"header\r\nFROM agentbox AS custom\r\nfooter\r\n",
	}

	for _, input := range inputs {
		agentbox, user, err := SplitAtCustomStage(input)
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", input, err)
		}
		if agentbox+user != input {
			t.Errorf("round-trip failed for input %q: got %q + %q", input, agentbox, user)
		}
	}
}

func TestSplitAtCustomStage_WindowsLineEndings(t *testing.T) {
	t.Parallel()

	content := "FROM debian:bookworm-slim AS agentbox\r\nWORKDIR /workspace\r\n\r\nFROM agentbox AS custom\r\n# User\r\n"
	agentbox, user, err := SplitAtCustomStage(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agentbox+user != content {
		t.Error("round-trip failed with Windows line endings")
	}
	if user != "FROM agentbox AS custom\r\n# User\r\n" {
		t.Errorf("user part = %q", user)
	}
}
