package nginxload

import (
	"os"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/docker"
)

// ---------- Чтение сертификатов ----------
// Пути из ssl_certificate (nginx.conf / conf.d); чтение: volume_map → хост → docker exec cat.
// Docker cat пробуем даже если конфиг читается с хоста (SSL часто 640 root-only).

// CertReadFile возвращает reader для AuditCertificates.
func CertReadFile(cfg config.Config) analyzer.CertReadFile {
	dctx, err := DockerContext(cfg)
	if err != nil {
		dctx = docker.Context{VolumeMap: cfg.Docker.VolumeMap, Binary: cfg.Docker.Binary}
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
		// Конфиг могли прочитать с хоста (UseExec=false), а PEM — только через docker.
		execCtx, exErr := dockerExecContext(cfg)
		if exErr == nil && execCtx.UseExec {
			if data, err := docker.ReadFile(execCtx, logicalPath); err == nil {
				return data, nil
			} else {
				return nil, err
			}
		}
		return nil, hostErr
	}
}

// dockerExecContext форсирует детект контейнера для чтения файлов.
func dockerExecContext(cfg config.Config) (docker.Context, error) {
	mode := cfg.Docker.Enabled
	if mode == "" || mode == "false" {
		return docker.Context{}, os.ErrNotExist
	}
	// Временно не подставляем путь хоста, чтобы BuildContext ушёл в UseExec.
	return docker.BuildContext(docker.Config{
		Enabled:      "true",
		Container:    cfg.Docker.Container,
		Binary:       cfg.Docker.Binary,
		ConfigInside: cfg.Docker.ConfigInside,
		VolumeMap:    nil,
	}, "/.__nginx_lens_missing_host_config__")
}
