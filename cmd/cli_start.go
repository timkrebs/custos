package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/custos/pkg/evaluator"
	"github.com/timkrebs/custos/pkg/parser"
	"github.com/timkrebs/custos/pkg/reporter"
	"github.com/timkrebs/custos/pkg/spec"
)

// CliStartCmd implements the "custos test" command.
type CliStartCmd struct {
	UI     cli.Ui
	Writer io.Writer // Output destination for the reporter. Defaults to os.Stdout.
}

func (c *CliStartCmd) Name() string     { return "test" }
func (c *CliStartCmd) Synopsis() string { return "Run test assertions against Vault policies" }
func (c *CliStartCmd) Help() string {
	return `Usage: custos test [options]

  Run test assertions against one or more Vault HCL policy files.
  Each test case in the spec file is evaluated against the loaded
  policies and the result is compared to the expected outcome.

  Options:
    -f, --file string      Path to test spec YAML file (required)
    --format string        Output format: terminal (default), junit, or json
    --compact              Emit compact single-line output (json only)
    --fail-on-warn         Exit non-zero on security warnings too
    -v, --verbose          Show detailed evaluation trace per test

  JUnit XML output is intended for CI systems such as GitHub Actions
  (dorny/test-reporter), GitLab, and Jenkins. Redirect stdout to a
  file and feed it to your reporter of choice:

    custos test -f spec.yaml --format=junit > results.xml

  JSON output is intended for programmatic consumption (jq filters,
  custom dashboards, policy drift detectors). The schema is stable
  within a major version:

    custos test -f spec.yaml --format=json > results.json
    custos test -f spec.yaml --format=json --compact | jq '.summary'
`
}

func (c *CliStartCmd) Run(args []string) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	specFile := fs.String("file", "", "Path to test spec YAML file (required)")
	failOnWarn := fs.Bool("fail-on-warn", false, "Exit non-zero on security warnings")
	verbose := fs.Bool("verbose", false, "Show detailed evaluation trace per test")
	format := fs.String("format", string(reporter.FormatTerminal), "Output format: terminal, junit, or json")
	compact := fs.Bool("compact", false, "Emit compact single-line output (json format only)")
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

	// Report results. The factory validates the --format value and
	// returns a descriptive error when the user passes an unknown format.
	w := c.Writer
	if w == nil {
		w = os.Stdout
	}
	rep, err := reporter.New(reporter.Format(*format), w, *verbose)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error selecting reporter: %s", err))
		return 1
	}
	// --compact is a JSON-only affordance: flip the reporter to
	// single-line mode after construction. Ignored for other formats so
	// users passing --compact accidentally with --format=terminal get a
	// harmless no-op instead of an error.
	if j, ok := rep.(*reporter.JSON); ok && *compact {
		j.Pretty = false
	}
	if err := rep.Report(suite); err != nil {
		c.UI.Error(fmt.Sprintf("Error writing report: %s", err))
		return 1
	}

	// Exit code logic.
	if suite.Failed > 0 {
		return 1
	}
	if *failOnWarn && len(suite.Warnings) > 0 {
		return 1
	}
	return 0
}
