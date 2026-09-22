package main

import (
	_ "embed"

	"amdl/internal/config"
	"amdl/internal/updater"
)

//go:embed config.yaml.example
var defaultConfigExample string

func init() {
	config.DefaultConfigTemplate = defaultConfigExample
	updater.DefaultConfigExample = defaultConfigExample
}

