// Package firewall manages domain allowlists for network isolation.
//
// The registry returned by [Registry] is a curated set of per-stack domain
// allowlists. Each call returns a shallow copy of the internal map so callers
// cannot mutate the canonical data. The [Allowlist] and [Domain] values within
// the map should be treated as read-only; while they are technically mutable
// (slices share underlying arrays), modifying them leads to undefined behavior.
package firewall

import "sort"

// Stack identifies a technology stack or a pseudo-stack (e.g., AlwaysOn).
type Stack string

const (
	// AlwaysOn contains domains required regardless of detected stacks
	// (GitHub API, Anthropic API, telemetry).
	AlwaysOn Stack = "always-on"
	// Go contains domains required for Go module resolution and downloads.
	Go Stack = "go"
	// Node contains domains required for npm package installation.
	Node Stack = "node"
	// Python contains domains required for pip package installation.
	Python Stack = "python"
	// Rust contains domains required for Cargo crate resolution and downloads.
	Rust Stack = "rust"
	// Ruby contains domains required for RubyGems and Bundler.
	Ruby Stack = "ruby"
)

// Category classifies a domain by how its DNS resolution should be handled
// at the firewall layer.
type Category string

const (
	// Static domains have stable IPs. They are resolved once at firewall
	// init and cached in an ipset. Cheaper at runtime.
	Static Category = "static"
	// Dynamic domains use CDN or rotating IPs. They are managed by dnsmasq
	// with periodic re-resolution. Required for services behind load
	// balancers or CDNs.
	Dynamic Category = "dynamic"
)

// Domain represents a single allowlisted domain with its resolution category
// and a human-readable rationale explaining why it is needed.
type Domain struct {
	Name      string
	Category  Category
	Rationale string
}

// Allowlist groups the curated domains for a single stack.
type Allowlist struct {
	Stack   Stack
	Domains []Domain
}

// registry is the canonical, package-level curated domain data. It is never
// exposed directly; callers go through Registry().
var registry = map[Stack]Allowlist{
	AlwaysOn: {
		Stack: AlwaysOn,
		Domains: []Domain{
			{Name: "api.github.com", Category: Static, Rationale: "GitHub REST API - required for git clone/push/pull over HTTPS"},
			{Name: "github.com", Category: Static, Rationale: "GitHub web and git-over-HTTPS"},
			{Name: "*.anthropic.com", Category: Dynamic, Rationale: "Anthropic API - required for Claude Code to function"},
			{Name: "sentry.io", Category: Static, Rationale: "Error reporting for Claude Code"},
			{Name: "statsig.com", Category: Static, Rationale: "Feature flags and experimentation for Claude Code"},
		},
	},
	Go: {
		Stack: Go,
		Domains: []Domain{
			{Name: "proxy.golang.org", Category: Dynamic, Rationale: "Default Go module proxy - serves go get / go mod download"},
			{Name: "sum.golang.org", Category: Dynamic, Rationale: "Go checksum database - verifies module integrity"},
			{Name: "storage.googleapis.com", Category: Dynamic, Rationale: "GCS backend for Go module proxy - actual module content delivery"},
		},
	},
	Node: {
		Stack: Node,
		Domains: []Domain{
			{Name: "registry.npmjs.org", Category: Static, Rationale: "npm package registry - required for npm install"},
			{Name: "cdn.jsdelivr.net", Category: Dynamic, Rationale: "jsDelivr CDN - serves package tarballs for some workflows"},
			{Name: "unpkg.com", Category: Dynamic, Rationale: "CDN for npm packages - used by some tooling for direct browser imports"},
		},
	},
	Python: {
		Stack: Python,
		Domains: []Domain{
			{Name: "pypi.org", Category: Static, Rationale: "Python Package Index - required for pip install"},
			{Name: "files.pythonhosted.org", Category: Static, Rationale: "Hosts actual package files for PyPI downloads"},
		},
	},
	Rust: {
		Stack: Rust,
		Domains: []Domain{
			{Name: "crates.io", Category: Static, Rationale: "Rust package registry - required for cargo build / cargo add"},
			{Name: "static.crates.io", Category: Static, Rationale: "Serves crate tarballs - actual package content delivery"},
		},
	},
	Ruby: {
		Stack: Ruby,
		Domains: []Domain{
			{Name: "rubygems.org", Category: Static, Rationale: "RubyGems package registry - required for gem install / bundle install"},
			{Name: "index.rubygems.org", Category: Static, Rationale: "Compact index for dependency resolution - used by Bundler"},
		},
	},
}

// Registry returns the full curated domain allowlist registry. Each call
// returns a fresh shallow copy of the internal map to prevent callers from
// mutating the canonical data.
func Registry() map[Stack]Allowlist {
	out := make(map[Stack]Allowlist, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}

// ForStack returns the allowlist for the given stack. The second return value
// is false if the stack is not found in the registry.
func ForStack(stack Stack) (Allowlist, bool) {
	al, ok := registry[stack]
	return al, ok
}

// Stacks returns all registered stack names in sorted (deterministic) order.
// Useful for iteration and display in the CLI wizard.
func Stacks() []Stack {
	stacks := make([]Stack, 0, len(registry))
	for k := range registry {
		stacks = append(stacks, k)
	}
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i] < stacks[j]
	})
	return stacks
}
