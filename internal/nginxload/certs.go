package nginxload

import (
	"os"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/docker"
)

// ---------- Чтение сертификатов ----------
// Пути из ssl_certificate (nginx.conf / conf.d); чтение: volume_map → хост → docker exec cat.

// CertReadFile возвращает reader для AuditCertificates.
func CertReadFile(cfg config.Config) analyzer.CertReadFile {
	dctx, err := DockerContext(cfg)
	if err != nil {
		dctx = docker.Context{VolumeMap: cfg.Docker.VolumeMap}
	}
	volumeMap := cfg.Docker.VolumeMap
	return func(logicalPath string) ([]byte, error) {
		hostPath := logicalPath
		if mapped, ok := docker.MapToHost(volumeMap, logicalPath); ok {
			hostPath = mapped
		}
		data, hostErr := os.ReadFile(hostPath)
		if hostErr == nil {
			return data, nil
		}
		if dctx.UseExec {
			return docker.ReadFile(dctx, logicalPath)
		}
		return nil, hostErr
	}
}
