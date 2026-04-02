package stack

import (
	"slices"
	"strings"
)

// StackID identifies a supported technology stack.
// It uses string values rather than integer enums because stack IDs appear
// in configuration files (.ccbox.yml), CLI flags (--stacks=go,node), and
// template output, where self-describing values avoid a marshaling layer.
type StackID string

const (
	Go     StackID = "go"
	Node   StackID = "node"
	Python StackID = "python"
	Ruby   StackID = "ruby"
	Rust   StackID = "rust"
)

// Runtime describes the mise tool name and version strategy for a stack.
type Runtime struct {
	Tool    string // mise tool name, e.g. "go", "node"
	Version string // Version strategy, e.g. "latest", "lts"
}

// LSP describes the language server configuration for a stack.
type LSP struct {
	Package    string // Language server package name, e.g. "gopls"
	InstallCmd string // Full install command, e.g. "go install golang.org/x/tools/gopls@latest"
	Plugin     string // Claude Code plugin identifier, e.g. "gopls"
}

// Stack holds all metadata for a supported technology stack.
// It serves as the single source of truth for stack-specific behavior
// across ccbox, consumed by the detect, firewall, and render packages.
type Stack struct {
	ID   StackID
	Name string // Display name, e.g. "Go", "Node/TypeScript"

	Runtime Runtime
	LSP     LSP

	// DefaultDomains lists static registry/package domains to allowlist
	// in the container firewall.
	// NOTE: Domain lists are provisional placeholders. The firewall bean
	// (ccbox-ztaa) will finalize the exact domain allowlists.
	DefaultDomains []string

	// DynamicDomains lists domains that need dnsmasq resolution because
	// their IPs change frequently (CDNs, etc.).
	// NOTE: Domain lists are provisional placeholders. The firewall bean
	// (ccbox-ztaa) will finalize the exact domain allowlists.
	DynamicDomains []string

	// MarkerFiles lists filenames whose presence in a project root
	// indicates this stack is in use. These are exact filenames only,
	// not glob patterns. Pattern-based detection (e.g., *.go files)
	// is the responsibility of the scanner (ccbox-6j8r), not the registry.
	MarkerFiles []string
}

// registry is the package-level lookup map, keyed by StackID.
// It is read-only after package initialization.
var registry = map[StackID]Stack{
	Go: {
		ID:   Go,
		Name: "Go",
		Runtime: Runtime{
			Tool:    "go",
			Version: "latest",
		},
		LSP: LSP{
			Package:    "gopls",
			InstallCmd: "go install golang.org/x/tools/gopls@latest",
			Plugin:     "gopls",
		},
		DefaultDomains: []string{"proxy.golang.org", "sum.golang.org", "storage.googleapis.com"},
		MarkerFiles:    []string{"go.mod"},
	},
	Node: {
		ID:   Node,
		Name: "Node/TypeScript",
		Runtime: Runtime{
			Tool:    "node",
			Version: "lts",
		},
		LSP: LSP{
			Package:    "typescript-language-server",
			InstallCmd: "npm install -g typescript-language-server typescript",
			Plugin:     "typescript",
		},
		DefaultDomains: []string{"registry.npmjs.org"},
		DynamicDomains: []string{"registry.yarnpkg.com"},
		MarkerFiles:    []string{"package.json"},
	},
	Python: {
		ID:   Python,
		Name: "Python",
		Runtime: Runtime{
			Tool:    "python",
			Version: "latest",
		},
		LSP: LSP{
			Package:    "pyright",
			InstallCmd: "pip install pyright",
			Plugin:     "pyright",
		},
		DefaultDomains: []string{"pypi.org", "files.pythonhosted.org"},
		MarkerFiles:    []string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile"},
	},
	Rust: {
		ID:   Rust,
		Name: "Rust",
		Runtime: Runtime{
			Tool:    "rust",
			Version: "latest",
		},
		LSP: LSP{
			Package:    "rust-analyzer",
			InstallCmd: "rustup component add rust-analyzer",
			Plugin:     "rust-analyzer",
		},
		DefaultDomains: []string{"crates.io", "static.crates.io", "index.crates.io"},
		DynamicDomains: []string{"static.rust-lang.org"},
		MarkerFiles:    []string{"Cargo.toml"},
	},
	Ruby: {
		ID:   Ruby,
		Name: "Ruby",
		Runtime: Runtime{
			Tool:    "ruby",
			Version: "latest",
		},
		LSP: LSP{
			Package:    "solargraph",
			InstallCmd: "gem install solargraph",
			Plugin:     "solargraph",
		},
		DefaultDomains: []string{"rubygems.org", "index.rubygems.org"},
		MarkerFiles:    []string{"Gemfile"},
	},
}

// Get returns the stack metadata for the given ID and a boolean indicating
// whether the stack was found. It follows the standard Go two-value lookup
// convention.
func Get(id StackID) (Stack, bool) {
	s, ok := registry[id]
	if !ok {
		return Stack{}, false
	}
	return copyStack(s), true
}

// All returns all registered stacks sorted by ID for deterministic output
// in templates and CLI displays. It returns copies to prevent callers from
// mutating the registry.
func All() []Stack {
	stacks := make([]Stack, 0, len(registry))
	for _, s := range registry {
		stacks = append(stacks, copyStack(s))
	}
	slices.SortFunc(stacks, func(a, b Stack) int {
		return strings.Compare(string(a.ID), string(b.ID))
	})
	return stacks
}

// IDs returns all registered stack IDs sorted alphabetically.
// It is useful for validating CLI --stacks flag values and displaying
// available options in help text.
func IDs() []StackID {
	ids := make([]StackID, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b StackID) int {
		return strings.Compare(string(a), string(b))
	})
	return ids
}

// copyStack returns a deep copy of a Stack, duplicating all slices so that
// callers cannot mutate registry data.
func copyStack(s Stack) Stack {
	cp := s
	cp.DefaultDomains = slices.Clone(s.DefaultDomains)
	cp.DynamicDomains = slices.Clone(s.DynamicDomains)
	cp.MarkerFiles = slices.Clone(s.MarkerFiles)
	return cp
}
