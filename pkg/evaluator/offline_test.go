package evaluator

import (
	"testing"

	"github.com/timkrebs/custos/pkg/parser"
	"github.com/timkrebs/custos/pkg/spec"
)

// ---------------------------------------------------------------------------
// matchPath tests
// ---------------------------------------------------------------------------

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name        string
		rulePath    string
		requestPath string
		wantMatch   bool
		wantType    matchType
	}{
		// Exact matches
		{
			name:        "exact match",
			rulePath:    "secret/foo",
			requestPath: "secret/foo",
			wantMatch:   true,
			wantType:    matchExact,
		},
		{
			name:        "exact miss longer name",
			rulePath:    "secret/foo",
			requestPath: "secret/food",
			wantMatch:   false,
		},
		{
			name:        "exact miss deeper path",
			rulePath:    "secret/foo",
			requestPath: "secret/foo/bar",
			wantMatch:   false,
		},
		{
			name:        "exact miss shorter path",
			rulePath:    "secret/foo/bar",
			requestPath: "secret/foo",
			wantMatch:   false,
		},

		// Prefix glob (trailing *)
		{
			name:        "prefix glob one level",
			rulePath:    "secret/bar/*",
			requestPath: "secret/bar/zip",
			wantMatch:   true,
			wantType:    matchPrefix,
		},
		{
			name:        "prefix glob multiple levels",
			rulePath:    "secret/bar/*",
			requestPath: "secret/bar/zip/zap",
			wantMatch:   true,
			wantType:    matchPrefix,
		},
		{
			name:        "prefix glob exact prefix boundary",
			rulePath:    "secret/bar/*",
			requestPath: "secret/bar/",
			wantMatch:   true,
			wantType:    matchPrefix,
		},
		{
			name:        "prefix glob miss different prefix",
			rulePath:    "secret/bar/*",
			requestPath: "secret/bars/zip",
			wantMatch:   false,
		},
		{
			name:        "inline prefix glob",
			rulePath:    "secret/zip-*",
			requestPath: "secret/zip-zap",
			wantMatch:   true,
			wantType:    matchPrefix,
		},
		{
			name:        "inline prefix glob deeper",
			rulePath:    "secret/zip-*",
			requestPath: "secret/zip-zap/baz",
			wantMatch:   true,
			wantType:    matchPrefix,
		},
		{
			name:        "inline prefix glob miss",
			rulePath:    "secret/zip-*",
			requestPath: "secret/zop-zap",
			wantMatch:   false,
		},
		{
			name:        "root wildcard matches everything",
			rulePath:    "*",
			requestPath: "anything/at/all",
			wantMatch:   true,
			wantType:    matchPrefix,
		},
		{
			name:        "root wildcard matches single segment",
			rulePath:    "*",
			requestPath: "foo",
			wantMatch:   true,
			wantType:    matchPrefix,
		},

		// Segment glob (+)
		{
			name:        "plus matches one segment",
			rulePath:    "secret/+/config",
			requestPath: "secret/team-a/config",
			wantMatch:   true,
			wantType:    matchGlob,
		},
		{
			name:        "plus does not match zero segments",
			rulePath:    "secret/+/config",
			requestPath: "secret/config",
			wantMatch:   false,
		},
		{
			name:        "plus does not match two segments",
			rulePath:    "secret/+/config",
			requestPath: "secret/a/b/config",
			wantMatch:   false,
		},
		{
			name:        "double plus matches two segments",
			rulePath:    "secret/+/+/config",
			requestPath: "secret/a/b/config",
			wantMatch:   true,
			wantType:    matchGlob,
		},
		{
			name:        "plus at start",
			rulePath:    "+/data/config",
			requestPath: "secret/data/config",
			wantMatch:   true,
			wantType:    matchGlob,
		},
		{
			name:        "plus miss wrong literal",
			rulePath:    "secret/+/config",
			requestPath: "secret/team-a/settings",
			wantMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, mtype, _ := matchPath(tt.rulePath, tt.requestPath)
			if matched != tt.wantMatch {
				t.Errorf("matchPath(%q, %q) matched = %v, want %v",
					tt.rulePath, tt.requestPath, matched, tt.wantMatch)
			}
			if matched && mtype != tt.wantType {
				t.Errorf("matchPath(%q, %q) type = %v, want %v",
					tt.rulePath, tt.requestPath, mtype, tt.wantType)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// selectBestMatches tests
// ---------------------------------------------------------------------------

func TestSelectBestMatches(t *testing.T) {
	t.Run("exact beats glob and prefix", func(t *testing.T) {
		candidates := []matchCandidate{
			{policyFile: "a.hcl", mtype: matchPrefix, specificity: 10},
			{policyFile: "b.hcl", mtype: matchGlob, specificity: 12},
			{policyFile: "c.hcl", mtype: matchExact, specificity: 14},
		}
		best := selectBestMatches(candidates)
		if len(best) != 1 || best[0].policyFile != "c.hcl" {
			t.Fatalf("expected exact match c.hcl, got %+v", best)
		}
	})

	t.Run("glob beats prefix", func(t *testing.T) {
		candidates := []matchCandidate{
			{policyFile: "a.hcl", mtype: matchPrefix, specificity: 15},
			{policyFile: "b.hcl", mtype: matchGlob, specificity: 10},
		}
		best := selectBestMatches(candidates)
		if len(best) != 1 || best[0].policyFile != "b.hcl" {
			t.Fatalf("expected glob match b.hcl, got %+v", best)
		}
	})

	t.Run("higher specificity wins within same type", func(t *testing.T) {
		candidates := []matchCandidate{
			{policyFile: "short.hcl", mtype: matchPrefix, specificity: 5},
			{policyFile: "long.hcl", mtype: matchPrefix, specificity: 15},
		}
		best := selectBestMatches(candidates)
		if len(best) != 1 || best[0].policyFile != "long.hcl" {
			t.Fatalf("expected longer prefix long.hcl, got %+v", best)
		}
	})

	t.Run("multiple matches at same tier are all returned", func(t *testing.T) {
		candidates := []matchCandidate{
			{policyFile: "a.hcl", mtype: matchExact, specificity: 10},
			{policyFile: "b.hcl", mtype: matchExact, specificity: 10},
		}
		best := selectBestMatches(candidates)
		if len(best) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(best))
		}
	})
}

// ---------------------------------------------------------------------------
// Evaluate tests
// ---------------------------------------------------------------------------

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name      string
		policies  []parser.Policy
		tc        spec.TestCase
		wantAllow bool
	}{
		{
			name: "single policy exact match capability granted",
			policies: []parser.Policy{{
				Filepath: "policy.hcl",
				Paths: []parser.PathRule{{
					Path:         "secret/foo",
					Capabilities: []string{"read", "list"},
				}},
			}},
			tc:        spec.TestCase{Name: "t1", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
			wantAllow: true,
		},
		{
			name: "single policy capability not granted",
			policies: []parser.Policy{{
				Filepath: "policy.hcl",
				Paths: []parser.PathRule{{
					Path:         "secret/foo",
					Capabilities: []string{"read"},
				}},
			}},
			tc:        spec.TestCase{Name: "t2", Path: "secret/foo", Capabilities: []string{"create"}, Expect: "deny"},
			wantAllow: false,
		},
		{
			name:      "no matching rule implicit deny",
			policies:  []parser.Policy{{Filepath: "policy.hcl", Paths: []parser.PathRule{{Path: "secret/other", Capabilities: []string{"read"}}}}},
			tc:        spec.TestCase{Name: "t3", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "deny"},
			wantAllow: false,
		},
		{
			name: "deny capability overrides all",
			policies: []parser.Policy{{
				Filepath: "policy.hcl",
				Paths: []parser.PathRule{{
					Path:         "secret/foo",
					Capabilities: []string{"deny"},
				}},
			}},
			tc:        spec.TestCase{Name: "t4", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "deny"},
			wantAllow: false,
		},
		{
			name: "two policies same path different caps union allows",
			policies: []parser.Policy{
				{Filepath: "a.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"read"}}}},
				{Filepath: "b.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"create"}}}},
			},
			tc:        spec.TestCase{Name: "t5", Path: "secret/foo", Capabilities: []string{"read", "create"}, Expect: "allow"},
			wantAllow: true,
		},
		{
			name: "two policies one allows one denies deny wins",
			policies: []parser.Policy{
				{Filepath: "a.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"read", "list"}}}},
				{Filepath: "b.hcl", Paths: []parser.PathRule{{Path: "secret/foo", Capabilities: []string{"deny"}}}},
			},
			tc:        spec.TestCase{Name: "t6", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "deny"},
			wantAllow: false,
		},
		{
			name: "multiple capabilities all present",
			policies: []parser.Policy{{
				Filepath: "policy.hcl",
				Paths: []parser.PathRule{{
					Path:         "secret/foo",
					Capabilities: []string{"read", "create", "update"},
				}},
			}},
			tc:        spec.TestCase{Name: "t7", Path: "secret/foo", Capabilities: []string{"read", "create"}, Expect: "allow"},
			wantAllow: true,
		},
		{
			name: "multiple capabilities one missing",
			policies: []parser.Policy{{
				Filepath: "policy.hcl",
				Paths: []parser.PathRule{{
					Path:         "secret/foo",
					Capabilities: []string{"read"},
				}},
			}},
			tc:        spec.TestCase{Name: "t8", Path: "secret/foo", Capabilities: []string{"read", "create"}, Expect: "deny"},
			wantAllow: false,
		},
		{
			name: "exact match beats glob on same path",
			policies: []parser.Policy{{
				Filepath: "policy.hcl",
				Paths: []parser.PathRule{
					{Path: "secret/*", Capabilities: []string{"read", "create"}},
					{Path: "secret/foo", Capabilities: []string{"read"}},
				},
			}},
			tc:        spec.TestCase{Name: "t9", Path: "secret/foo", Capabilities: []string{"create"}, Expect: "deny"},
			wantAllow: false, // exact match wins, only has "read"
		},
		{
			name:      "empty policy list implicit deny",
			policies:  nil,
			tc:        spec.TestCase{Name: "t10", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "deny"},
			wantAllow: false,
		},
		{
			name: "prefix glob match grants capability",
			policies: []parser.Policy{{
				Filepath: "policy.hcl",
				Paths: []parser.PathRule{{
					Path:         "secret/data/*",
					Capabilities: []string{"read", "list"},
				}},
			}},
			tc:        spec.TestCase{Name: "t11", Path: "secret/data/myapp/config", Capabilities: []string{"read"}, Expect: "allow"},
			wantAllow: true,
		},
		{
			name: "longer prefix beats shorter prefix",
			policies: []parser.Policy{{
				Filepath: "policy.hcl",
				Paths: []parser.PathRule{
					{Path: "secret/*", Capabilities: []string{"read"}},
					{Path: "secret/data/team-*", Capabilities: []string{"read", "create"}},
				},
			}},
			tc:        spec.TestCase{Name: "t12", Path: "secret/data/team-a/config", Capabilities: []string{"create"}, Expect: "allow"},
			wantAllow: true, // longer prefix wins, has "create"
		},
		{
			name: "deny on glob overrides allow on same glob",
			policies: []parser.Policy{
				{Filepath: "allow.hcl", Paths: []parser.PathRule{{Path: "secret/billing/*", Capabilities: []string{"read"}}}},
				{Filepath: "deny.hcl", Paths: []parser.PathRule{{Path: "secret/billing/*", Capabilities: []string{"deny"}}}},
			},
			tc:        spec.TestCase{Name: "t13", Path: "secret/billing/api-key", Capabilities: []string{"read"}, Expect: "deny"},
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Evaluate(tt.policies, tt.tc)
			if result.Allowed != tt.wantAllow {
				t.Errorf("Evaluate() allowed = %v, want %v\n  explanation: %s",
					result.Allowed, tt.wantAllow, result.Explanation)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Evaluate result metadata tests
// ---------------------------------------------------------------------------

func TestEvaluate_Metadata(t *testing.T) {
	t.Run("implicit deny has nil matched rule", func(t *testing.T) {
		result := Evaluate(nil, spec.TestCase{Name: "t", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "deny"})
		if result.MatchedRule != nil {
			t.Errorf("expected nil MatchedRule for implicit deny, got %+v", result.MatchedRule)
		}
		if result.Explanation == "" {
			t.Error("expected non-empty explanation")
		}
	})

	t.Run("matched rule contains policy file and rule path", func(t *testing.T) {
		policies := []parser.Policy{{
			Filepath: "policies/app.hcl",
			Paths:    []parser.PathRule{{Path: "secret/app/*", Capabilities: []string{"read"}}},
		}}
		result := Evaluate(policies, spec.TestCase{Name: "t", Path: "secret/app/config", Capabilities: []string{"read"}, Expect: "allow"})
		if result.MatchedRule == nil {
			t.Fatal("expected non-nil MatchedRule")
		}
		if result.MatchedRule.PolicyFile != "policies/app.hcl" {
			t.Errorf("PolicyFile = %q, want %q", result.MatchedRule.PolicyFile, "policies/app.hcl")
		}
		if result.MatchedRule.RulePath != "secret/app/*" {
			t.Errorf("RulePath = %q, want %q", result.MatchedRule.RulePath, "secret/app/*")
		}
	})
}

// ---------------------------------------------------------------------------
// EvaluateSuite tests
// ---------------------------------------------------------------------------

func TestEvaluateSuite(t *testing.T) {
	policies := []parser.Policy{{
		Filepath: "policy.hcl",
		Paths: []parser.PathRule{
			{Path: "secret/foo", Capabilities: []string{"read"}},
			{Path: "secret/bar/*", Capabilities: []string{"deny"}},
		},
	}}

	s := &spec.Spec{
		Suite: "test-suite",
		Tests: []spec.TestCase{
			{Name: "allow read", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
			{Name: "deny bar", Path: "secret/bar/baz", Capabilities: []string{"read"}, Expect: "deny"},
			{Name: "implicit deny", Path: "secret/other", Capabilities: []string{"read"}, Expect: "deny"},
			{Name: "fails expectation", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "deny"},
		},
	}

	sr := EvaluateSuite(policies, s)

	if sr.Suite != "test-suite" {
		t.Errorf("Suite = %q, want %q", sr.Suite, "test-suite")
	}
	if sr.Passed != 3 {
		t.Errorf("Passed = %d, want 3", sr.Passed)
	}
	if sr.Failed != 1 {
		t.Errorf("Failed = %d, want 1", sr.Failed)
	}
	if len(sr.Results) != 4 {
		t.Fatalf("Results count = %d, want 4", len(sr.Results))
	}

	// Check individual results
	if !sr.Results[0].Pass {
		t.Error("test 0 (allow read) should pass")
	}
	if !sr.Results[1].Pass {
		t.Error("test 1 (deny bar) should pass")
	}
	if !sr.Results[2].Pass {
		t.Error("test 2 (implicit deny) should pass")
	}
	if sr.Results[3].Pass {
		t.Error("test 3 (fails expectation) should fail")
	}
}
