package parser

import (
	"testing"
)

func TestParseHCLConfig(t *testing.T) {
	config := ParseHCLConfig("testdata/config.hcl")

	if config.IOMode != "async" {
		t.Errorf("Expected IOMode to be 'async', got '%s'", config.IOMode)
	}

}
