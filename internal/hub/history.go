package hub

import (
	"fmt"
	"sync"
	"time"
)

// ---------- KPI history ----------
// Кольцевой буфер сэмплов (~1ч) для deltas и availability uptime.

const (
	historyRetention = time.Hour
	historyMaxSamples = 200
)

// kpiSample — срез метрик на момент scrape.
type kpiSample struct {
	At              time.Time
	AgentsOnline    int
	AgentsTotal     int
	Critical        int
	Warnings        int
	UpstreamPct     float64 // -1 если нет данных
	AgentOnline     map[string]bool
}

var (
	historyMu      sync.Mutex
	historySamples []kpiSample
)

// recordKPISample сохраняет срез и подчищает старше 1ч.
func recordKPISample(s kpiSample) {
	historyMu.Lock()
	defer historyMu.Unlock()
	cutoff := time.Now().Add(-historyRetention)
	historySamples = append(historySamples, s)
	n := 0
	for _, sample := range historySamples {
		if sample.At.After(cutoff) {
			historySamples[n] = sample
			n++
		}
	}
	historySamples = historySamples[:n]
	if len(historySamples) > historyMaxSamples {
		historySamples = historySamples[len(historySamples)-historyMaxSamples:]
	}
}

// sampleNear1h возвращает сэмпл ~1ч назад (fallback — самый старый ≥5м).
func sampleNear1h() *kpiSample {
	historyMu.Lock()
	defer historyMu.Unlock()
	if len(historySamples) == 0 {
		return nil
	}
	target := time.Now().Add(-historyRetention)
	var best *kpiSample
	bestDist := time.Duration(1<<63 - 1)
	for i := range historySamples {
		d := historySamples[i].At.Sub(target)
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			bestDist = d
			best = &historySamples[i]
		}
	}
	if best != nil && bestDist <= 45*time.Minute {
		cp := *best
		return &cp
	}
	oldest := historySamples[0]
	if time.Since(oldest.At) < 5*time.Minute || len(historySamples) < 2 {
		return nil
	}
	cp := oldest
	return &cp
}

// agentAvailabilityPct — доля online-скрапов за час.
func agentAvailabilityPct(agentURL string) string {
	historyMu.Lock()
	defer historyMu.Unlock()
	if len(historySamples) < 2 {
		return ""
	}
	ok, total := 0, 0
	for _, s := range historySamples {
		v, exists := s.AgentOnline[agentURL]
		if !exists {
			continue
		}
		total++
		if v {
			ok++
		}
	}
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%.1f%%", float64(ok)/float64(total)*100)
}

// ---------- Delta helpers ----------
// Формирование HubKPIDelta относительно сэмпла ~1h ago.

func buildKPIDeltas(cur kpiSample, prev *kpiSample) *HubKPIDeltas {
	if prev == nil {
		return nil
	}
	return &HubKPIDeltas{
		AgentsOnline:    deltaInt(cur.AgentsOnline, prev.AgentsOnline, true),
		CriticalIssues:  deltaInt(cur.Critical, prev.Critical, false),
		Warnings:        deltaInt(cur.Warnings, prev.Warnings, false),
		UpstreamHealthy: deltaFloat(cur.UpstreamPct, prev.UpstreamPct, true),
	}
}

func deltaInt(cur, prev int, moreIsGood bool) *HubKPIDelta {
	d := cur - prev
	if d == 0 {
		return &HubKPIDelta{Value: "0", Trend: "flat", Positive: true}
	}
	trend := "up"
	if d < 0 {
		trend = "down"
	}
	positive := (d > 0) == moreIsGood
	sign := "+"
	if d < 0 {
		sign = ""
	}
	return &HubKPIDelta{
		Value:    fmt.Sprintf("%s%d", sign, d),
		Trend:    trend,
		Positive: positive,
	}
}

func deltaFloat(cur, prev float64, moreIsGood bool) *HubKPIDelta {
	if cur < 0 || prev < 0 {
		return nil
	}
	d := cur - prev
	if d > -0.05 && d < 0.05 {
		return &HubKPIDelta{Value: "0%", Trend: "flat", Positive: true}
	}
	trend := "up"
	if d < 0 {
		trend = "down"
	}
	positive := (d > 0) == moreIsGood
	return &HubKPIDelta{
		Value:    fmt.Sprintf("%+.1f%%", d),
		Trend:    trend,
		Positive: positive,
	}
}
