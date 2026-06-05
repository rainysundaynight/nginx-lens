package analyzer

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ---------- Route engine ----------
// Маршрутизация с приоритетами nginx location.

// RouteTraceStep — шаг объяснения маршрута.
type RouteTraceStep struct {
	Step    string `json:"step"`
	Detail  string `json:"detail"`
	Matched bool   `json:"matched"`
}

// ExplainResult — результат explain с trace.
type ExplainResult struct {
	URL        string           `json:"url"`
	Server     *parser.Node     `json:"server,omitempty"`
	Location   *parser.Node     `json:"location,omitempty"`
	ProxyPass  string           `json:"proxy_pass,omitempty"`
	Upstream   string           `json:"upstream,omitempty"`
	Trace      []RouteTraceStep `json:"trace"`
}

type locationCandidate struct {
	node     parser.Node
	modifier string
	path     string
	priority int
}

// ExplainRoute объясняет маршрутизацию URL с trace.
func ExplainRoute(tree *parser.ConfigTree, rawURL string) *ExplainResult {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	path := parsed.Path
	if path == "" {
		path = "/"
	}

	result := &ExplainResult{URL: rawURL}
	server, serverTrace := selectServer(tree, host, port)
	result.Trace = append(result.Trace, serverTrace...)
	if server == nil {
		return result
	}
	result.Server = server

	loc, locTrace, proxyPass, upstream := selectLocation(server, path)
	result.Trace = append(result.Trace, locTrace...)
	if loc != nil {
		result.Trace = append(result.Trace, traceLocationDirectives(loc)...)
		result.Location = loc
		result.ProxyPass = proxyPass
		result.Upstream = upstream
	}
	return result
}

func traceLocationDirectives(loc *parser.Node) []RouteTraceStep {
	var trace []RouteTraceStep
	for _, inner := range WalkNodes(loc.Directives, loc) {
		switch inner.Node.Directive {
		case "rewrite":
			trace = append(trace, RouteTraceStep{
				Step: "rewrite", Detail: inner.Node.Args, Matched: true,
			})
		case "try_files":
			trace = append(trace, RouteTraceStep{
				Step: "try_files", Detail: inner.Node.Args, Matched: true,
			})
		case "return":
			trace = append(trace, RouteTraceStep{
				Step: "return", Detail: inner.Node.Args, Matched: true,
			})
		}
	}
	return trace
}

func selectServer(tree *parser.ConfigTree, host, port string) (*parser.Node, []RouteTraceStep) {
	var trace []RouteTraceStep
	type candidate struct {
		node  parser.Node
		score int
		reason string
	}
	var candidates []candidate

	for _, item := range Walk(tree) {
		if item.Node.Block != "server" {
			continue
		}
		var names, listens []string
		defaultServer := false
		for _, sub := range WalkNodes(item.Node.Directives, &item.Node) {
			if sub.Node.Directive == "server_name" {
				names = append(names, strings.Fields(sub.Node.Args)...)
			}
			if sub.Node.Directive == "listen" {
				listens = append(listens, sub.Node.Args)
				if strings.Contains(sub.Node.Args, "default_server") {
					defaultServer = true
				}
			}
		}
		score := 0
		reason := ""
		for _, n := range names {
			if hostMatch(host, n) {
				score += 10
				reason = "server_name=" + n
				break
			}
		}
		for _, l := range listens {
			if listenMatchesPort(l, port) {
				score += 5
				if reason == "" {
					reason = "listen=" + l
				}
				break
			}
		}
		if defaultServer && score == 0 {
			score = 1
			reason = "default_server"
		}
		if score > 0 {
			node := item.Node
			node.DefaultServer = defaultServer
			candidates = append(candidates, candidate{node: node, score: score, reason: reason})
		}
	}
	if len(candidates) == 0 {
		trace = append(trace, RouteTraceStep{Step: "server", Detail: "нет подходящего server-блока", Matched: false})
		return nil, trace
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	best := candidates[0]
	trace = append(trace, RouteTraceStep{Step: "server", Detail: best.reason, Matched: true})
	node := best.node
	return &node, trace
}

func selectLocation(server *parser.Node, path string) (*parser.Node, []RouteTraceStep, string, string) {
	var trace []RouteTraceStep
	var candidates []locationCandidate

	for _, sub := range WalkNodes(server.Directives, server) {
		if sub.Node.Block != "location" {
			continue
		}
		mod := sub.Node.LocModifier
		locPath := sub.Node.Arg
		if mod == "" {
			mod, locPath = parser.ParseLocationArg(sub.Node.Arg)
		}

		matched := false
		priority := 0
		switch mod {
		case "=":
			matched = path == locPath
			priority = 1000
		case "^~":
			matched = strings.HasPrefix(path, locPath)
			priority = 800 + len(locPath)
		case "~", "~*":
			re, err := regexp.Compile(locPath)
			if mod == "~*" {
				re, err = regexp.Compile("(?i)" + locPath)
			}
			if err == nil {
				matched = re.MatchString(path)
			}
			priority = 600
		default:
			matched = strings.HasPrefix(path, locPath)
			priority = 400 + len(locPath)
		}
		if matched {
			node := sub.Node
			candidates = append(candidates, locationCandidate{node: node, modifier: mod, path: locPath, priority: priority})
			trace = append(trace, RouteTraceStep{
				Step: "location_candidate", Detail: mod + locPath + " (priority " + strconv.Itoa(priority) + ")", Matched: true,
			})
		}
	}
	if len(candidates) == 0 {
		trace = append(trace, RouteTraceStep{Step: "location", Detail: "нет подходящего location", Matched: false})
		return nil, trace, "", ""
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].priority > candidates[j].priority })
	best := candidates[0]
	trace = append(trace, RouteTraceStep{
		Step: "location", Detail: "выбран " + best.modifier + best.path, Matched: true,
	})
	node := best.node
	var proxyPass, upstream string
	for _, inner := range WalkNodes(node.Directives, &node) {
		if inner.Node.Directive == "proxy_pass" {
			proxyPass = inner.Node.Args
			upstream = extractUpstreamName(proxyPass)
		}
	}
	return &node, trace, proxyPass, upstream
}

func listenMatchesPort(listen, port string) bool {
	return strings.Contains(listen, ":"+port) || strings.HasSuffix(listen, " "+port) ||
		(port == "80" && !strings.Contains(listen, ":") && !strings.Contains(listen, "ssl"))
}

func extractUpstreamName(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		rest := strings.SplitN(value, "://", 2)[1]
		if idx := strings.Index(rest, "/"); idx >= 0 {
			rest = rest[:idx]
		}
		return rest
	}
	parts := strings.Fields(value)
	if len(parts) > 0 && !strings.Contains(parts[0], "://") {
		return parts[0]
	}
	return ""
}
