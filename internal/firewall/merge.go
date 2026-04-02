package firewall

import (
	"slices"
	"strings"

	"github.com/bjro/ccbox/internal/stack"
)

// MergedDomains holds the final deduplicated domain lists, split by category,
// ready for template rendering. Static domains are resolved once at firewall
// init (init-firewall.sh) and cached in an ipset. Dynamic domains are managed
// by dnsmasq with periodic re-resolution (dynamic-domains.conf).
//
// Callers own the returned slices and may mutate them freely; they are freshly
// allocated by [Merge] and share no backing array with the internal registry.
type MergedDomains struct {
	Static  []Domain // Domains with stable IPs, resolved once at firewall init
	Dynamic []Domain // Domains with rotating IPs, managed by dnsmasq
}

// Merge combines always-on domains, per-stack domains for the given stacks,
// and user-provided extra domains into two deduplicated, sorted lists (Static
// and Dynamic). Unknown stack IDs are silently skipped. User extras are
// classified as Dynamic by default.
//
// Deduplication uses first-occurrence-wins: always-on domains are processed
// first, then per-stack domains in the order given, then user extras. If a
// domain appears in multiple sources, the first entry's category and rationale
// are retained.
func Merge(stacks []stack.StackID, userExtras []string) MergedDomains {
	seen := make(map[string]bool)
	var staticDomains []Domain
	var dynamicDomains []Domain

	addDomain := func(d Domain) {
		if seen[d.Name] {
			return
		}
		seen[d.Name] = true
		switch d.Category {
		case Static:
			staticDomains = append(staticDomains, d)
		case Dynamic:
			dynamicDomains = append(dynamicDomains, d)
		}
	}

	// Step 1: Collect always-on domains.
	if al, ok := ForStack(AlwaysOn); ok {
		for _, d := range al.Domains {
			addDomain(d)
		}
	}

	// Step 2: Collect per-stack domains.
	for _, id := range stacks {
		if al, ok := ForStack(Stack(id)); ok {
			for _, d := range al.Domains {
				addDomain(d)
			}
		}
	}

	// Step 3: Collect user extras (always Dynamic).
	for _, extra := range userExtras {
		name := strings.TrimSpace(extra)
		if name == "" {
			continue
		}
		addDomain(Domain{
			Name:      name,
			Category:  Dynamic,
			Rationale: "User-specified domain",
		})
	}

	// Step 4: Sort both slices by domain name.
	cmp := func(a, b Domain) int {
		return strings.Compare(a.Name, b.Name)
	}
	slices.SortFunc(staticDomains, cmp)
	slices.SortFunc(dynamicDomains, cmp)

	// Return non-nil empty slices when no domains in a category.
	if staticDomains == nil {
		staticDomains = []Domain{}
	}
	if dynamicDomains == nil {
		dynamicDomains = []Domain{}
	}

	return MergedDomains{
		Static:  staticDomains,
		Dynamic: dynamicDomains,
	}
}
