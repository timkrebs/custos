package reporter

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/timkrebs/custos/pkg/evaluator"
)

// JUnit renders a custos SuiteResult as JUnit XML for consumption by CI
// test reporters such as dorny/test-reporter, Jenkins, GitLab, and GitHub
// Actions test dashboards. The emitted schema is the de facto "Ant JUnit"
// format accepted by every mainstream CI tool:
//
//	<testsuites>
//	  <testsuite name="..." tests="..." failures="..." errors="..." time="..." timestamp="...">
//	    <testcase name="..." classname="..." time="...">
//	      <failure message="..." type="AssertionError">...details...</failure>
//	    </testcase>
//	  </testsuite>
//	</testsuites>
//
// The reporter writes a UTF-8 XML declaration followed by an indented
// document so the output is both machine-parseable and human-readable.
type JUnit struct {
	Writer io.Writer

	// Now overrides the timestamp source. Tests set this to a fixed time
	// so the emitted XML is deterministic; production code leaves it nil
	// and falls back to time.Now.
	Now func() time.Time
}

// NewJUnit constructs a JUnit reporter writing to the given writer. Pass
// nil to write to os.Stdout, matching NewTerminal's convention.
func NewJUnit(w io.Writer) *JUnit {
	if w == nil {
		w = os.Stdout
	}
	return &JUnit{Writer: w}
}

// Report marshals the suite result as JUnit XML and writes it to the
// underlying writer. It returns an error if XML marshaling fails or the
// writer rejects the payload. The output is a complete document including
// the XML declaration; callers can redirect it directly to a file.
func (j *JUnit) Report(suite evaluator.SuiteResult) error {
	doc := j.buildDocument(suite)

	if _, err := io.WriteString(j.Writer, xml.Header); err != nil {
		return fmt.Errorf("writing XML header: %w", err)
	}

	enc := xml.NewEncoder(j.Writer)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encoding JUnit XML: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return fmt.Errorf("flushing JUnit XML: %w", err)
	}
	if _, err := io.WriteString(j.Writer, "\n"); err != nil {
		return fmt.Errorf("writing trailing newline: %w", err)
	}
	return nil
}

// buildDocument converts a SuiteResult into the JUnit XML object tree.
// Split out from Report so tests can inspect the tree without parsing XML.
func (j *JUnit) buildDocument(suite evaluator.SuiteResult) junitTestsuites {
	now := time.Now
	if j.Now != nil {
		now = j.Now
	}
	timestamp := now().UTC().Format("2006-01-02T15:04:05Z")

	suiteName := suite.Suite
	if suiteName == "" {
		suiteName = "custos"
	}

	testcases := make([]junitTestcase, 0, len(suite.Results))
	for _, tr := range suite.Results {
		tc := junitTestcase{
			Name:      tr.Test.Name,
			Classname: suiteName,
			Time:      formatSeconds(tr.Duration),
		}
		if !tr.Pass {
			tc.Failure = buildFailure(tr)
		}
		testcases = append(testcases, tc)
	}

	testsuite := junitTestsuite{
		Name:      suiteName,
		Tests:     len(suite.Results),
		Failures:  suite.Failed,
		Errors:    0,
		Skipped:   0,
		Time:      formatSeconds(suite.Duration),
		Timestamp: timestamp,
		Testcases: testcases,
	}

	return junitTestsuites{
		Name:     "custos",
		Tests:    len(suite.Results),
		Failures: suite.Failed,
		Errors:   0,
		Time:     formatSeconds(suite.Duration),
		Suites:   []junitTestsuite{testsuite},
	}
}

// buildFailure produces the <failure> child for a failing testcase. The
// short message attribute is designed to fit in CI dashboards at a glance;
// the longer chardata body carries expected/got, path, capabilities, the
// primary matched rule, and multi-policy provenance when available.
func buildFailure(tr evaluator.TestResult) *junitFailure {
	expected := tr.Test.Expect
	got := "deny"
	if tr.Result.Allowed {
		got = "allow"
	}

	message := fmt.Sprintf("expected %s, got %s at path %s", expected, got, tr.Test.Path)

	var body strings.Builder
	fmt.Fprintf(&body, "Expected: %s\n", expected)
	fmt.Fprintf(&body, "Got:      %s\n", got)
	fmt.Fprintf(&body, "Path:     %s\n", tr.Test.Path)
	fmt.Fprintf(&body, "Capabilities: %v\n", tr.Test.Capabilities)

	if tr.Result.MatchedRule != nil {
		policyName := strings.TrimSuffix(filepath.Base(tr.Result.MatchedRule.PolicyFile), ".hcl")
		fmt.Fprintf(&body, "Matched rule: %q in policy %q\n", tr.Result.MatchedRule.RulePath, policyName)
	}

	if tr.Result.Explanation != "" {
		fmt.Fprintf(&body, "Explanation: %s\n", tr.Result.Explanation)
	}

	// Surface multi-policy provenance so CI drill-downs show which policy
	// granted and which policy denied, matching the terminal reporter.
	if c := tr.Result.Composed; c != nil && len(c.Contributions) >= 2 {
		fmt.Fprintln(&body, "Contributions:")
		for _, contrib := range c.Contributions {
			name := strings.TrimSuffix(filepath.Base(contrib.PolicyFile), ".hcl")
			if contrib.IsDeny {
				fmt.Fprintf(&body, "  - %s (%s) DENIED\n", name, contrib.RulePath)
				continue
			}
			fmt.Fprintf(&body, "  - %s (%s) granted %v\n", name, contrib.RulePath, nonDenyCapabilities(contrib.Capabilities))
		}
	}

	return &junitFailure{
		Message: message,
		Type:    "AssertionError",
		Content: body.String(),
	}
}

// formatSeconds renders a duration as a fixed-point seconds string with
// microsecond precision. The format matches what mainstream JUnit parsers
// expect: a plain float without units and without scientific notation.
// Negative and zero durations render as "0.000000".
func formatSeconds(d time.Duration) string {
	if d <= 0 {
		return "0.000000"
	}
	return fmt.Sprintf("%.6f", d.Seconds())
}

// junitTestsuites is the document root. Using a dedicated root (as opposed
// to emitting a bare <testsuite>) is the format accepted by every CI tool
// we target, including dorny/test-reporter's Ant-JUnit parser.
type junitTestsuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Name     string           `xml:"name,attr"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Time     string           `xml:"time,attr"`
	Suites   []junitTestsuite `xml:"testsuite"`
}

// junitTestsuite maps to one custos suite. custos emits exactly one
// testsuite per run today because the test command evaluates a single
// spec file; future multi-spec runs would append additional entries.
type junitTestsuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      string          `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	Testcases []junitTestcase `xml:"testcase"`
}

// junitTestcase maps to one spec test assertion. The classname attribute is
// set to the suite name so CI dashboards that group by classname render a
// usable tree view even for flat custos specs.
type junitTestcase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

// junitFailure is the <failure> child of a failing testcase. The Message
// attribute appears in CI dashboard headers; the Content chardata body is
// shown on expanded failure views. Errors (infrastructure failures) are
// not currently modeled since custos reports setup failures through its
// exit code rather than per-case.
type junitFailure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}
