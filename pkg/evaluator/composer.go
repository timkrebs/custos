package evaluator

import (
	"github.com/timkrebs/custos/pkg/parser"
)

// RuleContribution records a single policy's contribution to a composed
// evaluation: the rule within that policy whose path most specifically
// matched the request, along with the capabilities that rule granted.
// A contribution with IsDeny set carries Vault's explicit deny capability
// and hard-overrides any grants across the entity's other policies.
type RuleContribution struct {
	PolicyFile   string
	RulePath     string
	Capabilities []string
	IsDeny       bool
}

// Composed is the result of evaluating a single request path against a set
// of policies using Vault's composition semantics:
//
//  1. For each policy independently, select the most specific matching path
//     rule (longest-prefix / most-literal match).
//  2. Union the capabilities from every per-policy winner.
//  3. If any winner includes the "deny" capability, the composed result is
//     denied regardless of other grants.
//
// The Contributions slice preserves per-policy provenance so callers can
// report which policy granted or denied which capability.
type Composed struct {
	// Path is the request path that was evaluated.
	Path string

	// Granted is the union of capabilities contributed by all matching
	// policies, excluding the "deny" sentinel. Keyed for O(1) lookup.
	Granted map[string]bool

	// Denied is true when at least one contribution carries the "deny"
	// capability. When true, Granted should be treated as advisory only;
	// the composed decision is deny.
	Denied bool

	// Contributions lists the per-policy winning rules in the order the
	// policies were supplied to Compose. Policies that had no matching
	// rule for the request path are omitted.
	Contributions []RuleContribution

	// GrantedBy maps each granted capability to the contributions that
	// granted it. Useful for provenance reporting such as "read granted
	// by readonly.hcl and payment-svc.hcl".
	GrantedBy map[string][]RuleContribution

	// DeniedBy lists every contribution that carried the deny capability.
	// Empty when Denied is false.
	DeniedBy []RuleContribution
}

// Compose evaluates a request path against a set of policies and returns the
// merged decision following Vault composition rules. When no policy has a
// matching rule, the returned Composed has zero Contributions; callers
// should treat that as an implicit deny.
//
// Compose is pure: it does not mutate the input policies and is safe to call
// concurrently.
func Compose(policies []parser.Policy, requestPath string) Composed {
	c := Composed{
		Path:      requestPath,
		Granted:   make(map[string]bool),
		GrantedBy: make(map[string][]RuleContribution),
	}

	for _, pol := range policies {
		best := bestMatchWithinPolicy(pol, requestPath)
		if best == nil {
			continue
		}

		contrib := RuleContribution{
			PolicyFile:   best.policyFile,
			RulePath:     best.rule.Path,
			Capabilities: best.rule.Capabilities,
		}

		for _, capability := range best.rule.Capabilities {
			if capability == "deny" {
				contrib.IsDeny = true
				continue
			}
			c.Granted[capability] = true
		}

		c.Contributions = append(c.Contributions, contrib)

		if contrib.IsDeny {
			c.Denied = true
			c.DeniedBy = append(c.DeniedBy, contrib)
			continue
		}

		for _, capability := range best.rule.Capabilities {
			if capability == "deny" {
				continue
			}
			c.GrantedBy[capability] = append(c.GrantedBy[capability], contrib)
		}
	}

	return c
}

// HasAll reports whether every requested capability is present in the
// composed grant set. Denied results always return false.
func (c Composed) HasAll(requested []string) bool {
	if c.Denied {
		return false
	}
	for _, capability := range requested {
		if !c.Granted[capability] {
			return false
		}
	}
	return true
}

// Missing returns the subset of requested capabilities that are not present
// in the composed grant set. For a denied result, every requested capability
// is considered missing since the hard deny overrides all grants.
func (c Composed) Missing(requested []string) []string {
	if c.Denied {
		out := make([]string, len(requested))
		copy(out, requested)
		return out
	}
	var missing []string
	for _, capability := range requested {
		if !c.Granted[capability] {
			missing = append(missing, capability)
		}
	}
	return missing
}

// bestMatchWithinPolicy returns the most specific matching rule inside a
// single policy, or nil if no rule in that policy matches requestPath.
// Selection priority matches Vault: exact > segment-wildcard > prefix, and
// within the same tier the rule with the longest literal portion wins.
func bestMatchWithinPolicy(policy parser.Policy, requestPath string) *matchCandidate {
	var best *matchCandidate
	for i := range policy.Paths {
		rule := policy.Paths[i]
		matched, mtype, specificity := matchPath(rule.Path, requestPath)
		if !matched {
			continue
		}
		cand := matchCandidate{
			policyFile:  policy.Filepath,
			rule:        rule,
			mtype:       mtype,
			specificity: specificity,
		}
		if best == nil {
			best = &cand
			continue
		}
		if cand.mtype > best.mtype {
			best = &cand
			continue
		}
		if cand.mtype == best.mtype && cand.specificity > best.specificity {
			best = &cand
		}
	}
	return best
}
