package main

import (
	_ "embed"

	"amdl/internal/updater"
)

//go:embed config.yaml.example
var defaultConfigExample string

func init() {
	updater.DefaultConfigExample = defaultConfigExample
}
