package nginxload

import (
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/docker"
)

// ---------- Резолв путей логов ----------
// Хост или volume_map для access/error.log.

// ResolveLogPath возвращает путь к логу на хосте.
func ResolveLogPath(cfg config.Config, hostPath string) string {
	if hostPath == "" {
		return ""
	}
	dctx, err := DockerContext(cfg)
	if err != nil {
		return hostPath
	}
	return docker.ResolveHostPath(dctx, hostPath)
}
