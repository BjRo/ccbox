package firewall

import (
	"fmt"
	"regexp"
	"strings"
)

// domainPattern matches valid DNS hostnames per RFC 1123, with an optional
// leading "*." for wildcard domains. Each label must be 1-63 alphanumeric
// characters or hyphens, must not start or end with a hyphen, and labels
// are separated by dots.
var domainPattern = regexp.MustCompile(
	`^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`,
)

// ValidateDomain validates that name is a well-formed DNS hostname, optionally
// prefixed with "*." for wildcard domains. It rejects empty strings, names
// containing shell metacharacters, names with labels exceeding 63 characters,
// and names exceeding 253 characters total. This provides defense-in-depth
// against shell injection when domain names are interpolated into generated
// shell scripts.
func ValidateDomain(name string) error {
	if name == "" {
		return fmt.Errorf("firewall: invalid domain name: empty string")
	}

	// Check total length. The maximum DNS name length is 253 characters
	// (excluding the trailing dot in the fully qualified form).
	if len(name) > 253 {
		return fmt.Errorf("firewall: invalid domain name %q: exceeds 253 characters", name)
	}

	// Check individual label lengths. Strip optional wildcard prefix first.
	bareLabels := strings.TrimPrefix(name, "*.")

	for _, label := range strings.Split(bareLabels, ".") {
		if len(label) > 63 {
			return fmt.Errorf("firewall: invalid domain name %q: label %q exceeds 63 characters", name, label)
		}
	}

	if !domainPattern.MatchString(name) {
		return fmt.Errorf("firewall: invalid domain name %q: must be a valid DNS hostname", name)
	}

	return nil
}
