package parser

// ---------- Типы узлов дерева конфигурации ----------
// Структуры данных для представления распарсенного nginx.conf.

// DirectiveOption — опция внутри upstream-блока (hash, keepalive и т.д.).
type DirectiveOption struct {
	Name string `json:"name"`
	Args string `json:"args"`
}

// Node — узел дерева конфигурации (блок, директива или upstream).
type Node struct {
	Block        string            `json:"block,omitempty"`
	Arg          string            `json:"arg,omitempty"`
	LocModifier  string            `json:"loc_modifier,omitempty"`
	Directive    string            `json:"directive,omitempty"`
	Args         string            `json:"args,omitempty"`
	Upstream     string            `json:"upstream,omitempty"`
	Servers      []string          `json:"servers,omitempty"`
	Options      []DirectiveOption `json:"options,omitempty"`
	Directives   []Node            `json:"directives,omitempty"`
	File         string            `json:"file,omitempty"`
	Line         int               `json:"line,omitempty"`
	DefaultServer bool             `json:"default_server,omitempty"`
}

// ConfigTree — корневое дерево конфигурации nginx.
type ConfigTree struct {
	Directives []Node            `json:"directives"`
	upstreams  map[string][]string

	dynamicEnabled bool
	dynamicAPIURL  string
	dynamicTimeout float64
}

// NewConfigTree создаёт пустое дерево конфигурации.
func NewConfigTree(directives []Node, upstreams map[string][]string) *ConfigTree {
	if upstreams == nil {
		upstreams = make(map[string][]string)
	}
	return &ConfigTree{Directives: directives, upstreams: upstreams}
}

// SetDynamicUpstreamConfig настраивает интеграцию с ngx_dynamic_upstream API.
func (t *ConfigTree) SetDynamicUpstreamConfig(enabled bool, apiURL string, timeout float64) {
	t.dynamicEnabled = enabled
	t.dynamicAPIURL = apiURL
	t.dynamicTimeout = timeout
}

// GetUpstreams возвращает upstream-серверы, обогащённые динамическими при необходимости.
func (t *ConfigTree) GetUpstreams() map[string][]string {
	raw := DedupeUpstreamMap(t.upstreams)
	if t.dynamicEnabled && t.dynamicAPIURL != "" {
		return enrichUpstreamsWithDynamic(raw, t.dynamicAPIURL, t.dynamicTimeout, true)
	}
	return raw
}

// RawUpstreams возвращает upstream без dynamic enrichment.
func (t *ConfigTree) RawUpstreams() map[string][]string {
	return t.upstreams
}
