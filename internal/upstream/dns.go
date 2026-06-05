package upstream

import (
	"net"
	"strings"
)

// ---------- DNS-резолвинг ----------
// Резолвинг upstream-адресов с CNAME chain.

// ResolveAddress резолвит адрес upstream в IP-адреса.
func ResolveAddress(address string, useCache bool, cacheTTL int, cacheDir string) []string {
	hostPort := strings.Fields(address)[0]
	if !strings.Contains(hostPort, ":") {
		return nil
	}
	host, port, err := splitHostPort(hostPort)
	if err != nil {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		return []string{hostPort}
	}
	if useCache && IsCacheEnabled() {
		cache := GetCache(cacheTTL, cacheDir)
		if cached := cache.Get(host, port); cached != nil {
			return cached
		}
	}
	result := resolveWithNet(host, port)
	if useCache && IsCacheEnabled() {
		cache := GetCache(cacheTTL, cacheDir)
		cache.Set(host, port, result)
	}
	return result
}

func resolveWithNet(host, port string) []string {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil
	}
	var result []string
	for _, ip := range ips {
		if ip.To4() != nil {
			result = append(result, net.JoinHostPort(ip.String(), port))
		}
	}
	return result
}
