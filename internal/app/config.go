package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"amdl/internal/download"
	"net/http"
	"os"
	"sort"
	"strings"
)

func topLevelKeys(data []byte) map[string]bool {
	keys := make(map[string]bool)
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return keys
	}
	for k := range raw {
		keys[k] = true
	}
	return keys
}

// getLiteRegions queries wrapper-lite's /status endpoint once and returns
// the regions reported by the service.

func (r *Runner) getLiteRegions() ([]string, error) {
	if r.Config.LiteServer == "" {
		return nil, errors.New("lite-server is not configured")
	}
	endpoint := strings.TrimRight(r.Config.LiteServer, "/") + "/status"
	resp, err := download.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	var envelope struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Regions []string `json:"regions"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("lite-server /status returned code=%d msg=%s", envelope.Code, envelope.Msg)
	}
	return envelope.Data.Regions, nil
}

// flagValueFromArgs scans raw os.Args for "--name=value" or "--name value"
// before pflag.Parse runs, so early startup logic can already use the override.

func flagValueFromArgs(args []string, name string) string {
	prefix := "--" + name + "="
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], prefix) {
			return strings.TrimPrefix(args[i], prefix)
		}
		if args[i] == "--"+name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func (r *Runner) loadConfig() error {
	userData, err := os.ReadFile("config.yaml")
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read config.yaml: %w", err)
		}
		userData = nil
	}

	exampleData, err := os.ReadFile("config.yaml.example")
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read config.yaml.example: %w", err)
		}
		exampleData = nil
	}

	// config.yaml.example supplies defaults when it is available. A valid
	// config.yaml remains enough to run without the example file.
	if userData == nil && exampleData == nil {
		return errors.New("config file not found: provide config.yaml")
	}
	if exampleData == nil && userData != nil {
		fmt.Println("Warning: config.yaml.example not found, using config.yaml only")
	}

	if err := yaml.Unmarshal(exampleData, &r.Config); err != nil {
		return fmt.Errorf("parse config.yaml.example: %w", err)
	}

	if userData != nil {
		if err := yaml.Unmarshal(userData, &r.Config); err != nil {
			return fmt.Errorf("parse config.yaml: %w", err)
		}

		exampleKeys := topLevelKeys(exampleData)
		userKeys := topLevelKeys(userData)
		var missing []string
		for k := range exampleKeys {
			if !userKeys[k] {
				missing = append(missing, k)
			}
		}
		sort.Strings(missing)

		if len(missing) > 0 {
			fmt.Println("Warning: config.yaml is missing fields, using defaults from config.yaml.example for them.")
			fmt.Println("  Missing fields:", strings.Join(missing, ", "))
		}
	} else if exampleData != nil {
		fmt.Println("Warning: config.yaml not found, using defaults from config.yaml.example")
	}

	if len(r.Config.Storefront) != 2 {
		r.Config.Storefront = "us"
	}
	if r.Config.AlacMax == 0 {
		r.Config.AlacMax = 192000
	}

	if r.Config.AtmosMax == 0 {
		r.Config.AtmosMax = 2768
	}

	if r.Config.AacType == "" {
		r.Config.AacType = "aac-lc"
	}

	if r.Config.MVAudioType == "" {
		r.Config.MVAudioType = "atmos"
	}
	return nil
}
