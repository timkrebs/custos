package evaluator

import (
	"sort"
	"testing"

	"github.com/timkrebs/custos/pkg/parser"
)

// TestCompose_CrossPolicyUnion_GrantsMostPermissive is a regression guard for
// the per-policy composition rule: when one policy grants broader
// capabilities via a less specific path and another policy grants narrower
// capabilities via a more specific path, the composed result must union the
// two. A naive global-best-match selector would drop the broader policy's
// contribution and produce a false deny.
func TestCompose_CrossPolicyUnion_GrantsMostPermissive(t *testing.T) {
	policies := []parser.Policy{
		{
			Filepath: "broad.hcl",
			Paths: []parser.PathRule{
				{Path: "secret/*", Capabilities: []string{"read", "create"}},
			},
		},
		{
			Filepath: "narrow.hcl",
			Paths: []parser.PathRule{
				{Path: "secret/foo", Capabilities: []string{"read"}},
			},
		},
	}

	c := Compose(policies, "secret/foo")

	if c.Denied {
		t.Fatalf("expected allow, got denied: %+v", c)
	}
	if !c.HasAll([]string{"read", "create"}) {
		t.Errorf("expected union to grant read+create, got %v", capSetToSlice(c.Granted))
	}
	if len(c.Contributions) != 2 {
		t.Errorf("expected 2 contributions, got %d: %+v", len(c.Contributions), c.Contributions)
	}
	if len(c.GrantedBy["create"]) != 1 || c.GrantedBy["create"][0].PolicyFile != "broad.hcl" {
		t.Errorf("expected create to be provenance-tracked to broad.hcl, got %+v", c.GrantedBy["create"])
	}
	if len(c.GrantedBy["read"]) != 2 {
		t.Errorf("expected read to be granted by two policies, got %+v", c.GrantedBy["read"])
	}
}

// TestCompose_CrossPolicyDeny_OverridesMoreSpecificAllow is a regression
// guard for Vault's deny-override rule at composition time: if any policy's
// per-policy winner carries deny, the composed result is denied, even when
// another policy has a more specific allow for the same request path.
func TestCompose_CrossPolicyDeny_OverridesMoreSpecificAllow(t *testing.T) {
	policies := []parser.Policy{
		{
			Filepath: "allow.hcl",
			Paths: []parser.PathRule{
				{Path: "secret/foo", Capabilities: []string{"read"}},
			},
		},
		{
			Filepath: "deny.hcl",
			Paths: []parser.PathRule{
				{Path: "secret/*", Capabilities: []string{"deny"}},
			},
		},
	}

	c := Compose(policies, "secret/foo")

	if !c.Denied {
		t.Fatalf("expected deny from deny.hcl to override allow.hcl, got %+v", c)
	}
	if len(c.DeniedBy) != 1 || c.DeniedBy[0].PolicyFile != "deny.hcl" {
		t.Errorf("expected DeniedBy to point at deny.hcl, got %+v", c.DeniedBy)
	}
	if c.HasAll([]string{"read"}) {
		t.Error("HasAll must return false on a denied composition")
	}
	missing := c.Missing([]string{"read"})
	if len(missing) != 1 || missing[0] != "read" {
		t.Errorf("Missing on denied composition should return all requested, got %v", missing)
	}
}

// TestCompose_NoMatchingRules returns an empty contribution list so callers
// can distinguish implicit deny (no rules matched) from explicit deny.
func TestCompose_NoMatchingRules(t *testing.T) {
	policies := []parser.Policy{
		{
			Filepath: "p.hcl",
			Paths: []parser.PathRule{
				{Path: "secret/other", Capabilities: []string{"read"}},
			},
		},
	}

	c := Compose(policies, "secret/foo")

	if c.Denied {
		t.Error("implicit deny should not set Denied; Denied is for explicit deny capability")
	}
	if len(c.Contributions) != 0 {
		t.Errorf("expected zero contributions, got %d", len(c.Contributions))
	}
	if len(c.Granted) != 0 {
		t.Errorf("expected empty grant set, got %v", c.Granted)
	}
}

// TestCompose_WithinPolicyMostSpecificWins verifies that selection inside a
// single policy still follows longest-prefix-match semantics — this is the
// invariant the old global-best selector already satisfied, and the
// refactor must not regress it.
func TestCompose_WithinPolicyMostSpecificWins(t *testing.T) {
	policies := []parser.Policy{
		{
			Filepath: "p.hcl",
			Paths: []parser.PathRule{
				{Path: "secret/*", Capabilities: []string{"read", "create"}},
				{Path: "secret/foo", Capabilities: []string{"read"}},
			},
		},
	}

	c := Compose(policies, "secret/foo")

	if c.Denied {
		t.Fatalf("unexpected deny: %+v", c)
	}
	if len(c.Contributions) != 1 {
		t.Fatalf("expected single contribution from most-specific rule, got %d", len(c.Contributions))
	}
	if c.Contributions[0].RulePath != "secret/foo" {
		t.Errorf("expected secret/foo to win within policy, got %q", c.Contributions[0].RulePath)
	}
	if c.Granted["create"] {
		t.Error("create should not leak from the less specific secret/* rule")
	}
}

// TestCompose_ContributionOrderMatchesInput confirms that Contributions
// preserves input policy order, so downstream reporting is deterministic.
func TestCompose_ContributionOrderMatchesInput(t *testing.T) {
	policies := []parser.Policy{
		{Filepath: "first.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"read"}}}},
		{Filepath: "second.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"update"}}}},
		{Filepath: "third.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"delete"}}}},
	}

	c := Compose(policies, "secret/foo")

	want := []string{"first.hcl", "second.hcl", "third.hcl"}
	got := make([]string, len(c.Contributions))
	for i, contrib := range c.Contributions {
		got[i] = contrib.PolicyFile
	}
	if !equalSlices(got, want) {
		t.Errorf("Contributions order = %v, want %v", got, want)
	}
}

// TestCompose_DenyRecordedInContributions asserts that a deny contribution
// is also recorded in the full Contributions list (not only in DeniedBy),
// so provenance reports can show the denying rule alongside grants.
func TestCompose_DenyRecordedInContributions(t *testing.T) {
	policies := []parser.Policy{
		{Filepath: "grant.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"read"}}}},
		{Filepath: "deny.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"deny"}}}},
	}

	c := Compose(policies, "secret/foo")

	if len(c.Contributions) != 2 {
		t.Fatalf("expected both contributions recorded, got %d", len(c.Contributions))
	}
	var sawDeny bool
	for _, contrib := range c.Contributions {
		if contrib.IsDeny && contrib.PolicyFile == "deny.hcl" {
			sawDeny = true
		}
	}
	if !sawDeny {
		t.Error("deny contribution missing from Contributions list")
	}
}

// TestCompose_HasAllAndMissingBehavior exercises the Composed helpers.
func TestCompose_HasAllAndMissingBehavior(t *testing.T) {
	policies := []parser.Policy{
		{Filepath: "p.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"read", "list"}}}},
	}
	c := Compose(policies, "secret/foo")

	if !c.HasAll([]string{"read"}) {
		t.Error("expected HasAll(read) to be true")
	}
	if c.HasAll([]string{"read", "update"}) {
		t.Error("expected HasAll(read, update) to be false")
	}
	missing := c.Missing([]string{"read", "update", "delete"})
	sort.Strings(missing)
	want := []string{"delete", "update"}
	if !equalSlices(missing, want) {
		t.Errorf("Missing = %v, want %v", missing, want)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
