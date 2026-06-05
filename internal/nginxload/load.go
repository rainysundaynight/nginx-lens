package nginxload

import (
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/docker"
	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Загрузка nginx.conf ----------
// Единая точка: хост, volume_map или docker exec.

// BuildTree парсит конфигурацию nginx с учётом docker.
func BuildTree(cfg config.Config) (*parser.ConfigTree, string, error) {
	dctx, err := docker.BuildContext(docker.Config{
		Enabled:      cfg.Docker.Enabled,
		Container:    cfg.Docker.Container,
		Binary:       cfg.Docker.Binary,
		ConfigInside: cfg.Docker.ConfigInside,
		VolumeMap:    cfg.Docker.VolumeMap,
	}, cfg.Defaults.NginxConfigPath)
	if err != nil {
		return nil, "", err
	}
	configPath := dctx.HostConfigPath
	if dctx.UseExec {
		configPath = dctx.ConfigInside
	}
	opts := parser.ParseOptions{Mode: cfg.Parser.Mode, NginxPath: cfg.Parser.NginxPath}
	var tree *parser.ConfigTree
	if dctx.UseExec {
		out, err := docker.NginxT(dctx, dctx.ConfigInside)
		if err == nil {
			tree, err = parser.ParseExpandedOutput(out)
			if err != nil {
				return nil, "", err
			}
		}
	}
	if tree == nil {
		path := dctx.HostConfigPath
		if path == "" {
			path = cfg.Defaults.NginxConfigPath
		}
		tree, err = parser.ParseConfig(path, opts)
		if err != nil {
			return nil, "", err
		}
	}
	dyn := cfg.DynamicUpstream
	tree.SetDynamicUpstreamConfig(dyn.Enabled, dyn.APIURL, dyn.Timeout)
	return tree, configPath, nil
}

// DockerContext возвращает docker-контекст для syntax/certs.
func DockerContext(cfg config.Config) (docker.Context, error) {
	return docker.BuildContext(docker.Config{
		Enabled:      cfg.Docker.Enabled,
		Container:    cfg.Docker.Container,
		Binary:       cfg.Docker.Binary,
		ConfigInside: cfg.Docker.ConfigInside,
		VolumeMap:    cfg.Docker.VolumeMap,
	}, cfg.Defaults.NginxConfigPath)
}
