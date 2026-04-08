package render

import (
	"strings"
	"testing"
)

func TestCustomStage_RendersWithoutError(t *testing.T) {
	_, err := CustomStage()
	if err != nil {
		t.Fatalf("CustomStage: %v", err)
	}
}

func TestCustomStage_ContainsFROMLine(t *testing.T) {
	out, err := CustomStage()
	if err != nil {
		t.Fatalf("CustomStage: %v", err)
	}

	if !strings.Contains(out, "FROM agentbox AS custom") {
		t.Error("output missing FROM agentbox AS custom")
	}
}

func TestCustomStage_ContainsHelpfulComments(t *testing.T) {
	out, err := CustomStage()
	if err != nil {
		t.Fatalf("CustomStage: %v", err)
	}

	if !strings.Contains(out, "USER CUSTOMIZATIONS") {
		t.Error("output missing USER CUSTOMIZATIONS comment")
	}
	if !strings.Contains(out, "agentbox update") {
		t.Error("output missing agentbox update reference")
	}
}

func TestCustomStage_Deterministic(t *testing.T) {
	out1, err := CustomStage()
	if err != nil {
		t.Fatalf("CustomStage (first): %v", err)
	}

	out2, err := CustomStage()
	if err != nil {
		t.Fatalf("CustomStage (second): %v", err)
	}

	if out1 != out2 {
		t.Error("CustomStage output is not deterministic; two renders differ")
	}
}

func TestCustomStage_NoTemplateArtifacts(t *testing.T) {
	out, err := CustomStage()
	if err != nil {
		t.Fatalf("CustomStage: %v", err)
	}

	artifacts := []string{"<no value>", "<nil>", "{{", "}}"}
	for _, a := range artifacts {
		if strings.Contains(out, a) {
			t.Errorf("CustomStage contains template artifact %q", a)
		}
	}
}

func TestCustomStage_NoWORKDIR(t *testing.T) {
	out, err := CustomStage()
	if err != nil {
		t.Fatalf("CustomStage: %v", err)
	}

	if strings.Contains(out, "WORKDIR") {
		t.Error("CustomStage should not contain WORKDIR (inherited from parent stage)")
	}
}
