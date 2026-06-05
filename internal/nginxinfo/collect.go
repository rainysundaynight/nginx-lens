package nginxinfo

import (
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/docker"
	"github.com/rainysundaynight/nginx-lens/internal/nginxload"
)

// ---------- Сбор nginx -V ----------
// Хост или docker exec в зависимости от config.docker.

// CollectBuildInfo возвращает сведения о сборке nginx.
func CollectBuildInfo(cfg config.Config) *BuildInfo {
	nginxPath := cfg.Parser.NginxPath
	if nginxPath == "" {
		nginxPath = cfg.Syntax.NginxPath
	}
	if nginxPath == "" {
		nginxPath = "nginx"
	}
	if dctx, err := nginxload.DockerContext(cfg); err == nil && dctx.UseExec {
		if out, err := docker.NginxVOutput(dctx); err == nil {
			if info, err := ParseVersionOutput(out); err == nil {
				return info
			}
		}
	}
	info, err := RunVersion(nginxPath)
	if err != nil {
		return nil
	}
	return info
}
