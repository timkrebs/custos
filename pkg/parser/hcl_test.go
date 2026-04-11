package parser

import (
	"fmt"
	"testing"
)

func TestParseHCLConfig(t *testing.T) {
	config := ParseHCLConfig("testdata/config.hcl")
	fmt.Printf("Parsed HCL Config: %+v\n", config)

}
