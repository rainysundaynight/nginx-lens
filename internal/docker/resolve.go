package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ---------- Volume map и резолв путей ----------
// Маппинг путей контейнера на хост для regex/certs/logs.

// Context — контекст работы с dockerized nginx.
type Context struct {
	Enabled        bool
	UseExec        bool
	Container      string
	Binary         string
	ConfigInside   string
	HostConfigPath string
	VolumeMap      map[string]string
}

// Config — настройки docker из config.yaml.
type Config struct {
	Enabled      string
	Container    string
	Binary       string
	ConfigInside string
	VolumeMap    map[string]string
}

// BuildContext собирает контекст: exec или host file через volume_map.
func BuildContext(cfg Config, hostConfigPath string) (Context, error) {
	ctx := Context{
		Binary:       cfg.Binary,
		ConfigInside: cfg.ConfigInside,
		VolumeMap:    cfg.VolumeMap,
	}
	if ctx.Binary == "" {
		ctx.Binary = "docker"
	}
	if ctx.ConfigInside == "" {
		ctx.ConfigInside = "/etc/nginx/nginx.conf"
	}
	mode := strings.ToLower(cfg.Enabled)
	if mode == "" {
		mode = "auto"
	}
	if mode == "false" {
		ctx.HostConfigPath = hostConfigPath
		return ctx, nil
	}
	if mapped, ok := MapToHost(cfg.VolumeMap, hostConfigPath); ok {
		if _, err := os.Stat(mapped); err == nil {
			ctx.HostConfigPath = mapped
			return ctx, nil
		}
	}
	if _, err := os.Stat(hostConfigPath); err == nil {
		ctx.HostConfigPath = hostConfigPath
		return ctx, nil
	}
	if !IsAvailable(ctx.Binary) {
		if mode == "true" {
			return ctx, fmt.Errorf("docker недоступен, enabled=true")
		}
		return ctx, fmt.Errorf("nginx.conf не найден: %s", hostConfigPath)
	}
	container, err := DetectContainer(ctx.Binary, cfg.Container)
	if err != nil {
		return ctx, err
	}
	if container == "" {
		if mode == "true" {
			return ctx, fmt.Errorf("nginx-контейнер не найден")
		}
		return ctx, fmt.Errorf("nginx.conf не найден: %s", hostConfigPath)
	}
	ctx.Enabled = true
	ctx.UseExec = true
	ctx.Container = container
	return ctx, nil
}

// MapToHost переводит путь внутри контейнера на путь хоста.
func MapToHost(volumeMap map[string]string, containerPath string) (string, bool) {
	if len(volumeMap) == 0 {
		return "", false
	}
	containerPath = filepath.ToSlash(containerPath)
	var bestPrefix, bestHost string
	for prefix, host := range volumeMap {
		p := filepath.ToSlash(prefix)
		if strings.HasPrefix(containerPath, p) && len(p) > len(bestPrefix) {
			bestPrefix = p
			bestHost = host
		}
	}
	if bestPrefix == "" {
		return "", false
	}
	suffix := strings.TrimPrefix(containerPath, bestPrefix)
	return filepath.Join(bestHost, filepath.FromSlash(strings.TrimPrefix(suffix, "/"))), true
}

// ResolveHostPath резолвит произвольный путь (certs, logs) через volume_map.
func ResolveHostPath(ctx Context, path string) string {
	if mapped, ok := MapToHost(ctx.VolumeMap, path); ok {
		return mapped
	}
	if ctx.HostConfigPath != "" && !ctx.UseExec {
		return path
	}
	return path
}
