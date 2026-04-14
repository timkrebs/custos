package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/timkrebs/custos/pkg/evaluator"
)

// JSONSchemaVersion identifies the JSON document schema version. Consumers
// are expected to version-lock on the major number ("1") so backward-
// compatible additions (new fields on existing objects) do not break
// existing tooling. Breaking changes bump the major number.
const JSONSchemaVersion = "1.0"

// JSON renders a custos SuiteResult as a structured JSON document intended
// for programmatic consumption. It is the format to use when writing
// tooling around custos results: jq filters, custom dashboards, policy
// drift detectors, and so on.
//
// The emitted schema is stable within a major version. All array-valued
// fields are emitted as empty arrays (never null) so consumers can apply
// jq filters like '.results[] | select(.pass == false)' without having
// to handle the empty case specially. Field order is deterministic
// because the document is built from Go structs, not maps.
type JSON struct {
	Writer io.Writer

	// Pretty controls indentation. When true (the default in the
	// factory) the document is indented with two spaces per level for
	// human readability; when false the output is a single-line compact
	// document ideal for line-oriented pipelines and log ingestion.
	Pretty bool
}

// NewJSON constructs a JSON reporter writing to w. Pass nil for w to
// default to os.Stdout, matching the other reporters. Pretty-printing is
// enabled by default; callers wanting compact single-line output flip the
// Pretty field to false or use the CLI --compact flag.
func NewJSON(w io.Writer, pretty bool) *JSON {
	if w == nil {
		w = os.Stdout
	}
	return &JSON{Writer: w, Pretty: pretty}
}

// Report serializes the suite as a JSON document and writes it to the
// underlying writer, followed by a trailing newline so the output is
// line-oriented tool friendly (many Unix utilities assume a final \n).
// It returns an error when encoding or writing fails.
func (j *JSON) Report(suite evaluator.SuiteResult) error {
	doc := buildJSONDocument(suite)

	var (
		data []byte
		err  error
	)
	if j.Pretty {
		data, err = json.MarshalIndent(doc, "", "  ")
	} else {
		data, err = json.Marshal(doc)
	}
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	if _, err := j.Writer.Write(data); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}
	if _, err := io.WriteString(j.Writer, "\n"); err != nil {
		return fmt.Errorf("writing trailing newline: %w", err)
	}
	return nil
}

// buildJSONDocument converts a SuiteResult into the JSON object tree.
// Split out from Report so tests can inspect the tree without re-parsing
// the serialized output.
func buildJSONDocument(suite evaluator.SuiteResult) jsonDocument {
	suiteName := suite.Suite
	if suiteName == "" {
		suiteName = "custos"
	}

	results := make([]jsonTestResult, 0, len(suite.Results))
	for _, tr := range suite.Results {
		results = append(results, buildJSONTestResult(tr))
	}

	// Normalize warnings: emit an empty array rather than null so jq
	// consumers do not have to special-case the no-warnings path.
	warnings := suite.Warnings
	if warnings == nil {
		warnings = []string{}
	}

	return jsonDocument{
		SchemaVersion: JSONSchemaVersion,
		Suite:         suiteName,
		Duration:      suite.Duration.Seconds(),
		Summary: jsonSummary{
			Total:    len(suite.Results),
			Passed:   suite.Passed,
			Failed:   suite.Failed,
			Warnings: len(warnings),
		},
		Warnings: warnings,
		Results:  results,
	}
}

// buildJSONTestResult converts one TestResult into its JSON form,
// deriving the "actual" outcome string from the evaluator's allow bit
// and flattening the optional matched rule and composed provenance into
// their JSON shapes.
func buildJSONTestResult(tr evaluator.TestResult) jsonTestResult {
	actual := "deny"
	if tr.Result.Allowed {
		actual = "allow"
	}

	// Capabilities in the spec request are preserved as-is so consumers
	// see the exact list the user wrote, not a normalized form.
	capabilities := tr.Test.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}

	out := jsonTestResult{
		Name:         tr.Test.Name,
		Path:         tr.Test.Path,
		Capabilities: capabilities,
		Expected:     tr.Test.Expect,
		Actual:       actual,
		Pass:         tr.Pass,
		Duration:     tr.Duration.Seconds(),
		Explanation:  tr.Result.Explanation,
	}

	if mr := tr.Result.MatchedRule; mr != nil {
		caps := mr.Capabilities
		if caps == nil {
			caps = []string{}
		}
		out.MatchedRule = &jsonMatchedRule{
			PolicyFile:   mr.PolicyFile,
			RulePath:     mr.RulePath,
			Capabilities: caps,
		}
	}

	if c := tr.Result.Composed; c != nil {
		out.Composed = buildJSONComposed(c)
	}

	return out
}

// buildJSONComposed flattens the composer's multi-policy provenance into
// its JSON form. Granted is rendered as a sorted slice (deterministic
// across runs) and every slice field defaults to an empty array so jq
// consumers never encounter null where an array is expected.
func buildJSONComposed(c *evaluator.Composed) *jsonComposed {
	granted := make([]string, 0, len(c.Granted))
	for cap := range c.Granted {
		granted = append(granted, cap)
	}
	sort.Strings(granted)

	contributions := make([]jsonContribution, 0, len(c.Contributions))
	for _, contrib := range c.Contributions {
		caps := contrib.Capabilities
		if caps == nil {
			caps = []string{}
		}
		contributions = append(contributions, jsonContribution{
			PolicyFile:   contrib.PolicyFile,
			RulePath:     contrib.RulePath,
			Capabilities: caps,
			IsDeny:       contrib.IsDeny,
		})
	}

	deniedBy := make([]jsonContribution, 0, len(c.DeniedBy))
	for _, contrib := range c.DeniedBy {
		caps := contrib.Capabilities
		if caps == nil {
			caps = []string{}
		}
		deniedBy = append(deniedBy, jsonContribution{
			PolicyFile:   contrib.PolicyFile,
			RulePath:     contrib.RulePath,
			Capabilities: caps,
			IsDeny:       contrib.IsDeny,
		})
	}

	return &jsonComposed{
		Denied:        c.Denied,
		Granted:       granted,
		Contributions: contributions,
		DeniedBy:      deniedBy,
	}
}

// jsonDocument is the top-level JSON object. Field order in the serialized
// output matches the struct declaration order, so downstream diffs and
// golden snapshots are stable across runs and across Go versions.
type jsonDocument struct {
	SchemaVersion string           `json:"schema_version"`
	Suite         string           `json:"suite"`
	Duration      float64          `json:"duration_seconds"`
	Summary       jsonSummary      `json:"summary"`
	Warnings      []string         `json:"warnings"`
	Results       []jsonTestResult `json:"results"`
}

// jsonSummary aggregates the headline counts consumers typically filter or
// chart on. Warnings is included in the summary in addition to the top-
// level warnings array so jq filters like '.summary.failed > 0' work
// without having to walk the results list.
type jsonSummary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
}

// jsonTestResult is the per-test payload. MatchedRule and Composed are
// pointers so they serialize as null rather than as an empty struct when
// absent; consumers should check for null before drilling into them.
type jsonTestResult struct {
	Name         string           `json:"name"`
	Path         string           `json:"path"`
	Capabilities []string         `json:"capabilities"`
	Expected     string           `json:"expected"`
	Actual       string           `json:"actual"`
	Pass         bool             `json:"pass"`
	Duration     float64          `json:"duration_seconds"`
	Explanation  string           `json:"explanation"`
	MatchedRule  *jsonMatchedRule `json:"matched_rule"`
	Composed     *jsonComposed    `json:"composed"`
}

// jsonMatchedRule is the primary attribution: one rule in one policy. It
// mirrors evaluator.MatchedRule with snake_case field names.
type jsonMatchedRule struct {
	PolicyFile   string   `json:"policy_file"`
	RulePath     string   `json:"rule_path"`
	Capabilities []string `json:"capabilities"`
}

// jsonComposed captures the full cross-policy composition picture. Granted
// is the union of non-deny capabilities across all contributing policies,
// sorted for deterministic output. Contributions preserves input policy
// order for deterministic provenance walks.
type jsonComposed struct {
	Denied        bool               `json:"denied"`
	Granted       []string           `json:"granted"`
	Contributions []jsonContribution `json:"contributions"`
	DeniedBy      []jsonContribution `json:"denied_by"`
}

// jsonContribution is one policy's contribution to a composed decision.
// IsDeny is true when the contribution carried the Vault deny capability;
// such contributions hard-override allows from every other policy.
type jsonContribution struct {
	PolicyFile   string   `json:"policy_file"`
	RulePath     string   `json:"rule_path"`
	Capabilities []string `json:"capabilities"`
	IsDeny       bool     `json:"is_deny"`
}
