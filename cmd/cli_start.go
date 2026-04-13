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

	// Report results.
	w := c.Writer
	if w == nil {
		w = os.Stdout
	}
	rep := reporter.NewTerminal(w, *verbose)
	rep.Report(suite)

	if suite.Failed > 0 {
		return 1
	}
	return 0
}
