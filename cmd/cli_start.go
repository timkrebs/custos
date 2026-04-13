package cmd

import (
	"flag"
	"fmt"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/custos/pkg/evaluator"
	"github.com/timkrebs/custos/pkg/parser"
	"github.com/timkrebs/custos/pkg/spec"
)

type CliStartCmd struct{ UI cli.Ui }

func (c *CliStartCmd) Name() string     { return "test" }
func (c *CliStartCmd) Synopsis() string { return "Run custos tests" }
func (c *CliStartCmd) Help() string {
	return `Usage: custos test [options]

  Options:
	-f, --file string      Path to test spec YAML file (required)
    --vault-addr string    Vault server address (enables online mode)
    --vault-token string   Vault authentication token
    --vault-namespace str  Vault namespace (Enterprise)
    --format string        Output format: terminal (default), junit, json
    --fail-on-warn         Exit non-zero on security warnings (not just test failures)
    --timeout duration     Timeout for online Vault requests (default: 10s)
  	-v, --verbose          Show detailed evaluation trace per test
	`
}

func (c *CliStartCmd) Run(args []string) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	specFile := fs.String("file", "", "Path to test spec YAML file (required)")
	verbose := fs.Bool("verbose", false, "Show detailed evaluation trace per test")
	fs.StringVar(specFile, "f", "", "Path to test spec YAML file (required)")
	fs.BoolVar(verbose, "v", false, "Show detailed evaluation trace per test")

	if err := fs.Parse(args); err != nil {
		c.UI.Error("Error parsing flags: " + err.Error())
		return 1
	}

	if *specFile == "" {
		c.UI.Error("Missing required flag: -f / --file")
		return 1
	}

	// Load the test spec.
	s, err := spec.LoadFile(*specFile)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error loading spec: %s", err))
		return 1
	}

	// Parse all referenced policies.
	var policies []parser.Policy
	for _, ref := range s.Policies {
		p, err := parser.ParsePolicyFile(ref.Path)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Error parsing policy %s: %s", ref.Path, err))
			return 1
		}
		policies = append(policies, *p)
	}

	// Run evaluation.
	suite := evaluator.EvaluateSuite(policies, s)

	// Print results.
	printSuiteResult(c.UI, suite, *verbose)

	if suite.Failed > 0 {
		return 1
	}
	return 0
}

func printSuiteResult(ui cli.Ui, suite evaluator.SuiteResult, verbose bool) {
	testCount := len(suite.Results)
	policyCount := countPolicies(suite)
	ui.Output(fmt.Sprintf("\n=== SUITE: %s (%d %s, %d %s)\n",
		suite.Suite,
		policyCount, pluralize("policy", "policies", policyCount),
		testCount, pluralize("test", "tests", testCount),
	))

	for _, tr := range suite.Results {
		path := tr.Test.Path
		if tr.Pass {
			ui.Info(fmt.Sprintf("--- PASS: %-40s (%s)", tr.Test.Name, path))
		} else {
			ui.Error(fmt.Sprintf("--- FAIL: %-40s (%s)", tr.Test.Name, path))
			expected := tr.Test.Expect
			got := "deny"
			if tr.Result.Allowed {
				got = "allow"
			}
			detail := fmt.Sprintf("         expected: %s, got: %s", expected, got)
			if tr.Result.MatchedRule != nil {
				detail += fmt.Sprintf(" (matched %s in %s)",
					tr.Result.MatchedRule.RulePath,
					tr.Result.MatchedRule.PolicyFile)
			}
			ui.Error(detail)
		}
		if verbose {
			ui.Output(fmt.Sprintf("         %s", tr.Result.Explanation))
		}
	}

	ui.Output("")
	if suite.Failed > 0 {
		ui.Error(fmt.Sprintf("=== RESULTS: %d passed, %d failed", suite.Passed, suite.Failed))
		ui.Error("FAIL")
	} else {
		ui.Info(fmt.Sprintf("=== RESULTS: %d passed, %d failed", suite.Passed, suite.Failed))
		ui.Output("ok")
	}
}

func countPolicies(suite evaluator.SuiteResult) int {
	seen := make(map[string]bool)
	for _, tr := range suite.Results {
		if tr.Result.MatchedRule != nil {
			seen[tr.Result.MatchedRule.PolicyFile] = true
		}
	}
	if len(seen) == 0 {
		return 0
	}
	return len(seen)
}

func pluralize(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}
