package evaluator

import (
	"fmt"
	"sort"
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

	// Composed exposes the full multi-policy composition, including
	// per-capability provenance and the list of policies that denied the
	// path. It is nil only for implicit-deny results where no policy had a
	// matching rule. Reporter and UI code can walk this for rich output;
	// MatchedRule remains the single primary attribution for back-compat.
	Composed *Composed
}

// MatchedRule identifies which policy and path rule produced the decision.
type MatchedRule struct {
	PolicyFile   string
	RulePath     string
	Capabilities []string
}

// SuiteResult aggregates results for the full test suite.
type SuiteResult struct {
	Suite    string
	Results  []TestResult
	Passed   int
	Failed   int
	Warnings []string
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
// Composition follows Vault semantics via Compose: per-policy most-specific
// match, then union across policies, with deny as a hard override.
func Evaluate(policies []parser.Policy, tc spec.TestCase) Result {
	composed := Compose(policies, tc.Path)

	// No policy had a matching rule -> implicit deny.
	if len(composed.Contributions) == 0 {
		return Result{
			TestName:    tc.Name,
			Path:        tc.Path,
			Allowed:     false,
			Explanation: "no policy rule matches path (implicit deny)",
			Composed:    &composed,
		}
	}

	// Explicit deny from any contributing policy wins over every grant.
	if composed.Denied {
		first := composed.DeniedBy[0]
		explanation := fmt.Sprintf("explicitly denied by rule %q in %s", first.RulePath, first.PolicyFile)
		if len(composed.DeniedBy) > 1 {
			explanation = fmt.Sprintf("%s (and %d other deny contribution(s))", explanation, len(composed.DeniedBy)-1)
		}
		return Result{
			TestName: tc.Name,
			Path:     tc.Path,
			Allowed:  false,
			MatchedRule: &MatchedRule{
				PolicyFile:   first.PolicyFile,
				RulePath:     first.RulePath,
				Capabilities: first.Capabilities,
			},
			Explanation: explanation,
			Composed:    &composed,
		}
	}

	primary := composed.Contributions[0]
	grantedSlice := capSetToSlice(composed.Granted)

	if composed.HasAll(tc.Capabilities) {
		explanation := fmt.Sprintf("allowed by rule %q in %s", primary.RulePath, primary.PolicyFile)
		if len(composed.Contributions) > 1 {
			explanation = fmt.Sprintf("%s (composed from %d policies)", explanation, len(composed.Contributions))
		}
		return Result{
			TestName: tc.Name,
			Path:     tc.Path,
			Allowed:  true,
			MatchedRule: &MatchedRule{
				PolicyFile:   primary.PolicyFile,
				RulePath:     primary.RulePath,
				Capabilities: grantedSlice,
			},
			Explanation: explanation,
			Composed:    &composed,
		}
	}

	missing := composed.Missing(tc.Capabilities)
	return Result{
		TestName: tc.Name,
		Path:     tc.Path,
		Allowed:  false,
		MatchedRule: &MatchedRule{
			PolicyFile:   primary.PolicyFile,
			RulePath:     primary.RulePath,
			Capabilities: grantedSlice,
		},
		Explanation: fmt.Sprintf("missing capabilities %v (granted: %v)", missing, grantedSlice),
		Composed:    &composed,
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

// capSetToSlice converts a capability set to a sorted slice so the output
// order is deterministic across runs (important for stable test assertions
// and reporter output).
func capSetToSlice(caps map[string]bool) []string {
	result := make([]string, 0, len(caps))
	for capability := range caps {
		result = append(result, capability)
	}
	sort.Strings(result)
	return result
}
