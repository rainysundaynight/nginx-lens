package cli

import (
	"fmt"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/docker"
	"github.com/rainysundaynight/nginx-lens/internal/nginxload"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
	"github.com/rainysundaynight/nginx-lens/internal/upstream"
)

func loadConfig() config.Config {
	return config.Get().Config
}

func requireConfig() (config.Config, error) {
	loader := config.Get()
	if err := config.RequireConfigFile(loader); err != nil {
		return config.Config{}, err
	}
	return loader.Config, nil
}

func dockerContext(cfg config.Config) (docker.Context, error) {
	return nginxload.DockerContext(cfg)
}

func nginxConfigPath(cfg config.Config) (string, error) {
	_, path, err := nginxload.BuildTree(cfg)
	return path, err
}

func parseConfigFromCfg(cfg config.Config) (*parser.ConfigTree, error) {
	tree, _, err := nginxload.BuildTree(cfg)
	return tree, err
}

func outputFormat(cfg config.Config) string {
	return cfg.Output.Format
}

func setupDNSCache(cfg config.Config, sectionSkipCache bool) {
	if sectionSkipCache || !cfg.Cache.Enabled {
		upstream.DisableCache()
		return
	}
	upstream.EnableCache()
}

func analyzeFilter(cfg config.Config) analyzer.FilterOptions {
	a := cfg.Analyze
	opts := analyzer.FilterOptions{
		SkipLow:    a.SkipLow,
		SkipMedium: a.SkipMedium,
	}
	if len(a.SkipTypes) > 0 {
		opts.SkipTypes = make(map[string]struct{})
		for _, t := range a.SkipTypes {
			opts.SkipTypes[t] = struct{}{}
		}
	}
	switch a.MinSeverity {
	case "high":
		opts.MinSeverity = analyzer.SeverityHigh
	case "medium":
		opts.MinSeverity = analyzer.SeverityMedium
	default:
		opts.MinSeverity = analyzer.SeverityLow
	}
	return opts
}

func checkNginxSyntax(cfg config.Config, skipWarns bool) (bool, []string, error) {
	dctx, err := dockerContext(cfg)
	if err != nil {
		return false, nil, err
	}
	if dctx.UseExec {
		valid, errs := docker.NginxTest(dctx, dctx.ConfigInside)
		return valid, errs, nil
	}
	path := dctx.HostConfigPath
	if path == "" {
		path = cfg.Defaults.NginxConfigPath
	}
	nginxPath := cfg.Syntax.NginxPath
	if nginxPath == "" {
		nginxPath = cfg.Validate.NginxPath
	}
	if nginxPath == "" {
		nginxPath = "nginx"
	}
	valid, errs := checkSyntaxLocal(path, nginxPath, skipWarns)
	return valid, errs, nil
}

func certVolumeMap(cfg config.Config) map[string]string {
	return cfg.Docker.VolumeMap
}

func requireNonEmpty(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s не задан в конфиге", field)
	}
	return nil
}

// firstNonEmpty возвращает первое непустое значение (CLI → config).
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func requireParam(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s не задан (укажите аргумент или флаг команды, либо значение в config.yaml)", name)
	}
	return nil
}

func validateFailOn(cfg config.Config) string {
	if cfg.Validate.FailOn != "" {
		return cfg.Validate.FailOn
	}
	if cfg.Validate.FailOnLow {
		return "low"
	}
	return "high"
}
