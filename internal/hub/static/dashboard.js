(function () {
  "use strict";

  const REFRESH_MS = window.NGINX_LENS_REFRESH || 30000;
  // ---------- Navigation ----------
  // Группы Monitor / Analyze и иконки пунктов сайдбара
  const NAV_GROUPS = [
    {
      label: "Мониторинг",
      items: [
        { label: "Обзор", path: "", icon: "activity", badge: null },
        { label: "Агенты", path: "agents", icon: "layers", badge: "agents" },
      ],
    },
    {
      label: "Анализ",
      items: [
        { label: "Конфигурации", path: "nodes", icon: "camera", badge: null },
        { label: "Корреляция", path: "correlation", icon: "git-branch", badge: null },
        { label: "Зона поражения", path: "blast-radius", icon: "radio", badge: "blast" },
      ],
    },
  ];
  const TABS = [
    { id: "Upstream", label: "Upstream" },
    { id: "Build", label: "Сборка" },
    { id: "Certs", label: "Сертификаты" },
    { id: "Blast-radius", label: "Зона поражения" },
    { id: "Errors", label: "Ошибки" },
    { id: "Explore", label: "Маршрут" },
  ];
  const ICONS = {
    activity: '<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>',
    layers: '<polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/>',
    camera: '<path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/>',
    "git-branch": '<line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>',
    radio: '<path d="M4.9 19.1C1 15.2 1 8.8 4.9 4.9"/><path d="M7.8 16.2c-2.3-2.3-2.3-6.1 0-8.5"/><circle cx="12" cy="12" r="2"/><path d="M16.2 7.8c2.3 2.3 2.3 6.1 0 8.5"/><path d="M19.1 4.9C23 8.8 23 15.1 19.1 19"/>',
    users: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>',
    "alert-circle": '<circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>',
    "alert-triangle": '<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>',
    "trending-up": '<polyline points="22 7 13.5 15.5 8.5 10.5 2 17"/><polyline points="16 7 22 7 22 13"/>',
    "trending-down": '<polyline points="22 17 13.5 8.5 8.5 13.5 2 7"/><polyline points="16 17 22 17 22 11"/>',
  };

  let state = null;
  let route = parseRoute();
  let detailTab = "Upstream";
  let searchQuery = "";
  let overviewTab = "fleet"; // fleet | services
  let animate = true;
  let viewMounted = false;

  const $ = (sel) => document.querySelector(sel);

  function animAttr(delay) {
    if (!animate) return delay != null ? ` style="animation-delay:${delay}ms"` : "";
    return delay != null ? ` class="animate-enter" style="animation-delay:${delay}ms"` : ` class="animate-enter"`;
  }

  function routeKey() {
    if (route.page === "snapshot-detail") return "snapshot-detail:" + route.id;
    return route.page;
  }

  function esc(s) {
    return String(s ?? "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function icon(name, cls) {
    const p = ICONS[name] || "";
    return `<svg class="${cls || "nav-icon"}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">${p}</svg>`;
  }

  function navBadge(kind) {
    if (!state) return "";
    if (kind === "agents") return String((state.snapshots || []).length || (state.meta && state.meta.agents_total) || 0);
    if (kind === "blast") return String((state.blast_radius || []).length || 0);
    return "";
  }

  function token() {
    return localStorage.getItem("nginx_lens_hub_token") || "";
  }

  function headers() {
    const h = {};
    const t = token();
    if (t) h["X-Nginx-Lens-Token"] = t;
    return h;
  }

  async function fetchState() {
    const r = await fetch("/api/v1/hub/state", { headers: headers() });
    if (r.status === 401) {
      const err = new Error("unauthorized");
      err.code = 401;
      throw err;
    }
    if (!r.ok) throw new Error("HTTP " + r.status);
    return r.json();
  }

  function parseRoute() {
    const hash = (location.hash || "#/").replace(/^#\/?/, "");
    const parts = hash.split("/").filter(Boolean);
    if ((parts[0] === "nodes" || parts[0] === "snapshots") && parts[1]) {
      return { page: "snapshot-detail", id: decodeURIComponent(parts[1]) };
    }
    if (parts[0] === "snapshots") return { page: "nodes" };
    if (parts[0]) return { page: parts[0] };
    return { page: "overview" };
  }

  function navigate(path) {
    location.hash = path ? "#/" + path : "#/";
  }

  function scoreTone(n) {
    if (n >= 90) return "t-highlight";
    if (n >= 70) return "t-primary";
    if (n >= 50) return "t-warning";
    return "t-danger";
  }

  function catBarClass(n) {
    if (n >= 70) return "t-primary";
    if (n >= 50) return "t-warning";
    return "t-danger";
  }

  function impactClass(v) {
    if (v >= 60) return "impact-high";
    if (v >= 20) return "impact-med";
    return "impact-low";
  }

  function impactText(v) {
    if (v >= 60) return "t-danger";
    if (v >= 20) return "t-warning";
    return "t-highlight";
  }

  function agentStatusLabel(st) {
    if (st === "offline") return { label: "CRITICAL", cls: "badge-critical", critical: true };
    if (st === "warning") return { label: "WARNING", cls: "badge-warning", critical: false };
    return { label: "ONLINE", cls: "badge-online", critical: false };
  }

  function statusPill(st) {
    const map = {
      OK: "ok",
      DOWN: "down",
      WARN: "warn",
      "—": "warn",
    };
    const cls = map[st] || "warn";
    return `<span class="status-pill ${cls}"><span style="width:6px;height:6px;border-radius:50%;background:currentColor"></span>${esc(st)}</span>`;
  }

  function filterSnapshots(list) {
    if (!searchQuery) return list;
    const q = searchQuery.toLowerCase();
    return list.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        s.id.toLowerCase().includes(q) ||
        s.url.toLowerCase().includes(q) ||
        s.host.toLowerCase().includes(q) ||
        (s.upstreams || []).some((u) => u.name.toLowerCase().includes(q))
    );
  }

  function snapPreview(s) {
    if (s.status === "offline") return { label: "Ошибка", text: s.error || "Connection refused", bad: true };
    if (s.access) {
      return {
        label: "Access",
        text: "5xx " + s.access.status_5xx + " · p95 " + Math.round(s.access.p95_ms) + "ms",
        bad: (s.access.status_5xx || 0) > 0,
      };
    }
    if ((s.upstreams || []).length) {
      return { label: "Upstream", text: s.upstreams[0].name + " · " + (s.upstreams[0].status || "—"), bad: false };
    }
    return { label: "Статус", text: "online", bad: false };
  }

  function renderNav() {
    const nav = $("#nav");
    nav.innerHTML = NAV_GROUPS.map((group) => {
      const items = group.items
        .map((item) => {
          const active =
            (item.path === "" && route.page === "overview") ||
            route.page === item.path ||
            (item.path === "nodes" && route.page === "snapshot-detail");
          const badge = item.badge ? navBadge(item.badge) : "";
          return `<button type="button" class="nav-item${active ? " active" : ""}" data-nav="${esc(item.path)}">
            ${icon(item.icon)}
            <span>${esc(item.label)}</span>
            ${badge ? `<span class="nav-badge">${esc(badge)}</span>` : ""}
          </button>`;
        })
        .join("");
      return `<div class="nav-group"><div class="nav-group-label">${esc(group.label)}</div>${items}</div>`;
    }).join("");
    nav.querySelectorAll("[data-nav]").forEach((el) => {
      el.addEventListener("click", () => navigate(el.dataset.nav));
    });
  }

  function pageHeader(eyebrow, title, desc, actions, tone) {
    const toneCls = eyebrowToneClass(tone);
    return `<div class="page-header"${animAttr()}>
      <div>
        ${eyebrow ? `<div class="page-eyebrow${toneCls}"><span class="page-eyebrow-dot"></span>${esc(eyebrow)}</div>` : ""}
        <h1 class="page-title">${esc(title)}</h1>
        ${desc ? `<p class="page-desc">${esc(desc)}</p>` : ""}
      </div>
      ${actions || ""}
    </div>`;
  }

  function eyebrowToneClass(tone) {
    if (tone === "offline" || tone === "critical") return " is-offline";
    if (tone === "warning" || tone === "degraded") return " is-warning";
    return "";
  }

  function renderKpiCards(k, delay) {
    const d = k.deltas || {};
    const cards = [
      { label: "Агенты Online", value: k.agents_online, suffix: k.agents_suffix, tone: "t-primary", iconTone: "primary", icon: "users", delta: d.agents_online },
      { label: "Upstream Healthy", value: k.upstream_healthy, tone: "t-highlight", iconTone: "highlight", icon: "activity", delta: d.upstream_healthy },
    ];
    return `<section class="kpi-grid kpi-grid-2">${cards
      .map(
        (c, i) => `<div class="kpi-card"${animAttr(delay + i * 80)}>
          <div class="kpi-ribbon ${c.iconTone}" aria-hidden="true"></div>
          <div class="kpi-top">
            <div class="kpi-icon ${c.iconTone}">${icon(c.icon)}</div>
          </div>
          <div class="kpi-body">
            <div class="kpi-label">${esc(c.label)}</div>
            <div class="kpi-row">
              <span class="kpi-value ${c.tone}">${esc(c.value)}</span>
              ${c.suffix ? `<span class="kpi-suffix">${esc(c.suffix)}</span>` : ""}
            </div>
            ${renderKpiDelta(c.delta)}
          </div>
        </div>`
      )
      .join("")}</section>`;
  }

  function renderKpiDelta(delta) {
    if (!delta || !delta.value) return "";
    const trend = delta.trend || "flat";
    const sens = delta.positive ? "good" : "bad";
    const trendIcon = trend === "down" ? "trending-down" : "trending-up";
    return `<div class="kpi-delta-row">
      <span class="kpi-delta ${trend} ${sens}">${icon(trendIcon)}${esc(delta.value)}</span>
      <span class="kpi-delta-hint">vs. 1h ago</span>
    </div>`;
  }

  function shortHost(name, maxLen) {
    const s = String(name || "");
    const max = maxLen || 16;
    if (s.length <= max) return s;
    const keep = Math.max(3, Math.floor((max - 1) / 2));
    return s.slice(0, keep) + "…" + s.slice(-keep);
  }

  function renderHealthOverview(bars) {
    if (!bars || !bars.length) {
      return `<div class="panel health-panel"${animAttr(120)}><div class="empty">Нет данных upstream</div></div>`;
    }
    const now = new Date();
    const labels = bars.map((_, i) => {
      if (i === bars.length - 1) return { text: "сейчас", now: true };
      const m = new Date(now - (bars.length - 1 - i) * 60000);
      return { text: m.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }), now: false };
    });
    const axisSparse = labels.filter((_, i) => i === 0 || i === Math.floor(labels.length / 2) || i === labels.length - 1);
    return `<div class="panel health-panel"${animAttr(120)}>
      <div class="panel-head">
        <div>
          <h3 class="panel-title">Состояние агентов</h3>
          <p class="panel-sub">Агрегированные health-сигналы по агентам · live</p>
        </div>
        <span class="panel-badge"><span class="panel-badge-dot"></span>Live</span>
      </div>
      <div class="health-legend">
        <div class="health-legend-item"><span class="health-legend-swatch ok"></span>Healthy</div>
        <div class="health-legend-item"><span class="health-legend-swatch bad"></span>Incident</div>
      </div>
      <div class="health-chart-wrap">
        <div class="health-grid" aria-hidden="true"><span></span><span></span><span></span><span></span></div>
        <div class="health-chart">${bars
          .map((b) => {
            const h = Math.max(8, Math.min(100, b.pct < 0 ? 50 : b.pct));
            return `<div class="health-bar ${b.bad ? "bad" : "ok"}" style="height:${h}%" title="${esc(b.agent)}: ${Math.round(b.pct)}%"></div>`;
          })
          .join("")}</div>
      </div>
      <div class="health-host-legend">${bars
        .map((b) => `<span class="health-host-tick" title="${esc(b.agent)}">${esc(shortHost(b.agent, 14))}</span>`)
        .join("")}</div>
      <div class="health-axis">${axisSparse
        .map((l) => `<span class="${l.now ? "now" : ""}">${esc(l.now ? "сейчас" : l.text)}</span>`)
        .join("")}</div>
    </div>`;
  }

  function renderSeverityPanel(sev) {
    const total = (sev.high || 0) + (sev.medium || 0) + (sev.low || 0);
    const rows = [
      { label: "High severity", pct: sev.high_pct, count: sev.high, fill: "severity-fill-high", text: "t-danger", dot: "high" },
      { label: "Medium severity", pct: sev.medium_pct, count: sev.medium, fill: "severity-fill-med", text: "t-warning", dot: "med" },
      { label: "Low severity", pct: sev.low_pct, count: sev.low, fill: "severity-fill-low", text: "t-primary", dot: "low" },
    ];
    return `<div class="panel severity-panel-wrap"${animAttr(180)}>
      <div class="panel-head">
        <div>
          <h3 class="panel-title">Severity Breakdown</h3>
          <p class="panel-sub">Активные issues по флоту</p>
        </div>
        <div class="panel-total">
          <div class="panel-total-val">${total}</div>
          <div class="panel-total-label">total</div>
        </div>
      </div>
      <div class="severity-panel">${rows
        .map(
          (r) => `<div>
            <div class="severity-row-head">
              <div class="severity-row-label"><span class="severity-row-dot ${r.dot}"></span>${esc(r.label)}</div>
              <div class="severity-row-vals">
                <span class="severity-row-count ${r.text}">${r.count}</span>
                <span class="severity-row-pct">${r.pct}%</span>
              </div>
            </div>
            <div class="severity-track"><div class="${r.fill}" style="width:${r.pct}%"></div></div>
          </div>`
        )
        .join("")}</div>
    </div>`;
  }

  function renderAgentsFeed(snaps) {
    const list = filterSnapshots(snaps);
    const shown = list.slice(0, 10);
    return `<section${animAttr(240)}>
      <div class="section-head">
        <div>
          <h2 class="section-title">Активные агенты</h2>
          <p class="section-sub">Мониторинг ключевых узлов · клик — детальная конфигурация</p>
        </div>
        <div class="section-meta">1–${shown.length} of ${list.length}</div>
      </div>
      <div class="agents-feed">${shown.length ? shown.map(renderAgentRow).join("") : '<div class="empty">Агенты не настроены</div>'}</div>
    </section>`;
  }

  function renderAgentRow(s, idx) {
    const st = agentStatusLabel(s.status);
    const prev = snapPreview(s);
    const i = String((idx ?? 0) + 1).padStart(2, "0");
    const action = st.critical ? "Корреляция" : "Открыть";
    const btnCls = st.critical ? "btn-outline btn-danger-outline" : "btn-outline";
    return `<div class="agent-row${st.critical ? " critical" : ""}" data-snap="${esc(s.id)}" data-action="${st.critical ? "correlation" : "detail"}">
      <div class="agent-idx ${st.critical ? "bad" : "ok"}">${i}</div>
      <div class="agent-main">
        <div class="agent-name-row">
          <span class="agent-name">${esc(s.name)}</span>
          <span class="badge ${st.cls}">${st.label}</span>
        </div>
        <div class="agent-sub${st.critical ? " bad" : ""}">${esc(s.host)} · ${esc(st.critical ? (s.error || "unreachable") : "v" + (s.version || "—"))}</div>
      </div>
      <div class="agent-preview">
        <div class="agent-preview-label">${esc(prev.label)}</div>
        <div class="agent-preview-text${prev.bad ? " bad" : prev.empty ? " empty" : ""}">${esc(prev.text)}</div>
      </div>
      <div class="agent-action"><button type="button" class="${btnCls}" data-snap="${esc(s.id)}" data-action="${st.critical ? "correlation" : "detail"}">${action}</button></div>
    </div>`;
  }

  function renderAnalyticsPanels() {
    const corr = (state.correlations || [])[0];
    const blastItems = [];
    for (const g of state.blast_radius || []) {
      for (const l of g.locations || []) {
        blastItems.push({ loc: g.upstream + " → " + l.loc, impact: l.impact });
      }
    }
    blastItems.sort((a, b) => b.impact - a.impact);
    const topBlast = blastItems.slice(0, 3);
    return `<div class="analytics-grid"${animAttr(300)}>
      <div class="analytics-panel">
        <h4 class="analytics-title">Корреляция error.log (upstream → error log)</h4>
        ${
          corr
            ? `<div class="analytics-log">
                <div class="t-danger" style="opacity:0.8;margin-bottom:0.25rem">[error] ${esc(corr.error)}</div>
                <div class="analytics-indent">↳ upstream [${esc(corr.upstream)}]<br/>↳ location: ${esc((corr.locations && corr.locations[0]) || "—")}</div>
              </div>`
            : '<div class="muted" style="font-family:var(--font-mono);font-size:0.6875rem">Нет ошибок для корреляции</div>'
        }
      </div>
      <div class="analytics-panel">
        <h4 class="analytics-title">Blast-radius (upstream → location)</h4>
        ${
          topBlast.length
            ? topBlast
                .map(
                  (b) => `<div class="blast-line">
                    <span>LOCATION ${esc(b.loc)}</span>
                    <span class="${impactText(b.impact)}">IMPACT: ${b.impact.toFixed(1)}%</span>
                  </div>`
                )
                .join("")
            : '<div class="muted" style="font-family:var(--font-mono);font-size:0.6875rem">Нет dependency graph</div>'
        }
      </div>
    </div>`;
  }

  function renderOverview() {
    return `
      ${renderOverviewTabs()}
      <div id="overview-pane">
        ${overviewTab === "services" ? renderServicesApps() : renderFleetOverview()}
      </div>
    `;
  }

  function renderOverviewTabs() {
    const svc = state.services || {};
    const meta = svc.has_data
      ? formatCount(svc.total_requests || 0) + " req · " + (svc.unique_services || 0) + " services"
      : "access.log";
    return `<div class="overview-tabs"${animAttr()}>
      <div class="overview-tab-list">
        <button type="button" class="overview-tab${overviewTab === "fleet" ? " active" : ""}" data-overview-tab="fleet">Обзор</button>
        <button type="button" class="overview-tab${overviewTab === "services" ? " active" : ""}" data-overview-tab="services">Приложения и сервисы</button>
      </div>
      <div class="overview-tab-meta">${esc(meta)}</div>
    </div>`;
  }

  function renderFleetOverview() {
    const snaps = state.snapshots || [];
    return `<div class="fleet-stack">
      ${renderKpiCards(state.kpi, 0)}
      ${renderHealthOverview(state.health_bars)}
      ${renderAgentsFeed(snaps)}
      ${renderAnalyticsPanels()}
    </div>`;
  }

  function formatCount(n) {
    n = Number(n) || 0;
    if (n >= 1e6) return (n / 1e6).toFixed(1) + "M";
    if (n >= 1e3) return (n / 1e3).toFixed(1) + "k";
    return String(n);
  }

  function renderServicesApps() {
    const svc = state.services || {};
    if (!svc.has_data) {
      return `<div class="empty-dashed"${animAttr(40)}>
        Нет данных access.log. Укажите <code>logs.path</code> у агентов и убедитесь, что в логе есть upstream_addr / URI.
      </div>`;
    }
    const tops = svc.upstreams || [];
    const maxUp = tops.length ? tops[0].count : 1;
    const ends = svc.endpoints || [];
    const maxEnd = ends.length ? ends[0].count : 1;
    const quality = svc.quality || [];
    const maxQ = quality.length ? quality[0].total : 1;

    return `
      <div class="svc-kpi-grid"${animAttr(40)}>
        <div class="svc-kpi">
          <div class="svc-kpi-label">Всего запросов</div>
          <div class="svc-kpi-value t-primary">${esc(formatCount(svc.total_requests))}</div>
          <div class="svc-kpi-sub">за окно access.log</div>
        </div>
        <div class="svc-kpi">
          <div class="svc-kpi-label">Уникальных сервисов</div>
          <div class="svc-kpi-value" style="color:oklch(0.55 0.14 250)">${esc(String(svc.unique_services || 0))}</div>
          <div class="svc-kpi-sub">upstream / backends</div>
        </div>
        <div class="svc-kpi">
          <div class="svc-kpi-label">Top share</div>
          <div class="svc-kpi-value t-highlight">${esc((svc.top_share_pct || 0).toFixed(1))}%</div>
          <div class="svc-kpi-sub">${esc(svc.top_service || "—")}</div>
        </div>
        <div class="svc-kpi">
          <div class="svc-kpi-label">Upstream traffic</div>
          <div class="svc-kpi-value t-warning">${esc((svc.upstream_share_pct || 0).toFixed(1))}%</div>
          <div class="svc-kpi-sub">5xx rate ${esc((svc.error_share_pct || 0).toFixed(1))}%</div>
        </div>
      </div>
      <div class="svc-charts"${animAttr(100)}>
        <div class="svc-chart">
          <h3 class="svc-chart-title">Upstreams</h3>
          <div class="svc-chart-sub">Backend-сервисы по числу запросов</div>
          <div class="svc-rows">${
            tops.length
              ? tops.map((r, i) => serviceBarRow(r, maxUp, "upstream", i === 0)).join("")
              : '<div class="empty">Нет upstream в логах</div>'
          }</div>
          <div class="svc-axis"><span>0</span><span>${esc(formatCount(Math.round(maxUp / 2)))}</span><span>${esc(formatCount(maxUp))}</span></div>
        </div>
        <div class="svc-chart">
          <h3 class="svc-chart-title">Endpoints</h3>
          <div class="svc-chart-sub">URI / locations по частоте</div>
          <div class="svc-rows">${
            ends.length
              ? ends.map((r, i) => serviceBarRow(r, maxEnd, "endpoint", i === 0)).join("")
              : '<div class="empty">Нет path-статистики</div>'
          }</div>
          <div class="svc-axis"><span>0</span><span>${esc(formatCount(Math.round(maxEnd / 2)))}</span><span>${esc(formatCount(maxEnd))}</span></div>
        </div>
        <div class="svc-chart">
          <h3 class="svc-chart-title">Качество ответов</h3>
          <div class="svc-chart-sub">OK vs 5xx по сервисам</div>
          <div class="svc-rows">${
            quality.length
              ? quality.map((r) => serviceDualRow(r, maxQ)).join("")
              : '<div class="empty">Нет данных</div>'
          }</div>
          <div class="svc-legend">
            <div class="svc-legend-item"><span class="svc-legend-swatch ok"></span>2xx–4xx</div>
            <div class="svc-legend-item"><span class="svc-legend-swatch err"></span>5xx</div>
          </div>
        </div>
      </div>
    `;
  }

  function serviceBarRow(r, max, tone, showVal) {
    const w = max > 0 ? Math.max(2, (r.count / max) * 100) : 0;
    return `<div class="svc-row">
      <div class="svc-row-head">
        <span class="svc-row-name" title="${esc(r.name)}">${esc(r.name)}</span>
        <span class="svc-row-val${showVal ? " strong" : ""}">${esc(formatCount(r.count))}${r.pct ? " · " + r.pct + "%" : ""}</span>
      </div>
      <div class="svc-track"><div class="svc-fill ${tone}" style="width:${w}%"></div></div>
    </div>`;
  }

  function serviceDualRow(r, max) {
    const totalW = max > 0 ? Math.max(4, (r.total / max) * 100) : 0;
    const okShare = r.total > 0 ? (r.ok / r.total) * 100 : 0;
    const errShare = r.total > 0 ? (r.errors / r.total) * 100 : 0;
    return `<div class="svc-row">
      <div class="svc-row-head">
        <span class="svc-row-name" title="${esc(r.name)}">${esc(r.name)}</span>
        <span class="svc-row-val">${esc(formatCount(r.total))}</span>
      </div>
      <div class="svc-track" style="width:${totalW}%">
        <div class="svc-fill ok" style="width:${okShare}%;border-radius:0"></div>
        <div class="svc-fill err" style="width:${errShare}%;border-radius:0"></div>
      </div>
    </div>`;
  }

  function renderAgents() {
    const snaps = filterSnapshots(state.snapshots || []);
    const total = snaps.length;
    const online = snaps.filter((s) => s.status === "online").length;
    const warning = snaps.filter((s) => s.status === "warning").length;
    const critical = snaps.filter((s) => s.status === "offline").length;
    const stats = [
      { label: "Total", value: String(total), tone: "t-highlight" },
      { label: "Online", value: String(online), tone: "t-primary" },
      { label: "Warning", value: String(warning), tone: "t-warning" },
      { label: "Critical", value: String(critical), tone: "t-danger" },
    ];
    return `
      ${pageHeader(
        "Раздел · 01",
        "Агенты",
        "Парк nginx-агентов из web.hub.agents. Кликните по строке для детальной конфигурации."
      )}
      <div class="stat-grid">${stats
        .map(
          (s) => `<div class="stat-card">
            <div class="stat-card-label">${esc(s.label)}</div>
            <div class="stat-card-val ${s.tone}">${esc(s.value)}</div>
          </div>`
        )
        .join("")}</div>
      <div class="agent-table-wrap">${renderAgentTable(snaps)}</div>
    `;
  }

  function renderAgentTable(snaps) {
    if (!snaps.length) return '<div class="empty">Агенты не настроены (web.hub.agents)</div>';
    const rows = snaps
      .map((s) => {
        const st = agentStatusLabel(s.status);
        const clickable = s.status !== "offline";
        const latency = s.scrape_ms != null
          ? s.scrape_ms + "ms"
          : s.access
            ? Math.round(s.access.p95_ms) + "ms"
            : "—";
        return `<tr class="${clickable ? "clickable" : ""}" ${clickable ? `data-snap="${esc(s.id)}"` : ""}>
          <td>${esc(s.name)}</td>
          <td class="muted">${esc(s.region || "—")}</td>
          <td class="muted">${esc(s.host)}</td>
          <td class="muted">${esc(s.version ? "v" + s.version : "—")}</td>
          <td>${esc(latency)}</td>
          <td>${esc(s.uptime || "—")}</td>
          <td><span class="badge ${st.cls}">${st.label}</span></td>
        </tr>`;
      })
      .join("");
    return `<table class="agent-table"><thead><tr><th>Agent ID</th><th>Region</th><th>IP</th><th>Version</th><th>Latency</th><th>Uptime</th><th>Status</th></tr></thead><tbody>${rows}</tbody></table>`;
  }

  function renderSnapshotsList() {
    const snaps = filterSnapshots(state.snapshots || []);
    return `
      ${pageHeader("Раздел · 02", "Конфигурации", "Конфигурации nginx с каждого агента. Откройте карточку для полного разбора.")}
      <div class="snap-grid">${snaps.length ? snaps.map(renderSnapCard).join("") : '<div class="empty">Нет агентов</div>'}</div>
    `;
  }

  function renderSnapCard(s) {
    if (s.status === "offline") {
      return `<div class="snap-card" data-snap="${esc(s.id)}">
        <div class="snap-card-head">
          <div>
            <div class="snap-card-name">${esc(s.name)} <span class="badge badge-critical">offline</span></div>
            <div class="snap-card-url">${esc(s.url)}</div>
          </div>
          <svg class="snap-card-chevron icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 18 6-6-6-6"/></svg>
        </div>
        <div class="corr-error">${esc(s.error || "unreachable")}</div>
      </div>`;
    }
    return `<div class="snap-card" data-snap="${esc(s.id)}">
      <div class="snap-card-head">
        <div>
          <div class="snap-card-name">${esc(s.name)} <span class="badge ${s.status === "warning" ? "badge-warning" : "badge-online"}">${esc(s.status)}</span></div>
          <div class="snap-card-url">${esc(s.url)}</div>
        </div>
        <svg class="snap-card-chevron icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 18 6-6-6-6"/></svg>
      </div>
      <div class="snap-scores">
        <div><div class="score-cell-label">Score</div><div class="score-cell-val t-primary">${s.config_score}<span class="kpi-suffix">/100</span></div></div>
      </div>
      <div class="snap-card-url">nginx/${esc(s.version || "—")} · upd: ${esc(s.updated_at)}</div>
    </div>`;
  }

  function renderSnapshotDetail(id) {
    const s = (state.snapshots || []).find((x) => x.id === id);
    if (!s) {
      return `${pageHeader("", "Конфигурация не найдена", "")}<a class="back-link" href="#/nodes">← К списку</a>`;
    }
    if (s.status === "offline") {
      return `
        <a class="back-link" href="#/nodes">← К списку конфигураций</a>
        ${renderDetailHeader(s)}
        <div class="panel"><div class="panel-title" style="margin-bottom:0.75rem">Агент недоступен</div><pre class="explore-json">${esc(s.error)}</pre></div>
      `;
    }
    return `
      <a class="back-link" href="#/nodes">← К списку конфигураций</a>
      ${renderDetailHeader(s)}
      <div class="score-banner"${animAttr()}>
        <div>
          <div class="kpi-label">Config Score</div>
          <div class="kpi-row"><span class="score-big ${scoreTone(s.config_score)}">${s.config_score}</span><span class="kpi-suffix" style="font-size:1.125rem">/100</span></div>
        </div>
        <div style="flex:1;min-width:280px;max-width:28rem">
          <div class="score-bar-track"><div class="score-bar-fill" style="width:${s.config_score}%"></div></div>
          <div class="score-axis"><span>0</span><span>POOR · 50</span><span>GOOD · 80</span><span>100</span></div>
        </div>
      </div>
      <div class="cat-grid">${[
        ["Security", (s.categories || {}).security],
        ["Reliability", (s.categories || {}).reliability],
        ["Performance", (s.categories || {}).performance],
        ["Maintainability", (s.categories || {}).maintainability],
        ["Observability", (s.categories || {}).observability],
      ]
        .map(([label, score]) => renderCategoryCard(label, score))
        .join("")}</div>
      <div>
        <div class="tabs">${TABS.map((t) => `<button type="button" class="tab-btn${detailTab === t.id ? " active" : ""}" data-tab="${esc(t.id)}">${esc(t.label)}</button>`).join("")}</div>
        <div class="tab-panel">${renderTabContent(s)}</div>
      </div>
    `;
  }

  function renderDetailHeader(s) {
    const tone = s.status === "offline" ? "offline" : s.status === "warning" ? "warning" : "";
    return `<div class="detail-header"${animAttr()}>
      <div>
        <div class="page-eyebrow${eyebrowToneClass(tone)}"><span class="page-eyebrow-dot"></span>Конфигурация · ${esc(s.id)}</div>
        <h1 class="page-title">${esc(s.name)}</h1>
        <a href="${esc(s.url)}" target="_blank" rel="noopener" class="detail-url">${esc(s.url)}
          <svg class="icon-xs" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
        </a>
      </div>
      <div class="detail-actions">
        <button type="button" class="btn-outline" id="btn-copy-url">Копировать URL</button>
        <button type="button" class="btn-primary" id="btn-rescan">Обновить</button>
      </div>
    </div>`;
  }

  function renderCategoryCard(label, score) {
    const n = Math.round(score || 0);
    const barCls = n >= 70 ? "impact-low" : n >= 50 ? "impact-med" : "impact-high";
    return `<div class="cat-card">
      <div class="cat-card-label">${esc(label)}</div>
      <div class="kpi-row"><span class="cat-card-score ${scoreTone(n)}">${n}</span><span class="kpi-suffix">/ 100</span></div>
      <div class="cat-card-bar"><div class="cat-card-bar-fill ${barCls}" style="width:${n}%"></div></div>
    </div>`;
  }

  function renderTabContent(s) {
    switch (detailTab) {
      case "Upstream":
        return renderDataTable(
          ["Upstream", "Адрес", "Статус", "Ошибки"],
          (s.upstreams || []).map((u) => [u.name, u.address, statusPill(u.status), esc(u.errors)])
        );
      case "Build":
        return renderDataTable(
          ["Параметр", "Значение"],
          (s.build || []).map((b) => [`<span class="muted">${esc(b.name)}</span>`, esc(b.value)])
        );
      case "Certs":
        return renderDataTable(
          ["Домен", "Путь", "Issuer", "Истекает", "Осталось"],
          (s.certs || []).map((c) => [
            esc(c.domain || "—"),
            `<span class="muted" style="font-family:var(--font-mono);font-size:0.75rem">${esc(c.path || "—")}</span>`,
            esc(c.issuer || "—"),
            esc(c.expires || "—"),
            c.status === "cert_not_found" || c.status === "cert_invalid_pem"
              ? `<span class="t-danger">${esc(c.status)}</span>`
              : `<span class="${c.days_left < 30 ? "t-danger" : "t-primary"}">${c.days_left}д</span>`,
          ])
        );
      case "Blast-radius":
        return renderBlastTab(s);
      case "Errors":
        return renderErrorsTab(s);
      case "Explore":
        return renderExploreTab(s);
      default:
        return "";
    }
  }

  function renderExploreTab(s) {
    return `<div class="explore-box">
      <div class="kpi-label" style="margin-bottom:0.5rem">Explain route</div>
      <p class="muted" style="margin:0 0 1rem">Интерактивный разбор маршрутизации: server → location → proxy_pass / upstream.</p>
      <div class="explore-form">
        <input type="url" id="explore-url" placeholder="https://example.com/api/v1/users" value=""/>
        <button type="button" class="btn-primary" id="explore-run" data-agent-url="${esc(s.url)}">Explain</button>
      </div>
      <div id="explore-result"><div class="empty-dashed">Введите URL и нажмите Explain</div></div>
      <button type="button" class="explore-toggle" id="explore-raw-toggle">Показать raw snapshot JSON</button>
      <pre class="explore-json hidden" id="explore-raw">${esc(JSON.stringify(s, null, 2))}</pre>
    </div>`;
  }

  function renderExplainResult(data) {
    if (!data) return '<div class="empty-dashed">Пустой ответ</div>';
    const nodeLabel = (n) => {
      if (!n) return "";
      return n.args || n.arg || n.directive || n.block || "";
    };
    const chips = [
      { label: "URL", value: data.url || "—" },
      { label: "Server", value: nodeLabel(data.server) || "—" },
      { label: "Location", value: nodeLabel(data.location) || "—" },
      { label: "Upstream", value: data.upstream || "—" },
      { label: "Proxy pass", value: data.proxy_pass || "—" },
    ];
    const steps = Array.isArray(data.trace) ? data.trace : [];
    return `
      <div class="explore-summary">${chips
        .map(
          (c) => `<div class="explore-chip">
            <div class="explore-chip-label">${esc(c.label)}</div>
            <div class="explore-chip-val">${esc(c.value)}</div>
          </div>`
        )
        .join("")}</div>
      <div class="explore-trace">${
        steps.length
          ? steps
              .map(
                (step, i) => `<div class="explore-step${step.matched ? " matched" : ""}">
                  <div class="explore-step-idx">${i + 1}</div>
                  <div class="explore-step-body">
                    <div class="explore-step-name">${esc(step.step || "step")}</div>
                    <div class="explore-step-detail">${esc(step.detail || "")}</div>
                  </div>
                </div>`
              )
              .join("")
          : '<div class="empty-dashed">Trace пуст</div>'
      }</div>`;
  }

  function renderDataTable(headers, rows) {
    if (!rows.length) return '<div class="empty-dashed">Нет данных</div>';
    const th = headers.map((h) => `<th>${esc(h)}</th>`).join("");
    const tr = rows.map((r) => `<tr>${r.map((c) => `<td>${c}</td>`).join("")}</tr>`).join("");
    return `<div class="data-table-wrap"><table class="data-table"><thead><tr>${th}</tr></thead><tbody>${tr}</tbody></table></div>`;
  }

  function renderIssuesTab(s) {
    if (!s.issues || !s.issues.length) return '<div class="empty-dashed">Нет данных</div>';
    return s.issues
      .map(
        (i) => `<div class="issue-card">
          <span class="issue-sev ${esc(i.severity)}">${esc(i.severity)}</span>
          <div class="issue-body"><div class="issue-title">${esc(i.title)}</div><div class="issue-meta">${esc(i.rule)} · ${esc(i.location)}</div></div>
          <span class="issue-id">${esc(i.id)}</span>
        </div>`
      )
      .join("");
  }

  function renderBlastTab(s) {
    if (!s.blast || !s.blast.length) return '<div class="empty-dashed">Нет данных</div>';
    return s.blast
      .map(
        (b) => `<div class="blast-card">
          <div class="blast-card-head"><span>${esc(b.upstream)} → ${esc(b.location)}</span><span class="${impactText(b.impact)}">IMPACT ${b.impact}%</span></div>
          <div class="impact-track"><div class="impact-fill ${impactClass(b.impact)}" style="width:${b.impact}%"></div></div>
        </div>`
      )
      .join("");
  }

  function renderErrorsTab(s) {
    if (!s.errors || !s.errors.length) return '<div class="empty-dashed">Нет данных</div>';
    return `<div class="error-log">${s.errors
      .map((e) => `<div class="error-line"><span class="muted">${esc(e.time)}</span><span class="t-danger">${esc(e.code)}</span><span>${esc(e.message)}</span></div>`)
      .join("")}</div>`;
  }

  function renderCorrelation() {
    const items = (state.correlations || []).filter((c) => {
      if (!searchQuery) return true;
      const q = searchQuery.toLowerCase();
      return c.upstream.toLowerCase().includes(q) || c.agent.toLowerCase().includes(q) || c.error.toLowerCase().includes(q);
    });
    return `
      ${pageHeader("Section · 03", "Error Log Correlation", "Сопоставление upstream сбоев с записями error log. Сгруппировано по incident-окнам.")}
      ${items.length ? items.map(renderCorrCard).join("") : '<div class="empty-dashed">Нет корреляций</div>'}
    `;
  }

  function renderCorrCard(c) {
    const sevCls = { high: "issue-sev high", med: "issue-sev med", low: "issue-sev low" }[c.severity] || "issue-sev med";
    const loc = (c.locations && c.locations[0]) || "—";
    return `<div class="corr-card">
      <div style="display:flex;justify-content:space-between;flex-wrap:wrap;gap:0.5rem;margin-bottom:0.75rem">
        <div style="display:flex;gap:0.75rem;align-items:center;flex-wrap:wrap">
          <span class="${sevCls}">${esc(c.severity)}</span>
          <span style="font-family:var(--font-mono);font-size:0.75rem">${esc(c.id)}</span>
          <span class="muted" style="font-family:var(--font-mono);font-size:0.6875rem">${esc(c.time)}</span>
        </div>
        <span class="t-primary" style="font-family:var(--font-mono);font-size:0.625rem">${c.matches} matches</span>
      </div>
      <div class="corr-error"><span class="t-danger">[error]</span> ${esc(c.error)}</div>
      <div style="display:flex;gap:0.75rem;flex-wrap:wrap;font-family:var(--font-mono);font-size:0.6875rem;align-items:center">
        <span class="module-tag">upstream: ${esc(c.upstream)}</span>
        <span class="muted">→</span>
        <span class="module-tag module-tag-hl">location: ${esc(loc)}</span>
        <span class="muted">${esc(c.agent)}</span>
      </div>
    </div>`;
  }

  function renderBlastRadius() {
    const groups = (state.blast_radius || []).filter((g) => {
      if (!searchQuery) return true;
      return g.upstream.toLowerCase().includes(searchQuery.toLowerCase());
    });
    return `
      ${pageHeader("Section · 04", "Blast-radius", "Какие location затрагивает каждый upstream. % impact = доля запросов с upstream-ошибками.")}
      ${groups.length ? groups.map(renderBlastGroup).join("") : '<div class="empty-dashed">Нет dependency graph</div>'}
    `;
  }

  function renderBlastGroup(g) {
    const hCls = { healthy: "badge-healthy", degraded: "badge-degraded", critical: "badge-critical" }[g.health] || "badge-healthy";
    return `<div class="blast-group">
      <div style="display:flex;justify-content:space-between;margin-bottom:1.25rem;flex-wrap:wrap;gap:0.5rem">
        <div><span style="font-family:var(--font-display);font-weight:700;text-transform:uppercase;font-size:0.875rem">upstream: </span><span class="t-primary" style="font-family:var(--font-mono)">${esc(g.upstream)}</span><span class="muted" style="font-family:var(--font-mono);font-size:0.625rem;margin-left:0.75rem">${esc(g.agent)}</span></div>
        <span class="badge ${hCls}">${esc(g.health)}</span>
      </div>
      ${(g.locations || []).map((l) => `
        <div class="blast-loc-row">
          <div class="blast-loc-head"><span>${esc(l.loc)}</span><div><span class="muted">${esc(l.requests)}</span> <span class="${impactText(l.impact)}">IMPACT ${l.impact}%</span></div></div>
          <div class="impact-track"><div class="impact-fill ${impactClass(l.impact)}" style="width:${l.impact}%"></div></div>
        </div>
      `).join("")}
    </div>`;
  }

  function renderViewContent() {
    switch (route.page) {
      case "overview":
        return renderOverview();
      case "agents":
        return renderAgents();
      case "nodes":
        return renderSnapshotsList();
      case "snapshot-detail":
        return renderSnapshotDetail(route.id);
      case "correlation":
        return renderCorrelation();
      case "blast-radius":
        return renderBlastRadius();
      default:
        return renderOverview();
    }
  }

  function updateKpiCardEl(card, value, suffix, tone, pulse, delta) {
    if (!card) return;
    const valEl = card.querySelector(".kpi-value");
    if (valEl) {
      valEl.className = "kpi-value " + tone;
      valEl.textContent = value;
    }
    const row = card.querySelector(".kpi-row");
    let suffixEl = card.querySelector(".kpi-suffix");
    if (suffix) {
      if (suffixEl) suffixEl.textContent = suffix;
      else if (row) row.insertAdjacentHTML("beforeend", `<span class="kpi-suffix">${esc(suffix)}</span>`);
    } else if (suffixEl) {
      suffixEl.remove();
    }
    let pulseEl = card.querySelector(".kpi-pulse");
    const top = card.querySelector(".kpi-top");
    if (pulse && !pulseEl && top) {
      top.insertAdjacentHTML("beforeend", '<span class="kpi-pulse"></span>');
    } else if (!pulse && pulseEl) {
      pulseEl.remove();
    }
    const body = card.querySelector(".kpi-body");
    if (body) {
      const existing = body.querySelector(".kpi-delta-row");
      const html = renderKpiDelta(delta);
      if (existing) {
        if (html) existing.outerHTML = html;
        else existing.remove();
      } else if (html) {
        body.insertAdjacentHTML("beforeend", html);
      }
    }
  }

  function patchKpiGrid(kpi) {
    const cards = document.querySelectorAll(".kpi-grid .kpi-card");
    if (cards.length < 2) return false;
    const d = kpi.deltas || {};
    updateKpiCardEl(cards[0], kpi.agents_online, kpi.agents_suffix, "t-primary", false, d.agents_online);
    updateKpiCardEl(cards[1], kpi.upstream_healthy, null, "t-highlight", false, d.upstream_healthy);
    return true;
  }

  function patchHealthOverview() {
    const panel = document.querySelector(".fleet-stack .health-panel") || document.querySelector(".overview-grid .health-panel");
    if (!panel) return false;
    const bars = state.health_bars;
    if (!bars || !bars.length) return false;
    const now = new Date();
    const labels = bars.map((_, i) => {
      if (i === bars.length - 1) return { text: "сейчас", now: true };
      const m = new Date(now - (bars.length - 1 - i) * 60000);
      return { text: m.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }), now: false };
    });
    const axisSparse = labels.filter((_, i) => i === 0 || i === Math.floor(labels.length / 2) || i === labels.length - 1);
    const chart = panel.querySelector(".health-chart");
    const axis = panel.querySelector(".health-axis");
    const hosts = panel.querySelector(".health-host-legend");
    if (!chart || !axis) return false;
    chart.innerHTML = bars
      .map((b) => {
        const h = Math.max(8, Math.min(100, b.pct < 0 ? 50 : b.pct));
        return `<div class="health-bar ${b.bad ? "bad" : "ok"}" style="height:${h}%" title="${esc(b.agent)}: ${Math.round(b.pct)}%"></div>`;
      })
      .join("");
    if (hosts) {
      hosts.innerHTML = bars
        .map((b) => `<span class="health-host-tick" title="${esc(b.agent)}">${esc(shortHost(b.agent, 14))}</span>`)
        .join("");
    }
    axis.innerHTML = axisSparse
      .map((l) => `<span class="${l.now ? "now" : ""}">${esc(l.now ? "сейчас" : l.text)}</span>`)
      .join("");
    return true;
  }

  function patchSeverityPanel() {
    const panel = document.querySelector(".overview-grid .severity-panel-wrap");
    if (!panel) return false;
    const sev = state.severity;
    const total = (sev.high || 0) + (sev.medium || 0) + (sev.low || 0);
    const totalEl = panel.querySelector(".panel-total-val");
    if (totalEl) totalEl.textContent = total;
    const rows = [
      { label: "High severity", pct: sev.high_pct, count: sev.high, fill: "severity-fill-high", text: "t-danger", dot: "high" },
      { label: "Medium severity", pct: sev.medium_pct, count: sev.medium, fill: "severity-fill-med", text: "t-warning", dot: "med" },
      { label: "Low severity", pct: sev.low_pct, count: sev.low, fill: "severity-fill-low", text: "t-primary", dot: "low" },
    ];
    const body = panel.querySelector(".severity-panel");
    if (!body) return false;
    body.innerHTML = rows
      .map(
        (r) => `<div>
            <div class="severity-row-head">
              <div class="severity-row-label"><span class="severity-row-dot ${r.dot}"></span>${esc(r.label)}</div>
              <div class="severity-row-vals">
                <span class="severity-row-count ${r.text}">${r.count}</span>
                <span class="severity-row-pct">${r.pct}%</span>
              </div>
            </div>
            <div class="severity-track"><div class="${r.fill}" style="width:${r.pct}%"></div></div>
          </div>`
      )
      .join("");
    return true;
  }

  function patchOverview() {
    const pane = $("#overview-pane");
    const tabs = document.querySelector(".overview-tabs");
    if (!pane || !tabs) return false;
    const tabMeta = document.querySelector(".overview-tab-meta");
    if (tabMeta) {
      const svc = state.services || {};
      tabMeta.textContent = svc.has_data
        ? formatCount(svc.total_requests || 0) + " req · " + (svc.unique_services || 0) + " services"
        : "access.log";
    }
    document.querySelectorAll("[data-overview-tab]").forEach((el) => {
      el.classList.toggle("active", el.dataset.overviewTab === overviewTab);
    });
    if (overviewTab === "services") {
      pane.innerHTML = renderServicesApps();
      bindViewEvents();
      return true;
    }
    if (!document.querySelector(".kpi-grid")) {
      pane.innerHTML = renderFleetOverview();
      bindViewEvents();
      return true;
    }
    if (!patchKpiGrid(state.kpi)) return false;
    if (!patchHealthOverview()) return false;
    const snaps = state.snapshots || [];
    const list = filterSnapshots(snaps);
    const shown = list.slice(0, 10);
    const feed = document.querySelector(".agents-feed");
    const meta = document.querySelector(".section-head .section-meta");
    if (feed) {
      feed.innerHTML = shown.length ? shown.map(renderAgentRow).join("") : '<div class="empty">Агенты не настроены</div>';
    }
    if (meta) meta.textContent = "1–" + shown.length + " of " + list.length;
    const analytics = document.querySelector(".analytics-grid");
    if (analytics) analytics.outerHTML = renderAnalyticsPanels();
    bindViewEvents();
    return true;
  }

  function patchAgents() {
    const snaps = filterSnapshots(state.snapshots || []);
    const wrap = document.querySelector(".agent-table-wrap");
    if (!wrap) return false;
    const total = snaps.length;
    const online = snaps.filter((s) => s.status === "online").length;
    const warning = snaps.filter((s) => s.status === "warning").length;
    const critical = snaps.filter((s) => s.status === "offline").length;
    const values = [String(total), String(online), String(warning), String(critical)];
    document.querySelectorAll(".stat-grid .stat-card-val").forEach((el, i) => {
      if (values[i] != null) el.textContent = values[i];
    });
    wrap.innerHTML = renderAgentTable(snaps);
    bindViewEvents();
    return true;
  }

  function patchSnapshotsList() {
    const grid = document.querySelector(".snap-grid");
    if (!grid) return false;
    const snaps = filterSnapshots(state.snapshots || []);
    grid.innerHTML = snaps.length ? snaps.map(renderSnapCard).join("") : '<div class="empty">Нет агентов</div>';
    bindViewEvents();
    return true;
  }

  function patchSnapshotDetail() {
    const s = (state.snapshots || []).find((x) => x.id === route.id);
    if (!s || s.status === "offline") return false;
    const scoreEl = document.querySelector(".score-big");
    const scoreBar = document.querySelector(".score-bar-fill");
    if (scoreEl) {
      scoreEl.className = "score-big " + scoreTone(s.config_score);
      scoreEl.textContent = s.config_score;
    }
    if (scoreBar) scoreBar.style.width = s.config_score + "%";
    document.querySelectorAll(".cat-card").forEach((card, i) => {
      const cats = s.categories || {};
      const ib = s.issues_breakdown || {};
      const entries = [
        ["Security", cats.security, ib.security],
        ["Reliability", cats.reliability, ib.reliability],
        ["Performance", cats.performance, ib.performance],
        ["Maintainability", cats.maintainability, ib.maintainability],
        ["Observability", cats.observability, ib.observability],
      ];
      const [label, score, issues] = entries[i] || [];
      if (label == null) return;
      const n = Math.round(score || 0);
      const barCls = n >= 70 ? "impact-low" : n >= 50 ? "impact-med" : "impact-high";
      const sc = card.querySelector(".cat-card-score");
      if (sc) {
        sc.className = "cat-card-score " + scoreTone(n);
        sc.textContent = n;
      }
      const fill = card.querySelector(".cat-card-bar-fill");
      if (fill) {
        fill.className = "cat-card-bar-fill " + barCls;
        fill.style.width = n + "%";
      }
      const iss = card.querySelector(".muted");
      if (iss) iss.textContent = (issues || 0) + " issues";
    });
    const sevCells = document.querySelectorAll(".severity-cell-val");
    if (sevCells.length >= 3) {
      sevCells[0].textContent = s.severity.high;
      sevCells[1].textContent = s.severity.med;
      sevCells[2].textContent = s.severity.low;
    }
    const tabPanel = document.querySelector(".tab-panel");
    if (tabPanel) tabPanel.innerHTML = renderTabContent(s);
    bindViewEvents();
    return true;
  }

  function patchCorrelation() {
    const container = $("#view");
    const header = container.querySelector(".page-header");
    if (!header) return false;
    const items = (state.correlations || []).filter((c) => {
      if (!searchQuery) return true;
      const q = searchQuery.toLowerCase();
      return c.upstream.toLowerCase().includes(q) || c.agent.toLowerCase().includes(q) || c.error.toLowerCase().includes(q);
    });
    container.querySelectorAll(".corr-card, .empty-dashed").forEach((el) => el.remove());
    header.insertAdjacentHTML("afterend", items.length ? items.map(renderCorrCard).join("") : '<div class="empty-dashed">Нет корреляций</div>');
    return true;
  }

  function patchBlastRadius() {
    const container = $("#view");
    const header = container.querySelector(".page-header");
    if (!header) return false;
    const groups = (state.blast_radius || []).filter((g) => {
      if (!searchQuery) return true;
      return g.upstream.toLowerCase().includes(searchQuery.toLowerCase());
    });
    container.querySelectorAll(".blast-group, .empty-dashed").forEach((el) => el.remove());
    header.insertAdjacentHTML("afterend", groups.length ? groups.map(renderBlastGroup).join("") : '<div class="empty-dashed">Нет dependency graph</div>');
    return true;
  }

  function patchView() {
    switch (route.page) {
      case "overview":
        return patchOverview();
      case "agents":
        return patchAgents();
      case "nodes":
        return patchSnapshotsList();
      case "snapshot-detail":
        return patchSnapshotDetail();
      case "correlation":
        return patchCorrelation();
      case "blast-radius":
        return patchBlastRadius();
      default:
        return false;
    }
  }

  function render(options) {
    const soft = options && options.soft === true;
    animate = !soft;
    const view = $("#view");
    const key = routeKey();
    if (soft && viewMounted && view.dataset.route === key && state && patchView()) {
      updateMeta();
      return;
    }
    view.dataset.route = key;
    view.innerHTML = renderViewContent();
    viewMounted = true;
    bindViewEvents();
    updateMeta();
  }

  function bindViewEvents() {
    document.querySelectorAll("[data-snap]").forEach((el) => {
      el.addEventListener("click", (e) => {
        if (e.target.closest("button")) return;
        const action = el.dataset.action;
        if (action === "correlation") navigate("correlation");
        else navigate("nodes/" + el.dataset.snap);
      });
    });
    document.querySelectorAll("button[data-action]").forEach((el) => {
      el.addEventListener("click", (e) => {
        e.stopPropagation();
        if (el.dataset.action === "correlation") navigate("correlation");
        else navigate("nodes/" + el.dataset.snap);
      });
    });
    document.querySelectorAll(".agent-table tr.clickable").forEach((el) => {
      el.addEventListener("click", () => navigate("nodes/" + el.dataset.snap));
    });
    document.querySelectorAll("[data-tab]").forEach((el) => {
      el.addEventListener("click", () => {
        detailTab = el.dataset.tab;
        render();
      });
    });
    document.querySelectorAll("[data-overview-tab]").forEach((el) => {
      el.addEventListener("click", () => {
        overviewTab = el.dataset.overviewTab;
        render();
      });
    });
    const copyBtn = $("#btn-copy-url");
    if (copyBtn) {
      copyBtn.addEventListener("click", () => {
        const s = (state.snapshots || []).find((x) => x.id === route.id);
        if (s && navigator.clipboard) navigator.clipboard.writeText(s.url);
      });
    }
    const rescanBtn = $("#btn-rescan");
    if (rescanBtn) rescanBtn.addEventListener("click", () => refresh(true));
    const exploreRun = $("#explore-run");
    if (exploreRun) {
      exploreRun.addEventListener("click", () => runExplore(exploreRun.dataset.agentUrl));
      const urlInput = $("#explore-url");
      if (urlInput) {
        urlInput.addEventListener("keydown", (e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            runExplore(exploreRun.dataset.agentUrl);
          }
        });
      }
    }
    const rawToggle = $("#explore-raw-toggle");
    if (rawToggle) {
      rawToggle.addEventListener("click", () => {
        const raw = $("#explore-raw");
        if (!raw) return;
        const open = !raw.classList.contains("hidden");
        raw.classList.toggle("hidden", open);
        rawToggle.textContent = open ? "Показать raw snapshot JSON" : "Скрыть raw snapshot JSON";
      });
    }
  }

  async function runExplore(agentURL) {
    const input = $("#explore-url");
    const out = $("#explore-result");
    if (!input || !out) return;
    const url = input.value.trim();
    if (!url) {
      out.innerHTML = '<div class="empty-dashed">Укажите URL маршрута</div>';
      return;
    }
    out.innerHTML = '<div class="muted" style="font-family:var(--font-mono);font-size:0.75rem">Loading…</div>';
    try {
      const q = new URLSearchParams({ agent: agentURL, url: url });
      const r = await fetch("/api/v1/explain?" + q.toString(), { headers: headers() });
      if (!r.ok) throw new Error("HTTP " + r.status);
      const data = await r.json();
      out.innerHTML = renderExplainResult(data);
    } catch (e) {
      out.innerHTML = `<div class="empty-dashed t-danger">${esc("Ошибка explain: " + e.message)}</div>`;
    }
  }

  function updateMeta() {
    if (!state) return;
    const meta = state.meta || {};
    const status = meta.system_status || "—";
    const statusNorm = String(status).toUpperCase();
    const statusLabel = statusNorm === "NOMINAL" || statusNorm === "OK" || statusNorm === "HEALTHY"
      ? "System nominal"
      : "System " + status;
    $("#meta-status").textContent = statusLabel;
    const pill = document.querySelector(".header-status-pill");
    if (pill) {
      pill.classList.remove("is-offline", "is-degraded", "is-nominal");
      if (statusNorm === "OFFLINE") pill.classList.add("is-offline");
      else if (statusNorm === "DEGRADED") pill.classList.add("is-degraded");
      else pill.classList.add("is-nominal");
    }
    const sec = meta.refresh_interval || 30;
    $("#meta-refresh").textContent = sec + "s";
    const online = meta.agents_online || 0;
    const offline = (meta.agents_total || 0) - online;
    $("#footer-stats").innerHTML = `
      <div class="footer-stat online"><span class="footer-dot" style="background:var(--primary);animation:pulse-glow 2.2s infinite ease-in-out"></span>${online} online</div>
      <div class="footer-stat offline"><span class="footer-dot" style="background:var(--destructive)"></span>${String(offline).padStart(2, "0")} offline</div>`;
    renderNav();
  }

  function showError(msg) {
    const b = $("#error-banner");
    b.textContent = msg;
    b.classList.remove("hidden");
  }

  function hideError() {
    $("#error-banner").classList.add("hidden");
  }

  const AUTH_REQUIRED = window.NGINX_LENS_AUTH_REQUIRED === true;
  let refreshTimer = null;
  let unlocked = !AUTH_REQUIRED;

  function saveToken(value) {
    const q = new URLSearchParams(location.search).get("token");
    if (q) {
      // токен из query один раз переносим в storage и чистим URL
      const url = new URL(location.href);
      url.searchParams.delete("token");
      history.replaceState({}, "", url.pathname + url.search + url.hash);
    }
    localStorage.setItem("nginx_lens_hub_token", value);
  }

  function clearToken() {
    localStorage.removeItem("nginx_lens_hub_token");
  }

  function showGate(message) {
    unlocked = false;
    $("#app").classList.add("hidden");
    const gate = $("#auth-gate");
    gate.classList.remove("hidden");
    const err = $("#auth-error");
    if (message) {
      err.textContent = message;
      err.classList.remove("hidden");
    } else {
      err.textContent = "";
      err.classList.add("hidden");
    }
    const input = $("#auth-token-input");
    if (input) {
      input.value = token();
      setTimeout(() => input.focus(), 50);
    }
    if (refreshTimer) {
      clearInterval(refreshTimer);
      refreshTimer = null;
    }
  }

  function enterApp() {
    unlocked = true;
    $("#auth-gate").classList.add("hidden");
    $("#app").classList.remove("hidden");
    const logout = $("#btn-logout");
    if (logout) logout.classList.toggle("hidden", !AUTH_REQUIRED);
  }

  async function refresh(soft) {
    if (!unlocked) return;
    const view = $("#view");
    const scrollTop = soft && view ? view.scrollTop : 0;
    try {
      state = await fetchState();
      hideError();
      render({ soft: soft === true });
      if (soft && view) view.scrollTop = scrollTop;
    } catch (e) {
      if (e.code === 401 && AUTH_REQUIRED) {
        clearToken();
        showGate("Токен неверный или устарел. Введите hub token снова.");
        return;
      }
      showError("Ошибка загрузки: " + e.message);
    }
  }

  function startRefreshLoop() {
    if (refreshTimer) clearInterval(refreshTimer);
    refreshTimer = setInterval(() => refresh(true), REFRESH_MS);
  }

  async function tryUnlockWithStoredToken() {
    if (!AUTH_REQUIRED) {
      enterApp();
      await refresh(false);
      startRefreshLoop();
      return;
    }
    if (!token()) {
      showGate();
      return;
    }
    try {
      state = await fetchState();
      enterApp();
      hideError();
      render({ soft: false });
      startRefreshLoop();
    } catch (e) {
      clearToken();
      showGate(e.code === 401 ? "Токен неверный. Попробуйте ещё раз." : "Не удалось проверить токен: " + e.message);
    }
  }

  function initAuthGate() {
    const form = $("#auth-form");
    if (!form) return;
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      const input = $("#auth-token-input");
      const err = $("#auth-error");
      const btn = $("#auth-submit");
      const value = (input.value || "").trim();
      if (!value) {
        err.textContent = "Введите токен";
        err.classList.remove("hidden");
        return;
      }
      btn.disabled = true;
      err.classList.add("hidden");
      saveToken(value);
      try {
        state = await fetchState();
        enterApp();
        render({ soft: false });
        startRefreshLoop();
      } catch (ex) {
        clearToken();
        err.textContent = ex.code === 401 ? "Неверный токен" : "Ошибка: " + ex.message;
        err.classList.remove("hidden");
        input.focus();
      } finally {
        btn.disabled = false;
      }
    });
    const logout = $("#btn-logout");
    if (logout) {
      logout.addEventListener("click", () => {
        clearToken();
        showGate();
      });
    }
  }

  window.addEventListener("hashchange", () => {
    if (!unlocked) return;
    route = parseRoute();
    render();
  });

  document.addEventListener("keydown", (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
      if (!unlocked) return;
      e.preventDefault();
      const input = $("#search-input");
      if (input) {
        input.focus();
        input.select();
      }
    }
  });

  $("#btn-refresh").addEventListener("click", () => refresh(true));
  $("#search-input").addEventListener("input", (e) => {
    searchQuery = e.target.value.trim();
    if (unlocked) render();
  });

  initAuthGate();
  // Если в URL есть ?token= — сохранить до проверки
  const bootToken = new URLSearchParams(location.search).get("token");
  if (bootToken) saveToken(bootToken.trim());
  tryUnlockWithStoredToken();
})();
