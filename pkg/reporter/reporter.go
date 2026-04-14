// Package reporter renders the outcome of a custos test run in a variety of
// formats. Human-oriented output uses the terminal reporter; machine-
// oriented output for CI systems uses the JUnit XML reporter. New formats
// (JSON, SARIF) can be added by implementing the Reporter interface and
// wiring them into New.
package reporter

import (
	"fmt"
	"io"

	"github.com/timkrebs/custos/pkg/evaluator"
)

// Format identifies a supported output format. It is a typed string so the
// CLI flag parser can validate user input with clear error messages.
type Format string

const (
	// FormatTerminal produces the default colored human-readable output
	// intended for interactive shells and local development.
	FormatTerminal Format = "terminal"

	// FormatJUnit produces JUnit XML intended for CI test reporters such
	// as dorny/test-reporter, Jenkins, and GitLab. The output contains
	// only XML so it can be safely redirected to a file.
	FormatJUnit Format = "junit"

	// FormatJSON produces a structured JSON document intended for
	// programmatic consumption: jq filters, custom dashboards, policy
	// drift detectors, and similar tooling. The schema is stable
	// within a major version (see JSONSchemaVersion).
	FormatJSON Format = "json"
)

// SupportedFormats lists every format New accepts, in the order they should
// appear in user-facing error messages and documentation.
var SupportedFormats = []Format{FormatTerminal, FormatJUnit, FormatJSON}

// Reporter renders a completed SuiteResult to an underlying writer. Report
// returns a non-nil error only for encoding failures that cannot produce a
// usable output (for example, XML marshaling errors). The terminal reporter
// always returns nil because fmt.Fprintf swallows write errors by design.
type Reporter interface {
	Report(suite evaluator.SuiteResult) error
}

// New constructs the reporter for the given format. The verbose flag is
// honored only by reporters that distinguish verbose output (currently just
// the terminal reporter). An unknown format returns a descriptive error
// listing the supported values so CLI users get actionable feedback.
func New(format Format, w io.Writer, verbose bool) (Reporter, error) {
	switch format {
	case FormatTerminal, "":
		return NewTerminal(w, verbose), nil
	case FormatJUnit:
		return NewJUnit(w), nil
	case FormatJSON:
		// Pretty-print by default. Callers wanting compact output
		// type-assert to *JSON and flip the Pretty field; the CLI
		// does this behind --compact.
		return NewJSON(w, true), nil
	default:
		return nil, fmt.Errorf("unknown reporter format %q (supported: %s)", format, joinFormats(SupportedFormats))
	}
}

// joinFormats renders the supported format list as a comma-separated string
// for error messages.
func joinFormats(formats []Format) string {
	out := ""
	for i, f := range formats {
		if i > 0 {
			out += ", "
		}
		out += string(f)
	}
	return out
}
