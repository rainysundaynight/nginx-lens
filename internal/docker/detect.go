package docker

import (
	"os/exec"
	"strings"
)

// ---------- Detect контейнера nginx ----------
// Auto-detect через docker ps.

// IsAvailable проверяет наличие docker CLI.
func IsAvailable(binary string) bool {
	if binary == "" {
		binary = "docker"
	}
	_, err := exec.LookPath(binary)
	return err == nil
}

// DetectContainer находит running nginx-контейнер.
func DetectContainer(binary, name string) (string, error) {
	if binary == "" {
		binary = "docker"
	}
	if name != "" {
		return name, nil
	}
	out, err := exec.Command(binary, "ps", "--format", "{{.Names}}").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		n := strings.TrimSpace(line)
		if n == "" {
			continue
		}
		lower := strings.ToLower(n)
		if strings.Contains(lower, "nginx") {
			return n, nil
		}
	}
	return "", nil
}
