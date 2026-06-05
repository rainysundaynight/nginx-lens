package nginxload

import (
	"os"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/docker"
)

// ---------- Чтение логов nginx ----------
// Хост, volume_map или docker exec tail.

// ReadLogTail возвращает последние maxLines строк access/error.log.
func ReadLogTail(cfg config.Config, logPath string, maxLines int) (string, error) {
	if logPath == "" {
		return "", nil
	}
	if maxLines <= 0 {
		maxLines = 50000
	}
	dctx, err := DockerContext(cfg)
	if err != nil {
		return readHostTail(logPath, maxLines)
	}
	hostPath := docker.ResolveHostPath(dctx, logPath)
	if st, err := os.Stat(hostPath); err == nil && !st.IsDir() {
		return readHostTail(hostPath, maxLines)
	}
	if dctx.UseExec {
		return docker.TailLog(dctx, logPath, maxLines)
	}
	return readHostTail(logPath, maxLines)
}

func readHostTail(path string, maxLines int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n"), nil
}
