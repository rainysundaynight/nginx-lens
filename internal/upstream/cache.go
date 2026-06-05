package upstream

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ---------- DNS-кэш ----------
// Файловый кэш результатов DNS-резолвинга.

// DNSCache хранит результаты DNS-резолвинга.
type DNSCache struct {
	ttl       int
	cacheDir  string
	cacheFile string
	data      map[string]cacheEntry
	mu        sync.Mutex
}

type cacheEntry struct {
	Timestamp float64  `json:"timestamp"`
	Result    []string `json:"result"`
	Host      string   `json:"host"`
	Port      string   `json:"port"`
}

var (
	globalCache   *DNSCache
	cacheEnabled  = true
	cacheInitOnce sync.Once
)

// GetCache возвращает глобальный экземпляр DNS-кэша.
func GetCache(ttl int, cacheDir string) *DNSCache {
	cacheInitOnce.Do(func() {
		globalCache = newDNSCache(ttl, cacheDir)
	})
	if ttl != globalCache.ttl {
		globalCache.ttl = ttl
	}
	return globalCache
}

func newDNSCache(ttl int, cacheDir string) *DNSCache {
	dir := cacheDir
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache", "nginx-lens")
	}
	_ = os.MkdirAll(dir, 0755)
	c := &DNSCache{
		ttl:       ttl,
		cacheDir:  dir,
		cacheFile: filepath.Join(dir, "dns_cache.json"),
		data:      make(map[string]cacheEntry),
	}
	c.load()
	return c
}

func (c *DNSCache) cacheKey(host, port string) string {
	h := md5.Sum([]byte(host + ":" + port))
	return hex.EncodeToString(h[:])
}

func (c *DNSCache) load() {
	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &c.data)
}

func (c *DNSCache) save() {
	data, _ := json.Marshal(c.data)
	_ = os.WriteFile(c.cacheFile, data, 0644)
}

// Get возвращает результат из кэша или nil.
func (c *DNSCache) Get(host, port string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.cacheKey(host, port)
	entry, ok := c.data[key]
	if !ok {
		return nil
	}
	if time.Now().Unix()-int64(entry.Timestamp) > int64(c.ttl) {
		delete(c.data, key)
		c.save()
		return nil
	}
	return entry.Result
}

// Set сохраняет результат в кэш.
func (c *DNSCache) Set(host, port string, result []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.cacheKey(host, port)
	c.data[key] = cacheEntry{
		Timestamp: float64(time.Now().Unix()),
		Result:    result,
		Host:      host,
		Port:      port,
	}
	c.save()
}

// Clear очищает кэш.
func (c *DNSCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheEntry)
	_ = os.Remove(c.cacheFile)
}

// DisableCache отключает кэширование.
func DisableCache() { cacheEnabled = false }

// EnableCache включает кэширование.
func EnableCache() { cacheEnabled = true }

// IsCacheEnabled проверяет, включён ли кэш.
func IsCacheEnabled() bool { return cacheEnabled }
