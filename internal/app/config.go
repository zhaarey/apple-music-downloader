package app

import (
	"strings"

	"amdl/internal/config"
	"amdl/internal/wrapper"
)

// getLiteRegions queries wrapper-lite's /status endpoint once and returns
// the regions reported by the service.
func (r *Runner) getLiteRegions() ([]string, error) {
	return wrapper.GetStatus(r.Config.General.LiteServer)
}

// flagValueFromArgs scans raw os.Args for "--name=value" or "--name value"
// before flag parsing runs, allowing early override lookups if needed.
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

// loadConfig loads application configuration through internal/config with koanf.
func (r *Runner) loadConfig(opts ...config.LoadOptions) error {
	var opt config.LoadOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	cfg, err := config.Load(opt)
	if err != nil {
		return err
	}
	r.Config = *cfg
	return nil
}
