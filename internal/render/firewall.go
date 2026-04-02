package render

import (
	"bytes"
	"embed"
	"strings"
	"text/template"
)

//go:embed templates/init-firewall.sh.tmpl templates/warmup-dns.sh.tmpl templates/dynamic-domains.conf.tmpl
var templatesFS embed.FS

// funcMap provides template helper functions for firewall script rendering.
var funcMap = template.FuncMap{
	// stripWildcard removes a leading "*." prefix from a domain name.
	// This is needed for dnsmasq ipset directives (dnsmasq natively treats
	// bare domains as matching all subdomains) and for dig resolution
	// (dig cannot resolve wildcard DNS names like *.example.com).
	"stripWildcard": func(name string) string {
		return strings.TrimPrefix(name, "*.")
	},
}

// firewallTemplates holds the parsed templates for all firewall scripts.
// Parsing at package level via template.Must ensures syntax errors are caught
// at program startup, which is the correct behavior for embedded templates
// that should always be valid.
var firewallTemplates = template.Must(
	template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.tmpl"),
)

// FirewallFiles holds the rendered output of the three firewall-related files.
// Each field contains the full file content ready for writing to disk.
type FirewallFiles struct {
	InitFirewall   []byte // init-firewall.sh content
	WarmupDNS      []byte // warmup-dns.sh content
	DynamicDomains []byte // dynamic-domains.conf content
}

// RenderFirewall executes the firewall templates against the given
// GenerationConfig and returns the rendered file contents. It is a pure
// transformation from config to bytes with no file I/O -- actual file
// writing is the responsibility of the orchestrator (ccbox init command).
func RenderFirewall(cfg GenerationConfig) (FirewallFiles, error) {
	var initBuf, warmupBuf, dynamicBuf bytes.Buffer

	if err := firewallTemplates.ExecuteTemplate(&initBuf, "init-firewall.sh.tmpl", cfg); err != nil {
		return FirewallFiles{}, err
	}

	if err := firewallTemplates.ExecuteTemplate(&warmupBuf, "warmup-dns.sh.tmpl", cfg); err != nil {
		return FirewallFiles{}, err
	}

	if err := firewallTemplates.ExecuteTemplate(&dynamicBuf, "dynamic-domains.conf.tmpl", cfg); err != nil {
		return FirewallFiles{}, err
	}

	return FirewallFiles{
		InitFirewall:   initBuf.Bytes(),
		WarmupDNS:      warmupBuf.Bytes(),
		DynamicDomains: dynamicBuf.Bytes(),
	}, nil
}
