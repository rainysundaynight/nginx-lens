package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ---------- Загрузчик конфигурации ----------
// Единый источник настроек для всех команд nginx-lens.

const envConfig = "NGINX_LENS_CONFIG"

// Config — полная конфигурация nginx-lens.
type Config struct {
	Defaults        DefaultsConfig        `yaml:"defaults"`
	Output          OutputConfig          `yaml:"output"`
	Cache           CacheConfig           `yaml:"cache"`
	Validate        ValidateConfig        `yaml:"validate"`
	Syntax          SyntaxConfig          `yaml:"syntax"`
	Analyze         AnalyzeConfig         `yaml:"analyze"`
	Health          HealthConfig          `yaml:"health"`
	Resolve         ResolveConfig         `yaml:"resolve"`
	Route           RouteConfig           `yaml:"route"`
	Logs            LogsConfig            `yaml:"logs"`
	Diff            DiffConfig            `yaml:"diff"`
	Metrics         MetricsConfig         `yaml:"metrics"`
	Upstreams       UpstreamsConfig       `yaml:"upstreams"`
	IncludeTree     IncludeTreeConfig     `yaml:"include_tree"`
	Tree            TreeConfig            `yaml:"tree"`
	Parser          ParserConfig          `yaml:"parser"`
	Explain         ExplainConfig         `yaml:"explain"`
	BlastRadius     BlastRadiusConfig     `yaml:"blast_radius"`
	Certs           CertsConfig           `yaml:"certs"`
	Policy          PolicyConfig          `yaml:"policy"`
	Score           ScoreConfig           `yaml:"score"`
	K8s             K8sConfig             `yaml:"k8s"`
	Docker          DockerConfig          `yaml:"docker"`
	DynamicUpstream DynamicUpstreamConfig `yaml:"dynamic_upstream"`
	Web             WebConfig             `yaml:"web"`
}

// DefaultsConfig — общие значения по умолчанию.
type DefaultsConfig struct {
	Timeout         float64 `yaml:"timeout"`
	Retries         int     `yaml:"retries"`
	Mode            string  `yaml:"mode"`
	MaxWorkers      int     `yaml:"max_workers"`
	DNSCacheTTL     int     `yaml:"dns_cache_ttl"`
	Top             int     `yaml:"top"`
	NginxConfigPath string  `yaml:"nginx_config_path"`
}

// OutputConfig — формат вывода результатов.
type OutputConfig struct {
	Colors bool   `yaml:"colors"`
	Format string `yaml:"format"`
}

// CacheConfig — DNS-кэш.
type CacheConfig struct {
	Enabled bool `yaml:"enabled"`
	TTL     int  `yaml:"ttl"`
}

// ValidateConfig — pipeline validate.
type ValidateConfig struct {
	SkipSyntax   bool   `yaml:"skip_syntax"`
	SkipAnalysis bool   `yaml:"skip_analysis"`
	SkipUpstream bool   `yaml:"skip_upstream"`
	SkipDNS      bool   `yaml:"skip_dns"`
	SkipCerts    bool   `yaml:"skip_certs"`
	SkipPolicy   bool   `yaml:"skip_policy"`
	SkipWarns    bool   `yaml:"skip_warns"`
	FailOnLow    bool   `yaml:"fail_on_low"`
	FailOn       string `yaml:"fail_on"`
	NginxPath    string `yaml:"nginx_path"`
}

// SyntaxConfig — проверка nginx -t.
type SyntaxConfig struct {
	SkipWarns bool   `yaml:"skip_warns"`
	NginxPath string `yaml:"nginx_path"`
}

// AnalyzeConfig — статический анализ.
type AnalyzeConfig struct {
	MinSeverity string   `yaml:"min_severity"`
	SkipTypes   []string `yaml:"skip_types"`
	SkipLow     bool     `yaml:"skip_low"`
	SkipMedium  bool     `yaml:"skip_medium"`
}

// HealthConfig — проверка upstream.
type HealthConfig struct {
	WithResolve         bool `yaml:"with_resolve"`
	SkipExitOnUnhealthy bool `yaml:"skip_exit_on_unhealthy"`
	SkipCache           bool `yaml:"skip_cache"`
}

// ResolveConfig — DNS-резолвинг upstream.
type ResolveConfig struct {
	SkipCache bool `yaml:"skip_cache"`
}

// RouteConfig — маршрутизация URL.
type RouteConfig struct {
	URL string `yaml:"url"`
}

// LogsConfig — анализ access-логов.
type LogsConfig struct {
	Path          string `yaml:"path"`
	ErrorPath     string `yaml:"error_path"`
	Top           int    `yaml:"top"`
	Since         string `yaml:"since"`
	Until         string `yaml:"until"`
	Status        string `yaml:"status"`
	SkipAnomalies bool   `yaml:"skip_anomalies"`
	FormatRegex   string `yaml:"format_regex"`
	TailLines     int    `yaml:"tail_lines"`
}

// DiffConfig — сравнение двух конфигов.
type DiffConfig struct {
	Config1 string `yaml:"config1"`
	Config2 string `yaml:"config2"`
}

// MetricsConfig — метрики конфигурации.
type MetricsConfig struct {
	ComparePath string `yaml:"compare_path"`
	Prometheus  bool   `yaml:"prometheus"`
}

// UpstreamsConfig — сводка upstream-блоков.
type UpstreamsConfig struct {
	Name         string `yaml:"name"`
	Health       bool   `yaml:"health"`
	SkipHealth   bool   `yaml:"skip_health"`
	HealthByDefault bool `yaml:"health_by_default"`
}

// IncludeTreeConfig — дерево include.
type IncludeTreeConfig struct {
	Directive string `yaml:"directive"`
}

// TreeConfig — визуализация дерева конфигурации.
type TreeConfig struct {
	Format string `yaml:"format"`
}

// ParserConfig — режим парсинга конфигурации.
type ParserConfig struct {
	Mode      string `yaml:"mode"`
	NginxPath string `yaml:"nginx_path"`
}

// ExplainConfig — пошаговое объяснение маршрутизации.
type ExplainConfig struct {
	URL string `yaml:"url"`
}

// BlastRadiusConfig — анализ blast-radius upstream.
type BlastRadiusConfig struct {
	UpstreamName string `yaml:"upstream_name"`
}

// CertsConfig — аудит SSL-сертификатов.
type CertsConfig struct {
	WarnDays      int  `yaml:"warn_days"`
	FailOnExpired bool `yaml:"fail_on_expired"`
}

// PolicyConfig — policy engine.
type PolicyConfig struct {
	Packs       []string       `yaml:"packs"`
	Rules       []PolicyRule   `yaml:"rules"`
	PolicyOnly  bool           `yaml:"policy_only"`
}

// PolicyRule — пользовательское правило.
type PolicyRule struct {
	ID       string `yaml:"id"`
	Match    string `yaml:"match"`
	Severity string `yaml:"severity"`
	Message  string `yaml:"message"`
	FixHint  string `yaml:"fix_hint"`
}

// ScoreConfig — рейтинг конфигурации.
type ScoreConfig struct {
	Enabled bool `yaml:"enabled"`
}

// K8sConfig — аудит ingress-nginx.
type K8sConfig struct {
	ManifestsPath string `yaml:"manifests_path"`
}

// DockerConfig — nginx в Docker-контейнере.
type DockerConfig struct {
	Enabled      string            `yaml:"enabled"`
	Container    string            `yaml:"container"`
	Binary       string            `yaml:"binary"`
	ConfigInside string            `yaml:"config_inside"`
	VolumeMap    map[string]string `yaml:"volume_map"`
}

// DynamicUpstreamConfig — ngx_dynamic_upstream API.
type DynamicUpstreamConfig struct {
	Enabled bool    `yaml:"enabled"`
	APIURL  string  `yaml:"api_url"`
	Timeout float64 `yaml:"timeout"`
}

// WebConfig — agent и hub.
type WebConfig struct {
	Agent WebAgentConfig `yaml:"agent"`
	Hub   WebHubConfig   `yaml:"hub"`
}

// WebAgentConfig — HTTP-агент.
type WebAgentConfig struct {
	Host  string `yaml:"host"`
	Port  int    `yaml:"port"`
	Token string `yaml:"token"`
}

// WebHubConfig — центральный hub.
type WebHubConfig struct {
	Host            string   `yaml:"host"`
	Port            int      `yaml:"port"`
	Token           string   `yaml:"token"`
	AgentToken      string   `yaml:"agent_token"`
	Agents          []string `yaml:"agents"`
	CORSOrigins     []string `yaml:"cors_origins"`
	RefreshInterval int      `yaml:"refresh_interval"`
}

// Loader загружает конфигурацию.
type Loader struct {
	Config     Config
	ConfigPath string
}

var (
	globalLoader *Loader
	loaderOnce   sync.Once
)

// DefaultConfig — значения по умолчанию при init.
func DefaultConfig() Config {
	return Config{
		Defaults: DefaultsConfig{
			Timeout:         2.0,
			Retries:         1,
			Mode:            "tcp",
			MaxWorkers:      10,
			DNSCacheTTL:     300,
			Top:             10,
			NginxConfigPath: "/etc/nginx/nginx.conf",
		},
		Output: OutputConfig{Colors: true, Format: "table"},
		Cache:  CacheConfig{Enabled: true, TTL: 300},
		Validate: ValidateConfig{
			SkipDNS:   true,
			NginxPath: "nginx",
		},
		Syntax:  SyntaxConfig{NginxPath: "nginx"},
		Analyze: AnalyzeConfig{MinSeverity: "low"},
		Logs: LogsConfig{
			Path:          "/var/log/nginx/access.log",
			SkipAnomalies: true,
		},
		Tree:   TreeConfig{Format: "text"},
		Parser: ParserConfig{Mode: "auto", NginxPath: "nginx"},
		Certs:  CertsConfig{WarnDays: 30},
		Score:  ScoreConfig{Enabled: true},
		Docker: DockerConfig{Enabled: "auto", Binary: "docker", ConfigInside: "/etc/nginx/nginx.conf"},
		DynamicUpstream: DynamicUpstreamConfig{
			APIURL:  "http://127.0.0.1:6000/dynamic",
			Timeout: 2.0,
		},
		Web: WebConfig{
			Agent: WebAgentConfig{Host: "0.0.0.0", Port: 8088},
			Hub: WebHubConfig{
				Host:            "0.0.0.0",
				Port:            8089,
				Agents:          []string{"http://localhost:8088"},
				CORSOrigins:     []string{"*"},
				RefreshInterval: 30,
			},
		},
	}
}

// Load загружает конфигурацию.
func Load() (*Loader, error) {
	cfg := DefaultConfig()
	path := findConfigFile()
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			_ = yaml.Unmarshal(data, &cfg)
		}
	}
	return &Loader{Config: cfg, ConfigPath: path}, nil
}

// Get возвращает глобальный загрузчик.
func Get() *Loader {
	loaderOnce.Do(func() {
		l, _ := Load()
		globalLoader = l
	})
	return globalLoader
}

// Reload перезагружает конфигурацию.
func Reload() {
	loaderOnce = sync.Once{}
	globalLoader = nil
	Get()
}

// RequireConfigFile проверяет наличие файла конфигурации nginx-lens.
func RequireConfigFile(loader *Loader) error {
	if loader.ConfigPath == "" {
		return fmt.Errorf("конфиг не найден: выполните nginx-lens init или задайте NGINX_LENS_CONFIG")
	}
	return nil
}

// NginxConfigPath возвращает путь к nginx.conf из конфига.
func NginxConfigPath(cfg *Config) (string, error) {
	path := cfg.Defaults.NginxConfigPath
	if path == "" {
		return "", fmt.Errorf("defaults.nginx_config_path не задан в конфиге")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("nginx.conf не найден: %s", path)
	}
	if info.IsDir() {
		return "", fmt.Errorf("defaults.nginx_config_path указывает на директорию: %s", path)
	}
	return path, nil
}

func findConfigFile() string {
	if env := os.Getenv(envConfig); env != "" {
		if info, err := os.Stat(env); err == nil && !info.IsDir() {
			return env
		}
	}
	candidates := []string{
		SystemConfigPath,
		"/opt/nginx-lens/config.yml",
		filepath.Join(".", ".nginx-lens.yaml"),
		filepath.Join(".", ".nginx-lens.yml"),
	}
	if p := UserConfigPath(); p != "" {
		candidates = append(candidates, p, filepath.Join(filepath.Dir(p), "config.yml"))
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}
