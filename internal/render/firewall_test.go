package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bjro/ccbox/internal/firewall"
	"github.com/bjro/ccbox/internal/stack"
)

func TestRenderFirewall_NoError(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	if len(files.InitFirewall) == 0 {
		t.Error("InitFirewall is empty")
	}
	if len(files.WarmupDNS) == 0 {
		t.Error("WarmupDNS is empty")
	}
	if len(files.DynamicDomains) == 0 {
		t.Error("DynamicDomains is empty")
	}
}

func TestRenderFirewall_InitFirewall_ContainsStaticDomains(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	for _, d := range cfg.Domains.Static {
		if !bytes.Contains(files.InitFirewall, []byte(d.Name)) {
			t.Errorf("InitFirewall missing static domain %q", d.Name)
		}
	}

	// Spot-checks for well-known static domains.
	spotChecks := []string{"github.com", "api.github.com", "registry.npmjs.org"}
	for _, name := range spotChecks {
		if !bytes.Contains(files.InitFirewall, []byte(name)) {
			t.Errorf("InitFirewall missing well-known static domain %q", name)
		}
	}
}

func TestRenderFirewall_InitFirewall_ContainsDynamicDomains(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	// Verify dnsmasq ipset directives for dynamic domains.
	for _, d := range cfg.Domains.Dynamic {
		stripped := strings.TrimPrefix(d.Name, "*.")
		directive := "ipset=/" + stripped + "/allowed_ips"
		if !bytes.Contains(files.InitFirewall, []byte(directive)) {
			t.Errorf("InitFirewall missing dnsmasq directive %q", directive)
		}
	}
}

func TestRenderFirewall_DynamicDomains_ContainsDomainNames(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	// Each dynamic domain (with wildcards stripped) should appear in the output.
	for _, d := range cfg.Domains.Dynamic {
		stripped := strings.TrimPrefix(d.Name, "*.")
		if !bytes.Contains(files.DynamicDomains, []byte(stripped)) {
			t.Errorf("DynamicDomains missing domain %q (stripped from %q)", stripped, d.Name)
		}
	}

	// Static domains should NOT appear in dynamic-domains.conf.
	for _, d := range cfg.Domains.Static {
		// Only check domains that are unambiguously static-only. Some
		// domain substrings could match parts of dynamic domain names,
		// so we check for the domain at word boundaries by checking
		// that it appears as the first word on a line.
		lines := strings.Split(string(files.DynamicDomains), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == d.Name {
				t.Errorf("DynamicDomains contains static domain %q", d.Name)
			}
		}
	}
}

func TestRenderFirewall_DynamicDomains_ContainsRationale(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	// At least one dynamic domain rationale should appear as inline comment.
	for _, d := range cfg.Domains.Dynamic {
		if !bytes.Contains(files.DynamicDomains, []byte(d.Rationale)) {
			t.Errorf("DynamicDomains missing rationale %q for domain %q", d.Rationale, d.Name)
		}
	}
}

func TestRenderFirewall_WarmupDNS_IsStatic(t *testing.T) {
	cfgGo, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge (Go): %v", err)
	}
	cfgGoNode, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge (Go+Node): %v", err)
	}

	filesGo, err := RenderFirewall(cfgGo)
	if err != nil {
		t.Fatalf("RenderFirewall (Go): %v", err)
	}
	filesGoNode, err := RenderFirewall(cfgGoNode)
	if err != nil {
		t.Fatalf("RenderFirewall (Go+Node): %v", err)
	}

	if !bytes.Equal(filesGo.WarmupDNS, filesGoNode.WarmupDNS) {
		t.Error("WarmupDNS output differs between Go-only and Go+Node configs; it should be static")
	}
}

func TestRenderFirewall_EmptyDomains(t *testing.T) {
	cfg := GenerationConfig{
		Stacks:   []stack.StackID{},
		Runtimes: []stack.Runtime{},
		LSPs:     []stack.LSP{},
		Domains: firewall.MergedDomains{
			Static:  []firewall.Domain{},
			Dynamic: []firewall.Domain{},
		},
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall with empty domains: %v", err)
	}

	// Should still have valid shebang and no template artifacts.
	if !bytes.HasPrefix(files.InitFirewall, []byte("#!/usr/bin/env bash")) {
		t.Error("InitFirewall missing shebang")
	}
	if bytes.Contains(files.InitFirewall, []byte("<no value>")) {
		t.Error("InitFirewall contains '<no value>' template artifact")
	}
	if bytes.Contains(files.DynamicDomains, []byte("<no value>")) {
		t.Error("DynamicDomains contains '<no value>' template artifact")
	}
}

func TestRenderFirewall_InitFirewall_ScriptStructure(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	output := string(files.InitFirewall)

	checks := []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"ipset create",
		"iptables",
		"dnsmasq",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("InitFirewall missing structural marker %q", check)
		}
	}
}

func TestRenderFirewall_WarmupDNS_ScriptStructure(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	output := string(files.WarmupDNS)

	if !strings.HasPrefix(output, "#!/usr/bin/env bash") {
		t.Error("WarmupDNS missing shebang")
	}
	if !strings.Contains(output, "dig") {
		t.Error("WarmupDNS missing 'dig' command")
	}
	if !strings.Contains(output, "dynamic-domains.conf") {
		t.Error("WarmupDNS missing reference to dynamic-domains.conf")
	}
}

func TestRenderFirewall_AllStacks(t *testing.T) {
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, []string{"extra.example.com"})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	// Every dynamic domain (stripped) should appear in DynamicDomains.
	for _, d := range cfg.Domains.Dynamic {
		stripped := strings.TrimPrefix(d.Name, "*.")
		if !bytes.Contains(files.DynamicDomains, []byte(stripped)) {
			t.Errorf("DynamicDomains missing dynamic domain %q", stripped)
		}
	}

	// Every static domain should appear in InitFirewall.
	for _, d := range cfg.Domains.Static {
		if !bytes.Contains(files.InitFirewall, []byte(d.Name)) {
			t.Errorf("InitFirewall missing static domain %q", d.Name)
		}
	}
}

func TestRenderFirewall_WildcardDomainHandling(t *testing.T) {
	// AlwaysOn includes *.anthropic.com as a Dynamic domain.
	cfg, err := Merge(nil, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	// DynamicDomains should contain the stripped form.
	if !bytes.Contains(files.DynamicDomains, []byte("anthropic.com")) {
		t.Error("DynamicDomains missing stripped 'anthropic.com'")
	}
	// DynamicDomains should NOT contain the raw wildcard form as the first field on any line.
	lines := strings.Split(string(files.DynamicDomains), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "*.anthropic.com" {
			t.Error("DynamicDomains contains raw wildcard '*.anthropic.com' as domain entry")
		}
	}

	// InitFirewall should contain dnsmasq directive with stripped wildcard.
	if !bytes.Contains(files.InitFirewall, []byte("ipset=/anthropic.com/allowed_ips")) {
		t.Error("InitFirewall missing 'ipset=/anthropic.com/allowed_ips' dnsmasq directive")
	}
	// InitFirewall should NOT contain the raw wildcard in ipset directives.
	if bytes.Contains(files.InitFirewall, []byte("ipset=/*.anthropic.com/")) {
		t.Error("InitFirewall contains raw wildcard form 'ipset=/*.anthropic.com/' in dnsmasq directive")
	}
}

func TestRenderFirewall_InitFirewall_SingleQuotedDomains(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderFirewall(cfg)
	if err != nil {
		t.Fatalf("RenderFirewall: %v", err)
	}

	// Verify at least one single-quoted domain in a dig command context.
	// Static domains should appear as dig +short '<domain>'.
	foundSingleQuoted := false
	for _, d := range cfg.Domains.Static {
		quoted := "'" + d.Name + "'"
		if bytes.Contains(files.InitFirewall, []byte(quoted)) {
			foundSingleQuoted = true
			break
		}
	}

	if !foundSingleQuoted {
		t.Error("InitFirewall does not contain any single-quoted domain in dig context; expected defense-in-depth quoting")
	}
}
