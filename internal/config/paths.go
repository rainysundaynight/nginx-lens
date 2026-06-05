package config

import (
	"os"
	"path/filepath"
)

// ---------- Пути конфигурации ----------
// Стандартные расположения config.yaml.

const (
	SystemConfigDir  = "/opt/nginx-lens"
	SystemConfigPath = "/opt/nginx-lens/config.yaml"
)

// UserConfigPath возвращает путь ~/.nginx-lens/config.yaml.
func UserConfigPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".nginx-lens", "config.yaml")
}
