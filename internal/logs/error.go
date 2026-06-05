package logs

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"regexp"
	"strings"
)

// ---------- Парсер error.log ----------
// Корреляция ошибок upstream с конфигурацией.

// ErrorEntry — запись error.log.
type ErrorEntry struct {
	Upstream string `json:"upstream,omitempty"`
	Message  string `json:"message"`
	Count    int    `json:"count"`
}

// ErrorStats — статистика error.log.
type ErrorStats struct {
	TotalErrors      int            `json:"total_errors"`
	UpstreamErrors   []ErrorEntry   `json:"upstream_errors"`
	ConnectFailed    int            `json:"connect_failed"`
	UpstreamTimed    int            `json:"upstream_timed_out"`
	SSLErrors        int            `json:"ssl_errors"`
	UpstreamConnect  map[string]int `json:"upstream_connect,omitempty"`
	UpstreamTimeout  map[string]int `json:"upstream_timeout,omitempty"`
}

var (
	reUpstreamConnect = regexp.MustCompile(`connect\(\) failed.*while connecting to upstream`)
	reUpstreamTimeout = regexp.MustCompile(`upstream timed out`)
	reUpstreamHost    = regexp.MustCompile(`upstream:\s*"([^"]+)"`)
	reSSLError        = regexp.MustCompile(`SSL_do_handshake|ssl_`)
)

// ParseErrorLog парсит nginx error.log с диска.
func ParseErrorLog(path string) (*ErrorStats, error) {
	reader, closer, err := openLogFile(path)
	if err != nil {
		return nil, err
	}
	defer closer()
	return parseErrorReader(reader)
}

// ParseErrorLogContent парсит error.log из строки (docker tail).
func ParseErrorLogContent(content string) (*ErrorStats, error) {
	return parseErrorReader(strings.NewReader(content))
}

func parseErrorReader(reader io.Reader) (*ErrorStats, error) {
	stats := &ErrorStats{
		UpstreamConnect: make(map[string]int),
		UpstreamTimeout: make(map[string]int),
	}
	upstreamCounts := make(map[string]int)
	upstreamSample := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if !isErrorLine(line) {
			continue
		}
		stats.TotalErrors++
		upHost := ""
		if m := reUpstreamHost.FindStringSubmatch(line); m != nil {
			upHost = m[1]
			upstreamCounts[upHost]++
			if _, ok := upstreamSample[upHost]; !ok {
				upstreamSample[upHost] = trimErrorMessage(line)
			}
		}
		if reUpstreamConnect.MatchString(line) {
			stats.ConnectFailed++
			if upHost != "" {
				stats.UpstreamConnect[upHost]++
			}
		}
		if reUpstreamTimeout.MatchString(line) {
			stats.UpstreamTimed++
			if upHost != "" {
				stats.UpstreamTimeout[upHost]++
			}
		}
		if reSSLError.MatchString(line) {
			stats.SSLErrors++
		}
	}
	for up, cnt := range upstreamCounts {
		msg := upstreamSample[up]
		if msg == "" {
			msg = "upstream errors"
		}
		stats.UpstreamErrors = append(stats.UpstreamErrors, ErrorEntry{Upstream: up, Message: msg, Count: cnt})
	}
	return stats, scanner.Err()
}

func trimErrorMessage(line string) string {
	line = strings.TrimSpace(line)
	if idx := strings.Index(line, "] "); idx >= 0 {
		line = strings.TrimSpace(line[idx+2:])
	}
	if len(line) > 160 {
		line = line[:160] + "…"
	}
	return line
}

func openLogFile(path string) (io.Reader, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, nil, err
		}
		return gz, func() { gz.Close(); f.Close() }, nil
	}
	return f, func() { f.Close() }, nil
}

func isErrorLine(line string) bool {
	return strings.Contains(line, "[error]") || strings.Contains(line, "[crit]") || strings.Contains(line, "[alert]")
}

// CorrelateWithUpstream считает ошибки для логического upstream.
func CorrelateWithUpstream(stats *ErrorStats, upstreamName string, index map[string]string) int {
	if stats == nil {
		return 0
	}
	total := 0
	for _, e := range stats.UpstreamErrors {
		if ResolveLogicalUpstream(e.Upstream, index) == upstreamName {
			total += e.Count
		}
	}
	for raw, cnt := range stats.UpstreamConnect {
		if ResolveLogicalUpstream(raw, index) == upstreamName {
			total += cnt
		}
	}
	return total
}
