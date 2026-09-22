package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// DefaultConfigTemplate holds the embedded content of config.yaml.example,
// populated at runtime from package main via init().
var DefaultConfigTemplate string

type bytesProvider []byte

func (b bytesProvider) ReadBytes() ([]byte, error) {
	return []byte(b), nil
}

func (b bytesProvider) Read() (map[string]any, error) {
	return nil, nil
}

func getExampleContent(exampleFile string) string {
	if fileExists(exampleFile) {
		if data, err := os.ReadFile(exampleFile); err == nil {
			return string(data)
		}
	}
	return DefaultConfigTemplate
}

// flagKeyMap maps flat command line flag names to nested configuration keys.
var flagKeyMap = map[string]string{
	"alac-max":      "media.alac-max",
	"atmos-max":     "media.atmos-max",
	"aac-type":      "media.aac-type",
	"mv-audio-type": "media.mv.audio-type",
	"mv-max":        "media.mv.max",
	"lite-server":   "general.lite-server",
}

// LoadOptions controls the configuration loading behavior.
type LoadOptions struct {
	// ConfigFile specifies the path to the user config file (default: "config.yaml").
	ConfigFile string
	// ExampleFile specifies the path to the example/default config file (default: "config.yaml.example").
	ExampleFile string
	// FlagSet optionally passes command line flags to override configuration values.
	FlagSet *pflag.FlagSet
	// DisableMissingWarnings disables warnings when config.yaml has missing fields.
	DisableMissingWarnings bool
}

// Load loads configuration using koanf with layered precedence:
// 1. In-code defaults (Default())
// 2. Example file or embedded template (config.yaml.example or DefaultConfigTemplate)
// 3. User configuration file (config.yaml or custom path)
// 4. Command line flags (*pflag.FlagSet via posflag)
func Load(opts LoadOptions) (*Config, error) {
	configFile := opts.ConfigFile
	if configFile == "" {
		configFile = "config.yaml"
	}
	exampleFile := opts.ExampleFile
	if exampleFile == "" {
		exampleFile = "config.yaml.example"
	}

	userFileExists := fileExists(configFile)
	exampleFileExists := fileExists(exampleFile)

	// Automatically create default config file if it does not exist.
	if !userFileExists && (opts.ConfigFile == "" || filepath.Base(configFile) == "config.yaml") {
		templateContent := getExampleContent(exampleFile)
		if templateContent != "" {
			if dir := filepath.Dir(configFile); dir != "" && dir != "." {
				_ = os.MkdirAll(dir, 0755)
			}
			if err := os.WriteFile(configFile, []byte(templateContent), 0644); err == nil {
				userFileExists = true
				if !opts.DisableMissingWarnings {
					fmt.Printf("Config file %s not found, created from default template.\n", configFile)
				}
			}
		}
	}

	// If a custom config file path was specified, it must exist.
	if opts.ConfigFile != "" && !userFileExists {
		return nil, fmt.Errorf("config file not found: %s", configFile)
	}

	// Neither user config nor example file/embedded template exists.
	if !userFileExists && !exampleFileExists && DefaultConfigTemplate == "" {
		return nil, errors.New("config file not found: provide " + configFile)
	}

	k := koanf.New(".")

	// Layer 1: In-code defaults
	if err := k.Load(structs.Provider(Default(), "koanf"), nil); err != nil {
		return nil, fmt.Errorf("load default config: %w", err)
	}

	// Layer 2: Example file or embedded template (if available)
	var exampleK *koanf.Koanf
	exampleContent := getExampleContent(exampleFile)
	if exampleContent != "" {
		exampleBytes := []byte(exampleContent)
		exampleK = koanf.New(".")
		if err := exampleK.Load(bytesProvider(exampleBytes), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("parse example config: %w", err)
		}
		if err := k.Load(bytesProvider(exampleBytes), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load example config: %w", err)
		}
	} else if userFileExists && !opts.DisableMissingWarnings {
		fmt.Printf("Warning: %s not found, using %s only\n", exampleFile, configFile)
	}

	// Layer 3: User configuration file
	if userFileExists {
		userK := koanf.New(".")
		if err := userK.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("parse %s: %w", configFile, err)
		}
		if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load %s: %w", configFile, err)
		}

		if !opts.DisableMissingWarnings {
			var refKeys []string
			if exampleK != nil {
				refKeys = exampleK.Keys()
			} else {
				refK := koanf.New(".")
				_ = refK.Load(structs.Provider(Default(), "koanf"), nil)
				refKeys = refK.Keys()
			}

			var missing []string
			for _, rk := range refKeys {
				if !userK.Exists(rk) {
					missing = append(missing, rk)
				}
			}
			sort.Strings(missing)
			if len(missing) > 0 {
				fmt.Printf("Warning: %s is missing fields, using defaults for them.\n", configFile)
				fmt.Println("  Missing fields:", strings.Join(missing, ", "))
			}
		}
	} else if exampleContent != "" && !opts.DisableMissingWarnings {
		fmt.Printf("Warning: %s not found, using defaults\n", configFile)
	}


	// Layer 4: Command line flags (mapped to nested keys)
	if opts.FlagSet != nil {
		provider := posflag.ProviderWithFlag(opts.FlagSet, ".", k, func(f *pflag.Flag) (string, any) {
			key := f.Name
			if mapped, ok := flagKeyMap[key]; ok {
				key = mapped
			}
			return key, posflag.FlagVal(opts.FlagSet, f)
		})
		if err := k.Load(provider, nil); err != nil {
			return nil, fmt.Errorf("load flags: %w", err)
		}
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate sanitizes and applies default fallbacks to the configuration.
func (c *Config) Validate() error {
	// General
	c.General.Storefront = strings.ToLower(strings.TrimSpace(c.General.Storefront))
	if len(c.General.Storefront) != 2 {
		c.General.Storefront = "us"
	}
	if c.General.MaxMemoryLimit == 0 {
		c.General.MaxMemoryLimit = 256
	}

	// Media
	if c.Media.AlacMax == 0 {
		c.Media.AlacMax = 192000
	}
	if c.Media.AtmosMax == 0 {
		c.Media.AtmosMax = 2768
	}
	if c.Media.AacType == "" {
		c.Media.AacType = "aac-lc"
	}
	if c.Media.MV.AudioType == "" {
		c.Media.MV.AudioType = "atmos"
	}
	if c.Media.MV.Max == 0 {
		c.Media.MV.Max = 2160
	}

	// Metadata
	if c.Metadata.Artwork.Size == "" {
		c.Metadata.Artwork.Size = "5000x5000"
	}
	if c.Metadata.Artwork.Format == "" {
		c.Metadata.Artwork.Format = "jpg"
	}
	if c.Metadata.Format.LimitMax == 0 {
		c.Metadata.Format.LimitMax = 200
	}

	return nil
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fi.IsDir()
}
