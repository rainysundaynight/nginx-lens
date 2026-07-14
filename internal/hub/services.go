package hub

import (
	"math"
	"sort"
	"strings"

	"github.com/rainysundaynight/nginx-lens/internal/logs"
)

// ---------- Services / Apps panel ----------
// Агрегация access.log: upstreams, endpoints, ok vs 5xx.

// HubServicesApps — вкладка «Приложения и сервисы» на Overview.
type HubServicesApps struct {
	TotalRequests   int              `json:"total_requests"`
	UniqueServices  int              `json:"unique_services"`
	TopSharePct     float64          `json:"top_share_pct"`
	TopService      string           `json:"top_service"`
	UpstreamSharePct float64         `json:"upstream_share_pct"`
	ErrorSharePct   float64          `json:"error_share_pct"`
	HasData         bool             `json:"has_data"`
	Upstreams       []HubServiceBar  `json:"upstreams"`
	Endpoints       []HubServiceBar  `json:"endpoints"`
	Quality         []HubServiceDual `json:"quality"`
}

// HubServiceBar — горизонтальный бар рейтинга.
type HubServiceBar struct {
	Name   string  `json:"name"`
	Count  int     `json:"count"`
	Pct    float64 `json:"pct"`
	Agent  string  `json:"agent,omitempty"`
	Errors int     `json:"errors,omitempty"`
}

// HubServiceDual — бар с двумя сегментами (ok / 5xx).
type HubServiceDual struct {
	Name    string  `json:"name"`
	Ok      int     `json:"ok"`
	Errors  int     `json:"errors"`
	Total   int     `json:"total"`
	OkPct   float64 `json:"ok_pct"`
	ErrPct  float64 `json:"err_pct"`
}

type serviceAgg struct {
	req, s5 int
	agent   string
}

// buildServicesApps агрегирует access_stats всех агентов.
func buildServicesApps(results []map[string]interface{}) HubServicesApps {
	upAgg := map[string]*serviceAgg{}
	pathAgg := map[string]*serviceAgg{}
	totalReq := 0
	total5xx := 0
	upstreamReq := 0
	directReq := 0

	for _, item := range results {
		online, _ := item["online"].(bool)
		if !online {
			continue
		}
		raw, _ := item["snapshot"].(map[string]interface{})
		if raw == nil {
			continue
		}
		agentName := agentLabelFromItem(item)
		access, _ := raw["access_stats"].(map[string]interface{})
		if access == nil {
			continue
		}
		totalReq += int(floatVal(access["total_requests"]))
		total5xx += int(floatVal(access["status_5xx"]))

		if byUp, ok := access["by_upstream"].(map[string]interface{}); ok {
			index := logs.BuildUpstreamIndex(rawUpstreams(raw))
			for key, v := range byUp {
				m, _ := v.(map[string]interface{})
				if m == nil {
					continue
				}
				req := int(floatVal(m["requests"]))
				s5 := int(floatVal(m["status_5xx"]))
				if key == "_direct" {
					directReq += req
					continue
				}
				upstreamReq += req
				name := logs.ResolveLogicalUpstream(key, index)
				if name == "" {
					name = prettyUpstreamName(key)
				}
				a := upAgg[name]
				if a == nil {
					a = &serviceAgg{agent: agentName}
					upAgg[name] = a
				}
				a.req += req
				a.s5 += s5
			}
		}

		if tops, ok := access["top_paths"].([]interface{}); ok {
			for _, row := range tops {
				m, _ := row.(map[string]interface{})
				if m == nil {
					continue
				}
				path, _ := m["path"].(string)
				if path == "" {
					continue
				}
				a := pathAgg[path]
				if a == nil {
					a = &serviceAgg{agent: agentName}
					pathAgg[path] = a
				}
				a.req += int(floatVal(m["requests"]))
				a.s5 += int(floatVal(m["status_5xx"]))
			}
		} else if byPath, ok := access["by_path"].(map[string]interface{}); ok {
			for path, v := range byPath {
				m, _ := v.(map[string]interface{})
				if m == nil {
					continue
				}
				a := pathAgg[path]
				if a == nil {
					a = &serviceAgg{agent: agentName}
					pathAgg[path] = a
				}
				a.req += int(floatVal(m["requests"]))
				a.s5 += int(floatVal(m["status_5xx"]))
			}
		}
	}

	out := HubServicesApps{
		TotalRequests:  totalReq,
		UniqueServices: len(upAgg),
		HasData:        totalReq > 0 || len(upAgg) > 0 || len(pathAgg) > 0,
	}
	if totalReq > 0 {
		out.ErrorSharePct = math.Round(float64(total5xx)/float64(totalReq)*1000) / 10
		routed := upstreamReq + directReq
		if routed > 0 {
			out.UpstreamSharePct = math.Round(float64(upstreamReq)/float64(routed)*1000) / 10
		}
	}

	out.Upstreams = rankServiceBars(upAgg, totalReq, 12)
	out.Endpoints = rankServiceBars(pathAgg, totalReq, 12)
	out.Quality = rankServiceDual(upAgg, 10)

	if len(out.Upstreams) > 0 {
		out.TopService = out.Upstreams[0].Name
		out.TopSharePct = out.Upstreams[0].Pct
	} else if len(out.Endpoints) > 0 {
		out.TopService = out.Endpoints[0].Name
		out.TopSharePct = out.Endpoints[0].Pct
		out.UniqueServices = len(pathAgg)
	}
	return out
}

func agentLabelFromItem(item map[string]interface{}) string {
	if label, _ := item["label"].(string); label != "" {
		return label
	}
	agentURL, _ := item["agent"].(string)
	raw, _ := item["snapshot"].(map[string]interface{})
	if raw != nil {
		if meta, _ := raw["meta"].(map[string]interface{}); meta != nil {
			if h, _ := meta["hostname"].(string); h != "" {
				return h
			}
		}
	}
	return agentSlug(agentURL)
}

func prettyUpstreamName(key string) string {
	key = strings.TrimSpace(key)
	if key == "" || key == "_direct" {
		return "direct"
	}
	// unix: или несколько адресов уже нормализованы в parser
	return key
}

func rankServiceBars(m map[string]*serviceAgg, total int, limit int) []HubServiceBar {
	out := make([]HubServiceBar, 0, len(m))
	for name, a := range m {
		if a.req <= 0 {
			continue
		}
		pct := 0.0
		if total > 0 {
			pct = math.Round(float64(a.req)/float64(total)*1000) / 10
		}
		out = append(out, HubServiceBar{
			Name: name, Count: a.req, Pct: pct, Agent: a.agent, Errors: a.s5,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	// Пересчёт pct относительно max для ширины бара (в UI можно использовать count/max)
	return out
}

func rankServiceDual(m map[string]*serviceAgg, limit int) []HubServiceDual {
	out := make([]HubServiceDual, 0, len(m))
	for name, a := range m {
		if a.req <= 0 {
			continue
		}
		ok := a.req - a.s5
		if ok < 0 {
			ok = 0
		}
		d := HubServiceDual{Name: name, Ok: ok, Errors: a.s5, Total: a.req}
		if a.req > 0 {
			d.OkPct = math.Round(float64(ok)/float64(a.req)*1000) / 10
			d.ErrPct = math.Round(float64(a.s5)/float64(a.req)*1000) / 10
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total == out[j].Total {
			return out[i].Name < out[j].Name
		}
		return out[i].Total > out[j].Total
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
