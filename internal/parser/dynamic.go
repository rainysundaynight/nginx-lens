package parser

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ---------- Dynamic upstream API ----------
// Обогащение upstream через ngx_dynamic_upstream HTTP API.

var reDynamicServer = regexp.MustCompile(`^server\s+([^;]+);`)

// getDynamicUpstreamServers получает серверы динамического upstream через HTTP API.
func getDynamicUpstreamServers(upstreamName, apiURL string, timeout float64) ([]string, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	q := parsed.Query()
	q.Set("upstream", upstreamName)
	parsed.RawQuery = q.Encode()

	client := &http.Client{Timeout: time.Duration(timeout * float64(time.Second))}
	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "nginx-lens/2.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var servers []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := reDynamicServer.FindStringSubmatch(line); m != nil {
			spec := strings.TrimSpace(m[1])
			parts := strings.Fields(spec)
			if len(parts) > 0 {
				servers = append(servers, parts[0])
			}
		}
	}
	return servers, nil
}

// enrichUpstreamsWithDynamic обогащает upstream пустые списки серверов из API.
func enrichUpstreamsWithDynamic(upstreams map[string][]string, apiURL string, timeout float64, enabled bool) map[string][]string {
	if !enabled || apiURL == "" {
		return upstreams
	}
	enriched := make(map[string][]string, len(upstreams))
	for name, servers := range upstreams {
		enriched[name] = servers
	}
	for name, servers := range upstreams {
		if len(servers) == 0 {
			dynamic, err := getDynamicUpstreamServers(name, apiURL, timeout)
			if err == nil && len(dynamic) > 0 {
				enriched[name] = dynamic
			}
		}
	}
	return enriched
}
