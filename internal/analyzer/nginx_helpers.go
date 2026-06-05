package analyzer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/parser"
)

// ParseListenPort извлекает номер порта из директивы listen.
func ParseListenPort(args string) int {
	for _, tok := range strings.Fields(strings.TrimSpace(args)) {
		if strings.HasPrefix(tok, "[") {
			if idx := strings.LastIndex(tok, "]:"); idx >= 0 {
				if p, err := strconv.Atoi(tok[idx+2:]); err == nil {
					return p
				}
			}
			continue
		}
		if idx := strings.Index(tok, ":"); idx > 0 {
			if p, err := strconv.Atoi(tok[idx+1:]); err == nil {
				return p
			}
		}
		if p, err := strconv.Atoi(tok); err == nil {
			return p
		}
	}
	return 0
}

// ListenHasSSL проверяет наличие ssl в listen.
func ListenHasSSL(args string) bool {
	for _, f := range strings.Fields(args) {
		if f == "ssl" {
			return true
		}
	}
	return false
}

// ListenPortIs443 — HTTPS-порт 443 (не 8443/1443).
func ListenPortIs443(args string) bool {
	return ParseListenPort(args) == 443
}

// NormalizeListenKey нормализует listen для сравнения server-блоков.
func NormalizeListenKey(args string) string {
	port := ParseListenPort(args)
	if port == 0 {
		port = 80
	}
	if ListenHasSSL(args) {
		return fmt.Sprintf("%d:ssl", port)
	}
	return fmt.Sprintf("%d", port)
}

// HasWeakSSLCiphers проверяет слабые шифры, игнорируя исключения (!MD5).
func HasWeakSSLCiphers(args string) bool {
	for _, part := range strings.Split(args, ":") {
		tok := strings.TrimSpace(part)
		if tok == "" || strings.HasPrefix(tok, "!") {
			continue
		}
		if strings.HasPrefix(tok, "+") {
			tok = strings.TrimSpace(tok[1:])
		}
		for _, weak := range []string{"RC4", "MD5", "DES"} {
			if tok == weak || strings.Contains(tok, weak) {
				return true
			}
		}
	}
	return false
}

// ServerScopeKey уникальный ключ server-блока для dead location и rewrite.
func ServerScopeKey(s parser.Node) string {
	if s.File != "" && s.Line > 0 {
		return fmt.Sprintf("%s:%d", s.File, s.Line)
	}
	return fmt.Sprintf("server:%s:%d", s.File, s.Line)
}

type configContext struct {
	httpTokensOff    bool
	httpLimitReq     bool
	httpLimitConn    bool
	hasLimitReqZone  bool
	hasLimitConnZone bool
}

func collectConfigContext(tree *parser.ConfigTree) configContext {
	var ctx configContext
	for _, item := range Walk(tree) {
		if isInsideServer(item) {
			continue
		}
		switch item.Node.Directive {
		case "server_tokens":
			if strings.TrimSpace(item.Node.Args) == "off" {
				ctx.httpTokensOff = true
			}
		case "limit_req":
			ctx.httpLimitReq = true
		case "limit_conn":
			ctx.httpLimitConn = true
		case "limit_req_zone":
			ctx.hasLimitReqZone = true
		case "limit_conn_zone":
			ctx.hasLimitConnZone = true
		}
	}
	return ctx
}

func isInsideServer(item WalkItem) bool {
	if item.Node.Block == "server" {
		return true
	}
	if item.Parent != nil && item.Parent.Block == "server" {
		return true
	}
	for _, a := range item.Ancestors {
		if a != nil && a.Block == "server" {
			return true
		}
	}
	return false
}

func serverHasSSLListen(s parser.Node) bool {
	for _, sub := range WalkNodes(s.Directives, &s) {
		if sub.Node.Directive == "listen" && ListenHasSSL(sub.Node.Args) {
			return true
		}
	}
	return false
}

func serverHasPublicListen(s parser.Node) bool {
	for _, sub := range WalkNodes(s.Directives, &s) {
		if sub.Node.Directive != "listen" {
			continue
		}
		port := ParseListenPort(sub.Node.Args)
		if port == 80 || port == 443 || port == 0 {
			return true
		}
	}
	return false
}

func LimitZonesPresent(tree *parser.ConfigTree) (reqZone, connZone bool) {
	c := collectConfigContext(tree)
	return c.hasLimitReqZone, c.hasLimitConnZone
}

func serverTokensEffectiveOff(s parser.Node, ctx configContext) bool {
	if ctx.httpTokensOff {
		return true
	}
	for _, sub := range WalkNodes(s.Directives, &s) {
		if sub.Node.Directive == "server_tokens" && strings.TrimSpace(sub.Node.Args) == "off" {
			return true
		}
	}
	return false
}
