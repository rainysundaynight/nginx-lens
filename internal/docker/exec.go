package docker

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ---------- Docker exec для nginx ----------
// nginx -T, nginx -t и tail логов внутри контейнера.

// NginxT выполняет nginx -T в контейнере.
func NginxT(ctx Context, configInside string) (string, error) {
	if configInside == "" {
		configInside = ctx.ConfigInside
	}
	cmd := exec.Command(ctx.Binary, "exec", ctx.Container, "nginx", "-T", "-c", configInside)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker exec nginx -T: %w: %s", err, string(out))
	}
	return string(out), nil
}

// NginxVOutput выполняет nginx -V в контейнере.
func NginxVOutput(ctx Context) (string, error) {
	cmd := exec.Command(ctx.Binary, "exec", ctx.Container, "nginx", "-V")
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", fmt.Errorf("docker exec nginx -V: %w", err)
	}
	return string(out), nil
}

// NginxTest выполняет nginx -t в контейнере.
func NginxTest(ctx Context, configInside string) (bool, []string) {
	if configInside == "" {
		configInside = ctx.ConfigInside
	}
	cmd := exec.Command(ctx.Binary, "exec", ctx.Container, "nginx", "-t", "-c", configInside)
	out, err := cmd.CombinedOutput()
	combined := string(out)
	if err != nil {
		return false, []string{combined}
	}
	return true, nil
}

// TailLog читает последние n строк файла внутри контейнера.
func TailLog(ctx Context, pathInside string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50000
	}
	cmd := exec.Command(ctx.Binary, "exec", ctx.Container, "tail", "-n", strconv.Itoa(lines), pathInside)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker exec tail: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
