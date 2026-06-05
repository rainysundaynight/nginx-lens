package k8s

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------- Ingress-nginx аудит ----------
// Сопоставление K8s Ingress с nginx server/location.

// IngressIssue — проблема ingress.
type IngressIssue struct {
	Type    string `json:"type"`
	Host    string `json:"host,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

// IngressRule — распарсенное правило ingress.
type IngressRule struct {
	Host  string `json:"host"`
	Path  string `json:"path"`
	File  string `json:"file"`
}

// AuditIngressManifests читает YAML-манифесты и извлекает Ingress rules.
func AuditIngressManifests(manifestsPath string, nginxServerNames map[string]struct{}) ([]IngressIssue, []IngressRule) {
	var issues []IngressIssue
	var rules []IngressRule
	_ = filepath.Walk(manifestsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		docs := strings.Split(string(data), "---")
		for _, doc := range docs {
			doc = strings.TrimSpace(doc)
			if doc == "" {
				continue
			}
			var obj map[string]interface{}
			if yaml.Unmarshal([]byte(doc), &obj) != nil {
				continue
			}
			if obj["kind"] != "Ingress" {
				continue
			}
			spec, _ := obj["spec"].(map[string]interface{})
			if spec == nil {
				continue
			}
			tlsList, _ := spec["tls"].([]interface{})
			hasTLS := len(tlsList) > 0
			ingressRules, _ := spec["rules"].([]interface{})
			for _, r := range ingressRules {
				rm, _ := r.(map[string]interface{})
				if rm == nil {
					continue
				}
				host, _ := rm["host"].(string)
				rule := IngressRule{Host: host, File: path}
				http, _ := rm["http"].(map[string]interface{})
				if http != nil {
					paths, _ := http["paths"].([]interface{})
					for _, p := range paths {
						pm, _ := p.(map[string]interface{})
						if pm == nil {
							continue
						}
						rule.Path, _ = pm["path"].(string)
						rules = append(rules, rule)
						if host != "" {
							if _, ok := nginxServerNames[host]; !ok {
								issues = append(issues, IngressIssue{
									Type: "orphaned_ingress", Host: host, Path: rule.Path,
									Message: "Ingress host не найден в nginx server_name",
									File: path,
								})
							}
						}
					}
				}
				if !hasTLS && host != "" {
					issues = append(issues, IngressIssue{
						Type: "ingress_no_tls", Host: host,
						Message: "Ingress без TLS секции", File: path,
					})
				}
			}
		}
		return nil
	})
	return issues, rules
}

// CollectServerNames собирает server_name из nginx конфига.
func CollectServerNames(serverNames []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range serverNames {
		m[s] = struct{}{}
	}
	return m
}
