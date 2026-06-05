package upstream

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ---------- Проверка upstream ----------
// TCP/HTTP health checks с параллельной обработкой.

// ServerHealth — результат проверки одного сервера.
type ServerHealth struct {
	Address string `json:"address"`
	Healthy bool   `json:"healthy"`
}

// ServerResolve — результат DNS-резолвинга.
type ServerResolve struct {
	Address  string   `json:"address"`
	Resolved []string `json:"resolved"`
}

// CheckTCP проверяет доступность сервера по TCP.
func CheckTCP(address string, timeout float64, retries int) bool {
	hostPort := strings.Fields(address)[0]
	host, port, err := splitHostPort(hostPort)
	if err != nil {
		return false
	}
	dialTimeout := time.Duration(timeout * float64(time.Second))
	for i := 0; i < retries; i++ {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), dialTimeout)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// CheckHTTP проверяет доступность сервера по HTTP GET /.
func CheckHTTP(address string, timeout float64, retries int) bool {
	hostPort := strings.Fields(address)[0]
	host, port, err := splitHostPort(hostPort)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: time.Duration(timeout * float64(time.Second))}
	url := fmt.Sprintf("http://%s/", net.JoinHostPort(host, port))
	for i := 0; i < retries; i++ {
		resp, err := client.Get(url)
		if err == nil {
			healthy := resp.StatusCode < 500
			resp.Body.Close()
			if healthy {
				return true
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// CheckUpstreams проверяет доступность всех upstream-серверов.
func CheckUpstreams(upstreams map[string][]string, timeout float64, retries int, mode string, maxWorkers int) map[string][]ServerHealth {
	results := make(map[string][]ServerHealth)
	type task struct {
		name string
		idx  int
		srv  string
	}
	var tasks []task
	for name, servers := range upstreams {
		results[name] = make([]ServerHealth, len(servers))
		for idx, srv := range servers {
			tasks = append(tasks, task{name, idx, srv})
		}
	}
	if len(tasks) == 0 {
		return results
	}
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Add(1)
		go func(t task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var healthy bool
			if strings.ToLower(mode) == "http" {
				healthy = CheckHTTP(t.srv, timeout, retries)
			} else {
				healthy = CheckTCP(t.srv, timeout, retries)
			}
			results[t.name][t.idx] = ServerHealth{Address: t.srv, Healthy: healthy}
		}(t)
	}
	wg.Wait()
	return results
}

// CheckUpstreamsMixed проверяет upstream: TCP для имён из stream, иначе defaultMode.
func CheckUpstreamsMixed(upstreams map[string][]string, tcpNames map[string]struct{}, timeout float64, retries int, defaultMode string, maxWorkers int) map[string][]ServerHealth {
	results := make(map[string][]ServerHealth)
	type task struct {
		name string
		idx  int
		srv  string
		tcp  bool
	}
	var tasks []task
	for name, servers := range upstreams {
		_, forceTCP := tcpNames[name]
		results[name] = make([]ServerHealth, len(servers))
		for idx, srv := range servers {
			tasks = append(tasks, task{name, idx, srv, forceTCP})
		}
	}
	if len(tasks) == 0 {
		return results
	}
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Add(1)
		go func(t task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			mode := defaultMode
			if t.tcp {
				mode = "tcp"
			}
			var healthy bool
			if strings.ToLower(mode) == "http" {
				healthy = CheckHTTP(t.srv, timeout, retries)
			} else {
				healthy = CheckTCP(t.srv, timeout, retries)
			}
			results[t.name][t.idx] = ServerHealth{Address: t.srv, Healthy: healthy}
		}(t)
	}
	wg.Wait()
	return results
}

// ResolveUpstreams резолвит DNS имена upstream-серверов.
func ResolveUpstreams(upstreams map[string][]string, maxWorkers int, useCache bool, cacheTTL int, cacheDir string) map[string][]ServerResolve {
	results := make(map[string][]ServerResolve)
	type task struct {
		name string
		idx  int
		srv  string
	}
	var tasks []task
	for name, servers := range upstreams {
		results[name] = make([]ServerResolve, len(servers))
		for idx, srv := range servers {
			tasks = append(tasks, task{name, idx, srv})
		}
	}
	if len(tasks) == 0 {
		return results
	}
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Add(1)
		go func(t task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			resolved := ResolveAddress(t.srv, useCache, cacheTTL, cacheDir)
			results[t.name][t.idx] = ServerResolve{Address: t.srv, Resolved: resolved}
		}(t)
	}
	wg.Wait()
	return results
}

func splitHostPort(hostPort string) (string, string, error) {
	if strings.HasPrefix(hostPort, "[") {
		if idx := strings.Index(hostPort, "]:"); idx >= 0 {
			return hostPort[1:idx], hostPort[idx+2:], nil
		}
		return "", "", fmt.Errorf("invalid ipv6 address")
	}
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		if strings.Count(hostPort, ":") == 1 {
			parts := strings.SplitN(hostPort, ":", 2)
			return parts[0], parts[1], nil
		}
		return "", "", err
	}
	return host, port, nil
}
