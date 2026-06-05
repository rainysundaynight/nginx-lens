package logs

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------- Парсер access-логов ----------
// Поддержка nginx combined, JSONL, LTSV и gzip.

// LogLine — распарсенная строка access-лога.
type LogLine struct {
	Time         time.Time `json:"time,omitempty"`
	Method       string    `json:"method,omitempty"`
	Path         string    `json:"path,omitempty"`
	Status       int       `json:"status,omitempty"`
	IP           string    `json:"ip,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	ResponseTime float64   `json:"response_time,omitempty"`
	Upstream     string    `json:"upstream,omitempty"`
}

// Stats — статистика по логам.
type Stats struct {
	TotalRequests int                `json:"total_requests"`
	StatusCodes   map[string]int     `json:"status_codes"`
	TopPaths      []CountEntry       `json:"top_paths"`
	TopIPs        []CountEntry       `json:"top_ips"`
	TopUserAgents []CountEntry       `json:"top_user_agents"`
	AvgResponse   float64            `json:"avg_response_time"`
	Anomalies     []Anomaly          `json:"anomalies,omitempty"`
}

// CountEntry — запись топ-N.
type CountEntry struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// Anomaly — обнаруженная аномалия.
type Anomaly struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

var combinedRe = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "(\S+) (\S+) [^"]*" (\d+) \d+ "[^"]*" "([^"]*)"`)

// ParseLogFile парсит access-лог и возвращает статистику.
func ParseLogFile(path string, top int, since, until string, statusFilter string, skipAnomalies bool) (*Stats, error) {
	reader, closer, err := openLog(path)
	if err != nil {
		return nil, err
	}
	defer closer()

	sinceT, untilT := parseTimeBounds(since, until)
	var lines []LogLine
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		parsed := parseLine(line)
		if parsed == nil {
			continue
		}
		if statusFilter != "" && fmt.Sprintf("%d", parsed.Status) != statusFilter {
			continue
		}
		if !sinceT.IsZero() && !parsed.Time.IsZero() && parsed.Time.Before(sinceT) {
			continue
		}
		if !untilT.IsZero() && !parsed.Time.IsZero() && parsed.Time.After(untilT) {
			continue
		}
		lines = append(lines, *parsed)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return computeStats(lines, top, skipAnomalies), nil
}

func openLog(path string) (io.Reader, func(), error) {
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

func parseLine(line string) *LogLine {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if strings.HasPrefix(line, "{") {
		var m map[string]interface{}
		if json.Unmarshal([]byte(line), &m) == nil {
			return parseJSONLine(m)
		}
	}
	if m := combinedRe.FindStringSubmatch(line); m != nil {
		status, _ := strconv.Atoi(m[5])
		l := &LogLine{
			Method:    m[3],
			Path:      m[4],
			Status:    status,
			IP:        m[1],
			UserAgent: m[6],
		}
		if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[2]); err == nil {
			l.Time = t
		}
		return l
	}
	return nil
}

func parseJSONLine(m map[string]interface{}) *LogLine {
	l := &LogLine{}
	if v, ok := m["status"].(float64); ok {
		l.Status = int(v)
	}
	if v, ok := m["request_uri"].(string); ok {
		l.Path = v
	}
	if v, ok := m["remote_addr"].(string); ok {
		l.IP = v
	}
	if v, ok := m["http_user_agent"].(string); ok {
		l.UserAgent = v
	}
	if v, ok := m["request_method"].(string); ok {
		l.Method = v
	}
	if v, ok := m["upstream_addr"].(string); ok {
		l.Upstream = v
	}
	if v, ok := m["request_time"].(float64); ok {
		l.ResponseTime = v * 1000
	}
	return l
}

func computeStats(lines []LogLine, top int, skipAnomalies bool) *Stats {
	stats := &Stats{
		TotalRequests: len(lines),
		StatusCodes:   make(map[string]int),
	}
	paths, ips, agents := make(map[string]int), make(map[string]int), make(map[string]int)
	var totalRT float64
	var rtCount int
	for _, l := range lines {
		stats.StatusCodes[fmt.Sprintf("%d", l.Status)]++
		if l.Path != "" {
			paths[l.Path]++
		}
		if l.IP != "" {
			ips[l.IP]++
		}
		if l.UserAgent != "" {
			agents[l.UserAgent]++
		}
		if l.ResponseTime > 0 {
			totalRT += l.ResponseTime
			rtCount++
		}
	}
	if rtCount > 0 {
		stats.AvgResponse = totalRT / float64(rtCount)
	}
	stats.TopPaths = topN(paths, top)
	stats.TopIPs = topN(ips, top)
	stats.TopUserAgents = topN(agents, top)
	if !skipAnomalies {
		stats.Anomalies = detectAnomalies(lines)
	}
	return stats
}

func topN(m map[string]int, n int) []CountEntry {
	var entries []CountEntry
	for k, v := range m {
		entries = append(entries, CountEntry{Key: k, Count: v})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })
	if len(entries) > n {
		entries = entries[:n]
	}
	return entries
}

func parseTimeBounds(since, until string) (time.Time, time.Time) {
	var sinceT, untilT time.Time
	if since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			sinceT = t
		} else if t, err := time.Parse("2006-01-02", since); err == nil {
			sinceT = t
		}
	}
	if until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			untilT = t
		} else if t, err := time.Parse("2006-01-02", until); err == nil {
			untilT = t.Add(24*time.Hour - time.Second)
		}
	}
	return sinceT, untilT
}

func detectAnomalies(lines []LogLine) []Anomaly {
	var anomalies []Anomaly
	errors := 0
	for _, l := range lines {
		if l.Status >= 500 {
			errors++
		}
	}
	if len(lines) > 0 && float64(errors)/float64(len(lines)) > 0.1 {
		anomalies = append(anomalies, Anomaly{
			Type:    "error_rate_spike",
			Message: fmt.Sprintf("Высокий процент 5xx ошибок: %d/%d", errors, len(lines)),
		})
	}
	return anomalies
}
