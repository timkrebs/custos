package evaluator

import (
	"fmt"
	"strings"

	"github.com/timkrebs/custos/pkg/parser"
	"github.com/timkrebs/custos/pkg/spec"
)

// Result captures the outcome of evaluating a single test case.
type Result struct {
	TestName    string
	Path        string
	Allowed     bool
	MatchedRule *MatchedRule // nil if no rule matched (implicit deny)
	Explanation string
}

// MatchedRule identifies which policy and path rule produced the decision.
type MatchedRule struct {
	PolicyFile   string
	RulePath     string
	Capabilities []string
}

// SuiteResult aggregates results for the full test suite.
type SuiteResult struct {
	Suite   string
	Results []TestResult
	Passed  int
	Failed  int
}

// TestResult pairs the spec expectation with the engine result.
type TestResult struct {
	Test   spec.TestCase
	Result Result
	Pass   bool
}

// matchType encodes the priority of a path match.
type matchType int

const (
	matchPrefix matchType = 1 // trailing * (e.g. secret/bar/*)
	matchGlob   matchType = 2 // segment wildcard + (e.g. secret/+/config)
	matchExact  matchType = 3 // exact literal match
)

// matchCandidate tracks a rule that matched a request path during evaluation.
type matchCandidate struct {
	policyFile  string
	rule        parser.PathRule
	mtype       matchType
	specificity int // length of the non-wildcard portion
}

// EvaluateSuite runs all test cases in a spec against the given policies.
func EvaluateSuite(policies []parser.Policy, s *spec.Spec) SuiteResult {
	sr := SuiteResult{Suite: s.Suite}
	for _, tc := range s.Tests {
		result := Evaluate(policies, tc)
		pass := (tc.Expect == "allow") == result.Allowed
		tr := TestResult{
			Test:   tc,
			Result: result,
			Pass:   pass,
		}
		sr.Results = append(sr.Results, tr)
		if pass {
			sr.Passed++
		} else {
			sr.Failed++
		}
	}
	return sr
}

// Evaluate runs a single test case against a set of parsed policies.
func Evaluate(policies []parser.Policy, tc spec.TestCase) Result {
	// Collect all matching rules across all policies.
	var candidates []matchCandidate
	for _, pol := range policies {
		for _, rule := range pol.Paths {
			matched, mtype, specificity := matchPath(rule.Path, tc.Path)
			if matched {
				candidates = append(candidates, matchCandidate{
					policyFile:  pol.Filepath,
					rule:        rule,
					mtype:       mtype,
					specificity: specificity,
				})
			}
		}
	}

	// No matching rule → implicit deny.
	if len(candidates) == 0 {
		return Result{
			TestName:    tc.Name,
			Path:        tc.Path,
			Allowed:     false,
			Explanation: "no policy rule matches path (implicit deny)",
		}
	}

	// Select the best matches (highest priority tier).
	best := selectBestMatches(candidates)

	// Check for deny override: if any best-match has "deny" in capabilities,
	// the result is always deny regardless of other grants.
	for _, c := range best {
		for _, cap := range c.rule.Capabilities {
			if cap == "deny" {
				return Result{
					TestName: tc.Name,
					Path:     tc.Path,
					Allowed:  false,
					MatchedRule: &MatchedRule{
						PolicyFile:   c.policyFile,
						RulePath:     c.rule.Path,
						Capabilities: c.rule.Capabilities,
					},
					Explanation: fmt.Sprintf(
						"explicitly denied by rule %q in %s",
						c.rule.Path, c.policyFile,
					),
				}
			}
		}
	}

	// Merge capabilities from all best-matched rules.
	granted := mergeCapabilities(best)

	// Check if all requested capabilities are present.
	if hasAllCapabilities(granted, tc.Capabilities) {
		first := best[0]
		return Result{
			TestName: tc.Name,
			Path:     tc.Path,
			Allowed:  true,
			MatchedRule: &MatchedRule{
				PolicyFile:   first.policyFile,
				RulePath:     first.rule.Path,
				Capabilities: capSetToSlice(granted),
			},
			Explanation: fmt.Sprintf(
				"allowed by rule %q in %s",
				first.rule.Path, first.policyFile,
			),
		}
	}

	// Some requested capabilities are missing.
	missing := missingCapabilities(granted, tc.Capabilities)
	first := best[0]
	return Result{
		TestName: tc.Name,
		Path:     tc.Path,
		Allowed:  false,
		MatchedRule: &MatchedRule{
			PolicyFile:   first.policyFile,
			RulePath:     first.rule.Path,
			Capabilities: capSetToSlice(granted),
		},
		Explanation: fmt.Sprintf(
			"missing capabilities %v on rule %q in %s",
			missing, first.rule.Path, first.policyFile,
		),
	}
}

// matchPath checks if a rule path matches a request path.
// Returns whether it matched, the match type, and a specificity score.
func matchPath(rulePath, requestPath string) (bool, matchType, int) {
	// No wildcards → exact match only.
	if !strings.Contains(rulePath, "*") && !strings.Contains(rulePath, "+") {
		if rulePath == requestPath {
			return true, matchExact, len(rulePath)
		}
		return false, 0, 0
	}

	// Trailing * → prefix match.
	// In Vault, a trailing * matches zero or more characters including path separators.
	if strings.HasSuffix(rulePath, "*") && !strings.Contains(rulePath, "+") {
		prefix := rulePath[:len(rulePath)-1]
		if strings.HasPrefix(requestPath, prefix) {
			return true, matchPrefix, len(prefix)
		}
		return false, 0, 0
	}

	// Contains + (segment wildcard) → segment-by-segment matching.
	ruleSegs := strings.Split(rulePath, "/")
	reqSegs := strings.Split(requestPath, "/")

	if matchGlobSegments(ruleSegs, reqSegs) {
		specificity := 0
		for _, seg := range ruleSegs {
			if seg != "+" && seg != "*" {
				specificity += len(seg)
			}
		}
		return true, matchGlob, specificity
	}

	return false, 0, 0
}

// matchGlobSegments performs segment-by-segment matching for paths containing
// + (single segment wildcard) and optional trailing * (rest wildcard).
func matchGlobSegments(ruleSegs, reqSegs []string) bool {
	ri := 0
	qi := 0
	for ri < len(ruleSegs) && qi < len(reqSegs) {
		seg := ruleSegs[ri]
		switch {
		case seg == "+":
			// + matches exactly one non-empty segment.
			if reqSegs[qi] == "" {
				return false
			}
			ri++
			qi++
		case seg == "*":
			// Trailing * consumes all remaining segments.
			return true
		default:
			// Literal segment must match exactly.
			if seg != reqSegs[qi] {
				return false
			}
			ri++
			qi++
		}
	}

	// Both must be exhausted for a match (unless trailing *).
	return ri == len(ruleSegs) && qi == len(reqSegs)
}

// selectBestMatches filters candidates to only those in the highest priority tier.
// Priority: exact > glob > prefix. Within the same type, highest specificity wins.
func selectBestMatches(candidates []matchCandidate) []matchCandidate {
	if len(candidates) == 0 {
		return nil
	}

	// Find the best match type and specificity.
	bestType := candidates[0].mtype
	bestSpec := candidates[0].specificity
	for _, c := range candidates[1:] {
		if c.mtype > bestType {
			bestType = c.mtype
			bestSpec = c.specificity
		} else if c.mtype == bestType && c.specificity > bestSpec {
			bestSpec = c.specificity
		}
	}

	// Collect all candidates that match the best tier.
	var best []matchCandidate
	for _, c := range candidates {
		if c.mtype == bestType && c.specificity == bestSpec {
			best = append(best, c)
		}
	}
	return best
}

// mergeCapabilities unions capabilities from all candidates into a single set.
func mergeCapabilities(candidates []matchCandidate) map[string]bool {
	caps := make(map[string]bool)
	for _, c := range candidates {
		for _, cap := range c.rule.Capabilities {
			caps[cap] = true
		}
	}
	return caps
}

// hasAllCapabilities checks if every requested capability exists in the granted set.
func hasAllCapabilities(granted map[string]bool, requested []string) bool {
	for _, cap := range requested {
		if !granted[cap] {
			return false
		}
	}
	return true
}

// missingCapabilities returns the requested capabilities not present in granted.
func missingCapabilities(granted map[string]bool, requested []string) []string {
	var missing []string
	for _, cap := range requested {
		if !granted[cap] {
			missing = append(missing, cap)
		}
	}
	return missing
}

// capSetToSlice converts a capability set to a sorted slice.
func capSetToSlice(caps map[string]bool) []string {
	var result []string
	for cap := range caps {
		result = append(result, cap)
	}
	return result
}
