package cmd

import (
	"bytes"
	"testing"
)

func TestInitCommand_Stub(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	expected := "ccbox init: not yet implemented"
	if !contains(output, expected) {
		t.Errorf("expected output to contain %q, got %q", expected, output)
	}
}
