package parser

import (
	"log"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Config struct {
}

type PolicyConfig struct {
}

type ProcessConfig struct {
	Type    string   `hcl:"type,label"`
	Command []string `hcl:"command"`
}

func ParseHCLConfig(path string) Config {
	var config Config

	if path == "" {
		log.Fatal("No configuration file path provided")
	} else {
		log.Printf("Loading configuration from %s", path)
	}
	err := hclsimple.DecodeFile(path, nil, &config)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}
	log.Printf("Configuration is %#v", config)

	return config
}
