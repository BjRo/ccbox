// Package stack defines the metadata registry for all supported technology
// stacks. It serves as the single source of truth for stack-specific behavior
// across agentbox, including runtime versions, language servers, domain
// allowlists, and marker files.
//
// The registry is consumed by multiple packages (detect, firewall, render) and
// lives in its own package to avoid import cycles. It is read-only after
// initialization and all accessor functions return copies to prevent mutation.
package stack
