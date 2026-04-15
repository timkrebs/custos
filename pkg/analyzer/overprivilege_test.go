package analyzer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/timkrebs/custos/pkg/parser"
	"github.com/timkrebs/custos/pkg/spec"
)

// TestAnalyze_OverprivilegedFixture walks the real testdata policy that
// is intentionally crafted to trip every built-in check and asserts the
// full set of findings end-to-end. This doubles as a regression fence:
// if someone tightens a check later, the fixture counts have to move
// with it, which forces a deliberate decision.
func TestAnalyze_OverprivilegedFixture(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "policies", "overprivileged.hcl")
	p, err := parser.ParsePolicyFile(fixture)
	if err != nil {
		t.Fatalf("ParsePolicyFile: %v", err)
	}

	findings := Analyze([]parser.Policy{*p}, nil)

	// Expected hits on the fixture, keyed by (check, path) so ordering
	// changes don't flap the assertion.
	want := map[string]bool{
		CheckWildcardPaths + "|secret/*":              true,
		CheckWildcardPaths + "|database/config/*":     true,
		CheckSudoCapability + "|secret/*":             true,
		CheckSudoCapability + "|database/config/*":    true,
		CheckRootTokenCreate + "|auth/token/create":   true,
		CheckPolicyEscalation + "|sys/policies/acl/*": true,
		CheckSecretDestroy + "|secret/destroy/*":      true,
		CheckSecretDestroy + "|secret/metadata/*":     true,
	}

	got := make(map[string]bool)
	for _, f := range findings {
		got[f.Check+"|"+f.Path] = true

		if f.File == "" {
			t.Errorf("finding %+v missing file", f)
		}
		if f.Line == 0 {
			t.Errorf("finding %+v missing line", f)
		}
		if f.Message == "" {
			t.Errorf("finding %+v missing message", f)
		}
		if f.Severity == "" {
			t.Errorf("finding %+v missing severity", f)
		}
	}

	for key := range want {
		if !got[key] {
			t.Errorf("expected finding %s not emitted", key)
		}
	}

	// sys/seal and sys/unseal carry sudo but are under sys/ — they must
	// NOT trip the sudo_capability check, proving the sys/ exemption.
	for _, f := range findings {
		if f.Check == CheckSudoCapability && strings.HasPrefix(f.Path, "sys/") {
			t.Errorf("sudo_capability should not flag sys/ path, got %q", f.Path)
		}
	}
}

// TestAnalyze_DefaultSeverities pins the severity column of the check
// table to the constants so a future reshuffle of the map is intentional.
func TestAnalyze_DefaultSeverities(t *testing.T) {
	cases := map[string]Severity{
		CheckWildcardPaths:    SeverityWarning,
		CheckSudoCapability:   SeverityError,
		CheckRootTokenCreate:  SeverityError,
		CheckPolicyEscalation: SeverityError,
		CheckSecretDestroy:    SeverityWarning,
	}
	for id, want := range cases {
		if got := defaultSeverities[id]; got != want {
			t.Errorf("default severity %s = %q, want %q", id, got, want)
		}
	}
}

// TestAnalyze_WildcardThreshold verifies the "3+ capabilities" rule —
// two caps on a wildcard path is a legitimate browse grant and should
// not emit a finding.
func TestAnalyze_WildcardThreshold(t *testing.T) {
	policies := []parser.Policy{{
		Filepath: "t.hcl",
		Paths: []parser.PathRule{
			{Path: "secret/browse/*", Capabilities: []string{"read", "list"}, Line: 1},
			{Path: "secret/broad/*", Capabilities: []string{"read", "list", "create"}, Line: 2},
		},
	}}
	findings := Analyze(policies, nil)

	var wild []Finding
	for _, f := range findings {
		if f.Check == CheckWildcardPaths {
			wild = append(wild, f)
		}
	}
	if len(wild) != 1 || wild[0].Path != "secret/broad/*" {
		t.Errorf("wildcard findings = %+v, want only secret/broad/*", wild)
	}
}

// TestAnalyze_SudoAuthTokenExempt checks the auth/token/ exemption for
// sudo, which mirrors Vault's own documentation on legitimate sudo use.
func TestAnalyze_SudoAuthTokenExempt(t *testing.T) {
	policies := []parser.Policy{{
		Filepath: "t.hcl",
		Paths: []parser.PathRule{
			{Path: "auth/token/revoke-orphan", Capabilities: []string{"sudo", "update"}, Line: 1},
		},
	}}
	for _, f := range Analyze(policies, nil) {
		if f.Check == CheckSudoCapability {
			t.Errorf("auth/token/ path should not flag sudo_capability, got %+v", f)
		}
	}
}

// TestAnalyze_LegacyPolicyEscalation exercises the sys/policy/ legacy
// prefix in addition to the modern sys/policies/acl/ path.
func TestAnalyze_LegacyPolicyEscalation(t *testing.T) {
	policies := []parser.Policy{{
		Filepath: "t.hcl",
		Paths: []parser.PathRule{
			{Path: "sys/policy/my-policy", Capabilities: []string{"update"}, Line: 1},
		},
	}}
	findings := Analyze(policies, nil)
	if len(findings) != 1 || findings[0].Check != CheckPolicyEscalation {
		t.Errorf("want one policy_escalation finding, got %+v", findings)
	}
}

// TestAnalyze_ConfigDisabled ensures a user can turn off a check with
// `disabled: true` in the spec analyze section.
func TestAnalyze_ConfigDisabled(t *testing.T) {
	policies := []parser.Policy{{
		Filepath: "t.hcl",
		Paths: []parser.PathRule{
			{Path: "auth/token/create", Capabilities: []string{"create"}, Line: 1},
		},
	}}
	rules := []spec.AnalyzeCheck{
		{Check: CheckRootTokenCreate, Disabled: true},
	}
	if findings := Analyze(policies, rules); len(findings) != 0 {
		t.Errorf("want zero findings after disabling check, got %+v", findings)
	}
}

// TestAnalyze_ConfigAllowPaths verifies the allowlist — this is the
// escape hatch for legitimate sys/seal sudo grants and similar.
func TestAnalyze_ConfigAllowPaths(t *testing.T) {
	policies := []parser.Policy{{
		Filepath: "t.hcl",
		Paths: []parser.PathRule{
			// Not under sys/ so it would normally flag, but the allow
			// list exempts database/config/* specifically.
			{Path: "database/config/rotate", Capabilities: []string{"sudo", "update"}, Line: 1},
			// And this one still trips.
			{Path: "kv/foo", Capabilities: []string{"sudo"}, Line: 2},
		},
	}}
	rules := []spec.AnalyzeCheck{
		{Check: CheckSudoCapability, AllowPaths: []string{"database/config/*"}},
	}
	findings := Analyze(policies, rules)
	if len(findings) != 1 || findings[0].Path != "kv/foo" {
		t.Errorf("want single finding on kv/foo, got %+v", findings)
	}
}

// TestAnalyze_ConfigSeverityOverride checks that an operator can bump a
// warning check up to error (or vice versa) via spec config.
func TestAnalyze_ConfigSeverityOverride(t *testing.T) {
	policies := []parser.Policy{{
		Filepath: "t.hcl",
		Paths: []parser.PathRule{
			{Path: "secret/destroy/foo", Capabilities: []string{"update"}, Line: 1},
		},
	}}
	rules := []spec.AnalyzeCheck{
		{Check: CheckSecretDestroy, Severity: "error"},
	}
	findings := Analyze(policies, rules)
	if len(findings) != 1 || findings[0].Severity != SeverityError {
		t.Errorf("want single finding with error severity, got %+v", findings)
	}
}

// TestPathMatches exercises the allow-list matcher directly because it
// has enough branches (exact, trailing-*, + segment) to warrant its own
// table.
func TestPathMatches(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"secret/foo", "secret/foo", true},
		{"secret/foo", "secret/bar", false},
		{"secret/*", "secret/anything/deep", true},
		{"secret/+/config", "secret/svc/config", true},
		{"secret/+/config", "secret/svc/other", false},
		{"secret/+/config", "secret//config", false},
		{"", "secret/foo", false},
	}
	for _, tc := range cases {
		if got := pathMatches(tc.pattern, tc.path); got != tc.want {
			t.Errorf("pathMatches(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}
