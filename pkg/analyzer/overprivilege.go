// Package analyzer inspects parsed Vault policies for dangerous privilege
// patterns without evaluating any test assertions. Analyzer output
// (Findings) is independent of test pass/fail state: a policy can pass
// every behavioral test and still produce findings, and vice versa.
//
// The checks implemented here target well-known Vault misconfigurations:
// overbroad wildcards, sudo on non-system paths, root token minting,
// policy self-escalation, and destructive operations against KV v2
// versioning metadata. Operators who legitimately need one of these
// grants (a break-glass admin policy, an unseal operator) suppress the
// finding via the `analyze` section in the spec YAML — see AnalyzeCheck
// in pkg/spec for the `disabled` and `allow_paths` knobs.
package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/timkrebs/custos/pkg/parser"
	"github.com/timkrebs/custos/pkg/spec"
)

// Severity classifies a finding. It is a typed string so reporters and
// CLI exit-code logic can filter on it without string literals.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Check identifiers. Stable IDs so spec authors can reference them in
// the analyze section and so reporter output stays grep-friendly across
// custos versions.
const (
	CheckWildcardPaths    = "wildcard_paths"
	CheckSudoCapability   = "sudo_capability"
	CheckRootTokenCreate  = "root_token_create"
	CheckPolicyEscalation = "policy_escalation"
	CheckSecretDestroy    = "secret_destroy"
)

// wildcardMinCapabilities is the threshold at which a trailing-wildcard
// path is considered broad enough to flag. Three or more capabilities on
// a `foo/*` path means read-plus-mutate, which is the anti-pattern we
// care about; two capabilities (e.g. read+list) is a reasonable browse
// grant and should not warn.
const wildcardMinCapabilities = 3

// Finding is one analyzer hit. File and Line point at the offending path
// block in the source HCL so editors and CI annotators can navigate.
// RuleCapabilities is the raw capability set from the policy — handy for
// reporters that want to render the violating grant inline.
type Finding struct {
	Check            string   `json:"check"`
	Severity         Severity `json:"severity"`
	Message          string   `json:"message"`
	File             string   `json:"file"`
	Line             int      `json:"line"`
	Path             string   `json:"path"`
	RuleCapabilities []string `json:"rule_capabilities,omitempty"`
}

// checkConfig is the resolved per-check configuration after merging
// defaults with the user's `analyze` section.
type checkConfig struct {
	disabled     bool
	severity     Severity
	allowPaths   []string
	defaultSev   Severity
}

// defaultSeverities fixes the baseline severity for each built-in check.
// Spec authors can override these via `analyze[].severity`.
var defaultSeverities = map[string]Severity{
	CheckWildcardPaths:    SeverityWarning,
	CheckSudoCapability:   SeverityError,
	CheckRootTokenCreate:  SeverityError,
	CheckPolicyEscalation: SeverityError,
	CheckSecretDestroy:    SeverityWarning,
}

// BuiltinChecks lists every check this package knows about, in a stable
// order suitable for documentation and listing commands.
var BuiltinChecks = []string{
	CheckWildcardPaths,
	CheckSudoCapability,
	CheckRootTokenCreate,
	CheckPolicyEscalation,
	CheckSecretDestroy,
}

// Analyze runs every enabled overprivilege check against every parsed
// policy and returns the aggregated findings. Findings are ordered by
// policy input order, then by source line, then by check ID so reporter
// output is deterministic across runs.
//
// The `rules` argument is the `analyze:` section from the spec YAML and
// is optional — passing nil runs every built-in check at default
// severity with no exceptions.
func Analyze(policies []parser.Policy, rules []spec.AnalyzeCheck) []Finding {
	cfg := loadConfig(rules)

	var findings []Finding
	for _, p := range policies {
		for _, rule := range p.Paths {
			findings = append(findings, runChecks(p.Filepath, rule, cfg)...)
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Check < findings[j].Check
	})
	return findings
}

// loadConfig materializes the effective configuration for every
// built-in check, applying user overrides from the spec on top of the
// defaults. Unknown check names in the spec are silently ignored — the
// validator in pkg/spec only enforces structural validity, not the list
// of known checks, so an analyze entry for a check this version of
// custos does not know is treated as a no-op rather than a hard error.
func loadConfig(rules []spec.AnalyzeCheck) map[string]*checkConfig {
	cfg := make(map[string]*checkConfig, len(BuiltinChecks))
	for _, id := range BuiltinChecks {
		cfg[id] = &checkConfig{
			severity:   defaultSeverities[id],
			defaultSev: defaultSeverities[id],
		}
	}

	for _, r := range rules {
		c, ok := cfg[r.Check]
		if !ok {
			continue
		}
		c.disabled = r.Disabled
		if sev := normalizeSeverity(r.Severity); sev != "" {
			c.severity = sev
		}
		c.allowPaths = append(c.allowPaths, r.AllowPaths...)
	}
	return cfg
}

// normalizeSeverity maps the spec's accepted severity aliases onto the
// analyzer's typed Severity constants. An empty input is returned as
// empty so callers can tell "unset" from "explicitly set to error".
func normalizeSeverity(s string) Severity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return ""
	case "error":
		return SeverityError
	case "warning", "warn":
		return SeverityWarning
	case "info":
		return SeverityInfo
	default:
		return ""
	}
}

// runChecks evaluates all five rules against a single path rule. Each
// individual check is a small closure so the dispatch list reads like
// the table in the issue description.
func runChecks(file string, rule parser.PathRule, cfg map[string]*checkConfig) []Finding {
	var out []Finding
	emit := func(id, msg string) {
		c := cfg[id]
		if c == nil || c.disabled {
			return
		}
		if pathMatchesAny(rule.Path, c.allowPaths) {
			return
		}
		out = append(out, Finding{
			Check:            id,
			Severity:         c.severity,
			Message:          msg,
			File:             file,
			Line:             rule.Line,
			Path:             rule.Path,
			RuleCapabilities: append([]string(nil), rule.Capabilities...),
		})
	}

	if isBroadWildcard(rule) {
		emit(CheckWildcardPaths, fmt.Sprintf(
			"wildcard path %q grants %d capabilities (%s); narrow the path or split by access level",
			rule.Path, len(rule.Capabilities), strings.Join(rule.Capabilities, ", "),
		))
	}
	if hasSudoOnNonSystemPath(rule) {
		emit(CheckSudoCapability, fmt.Sprintf(
			"path %q grants sudo on a non-system path; sudo should be limited to sys/* or auth/token/*",
			rule.Path,
		))
	}
	if isRootTokenCreate(rule) {
		emit(CheckRootTokenCreate, fmt.Sprintf(
			"path %q permits token minting via auth/token/create; treat this as a privileged grant",
			rule.Path,
		))
	}
	if isPolicyEscalation(rule) {
		emit(CheckPolicyEscalation, fmt.Sprintf(
			"path %q permits writing ACL policies; any holder can grant themselves arbitrary capabilities",
			rule.Path,
		))
	}
	if isSecretDestroy(rule) {
		emit(CheckSecretDestroy, fmt.Sprintf(
			"path %q permits destructive KV v2 operations (version destroy or metadata delete); prefer soft-delete",
			rule.Path,
		))
	}
	return out
}

// isBroadWildcard reports whether a path ends in a trailing * and grants
// three or more capabilities. The threshold is defined as
// wildcardMinCapabilities so teams can reason about it in one place.
func isBroadWildcard(rule parser.PathRule) bool {
	if !strings.HasSuffix(rule.Path, "*") {
		return false
	}
	return len(uniqueCapabilities(rule.Capabilities)) >= wildcardMinCapabilities
}

// hasSudoOnNonSystemPath reports whether a rule grants sudo outside of
// the sys/ and auth/token/ trees. Sudo on sys/* or auth/token/* is how
// operators legitimately call privileged endpoints; sudo anywhere else
// almost always indicates a misunderstanding of the capability.
func hasSudoOnNonSystemPath(rule parser.PathRule) bool {
	if !containsCap(rule.Capabilities, "sudo") {
		return false
	}
	if strings.HasPrefix(rule.Path, "sys/") {
		return false
	}
	if strings.HasPrefix(rule.Path, "auth/token/") {
		return false
	}
	return true
}

// isRootTokenCreate detects direct grants on auth/token/create (the root
// token-minting endpoint family). Any create grant here is flagged —
// there is no benign "read-only" use of token creation.
func isRootTokenCreate(rule parser.PathRule) bool {
	if !containsCap(rule.Capabilities, "create") {
		return false
	}
	return rule.Path == "auth/token/create" ||
		rule.Path == "auth/token/create/*" ||
		strings.HasPrefix(rule.Path, "auth/token/create/")
}

// isPolicyEscalation detects update-on-policy endpoints. Update on
// sys/policy/* (legacy) or sys/policies/acl/* (current) lets the holder
// rewrite arbitrary ACL policies, which trivially composes into full
// root access.
func isPolicyEscalation(rule parser.PathRule) bool {
	if !containsCap(rule.Capabilities, "update") &&
		!containsCap(rule.Capabilities, "create") {
		return false
	}
	return strings.HasPrefix(rule.Path, "sys/policy/") ||
		strings.HasPrefix(rule.Path, "sys/policies/acl/")
}

// isSecretDestroy detects destructive operations on KV v2 versioning
// endpoints. `delete` on secret/destroy/* permanently removes versions,
// and `delete` on secret/metadata/* tombstones every version at once —
// both are irreversible and warrant a warning even when intentional.
// We also flag `update` on secret/destroy/* because that endpoint's
// destroy operation is invoked with update, not delete.
func isSecretDestroy(rule parser.PathRule) bool {
	if strings.HasPrefix(rule.Path, "secret/destroy/") {
		return containsCap(rule.Capabilities, "update") ||
			containsCap(rule.Capabilities, "delete")
	}
	if strings.HasPrefix(rule.Path, "secret/metadata/") {
		return containsCap(rule.Capabilities, "delete")
	}
	return false
}

// containsCap is a small helper that avoids pulling in a set type for a
// handful of membership checks.
func containsCap(caps []string, want string) bool {
	for _, c := range caps {
		if c == want {
			return true
		}
	}
	return false
}

// uniqueCapabilities deduplicates a capability list so policies that
// accidentally list a capability twice do not trip the wildcard
// threshold on the dupe alone.
func uniqueCapabilities(caps []string) []string {
	seen := make(map[string]struct{}, len(caps))
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

// pathMatchesAny reports whether the given path matches any entry in
// the allow-list. Entries use Vault-style glob semantics (trailing *
// is prefix match, + is single segment, otherwise exact). This mirrors
// the match semantics the evaluator uses so operators do not have to
// learn a second matcher just for exceptions.
func pathMatchesAny(path string, patterns []string) bool {
	for _, pat := range patterns {
		if pathMatches(pat, path) {
			return true
		}
	}
	return false
}

// pathMatches is the allow-list matcher. It intentionally supports only
// the two common wildcard forms — a full Vault-style matcher lives in
// the evaluator and is scoped to request-path resolution, which has
// different specificity-ranking needs than a simple allowlist.
func pathMatches(pattern, path string) bool {
	if pattern == "" {
		return false
	}
	if pattern == path {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, pattern[:len(pattern)-1])
	}
	if !strings.Contains(pattern, "+") {
		return false
	}
	patSegs := strings.Split(pattern, "/")
	pathSegs := strings.Split(path, "/")
	if len(patSegs) != len(pathSegs) {
		return false
	}
	for i, seg := range patSegs {
		if seg == "+" {
			if pathSegs[i] == "" {
				return false
			}
			continue
		}
		if seg != pathSegs[i] {
			return false
		}
	}
	return true
}
