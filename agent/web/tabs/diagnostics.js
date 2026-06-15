// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Diagnostics tab (custom 604 LOC, vertical pills layout). Audit pass — esc() on all 8 section renderers, agent_id hardcoded mr-flow, encodeURIComponent on query..

import { esc, escAttr, fetchJSON, loadStyle } from '../js/utils.js';

// Agent Diagnostics — vertical-pills + content panel. Now PER-AGENT: opened from
// each agent's card (sidebar tab removed), render(root, agentId) scopes every
// section to that agent. Defaults to 'mr-flow' when no id is given (back-compat).

let AGENT_ID = 'mr-flow';

const SECTIONS = [
  { key: 'interactions',   icon: '💬', label: 'Interactions',  sub: 'Chat log in/out · Section 1',
    endpoint: '/api/agents/interactions', render: renderInteraction,
    search: (it, q) => (it.content || '').toLowerCase().includes(q) },
  { key: 'decisions',      icon: '🧭', label: 'Decisions',     sub: 'Audit verdict + rationale · Section 3',
    endpoint: '/api/agents/decisions', render: renderDecision,
    search: (d, q) => ((d.decision_type || '') + ' ' + (d.rationale || '')).toLowerCase().includes(q) },
  { key: 'mistakes',       icon: '📓', label: 'Mistakes',      sub: 'Halu/error journal + tier · Section 2/7',
    endpoint: '/api/agents/mistakes', render: renderMistake,
    search: (m, q) => ((m.title || '') + ' ' + (m.content || '')).toLowerCase().includes(q) },
  { key: 'karma',          icon: '⚡', label: 'Karma',         sub: 'Per-metric counters · Section 5',
    endpoint: '/api/agents/karma', render: renderKarma,
    search: (k, q) => (k.metric_key || '').toLowerCase().includes(q) },
  { key: 'death-letter',   icon: '🕯️', label: 'Death Letter',  sub: 'Wasiat warga AI · Section 4',
    endpoint: '/api/agents/death-letter', render: renderDeathLetter,
    search: (d, q) => ((d.subject || '') + ' ' + (d.body || '')).toLowerCase().includes(q) },
  { key: 'workspace-meta', icon: '📁', label: 'Workspace',     sub: 'Resource catalog · Section 6',
    endpoint: '/api/agents/workspace-meta', render: renderWorkspace,
    search: (w, q) => (w.path || '').toLowerCase().includes(q) },
  { key: 'tool-audit',     icon: '🔧', label: 'Tool Audit',    sub: 'Sandbox tool calls · Section 26',
    endpoint: '/api/agents/tool-audit', render: renderToolAudit,
    search: (a, q) => ((a.tool_name || '') + ' ' + (a.caller || '')).toLowerCase().includes(q) },
  { key: 'slash',          icon: '⌘', label: 'Slash',         sub: 'Slash command history · Section 13',
    endpoint: '/api/agents/slash-invocations', render: renderSlash,
    search: (s, q) => ((s.command || '') + ' ' + (s.result_text || '')).toLowerCase().includes(q) },
];

const CSS = `
.dg-root {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 100px);
  min-height: 560px;
  padding: 14px 18px;
  gap: 12px;
}

/* ── Top bar ── */
.dg-topbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background: rgba(15, 17, 26, 0.65);
  border: 1px solid var(--glass-border);
  border-radius: 12px;
  flex-shrink: 0;
}
.dg-title {
  font-family: var(--font-heading, 'Outfit', sans-serif);
  font-size: 1.05rem;
  color: #e2e8f0;
  display: flex;
  align-items: center;
  gap: 9px;
  margin: 0;
}
.dg-pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
  box-shadow: 0 0 10px #10b981;
  animation: dg-pulse 1.8s infinite ease-in-out;
}
@keyframes dg-pulse {
  0%, 100% { opacity: 0.35; }
  50% { opacity: 1; }
}
.dg-agent-tag {
  margin-left: auto;
  font-family: ui-monospace, monospace;
  font-size: 0.74rem;
  color: #c4b5fd;
  background: rgba(139, 92, 246, 0.14);
  padding: 4px 10px;
  border-radius: 6px;
  border: 1px solid rgba(139, 92, 246, 0.26);
}
.dg-refresh {
  background: rgba(139, 92, 246, 0.16);
  border: 1px solid rgba(139, 92, 246, 0.34);
  color: #c4b5fd;
  padding: 6px 12px;
  border-radius: 7px;
  font-size: 0.78rem;
  cursor: pointer;
  font-family: inherit;
}
.dg-refresh:hover { background: rgba(139, 92, 246, 0.30); }
.dg-refresh:active { transform: scale(0.97); }

/* ── Two-column layout ── */
.dg-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 12px;
  flex: 1;
  min-height: 0;
}

/* ── Left pills column ── */
.dg-side {
  background: rgba(15, 17, 26, 0.65);
  border: 1px solid var(--glass-border);
  border-radius: 12px;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
}
.dg-pill {
  display: grid;
  grid-template-columns: 22px 1fr auto;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  background: transparent;
  border: 1px solid transparent;
  color: #94a3b8;
  font-size: 0.86rem;
  font-family: inherit;
  cursor: pointer;
  text-align: left;
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}
.dg-pill:hover {
  background: rgba(148, 163, 184, 0.07);
  color: #cbd5e1;
}
.dg-pill.active {
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.22), rgba(124, 58, 237, 0.10));
  border-color: rgba(139, 92, 246, 0.38);
  color: #e2e8f0;
  box-shadow: inset 3px 0 0 #8b5cf6;
}
.dg-pill .dg-pill-icon { font-size: 1.05rem; line-height: 1; text-align: center; }
.dg-pill .dg-pill-label { font-weight: 500; }
.dg-pill .dg-pill-count {
  background: rgba(148, 163, 184, 0.18);
  color: #94a3b8;
  font-family: ui-monospace, monospace;
  font-size: 0.7rem;
  padding: 1px 7px;
  border-radius: 999px;
  min-width: 22px;
  text-align: center;
  font-weight: 600;
}
.dg-pill.active .dg-pill-count {
  background: rgba(139, 92, 246, 0.34);
  color: #ddd6fe;
}

/* ── Right content panel ── */
.dg-content {
  display: flex;
  flex-direction: column;
  background: rgba(15, 17, 26, 0.65);
  border: 1px solid var(--glass-border);
  border-radius: 12px;
  overflow: hidden;
  min-width: 0;
}
.dg-content-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.10);
}
.dg-content-head h3 {
  margin: 0;
  font-size: 0.96rem;
  color: #e2e8f0;
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-heading, 'Outfit', sans-serif);
}
.dg-content-head .dg-content-sub {
  font-size: 0.74rem;
  color: #94a3b8;
}
.dg-filter {
  margin-left: auto;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 7px;
  padding: 6px 12px;
  font: inherit;
  font-size: 0.8rem;
  color: #e2e8f0;
  width: 220px;
}
.dg-filter:focus {
  outline: none;
  border-color: #7c3aed;
  box-shadow: 0 0 0 3px rgba(124, 58, 237, 0.22);
}

.dg-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* ── Row ── */
.dg-row {
  background: rgba(15, 23, 42, 0.42);
  border: 1px solid rgba(148, 163, 184, 0.08);
  border-radius: 9px;
  padding: 10px 14px;
  transition: background 0.12s, border-color 0.12s;
}
.dg-row:hover {
  background: rgba(15, 23, 42, 0.62);
  border-color: rgba(139, 92, 246, 0.22);
}
.dg-row-head {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 5px;
  flex-wrap: wrap;
}
.dg-tag {
  font-size: 0.64rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(139, 92, 246, 0.16);
  color: #c4b5fd;
  font-family: ui-monospace, monospace;
  font-weight: 600;
}
.dg-tag.ok       { background: rgba(16, 185, 129, 0.18); color: #6ee7b7; }
.dg-tag.warn     { background: rgba(245, 158, 11, 0.20); color: #fcd34d; }
.dg-tag.err      { background: rgba(239, 68, 68, 0.20);  color: #fca5a5; }
.dg-tag.muted    { background: rgba(148, 163, 184, 0.14); color: #94a3b8; }
.dg-tag.tier-promoted { background: rgba(245, 158, 11, 0.22); color: #fcd34d; }
.dg-tag.tier-raw      { background: rgba(148, 163, 184, 0.14); color: #94a3b8; }
.dg-meta {
  font-size: 0.72rem;
  color: #94a3b8;
}
.dg-time {
  margin-left: auto;
  font-size: 0.7rem;
  color: #64748b;
  font-family: ui-monospace, monospace;
}
.dg-title-row {
  font-weight: 600;
  color: #e2e8f0;
  margin-bottom: 3px;
  font-size: 0.88rem;
}
.dg-body {
  color: #cbd5e1;
  font-size: 0.82rem;
  line-height: 1.5;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.dg-path {
  font-family: ui-monospace, monospace;
  font-size: 0.76rem;
  color: #93c5fd;
  word-break: break-all;
}

/* ── Karma metric row ── */
.dg-metric {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: rgba(15, 23, 42, 0.42);
  border: 1px solid rgba(148, 163, 184, 0.08);
  border-radius: 9px;
  padding: 12px 16px;
}
.dg-metric-k {
  color: #cbd5e1;
  font-family: ui-monospace, monospace;
  font-size: 0.86rem;
}
.dg-metric-v {
  color: #c4b5fd;
  font-family: ui-monospace, monospace;
  font-size: 1rem;
  font-weight: 700;
}
.dg-metric-v .dg-metric-count {
  color: #64748b;
  font-weight: 400;
  font-size: 0.74rem;
  margin-left: 6px;
}

/* ── States ── */
.dg-empty, .dg-loading, .dg-err {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 50px 20px;
  color: #94a3b8;
  font-size: 0.84rem;
  text-align: center;
}
.dg-empty-icon { font-size: 2.2rem; margin-bottom: 10px; opacity: 0.5; }
.dg-empty-msg { max-width: 360px; line-height: 1.55; color: #94a3b8; }
.dg-loading::before {
  content: '';
  width: 13px; height: 13px;
  border: 2px solid rgba(139, 92, 246, 0.32);
  border-top-color: #c4b5fd;
  border-radius: 50%;
  margin-right: 10px;
  animation: dg-spin 0.9s linear infinite;
}
@keyframes dg-spin { to { transform: rotate(360deg); } }
.dg-err {
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.26);
  color: #fca5a5;
  border-radius: 9px;
  padding: 14px 18px;
  margin: 14px;
  flex-direction: row;
}

@media (max-width: 920px) {
  .dg-layout { grid-template-columns: 64px 1fr; }
  .dg-pill { grid-template-columns: 22px; }
  .dg-pill .dg-pill-label, .dg-pill .dg-pill-count { display: none; }
}
`;

function fmtTime(s) {
  if (!s) return '—';
  try {
    return new Date(s).toLocaleString('id-ID', { hour12: false, dateStyle: 'short', timeStyle: 'short' });
  } catch { return s; }
}

function classify(status) {
  const s = (status || '').toLowerCase();
  if (!s) return 'muted';
  if (s === 'success' || s === 'ok' || s === 'allowed' || s === 'approved') return 'ok';
  if (s.includes('pending') || s.includes('warn'))                          return 'warn';
  if (s.includes('error') || s.includes('fail') || s.includes('denied') || s.includes('blocked')) return 'err';
  return 'muted';
}

// ── Renderers per Section ──

function renderInteraction(it) {
  const dir = (it.direction || '').toLowerCase();
  const tag = dir === 'in' ? 'IN' : dir === 'out' ? 'OUT' : (it.channel || '?').toUpperCase();
  const tagClass = dir === 'in' ? 'ok' : dir === 'out' ? '' : 'muted';
  return `
    <div class="dg-row">
      <div class="dg-row-head">
        <span class="dg-tag ${tagClass}">${esc(tag)}</span>
        <span class="dg-meta">${esc(it.channel || '')} · ${esc(it.actor || '—')}</span>
        <span class="dg-time">${esc(fmtTime(it.created_at || it.ts))}</span>
      </div>
      <div class="dg-body">${esc(it.content || '—')}</div>
    </div>`;
}

function renderDecision(d) {
  const cls = classify(d.outcome);
  return `
    <div class="dg-row">
      <div class="dg-row-head">
        <span class="dg-tag">${esc(d.decision_type || 'decision')}</span>
        <span class="dg-tag ${cls}">${esc(d.outcome || '—')}</span>
        <span class="dg-time">${esc(fmtTime(d.occurred_at || d.created_at))}</span>
      </div>
      <div class="dg-body">${esc(d.rationale || '—')}</div>
    </div>`;
}

function renderMistake(m) {
  const tierClass = `tier-${(m.tier || 'raw').toLowerCase()}`;
  return `
    <div class="dg-row">
      <div class="dg-row-head">
        <span class="dg-tag ${tierClass}">${esc(m.tier || 'raw')}</span>
        <span class="dg-tag muted">${esc(m.category || 'misc')}</span>
        <span class="dg-meta">hit×${esc(String(m.hit_count || 0))}</span>
        <span class="dg-time">${esc(fmtTime(m.last_hit_at || m.created_at))}</span>
      </div>
      <div class="dg-title-row">${esc(m.title || '—')}</div>
      <div class="dg-body">${esc(m.content || '')}</div>
    </div>`;
}

function renderKarma(k) {
  const hasCount = Number(k.metric_count) > 0;
  const v = Number(k.metric_value);
  const disp = Number.isFinite(v) ? (Number.isInteger(v) ? String(v) : v.toFixed(2)) : esc(String(k.metric_value));
  return `
    <div class="dg-metric">
      <span class="dg-metric-k">${esc(k.metric_key)}</span>
      <span class="dg-metric-v">${disp}${hasCount ? `<span class="dg-metric-count">×${esc(String(k.metric_count))} samples</span>` : ''}</span>
    </div>`;
}

function renderDeathLetter(d) {
  return `
    <div class="dg-row">
      <div class="dg-row-head">
        <span class="dg-tag">${esc(d.letter_type || 'reflection')}</span>
        <span class="dg-meta">→ ${esc(d.recipient || 'all')}</span>
        <span class="dg-time">${esc(fmtTime(d.written_at))}</span>
      </div>
      <div class="dg-title-row">${esc(d.subject || '—')}</div>
      <div class="dg-body">${esc(d.body || '')}</div>
    </div>`;
}

function renderWorkspace(w) {
  return `
    <div class="dg-row">
      <div class="dg-row-head">
        <span class="dg-tag">${esc(w.category || 'misc')}</span>
        <span class="dg-meta">${esc(String(w.size_bytes || 0))} B${w.shareable ? ' · shared' : ''}</span>
        <span class="dg-time">${esc(fmtTime(w.updated_at))}</span>
      </div>
      <div class="dg-path">${esc(w.path || '—')}</div>
      ${w.description ? `<div class="dg-body" style="margin-top:4px">${esc(w.description)}</div>` : ''}
    </div>`;
}

function renderToolAudit(a) {
  const cls = classify(a.decision);
  return `
    <div class="dg-row">
      <div class="dg-row-head">
        <span class="dg-tag">${esc(a.tool_name || '?')}</span>
        <span class="dg-tag ${cls}">${esc(a.decision || '—')}</span>
        <span class="dg-meta">${esc(a.caller || '—')}</span>
        <span class="dg-time">${esc(fmtTime(a.occurred_at))}</span>
      </div>
      <div class="dg-body">${esc(a.reason || '—')}</div>
    </div>`;
}

function renderSlash(s) {
  const preview = (s.result_text || '').slice(0, 200);
  return `
    <div class="dg-row">
      <div class="dg-row-head">
        <span class="dg-tag">/${esc(s.command || '?')}</span>
        ${s.args ? `<span class="dg-meta">${esc(s.args)}</span>` : ''}
        <span class="dg-meta">${esc(s.caller || '—')}</span>
        <span class="dg-meta">${esc(String(s.duration_ms || 0))}ms</span>
        <span class="dg-time">${esc(fmtTime(s.invoked_at))}</span>
      </div>
      ${preview ? `<div class="dg-body">${esc(preview)}</div>` : ''}
    </div>`;
}

// ── Orchestration ──

export async function render(root, agentId) {
  if (agentId) AGENT_ID = agentId;
  loadStyle('diagnostics', CSS);

  const pillsHTML = SECTIONS.map((s, i) => `
    <button class="dg-pill${i === 0 ? ' active' : ''}" data-key="${s.key}" title="${escAttr(s.label)}">
      <span class="dg-pill-icon">${s.icon}</span>
      <span class="dg-pill-label">${esc(s.label)}</span>
      <span class="dg-pill-count" data-count="${s.key}">·</span>
    </button>
  `).join('');

  root.innerHTML = `
    <div class="dg-root">
      <header class="dg-topbar">
        <h2 class="dg-title"><span class="dg-pulse"></span> Diagnostics</h2>
        <span class="dg-agent-tag">agent=${esc(AGENT_ID)}</span>
        <button class="dg-refresh" id="dgRefresh">⟳ Refresh</button>
      </header>

      <div class="dg-layout">
        <nav class="dg-side">${pillsHTML}</nav>
        <section class="dg-content">
          <div class="dg-content-head">
            <h3 id="dgTitle">…</h3>
            <span class="dg-content-sub" id="dgSub"></span>
            <input class="dg-filter" id="dgFilter" type="text" placeholder="Filter…" />
          </div>
          <div class="dg-list" id="dgList"><div class="dg-loading">Loading…</div></div>
        </section>
      </div>
    </div>
  `;

  const cache = new Map();
  let activeKey = SECTIONS[0].key;
  let filterText = '';

  async function loadSection(key) {
    const sec = SECTIONS.find((s) => s.key === key);
    const listEl = root.querySelector('#dgList');
    const titleEl = root.querySelector('#dgTitle');
    const subEl = root.querySelector('#dgSub');
    const countEl = root.querySelector(`[data-count="${key}"]`);
    if (key === activeKey) {
      titleEl.innerHTML = `${sec.icon} ${esc(sec.label)}`;
      subEl.textContent = sec.sub;
    }
    try {
      const data = await fetchJSON(`${sec.endpoint}?id=${encodeURIComponent(AGENT_ID)}&limit=100`);
      const items = Array.isArray(data.items) ? data.items : [];
      cache.set(key, { sec, items });
      if (countEl) countEl.textContent = items.length;
      if (key === activeKey) paintList(key);
    } catch (err) {
      if (countEl) countEl.textContent = '!';
      if (key === activeKey) {
        listEl.innerHTML = `<div class="dg-err">Gagal load ${esc(sec.label)}: ${esc(err.message || String(err))}</div>`;
      }
    }
  }

  function paintList(key) {
    const entry = cache.get(key);
    if (!entry) return;
    const { sec, items } = entry;
    const listEl = root.querySelector('#dgList');
    const q = (filterText || '').trim().toLowerCase();
    const filtered = q ? items.filter((it) => sec.search(it, q)) : items;
    if (filtered.length === 0) {
      const msg = q
        ? `Ngga ada entry cocok "${q}".`
        : `Belum ada data ${sec.label.toLowerCase()} untuk ${esc(AGENT_ID)}.`;
      listEl.innerHTML = `
        <div class="dg-empty">
          <div class="dg-empty-icon">${sec.icon}</div>
          <div class="dg-empty-msg">${esc(msg)}</div>
        </div>`;
      return;
    }
    listEl.innerHTML = filtered.map(sec.render).join('');
  }

  function setActive(key) {
    activeKey = key;
    filterText = '';
    root.querySelectorAll('.dg-pill').forEach((b) => b.classList.toggle('active', b.dataset.key === key));
    const filterEl = root.querySelector('#dgFilter');
    if (filterEl) filterEl.value = '';
    const sec = SECTIONS.find((s) => s.key === key);
    root.querySelector('#dgTitle').innerHTML = `${sec.icon} ${esc(sec.label)}`;
    root.querySelector('#dgSub').textContent = sec.sub;
    if (cache.has(key)) {
      paintList(key);
      loadSection(key).catch(() => {});
    } else {
      root.querySelector('#dgList').innerHTML = `<div class="dg-loading">Loading ${esc(sec.label)}…</div>`;
      loadSection(key);
    }
  }

  root.querySelectorAll('.dg-pill').forEach((b) => {
    b.onclick = () => setActive(b.dataset.key);
  });
  root.querySelector('#dgFilter').oninput = (e) => {
    filterText = e.target.value;
    paintList(activeKey);
  };
  root.querySelector('#dgRefresh').onclick = async () => {
    cache.clear();
    SECTIONS.forEach((s) => {
      const el = root.querySelector(`[data-count="${s.key}"]`);
      if (el) el.textContent = '·';
    });
    await loadSection(activeKey);
    SECTIONS.filter((s) => s.key !== activeKey).forEach((s) => loadSection(s.key).catch(() => {}));
  };

  for (const s of SECTIONS) loadSection(s.key).catch(() => {});
}
