package cmd

import (
	cli "github.com/timkrebs/gocli"
)

type CliStartCmd struct{ Ui cli.Ui }

func (c *CliStartCmd) Name() string     { return "test" }
func (c *CliStartCmd) Synopsis() string { return "Run vaultspec tests" }
func (c *CliStartCmd) Help() string {
	return `Usage: vaultspec test [options]

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
	//fs := flag.NewFlagSet("test", flag.ContinueOnError)
	//specFile := fs.String("file", "", "Path to test spec YAML file (required)")
	//vaultAddr := fs.String("vault-addr", "", "Vault server address (enables online mode)")
	//vaultToken := fs.String("vault-token", "", "Vault authentication token")
	//vaultNamespace := fs.String("vault-namespace", "", "Vault namespace (Enterprise)")
	//outputFormat := fs.String("format", "terminal", "Output format: terminal (default), junit, json")
	//failOnWarn := fs.Bool("fail-on-warn", false, "Exit non-zero on security warnings (not just test failures)")
	//timeout := fs.Duration("timeout", 10, "Timeout for online Vault requests (default: 10s)")
	//verbose := fs.Bool("verbose", false, "Show detailed evaluation trace per test")
	//fs.StringVar(specFile, "f", "", "Path to test spec YAML file (required)")
	//fs.BoolVar(verbose, "v", false, "Show detailed evaluation trace per test")
	//
	//if err := fs.Parse(args); err != nil {
	//	c.Ui.Error("Error parsing flags: " + err.Error())
	//	return 1
	//}
	return 0
}
