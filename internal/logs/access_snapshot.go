package logs

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------- Access snapshot для agent ----------
// RPS, p95, 5xx/404 и статистика по upstream.

// AccessSnapshot — агрегированная статистика access.log.
type AccessSnapshot struct {
	TotalRequests int                       `json:"total_requests"`
	RPS           float64                   `json:"rps"`
	P95Latency    float64                   `json:"p95_latency_ms"`
	Status404     int                       `json:"status_404"`
	Status5xx     int                       `json:"status_5xx"`
	WindowSeconds float64                   `json:"window_seconds"`
	ByUpstream    map[string]UpstreamAccess `json:"by_upstream,omitempty"`
	ByPath        map[string]PathAccess     `json:"by_path,omitempty"`
}

// PathAccess — метрики access по URI (запросы без upstream_addr в логе).
type PathAccess struct {
	Requests  int `json:"requests"`
	Status5xx int `json:"status_5xx"`
	Status502 int `json:"status_502"`
}

// UpstreamAccess — метрики access по upstream.
type UpstreamAccess struct {
	Requests   int     `json:"requests"`
	Status5xx  int     `json:"status_5xx"`
	Status502  int     `json:"status_502"`
	ErrorPct   float64 `json:"error_pct"`
	P95Latency float64 `json:"p95_latency_ms"`
}

var extendedCombinedRe = regexp.MustCompile(
	`^(\S+) \S+ \S+ \[([^\]]+)\] "(\S+) (\S+) [^"]*" (\d+) \d+ "[^"]*" "([^"]*)"(?:\s+"?([\d.]+)"?(?:\s+"?([^"]*)"?)?)?`,
)

// ParseAccessSnapshot парсит access.log с диска (tail последних maxLines строк).
func ParseAccessSnapshot(path string, since, until, statusFilter, formatRegex string, maxLines int) (*AccessSnapshot, error) {
	spec, _ := NewFormatSpec(formatRegex)
	reader, closer, err := openLogTail(path, maxLines)
	if err != nil {
		return nil, err
	}
	defer closer()
	return parseAccessReader(reader, since, until, statusFilter, spec)
}

// ParseAccessSnapshotContent парсит access.log из строки (docker tail).
func ParseAccessSnapshotContent(content, since, until, statusFilter, formatRegex string) (*AccessSnapshot, error) {
	spec, _ := NewFormatSpec(formatRegex)
	return parseAccessReader(strings.NewReader(content), since, until, statusFilter, spec)
}

func parseAccessReader(reader io.Reader, since, until, statusFilter string, spec *FormatSpec) (*AccessSnapshot, error) {
	sinceT, untilT := parseTimeBounds(since, until)
	var lines []LogLine
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		parsed := parseAccessLine(scanner.Text(), spec)
		if parsed == nil {
			continue
		}
		if statusFilter != "" && !matchStatusFilter(parsed.Status, statusFilter) {
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
	return computeAccessSnapshot(lines), nil
}

func matchStatusFilter(status int, filter string) bool {
	s := strconv.Itoa(status)
	for _, part := range strings.Split(filter, ",") {
		if strings.TrimSpace(part) == s {
			return true
		}
	}
	return false
}

func parseAccessLine(line string, spec *FormatSpec) *LogLine {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if spec != nil {
		if l := spec.parse(line); l != nil {
			return l
		}
	}
	if strings.HasPrefix(line, "{") {
		var m map[string]interface{}
		if json.Unmarshal([]byte(line), &m) == nil {
			return parseJSONLine(m)
		}
	}
	if l := parseCombinedExtended(line); l != nil {
		return l
	}
	return parseLine(line)
}

func parseCombinedExtended(line string) *LogLine {
	if m := extendedCombinedRe.FindStringSubmatch(line); m != nil {
		status, _ := strconv.Atoi(m[5])
		l := &LogLine{
			Method:    m[3],
			Path:      m[4],
			Status:    status,
			IP:        m[1],
			UserAgent: m[6],
		}
		if len(m) > 7 && m[7] != "" {
			if rt, err := strconv.ParseFloat(m[7], 64); err == nil {
				l.ResponseTime = rt * 1000
			}
		}
		if len(m) > 8 && m[8] != "" && m[8] != "-" {
			l.Upstream = normalizeUpstreamAddr(m[8])
		}
		if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[2]); err == nil {
			l.Time = t
		}
		return l
	}
	return nil
}

func normalizeUpstreamAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if idx := strings.Index(addr, ","); idx >= 0 {
		addr = addr[:idx]
	}
	return addr
}

func computeAccessSnapshot(lines []LogLine) *AccessSnapshot {
	snap := &AccessSnapshot{ByUpstream: make(map[string]UpstreamAccess), ByPath: make(map[string]PathAccess)}
	if len(lines) == 0 {
		return snap
	}
	snap.TotalRequests = len(lines)
	var minT, maxT time.Time
	var rts []float64
	upRT := make(map[string][]float64)
	upCounts := make(map[string]*struct{ req, s5, s502 int })
	pathDirect := make(map[string]*struct{ req, s5, s502 int })

	for _, l := range lines {
		if l.Status == 404 {
			snap.Status404++
		}
		if l.Status >= 500 {
			snap.Status5xx++
		}
		if !l.Time.IsZero() {
			if minT.IsZero() || l.Time.Before(minT) {
				minT = l.Time
			}
			if maxT.IsZero() || l.Time.After(maxT) {
				maxT = l.Time
			}
		}
		if l.ResponseTime > 0 {
			rts = append(rts, l.ResponseTime)
		}
		upKey := l.Upstream
		if upKey == "" {
			upKey = "_direct"
		}
		if upCounts[upKey] == nil {
			upCounts[upKey] = &struct{ req, s5, s502 int }{}
		}
		upCounts[upKey].req++
		if l.Status >= 500 {
			upCounts[upKey].s5++
		}
		if l.Status == 502 {
			upCounts[upKey].s502++
		}
		if l.ResponseTime > 0 {
			upRT[upKey] = append(upRT[upKey], l.ResponseTime)
		}
		if l.Upstream == "" {
			path := normalizeAccessPath(l.Path)
			if path != "" {
				if pathDirect[path] == nil {
					pathDirect[path] = &struct{ req, s5, s502 int }{}
				}
				pathDirect[path].req++
				if l.Status >= 500 {
					pathDirect[path].s5++
				}
				if l.Status == 502 {
					pathDirect[path].s502++
				}
			}
		}
	}
	if !minT.IsZero() && !maxT.IsZero() {
		snap.WindowSeconds = maxT.Sub(minT).Seconds()
		if snap.WindowSeconds < 1 {
			snap.WindowSeconds = 1
		}
		snap.RPS = float64(snap.TotalRequests) / snap.WindowSeconds
	}
	snap.P95Latency = percentile(rts, 0.95)
	for name, c := range upCounts {
		ua := UpstreamAccess{
			Requests:  c.req,
			Status5xx: c.s5,
			Status502: c.s502,
		}
		if c.req > 0 {
			ua.ErrorPct = math.Round(float64(c.s5)/float64(c.req)*1000) / 10
		}
		ua.P95Latency = percentile(upRT[name], 0.95)
		snap.ByUpstream[name] = ua
	}
	for path, c := range pathDirect {
		snap.ByPath[path] = PathAccess{Requests: c.req, Status5xx: c.s5, Status502: c.s502}
	}
	return snap
}

func normalizeAccessPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	return path
}

func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]float64(nil), vals...)
	sort.Float64s(cp)
	idx := int(math.Ceil(p*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return math.Round(cp[idx]*10) / 10
}

func openLogTail(path string, maxLines int) (io.Reader, func(), error) {
	if maxLines <= 0 {
		maxLines = 50000
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	var reader io.Reader = f
	closer := func() { f.Close() }
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, nil, err
		}
		reader = gz
		closer = func() { gz.Close(); f.Close() }
	}
	lines, err := readLastLines(reader, maxLines)
	if err != nil {
		closer()
		return nil, nil, err
	}
	closer()
	return strings.NewReader(strings.Join(lines, "\n")), func() {}, nil
}

func readLastLines(r io.Reader, max int) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	all := strings.Split(string(data), "\n")
	if len(all) <= max {
		return all, nil
	}
	return all[len(all)-max:], nil
}

