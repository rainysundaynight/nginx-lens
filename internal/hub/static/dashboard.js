(function () {
  "use strict";

  const REFRESH_MS = window.NGINX_LENS_REFRESH || 30000;
  const NAV = [
    { label: "Overview", path: "", icon: "activity" },
    { label: "Agents", path: "agents", icon: "layers" },
    { label: "Snapshots", path: "snapshots", icon: "camera" },
    { label: "Correlation", path: "correlation", icon: "git-branch" },
    { label: "Blast-radius", path: "blast-radius", icon: "radio" },
  ];
  const TABS = ["Upstream", "Build", "Issues", "Certs", "Blast-radius", "Errors", "Explore"];

  const ICONS = {
    activity: '<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>',
    layers: '<polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/>',
    camera: '<path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/>',
    "git-branch": '<line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>',
    radio: '<path d="M4.9 19.1C1 15.2 1 8.8 4.9 4.9"/><path d="M7.8 16.2c-2.3-2.3-2.3-6.1 0-8.5"/><circle cx="12" cy="12" r="2"/><path d="M16.2 7.8c2.3 2.3 2.3 6.1 0 8.5"/><path d="M19.1 4.9C23 8.8 23 15.1 19.1 19"/>',
  };

  let state = null;
  let route = parseRoute();
  let detailTab = "Upstream";
  let searchQuery = "";
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

  function icon(name) {
    const p = ICONS[name] || "";
    return `<svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">${p}</svg>`;
  }

  function token() {
    const q = new URLSearchParams(location.search).get("token");
    if (q) return q;
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
    if (!r.ok) throw new Error("HTTP " + r.status);
    return r.json();
  }

  function parseRoute() {
    const hash = (location.hash || "#/").replace(/^#\/?/, "");
    const parts = hash.split("/").filter(Boolean);
    if (parts[0] === "snapshots" && parts[1]) return { page: "snapshot-detail", id: decodeURIComponent(parts[1]) };
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
    if (s.status === "offline") return { label: "Error Log", text: s.error || "Connection refused", bad: true };
    if (s.issues && s.issues.length) return { label: "Latest Snapshot", text: s.issues[0].title, bad: false };
    if (s.note) return { label: "Note", text: s.note, bad: false, empty: true };
    return { label: "Latest Snapshot", text: "Нет новых issues", bad: false, empty: true };
  }

  function renderNav() {
    const nav = $("#nav");
    nav.innerHTML = NAV.map((item) => {
      const active =
        (item.path === "" && route.page === "overview") ||
        route.page === item.path ||
        (item.path === "snapshots" && route.page === "snapshot-detail");
      return `<button type="button" class="nav-item${active ? " active" : ""}" data-nav="${esc(item.path)}">
        ${active ? '<span class="nav-dot"></span>' : icon(item.icon)}
        ${esc(item.label)}
      </button>`;
    }).join("");
    nav.querySelectorAll("[data-nav]").forEach((el) => {
      el.addEventListener("click", () => navigate(el.dataset.nav));
    });
  }

  function pageHeader(eyebrow, title, desc, actions) {
    return `<div class="page-header"${animAttr()}>
      <div>
        ${eyebrow ? `<div class="page-eyebrow">${esc(eyebrow)}</div>` : ""}
        <h1 class="page-title">${esc(title)}</h1>
        ${desc ? `<p class="page-desc">${esc(desc)}</p>` : ""}
      </div>
      ${actions || ""}
    </div>`;
  }

  function renderKpiCards(k, delay) {
    const critical = parseInt(k.critical_issues, 10) || 0;
    const cards = [
      { label: "Агенты Online", value: k.agents_online, suffix: k.agents_suffix, tone: "t-primary" },
      { label: "Critical Issues", value: k.critical_issues, tone: "t-danger", pulse: critical > 0 },
      { label: "Warnings", value: k.warnings, tone: "t-warning" },
      { label: "Upstream Healthy", value: k.upstream_healthy, tone: "t-highlight" },
    ];
    return `<section class="kpi-grid">${cards
      .map(
        (c, i) => `<div class="kpi-card"${animAttr(delay + i * 60)}>
          <div class="kpi-label">${esc(c.label)}</div>
          <div class="kpi-row">
            <span class="kpi-value ${c.tone}">${esc(c.value)}</span>
            ${c.suffix ? `<span class="kpi-suffix">${esc(c.suffix)}</span>` : ""}
            ${c.pulse ? '<span class="kpi-pulse"></span>' : ""}
          </div>
        </div>`
      )
      .join("")}</section>`;
  }

  function renderHealthOverview(bars) {
    if (!bars || !bars.length) {
      return `<div class="panel"${animAttr(120)}><div class="empty">Нет данных upstream</div></div>`;
    }
    const now = new Date();
    const labels = bars.map((_, i) => {
      if (i === bars.length - 1) return "NOW";
      const m = new Date(now - (bars.length - 1 - i) * 60000);
      return m.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    });
    return `<div class="panel"${animAttr(120)}>
      <div class="panel-head">
        <h3 class="panel-title">Health Overview по агентам</h3>
        <span class="panel-badge">LIVE FEED</span>
      </div>
      <div class="health-chart">${bars
        .map((b) => {
          const h = Math.max(8, Math.min(100, b.pct < 0 ? 50 : b.pct));
          const op = 0.4 + (h / 100) * 0.5;
          return `<div class="health-bar ${b.bad ? "bad" : "ok"}" style="height:${h}%;opacity:${op}" title="${esc(b.agent)}: ${Math.round(b.pct)}%"></div>`;
        })
        .join("")}</div>
      <div class="health-axis">${labels.map((l) => `<span>${esc(l)}</span>`).join("")}</div>
    </div>`;
  }

  function renderSeverityPanel(sev) {
    const rows = [
      { label: "HIGH SEVERITY", pct: sev.high_pct, count: sev.high, fill: "severity-fill-high", text: "t-danger" },
      { label: "MEDIUM SEVERITY", pct: sev.medium_pct, count: sev.medium, fill: "severity-fill-med", text: "t-warning" },
      { label: "LOW SEVERITY", pct: sev.low_pct, count: sev.low, fill: "severity-fill-low", text: "t-primary" },
    ];
    return `<div class="panel"${animAttr(180)}>
      <h3 class="panel-title" style="margin-bottom:1.5rem">Severity Breakdown</h3>
      <div class="severity-panel">${rows
        .map(
          (r) => `<div>
            <div class="severity-row-head">
              <span>${esc(r.label)}</span>
              <span class="${r.text}">${r.pct}% (${r.count})</span>
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
        <h2 class="section-title">Active Agents Detail</h2>
        <div class="section-meta">Showing 1-${shown.length} of ${list.length} agents</div>
      </div>
      <div class="agents-feed">${shown.length ? shown.map(renderAgentRow).join("") : '<div class="empty">Агенты не настроены</div>'}</div>
    </section>`;
  }

  function renderAgentRow(s, idx) {
    const st = agentStatusLabel(s.status);
    const prev = snapPreview(s);
    const i = String((idx ?? 0) + 1).padStart(2, "0");
    const action = st.critical ? "CORRELATE" : "VIEW RAW";
    const btnCls = st.critical ? "btn-outline btn-danger-outline" : "btn-outline";
    return `<div class="agent-row${st.critical ? " critical" : ""}" data-snap="${esc(s.id)}" data-action="${st.critical ? "correlation" : "detail"}">
      <div class="agent-idx ${st.critical ? "bad" : "ok"}">${i}</div>
      <div class="agent-main">
        <div class="agent-name-row">
          <span class="agent-name">${esc(s.name)}</span>
          <span class="badge ${st.cls}">${st.label}</span>
        </div>
        <div class="agent-sub${st.critical ? " bad" : ""}">IP: ${esc(s.host)} • ${esc(st.critical ? (s.error || "unreachable") : "nginx/" + (s.version || "—"))}</div>
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
        <h4 class="analytics-title">Error Log Correlation (upstream → error log)</h4>
        ${
          corr
            ? `<div class="analytics-log">
                <div class="t-danger" style="opacity:0.8;margin-bottom:0.25rem">[error] ${esc(corr.error)}</div>
                <div class="analytics-indent">↳ Found match in upstream [${esc(corr.upstream)}]<br/>↳ Target Location: ${esc((corr.locations && corr.locations[0]) || "—")}</div>
              </div>`
            : '<div class="muted" style="font-family:var(--font-mono);font-size:0.6875rem">Нет корреляций</div>'
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
    const snaps = state.snapshots || [];
    return `
      ${renderKpiCards(state.kpi, 0)}
      <div class="overview-grid">
        ${renderHealthOverview(state.health_bars)}
        ${renderSeverityPanel(state.severity)}
      </div>
      ${renderAgentsFeed(snaps)}
      ${renderAnalyticsPanels()}
    `;
  }

  function kpiFromSnapshots(snaps) {
    let high = 0;
    let med = 0;
    for (const s of snaps) {
      if (s.status === "offline") continue;
      high += s.severity?.high || 0;
      med += s.severity?.med || 0;
    }
    const online = snaps.filter((s) => s.status !== "offline").length;
    return {
      agents_online: String(online),
      agents_suffix: "/ " + snaps.length,
      critical_issues: String(high).padStart(2, "0"),
      warnings: String(med).padStart(2, "0"),
      upstream_healthy: state.kpi.upstream_healthy,
    };
  }

  function renderAgents() {
    const snaps = filterSnapshots(state.snapshots || []);
    return `
      ${pageHeader(
        "Section · 01",
        "Agents",
        "Парк nginx-агентов. Кликните по строке для детального snapshot.",
        '<button type="button" class="btn-primary" disabled style="opacity:0.5;cursor:not-allowed">+ Add agent</button>'
      )}
      ${renderKpiCards(kpiFromSnapshots(snaps), 0)}
      <div class="agent-table-wrap">${renderAgentTable(snaps)}</div>
    `;
  }

  function renderAgentTable(snaps) {
    if (!snaps.length) return '<div class="empty">Агенты не настроены (web.hub.agents)</div>';
    const rows = snaps
      .map((s) => {
        const st = agentStatusLabel(s.status);
        const clickable = s.status !== "offline";
        return `<tr class="${clickable ? "clickable" : ""}" ${clickable ? `data-snap="${esc(s.id)}"` : ""}>
          <td>${esc(s.name)}</td>
          <td class="muted">—</td>
          <td class="muted">${esc(s.host)}</td>
          <td class="muted">${esc(s.version ? "v" + s.version : "—")}</td>
          <td>${s.access ? esc(s.access.p95_ms.toFixed(0) + "ms") : "—"}</td>
          <td>—</td>
          <td><span class="badge ${st.cls}">${st.label}</span></td>
        </tr>`;
      })
      .join("");
    return `<table class="agent-table"><thead><tr><th>Agent ID</th><th>Region</th><th>IP</th><th>Version</th><th>Latency</th><th>Uptime</th><th>Status</th></tr></thead><tbody>${rows}</tbody></table>`;
  }

  function renderSnapshotsList() {
    const snaps = filterSnapshots(state.snapshots || []);
    return `
      ${pageHeader("Section · 02", "Snapshots", "Снимки конфигурации nginx по каждому агенту. Откройте карточку для полного анализа.")}
      <div class="snap-grid">${snaps.length ? snaps.map(renderSnapCard).join("") : '<div class="empty">Нет агентов</div>'}</div>
    `;
  }

  function renderSnapCard(s) {
    if (s.status === "offline") {
      return `<div class="snap-card" data-snap="${esc(s.id)}">
        <div class="snap-card-head"><div><div class="snap-card-name">${esc(s.name)} <span class="badge badge-critical">offline</span></div><div class="snap-card-url">${esc(s.url)}</div></div><span class="muted">→</span></div>
        <div class="corr-error">${esc(s.error || "unreachable")}</div>
      </div>`;
    }
    return `<div class="snap-card" data-snap="${esc(s.id)}">
      <div class="snap-card-head">
        <div><div class="snap-card-name">${esc(s.name)} <span class="badge ${s.status === "warning" ? "badge-warning" : "badge-online"}">${esc(s.status)}</span></div><div class="snap-card-url">${esc(s.url)}</div></div>
        <span class="muted">→</span>
      </div>
      <div class="snap-scores">
        <div><div class="score-cell-label">Score</div><div class="score-cell-val t-primary">${s.config_score}<span class="kpi-suffix">/100</span></div></div>
        <div><div class="score-cell-label">High</div><div class="score-cell-val t-danger">${s.severity.high}</div></div>
        <div><div class="score-cell-label">Med</div><div class="score-cell-val t-warning">${s.severity.med}</div></div>
        <div><div class="score-cell-label">Low</div><div class="score-cell-val t-highlight">${s.severity.low}</div></div>
      </div>
      <div class="snap-card-url">nginx/${esc(s.version || "—")} · upd: ${esc(s.updated_at)}</div>
    </div>`;
  }

  function renderSnapshotDetail(id) {
    const s = (state.snapshots || []).find((x) => x.id === id);
    if (!s) {
      return `${pageHeader("", "Snapshot не найден", "")}<a class="back-link" href="#/snapshots">← Назад к списку</a>`;
    }
    if (s.status === "offline") {
      return `
        <a class="back-link" href="#/snapshots">← Back to Snapshots</a>
        ${renderDetailHeader(s)}
        <div class="panel"><div class="panel-title" style="margin-bottom:0.75rem">Agent Unreachable</div><pre class="explore-json">${esc(s.error)}</pre></div>
      `;
    }
    const cats = s.categories || {};
    const ib = s.issues_breakdown || {};
    return `
      <a class="back-link" href="#/snapshots">← Back to Snapshots</a>
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
        ["Security", cats.security, ib.security],
        ["Reliability", cats.reliability, ib.reliability],
        ["Performance", cats.performance, ib.performance],
        ["Maintainability", cats.maintainability, ib.maintainability],
        ["Observability", cats.observability, ib.observability],
      ]
        .map(([label, score, issues]) => renderCategoryCard(label, score, issues))
        .join("")}</div>
      <div class="detail-summary">
        <div class="panel">
          <div class="kpi-label" style="margin-bottom:1rem">Severity Breakdown</div>
          <div class="severity-cells">
            <div class="severity-cell"><div class="severity-cell-label">High</div><div class="severity-cell-val t-danger">${s.severity.high}</div></div>
            <div class="severity-cell"><div class="severity-cell-label">Med</div><div class="severity-cell-val t-warning">${s.severity.med}</div></div>
            <div class="severity-cell"><div class="severity-cell-label">Low</div><div class="severity-cell-val t-highlight">${s.severity.low}</div></div>
          </div>
          ${s.note ? `<div class="note-box">⚠ ${esc(s.note)}</div>` : ""}
        </div>
        <div class="panel">
          <div style="display:flex;justify-content:space-between;margin-bottom:1rem">
            <div class="kpi-label" style="margin:0">Loaded Modules</div>
            <span class="t-primary" style="font-family:var(--font-mono);font-size:0.625rem">${(s.modules || []).length} active</span>
          </div>
          <div class="module-tags">${(s.modules || []).map((m) => `<span class="module-tag">${esc(m)}</span>`).join("")}</div>
        </div>
      </div>
      <div>
        <div class="tabs">${TABS.map((t) => `<button type="button" class="tab-btn${detailTab === t ? " active" : ""}" data-tab="${esc(t)}">${esc(t)}</button>`).join("")}</div>
        <div class="tab-panel">${renderTabContent(s)}</div>
      </div>
    `;
  }

  function renderDetailHeader(s) {
    return `<div class="detail-header"${animAttr()}>
      <div>
        <div class="page-eyebrow">Snapshot · ${esc(s.id)}</div>
        <h1 class="page-title" style="font-size:1.875rem">${esc(s.name)}</h1>
        <a href="${esc(s.url)}" target="_blank" rel="noopener" class="detail-url">${esc(s.url)} ↗</a>
      </div>
      <div class="detail-actions">
        <button type="button" class="btn-outline" id="btn-copy-url">COPY URL</button>
        <button type="button" class="btn-primary" id="btn-rescan">RE-SCAN</button>
      </div>
    </div>`;
  }

  function renderCategoryCard(label, score, issues) {
    const n = Math.round(score || 0);
    const barCls = n >= 70 ? "impact-low" : n >= 50 ? "impact-med" : "impact-high";
    return `<div class="cat-card">
      <div class="cat-card-label">${esc(label)}</div>
      <div class="kpi-row"><span class="cat-card-score ${scoreTone(n)}">${n}</span><span class="kpi-suffix">/ 100</span></div>
      <div class="cat-card-bar"><div class="cat-card-bar-fill ${barCls}" style="width:${n}%"></div></div>
      <div class="muted" style="font-family:var(--font-mono);font-size:0.625rem">${issues || 0} issues</div>
    </div>`;
  }

  function renderTabContent(s) {
    switch (detailTab) {
      case "Upstream":
        return renderDataTable(
          ["Upstream", "Address", "Status", "Errors"],
          (s.upstreams || []).map((u) => [u.name, u.address, statusPill(u.status), esc(u.errors)])
        );
      case "Build":
        return renderDataTable(
          ["Parameter", "Value"],
          (s.build || []).map((b) => [`<span class="muted">${esc(b.name)}</span>`, esc(b.value)])
        );
      case "Issues":
        return renderIssuesTab(s);
      case "Certs":
        return renderDataTable(
          ["Domain", "Issuer", "Expires", "Days Left"],
          (s.certs || []).map((c) => [
            esc(c.domain),
            esc(c.issuer),
            esc(c.expires),
            `<span class="${c.days_left < 30 ? "t-danger" : "t-primary"}">${c.days_left}d</span>`,
          ])
        );
      case "Blast-radius":
        return renderBlastTab(s);
      case "Errors":
        return renderErrorsTab(s);
      case "Explore":
        return `<div class="explore-box">
          <div class="muted">Explore: интерактивный обзор конфигурации nginx (директивы, includes, locations).</div>
          <pre class="explore-json">${esc(JSON.stringify(s, null, 2))}</pre>
        </div>`;
      default:
        return "";
    }
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
        <span class="module-tag" style="color:var(--highlight);border-color:oklch(0.92 0.18 155 / 0.2);background:oklch(0.92 0.18 155 / 0.1)">location: ${esc(loc)}</span>
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
      case "snapshots":
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

  function updateKpiCardEl(card, value, suffix, tone, pulse) {
    if (!card) return;
    const valEl = card.querySelector(".kpi-value");
    if (valEl) {
      valEl.className = "kpi-value " + tone;
      valEl.textContent = value;
    }
    const suffixEl = card.querySelector(".kpi-suffix");
    if (suffix) {
      if (suffixEl) suffixEl.textContent = suffix;
      else if (valEl) valEl.insertAdjacentHTML("afterend", `<span class="kpi-suffix">${esc(suffix)}</span>`);
    } else if (suffixEl) suffixEl.remove();
    let pulseEl = card.querySelector(".kpi-pulse");
    if (pulse && !pulseEl) {
      card.querySelector(".kpi-row").insertAdjacentHTML("beforeend", '<span class="kpi-pulse"></span>');
    } else if (!pulse && pulseEl) {
      pulseEl.remove();
    }
  }

  function patchKpiGrid(kpi, delay) {
    const cards = document.querySelectorAll(".kpi-grid .kpi-card");
    if (cards.length < 4) return false;
    const critical = parseInt(kpi.critical_issues, 10) || 0;
    updateKpiCardEl(cards[0], kpi.agents_online, kpi.agents_suffix, "t-primary", false);
    updateKpiCardEl(cards[1], kpi.critical_issues, null, "t-danger", critical > 0);
    updateKpiCardEl(cards[2], kpi.warnings, null, "t-warning", false);
    updateKpiCardEl(cards[3], kpi.upstream_healthy, null, "t-highlight", false);
    return true;
  }

  function patchHealthOverview() {
    const panel = document.querySelector(".overview-grid .panel");
    if (!panel) return false;
    const bars = state.health_bars;
    if (!bars || !bars.length) return false;
    const now = new Date();
    const labels = bars.map((_, i) => {
      if (i === bars.length - 1) return "NOW";
      const m = new Date(now - (bars.length - 1 - i) * 60000);
      return m.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    });
    const chart = panel.querySelector(".health-chart");
    const axis = panel.querySelector(".health-axis");
    if (!chart || !axis) return false;
    chart.innerHTML = bars
      .map((b) => {
        const h = Math.max(8, Math.min(100, b.pct < 0 ? 50 : b.pct));
        const op = 0.4 + (h / 100) * 0.5;
        return `<div class="health-bar ${b.bad ? "bad" : "ok"}" style="height:${h}%;opacity:${op}" title="${esc(b.agent)}: ${Math.round(b.pct)}%"></div>`;
      })
      .join("");
    axis.innerHTML = labels.map((l) => `<span>${esc(l)}</span>`).join("");
    return true;
  }

  function patchSeverityPanel() {
    const panels = document.querySelectorAll(".overview-grid .panel");
    const panel = panels[1];
    if (!panel) return false;
    const sev = state.severity;
    const rows = [
      { label: "HIGH SEVERITY", pct: sev.high_pct, count: sev.high, fill: "severity-fill-high", text: "t-danger" },
      { label: "MEDIUM SEVERITY", pct: sev.medium_pct, count: sev.medium, fill: "severity-fill-med", text: "t-warning" },
      { label: "LOW SEVERITY", pct: sev.low_pct, count: sev.low, fill: "severity-fill-low", text: "t-primary" },
    ];
    const body = panel.querySelector(".severity-panel");
    if (!body) return false;
    body.innerHTML = rows
      .map(
        (r) => `<div>
            <div class="severity-row-head">
              <span>${esc(r.label)}</span>
              <span class="${r.text}">${r.pct}% (${r.count})</span>
            </div>
            <div class="severity-track"><div class="${r.fill}" style="width:${r.pct}%"></div></div>
          </div>`
      )
      .join("");
    return true;
  }

  function patchOverview() {
    if (!patchKpiGrid(state.kpi, 0)) return false;
    if (!patchHealthOverview()) return false;
    if (!patchSeverityPanel()) return false;
    const snaps = state.snapshots || [];
    const list = filterSnapshots(snaps);
    const shown = list.slice(0, 10);
    const feed = document.querySelector(".agents-feed");
    const meta = document.querySelector(".section-head .section-meta");
    if (feed) {
      feed.innerHTML = shown.length ? shown.map(renderAgentRow).join("") : '<div class="empty">Агенты не настроены</div>';
    }
    if (meta) meta.textContent = "Showing 1-" + shown.length + " of " + list.length + " agents";
    const analytics = document.querySelector(".analytics-grid");
    if (analytics) analytics.outerHTML = renderAnalyticsPanels();
    bindViewEvents();
    return true;
  }

  function patchAgents() {
    const snaps = filterSnapshots(state.snapshots || []);
    if (!patchKpiGrid(kpiFromSnapshots(snaps), 0)) {
      return false;
    }
    const wrap = document.querySelector(".agent-table-wrap");
    if (!wrap) return false;
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
      case "snapshots":
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
    if (!soft) renderNav();
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
        else navigate("snapshots/" + el.dataset.snap);
      });
    });
    document.querySelectorAll("button[data-action]").forEach((el) => {
      el.addEventListener("click", (e) => {
        e.stopPropagation();
        if (el.dataset.action === "correlation") navigate("correlation");
        else navigate("snapshots/" + el.dataset.snap);
      });
    });
    document.querySelectorAll(".agent-table tr.clickable").forEach((el) => {
      el.addEventListener("click", () => navigate("snapshots/" + el.dataset.snap));
    });
    document.querySelectorAll("[data-tab]").forEach((el) => {
      el.addEventListener("click", () => {
        detailTab = el.dataset.tab;
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
  }

  function updateMeta() {
    if (!state) return;
    const meta = state.meta || {};
    $("#meta-status").textContent = "SYSTEM " + (meta.system_status || "—");
    const sec = meta.refresh_interval || 30;
    $("#meta-refresh").textContent = "AUTO-REFRESH: " + sec + "S";
    const online = meta.agents_online || 0;
    const offline = (meta.agents_total || 0) - online;
    $("#footer-stats").innerHTML = `
      <div class="footer-stat"><span class="footer-dot" style="background:var(--primary)"></span><span class="t-primary">${online} ONLINE</span></div>
      <div class="footer-stat"><span class="footer-dot" style="background:var(--destructive)"></span><span class="t-danger">${offline} OFFLINE</span></div>`;
  }

  function updateTokenPreview() {
    const t = token();
    $("#token-preview").textContent = t ? t.slice(0, 14) + "…" + t.slice(-2) : "не задан";
  }

  function showError(msg) {
    const b = $("#error-banner");
    b.textContent = msg;
    b.classList.remove("hidden");
  }

  function hideError() {
    $("#error-banner").classList.add("hidden");
  }

  async function refresh(soft) {
    const view = $("#view");
    const scrollTop = soft && view ? view.scrollTop : 0;
    try {
      state = await fetchState();
      hideError();
      render({ soft: soft === true });
      if (soft && view) view.scrollTop = scrollTop;
    } catch (e) {
      showError("Ошибка загрузки: " + e.message);
    }
  }

  function initTokenModal() {
    const modal = $("#token-modal");
    const input = $("#token-input");
    $("#btn-token").addEventListener("click", () => {
      input.value = token();
      modal.showModal();
    });
    modal.querySelector("form").addEventListener("submit", (e) => {
      e.preventDefault();
      localStorage.setItem("nginx_lens_hub_token", input.value.trim());
      modal.close();
      updateTokenPreview();
      refresh(true);
    });
    $("#token-clear").addEventListener("click", () => {
      localStorage.removeItem("nginx_lens_hub_token");
      input.value = "";
      updateTokenPreview();
    });
  }

  window.addEventListener("hashchange", () => {
    route = parseRoute();
    render();
  });

  $("#btn-refresh").addEventListener("click", () => refresh(true));
  $("#search-input").addEventListener("input", (e) => {
    searchQuery = e.target.value.trim();
    render();
  });

  initTokenModal();
  updateTokenPreview();
  refresh(false);
  setInterval(() => refresh(true), REFRESH_MS);
})();
