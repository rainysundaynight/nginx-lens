package logs

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------- Custom log_format ----------
// Парсинг по regex из config.logs.format_regex.

// FormatSpec — шаблон парсинга строки лога.
type FormatSpec struct {
	Regex string
	re    *regexp.Regexp
	names []string
}

// NewFormatSpec создаёт парсер по именованным группам (?P<name>...).
func NewFormatSpec(pattern string) (*FormatSpec, error) {
	if pattern == "" {
		return nil, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	names := re.SubexpNames()
	var out []string
	for _, n := range names {
		if n != "" {
			out = append(out, n)
		}
	}
	return &FormatSpec{Regex: pattern, re: re, names: out}, nil
}

func (f *FormatSpec) parse(line string) *LogLine {
	if f == nil || f.re == nil {
		return nil
	}
	m := f.re.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	l := &LogLine{}
	for i, name := range f.re.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		val := m[i]
		switch name {
		case "status", "status_code":
			if n, err := strconv.Atoi(val); err == nil {
				l.Status = n
			}
		case "request_time", "rt", "request_time_s":
			if rt, err := strconv.ParseFloat(val, 64); err == nil {
				l.ResponseTime = rt
			}
		case "request_uri", "uri", "path":
			l.Path = val
		case "remote_addr", "ip":
			l.IP = val
		case "http_user_agent", "user_agent":
			l.UserAgent = val
		case "request_method", "method":
			l.Method = val
		case "upstream_addr", "upstream":
			l.Upstream = val
		case "time_local", "time":
			if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", val); err == nil {
				l.Time = t
			} else if t, err := time.Parse(time.RFC3339, val); err == nil {
				l.Time = t
			}
		default:
			if strings.Contains(name, "upstream") && l.Upstream == "" {
				l.Upstream = val
			}
		}
	}
	if l.Status == 0 && l.Path == "" && l.Upstream == "" {
		return nil
	}
	return l
}
