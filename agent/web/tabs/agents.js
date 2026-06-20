// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Agents main tab (922 LOC). Audit pass — esc() on agent fields, CSS.escape on dynamic id, encodeURIComponent on agent_id param, modal close via ESC + button..
// 2026-06-20 (owner-approved): Tools section (Setting modal) sekarang nampilin
//   "Active capabilities (manifest, read-only)" dari a.capabilities_required — biar
//   menu JUJUR (sebelumnya cuma config.tools override yg KOSONG buat agent manifest-
//   defined kayak mr-flow → keliatan "0 tool" padahal punya 17 cap). GUI-only,
//   pakai data yg udah ada di `a`. Verified live screenshot. Re-locked.

// AI Agent gallery — kumpulan AI yang hidup di kernel.
//
// Vibe: tiap agent muncul sebagai "warga" dengan avatar unik
// (deterministik dari id-nya). Status indikator hidup, glow saat ready,
// dimm saat idle. Tombol Setting per kartu buka popup 4 section
// (Router / Prompt / Tools / Schedule + Telegram).

// IMPORTANT: import path MUST exactly match what /js/app.js uses
// (resolved to "/js/i18n.js" — no query string). ES modules are keyed
// by full URL, so `/js/i18n.js?v=11` would be a DIFFERENT instance
// with empty dict → t() balikin key mentah.
import { t } from '/js/i18n.js';
import { openRouterSkillBrowser } from './agents_router_skills.js';
import { openSlashModal } from './agents_slash_modal.js';
import { renderToolCatalog } from './agents_tool_catalog.js';

const API_LIST   = '/api/kernel/agents';
const API_UPLOAD = '/api/agents/upload';
const API_REMOVE = '/api/agents/remove';
const API_CFG    = '/api/agents/config';
const API_SCHEMA = '/api/agents/ui-schema';

const TOOL_FLAGS = ['telegram', 'router', 'kv', 'fs', 'net'];

// Palette warna seed untuk avatar — deterministik per agent id.
const AVATAR_PALETTES = [
  ['#7c3aed', '#a855f7'], // violet
  ['#0ea5e9', '#22d3ee'], // sky/cyan
  ['#10b981', '#34d399'], // emerald
  ['#f59e0b', '#fbbf24'], // amber
  ['#ec4899', '#f472b6'], // pink
  ['#6366f1', '#818cf8'], // indigo
  ['#ef4444', '#f87171'], // red
  ['#14b8a6', '#5eead4'], // teal
];

// Emoji "wajah" — pool kecil supaya tiap agent terlihat distinct.
const AVATAR_FACES = ['🤖', '🪄', '🦊', '🛸', '👾', '🐙', '🦉', '🐉', '🐺', '🪐', '⚡', '🧊'];

// ── helpers ────────────────────────────────────────────────────────────────

function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[c]));
}
// esc() di file ini SUDAH escape kutip → attribute-safe. escAttr = alias biar
// konteks atribut eksplisit (konsisten sama util global).
const escAttr = esc;

async function fetchJSON(url, opts) {
  const r = await fetch(url, opts);
  const body = await r.json().catch(() => ({}));
  if (!r.ok || body.error) throw new Error(body.error || r.statusText);
  return body;
}

// hashStr — FNV-1a 32-bit. Cukup buat seed avatar.
function hashStr(s) {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = (h + ((h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24))) >>> 0;
  }
  return h >>> 0;
}

// avatar(id) → { face, palette[2] } deterministik.
function avatarFor(id) {
  const h = hashStr(id || 'unknown');
  return {
    face:    AVATAR_FACES[h % AVATAR_FACES.length],
    palette: AVATAR_PALETTES[(h >>> 8) % AVATAR_PALETTES.length],
  };
}

function avatarHTML(id, size = 64) {
  const { face, palette } = avatarFor(id);
  return `
    <div class="ag-avatar" style="
      width:${size}px; height:${size}px;
      background: radial-gradient(circle at 30% 25%, ${palette[1]} 0%, ${palette[0]} 70%, #0f172a 110%);
      box-shadow: 0 0 18px ${palette[0]}66, inset 0 0 14px rgba(255,255,255,0.08);
    ">
      <span class="ag-avatar-face" style="font-size:${Math.round(size * 0.55)}px">${face}</span>
      <span class="ag-avatar-ring" style="border-color:${palette[1]}aa"></span>
    </div>
  `;
}

// ── render ─────────────────────────────────────────────────────────────────

export async function render(root) {
  root.innerHTML = `
    <section class="ag-tab">
      <header class="ag-hero">
        <div class="ag-hero-pulse"></div>
        <div class="ag-hero-text">
          <div class="ag-hero-eyebrow">FLOWORK · MICROKERNEL</div>
          <h2 class="ag-hero-title">${esc(t('menu.tab.agents.title'))}</h2>
          <p class="ag-hero-sub">${esc(t('menu.tab.agents.desc'))}</p>
        </div>
        <div class="ag-hero-stats">
          <div><span id="ag-stat-total">·</span><label>${t('agents.warga')}</label></div>
          <div><span id="ag-stat-ready">·</span><label>Hidup</label></div>
        </div>
      </header>

      <div class="ag-toolbar">
        <button id="ag-refresh" class="ag-btn ghost">${esc(t('menu.tab.agents.btn_refresh'))}</button>
        <button id="ag-compact-all" class="ag-btn ghost" title="Compact All — semua agent: digest pengalaman ke brain + trim konteks (anti-halu)">🧠 Compact All</button>
        <label for="ag-file" id="ag-drop" class="ag-drop">
          <span class="ag-drop-icon">📥</span>
          <span class="ag-drop-strong">${esc(t('menu.tab.agents.drop_label'))}</span>
          <span class="ag-drop-hint">${esc(t('menu.tab.agents.drop_hint'))}</span>
          <input type="file" id="ag-file" accept=".zip,.fwagent" hidden>
        </label>
      </div>
      <div id="ag-upload-msg" class="ag-msg"></div>

      <div id="ag-grid" class="ag-grid"></div>
      <div id="ag-modal-root"></div>
    </section>

    ${styles()}
  `;

  wireUpload(root);
  document.getElementById('ag-refresh').onclick = () => refreshList(root);
  const caBtn = document.getElementById('ag-compact-all');
  if (caBtn) caBtn.onclick = () => compactAllAgents(caBtn);
  refreshList(root);
}

// LOCKED (soft, owner-approved 2026-06-20): compact di menu agent (per-agent + all).
// compactOneAgent — Compact Now 1 agent: digest pengalaman → brain + trim konteks (force=abaikan
// ambang). Owner 2026-06-20 (pindah dari menu cognitive). Aman: cuma yg udah ke-brain yg di-trim.
async function compactOneAgent(id, btn) {
  if (!confirm(`Compact "${id}" sekarang?\nDigest pengalaman → brain + trim interaksi lama (sisain yg terbaru). Pengalaman ga ilang (pindah ke brain).`)) return;
  const orig = btn ? btn.textContent : '';
  if (btn) { btn.disabled = true; btn.textContent = '⏳'; }
  try {
    const r = await fetchJSON(`/api/agents/compact?id=${encodeURIComponent(id)}&force=1`, { method: 'POST' });
    alert(`🧠 ${id}: digested=${r.digested}, trimmed=${r.trimmed} (${r.note})`);
  } catch (e) { alert('❌ compact: ' + (e.message || e)); }
  finally { if (btn) { btn.disabled = false; btn.textContent = orig; } }
}

// compactAllAgents — Compact All: semua agent sekaligus (force). Bisa makan waktu (LLM digest per
// agent over-threshold). Resilient per-agent di server.
async function compactAllAgents(btn) {
  if (!confirm('Compact ALL agent sekarang?\nTiap agent: digest pengalaman → brain + trim konteks lama. Bisa beberapa menit (LLM). Pengalaman ga ilang.')) return;
  const orig = btn ? btn.textContent : '';
  if (btn) { btn.disabled = true; btn.textContent = '⏳ compacting…'; }
  try {
    const r = await fetchJSON('/api/agents/compact-all?force=1', { method: 'POST' });
    alert(`🧠 ${r.note || ('Compact All jalan di background (' + (r.agents || '?') + ' agent).')}`);
  } catch (e) { alert('❌ compact-all: ' + (e.message || e)); }
  finally { if (btn) { btn.disabled = false; btn.textContent = orig; } }
}

function styles() {
  return `
    <style>
      .ag-tab { padding: 24px 32px 60px; color: #e2e8f0; }

      /* ── Hero ── */
      .ag-hero { position: relative; overflow: hidden;
                 padding: 32px 38px; border-radius: 18px; margin-bottom: 26px;
                 background: linear-gradient(135deg, rgba(124,58,237,0.22) 0%, rgba(14,165,233,0.18) 50%, rgba(16,185,129,0.16) 100%);
                 border: 1px solid rgba(148,163,184,0.22);
                 display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 22px;
                 box-shadow: 0 18px 60px -28px rgba(124,58,237,0.45); }
      .ag-hero-pulse { position: absolute; inset: 0; pointer-events: none;
                       background:
                         radial-gradient(circle at 12% 30%, rgba(124,58,237,0.32), transparent 36%),
                         radial-gradient(circle at 88% 70%, rgba(14,165,233,0.28), transparent 34%),
                         radial-gradient(circle at 60% 0%,  rgba(16,185,129,0.18), transparent 28%);
                       animation: ag-pulse 9s ease-in-out infinite alternate; }
      @keyframes ag-pulse { from { opacity: 0.55; } to { opacity: 1; } }
      .ag-hero-text { position: relative; z-index: 1; }
      .ag-hero-eyebrow { font-size: 0.74rem; letter-spacing: 0.32em; color: #a78bfa;
                         text-transform: uppercase; margin-bottom: 8px; font-weight: 600; }
      .ag-hero-title { margin: 0; font-size: 2.3rem; line-height: 1.1; font-weight: 700;
                       background: linear-gradient(90deg, #c4b5fd, #67e8f9 55%, #6ee7b7);
                       -webkit-background-clip: text; background-clip: text; color: transparent; }
      .ag-hero-sub { margin: 10px 0 0; color: #cbd5e1; max-width: 70ch;
                     line-height: 1.55; font-size: 1rem; }
      .ag-hero-stats { display: flex; gap: 14px; position: relative; z-index: 1; }
      .ag-hero-stats > div { background: rgba(15,23,42,0.6);
                              border: 1px solid rgba(148,163,184,0.2);
                              padding: 14px 20px; border-radius: 14px;
                              text-align: center; min-width: 92px;
                              backdrop-filter: blur(4px); }
      .ag-hero-stats span { display: block; font-size: 1.9rem; font-weight: 700; color: #c4b5fd; line-height: 1; }
      .ag-hero-stats label { font-size: 0.7rem; letter-spacing: 0.2em;
                              text-transform: uppercase; color: #94a3b8; margin-top: 4px; display: block; }

      /* ── Toolbar ── */
      .ag-toolbar { display: grid; grid-template-columns: auto 1fr; gap: 14px;
                    align-items: stretch; margin-bottom: 18px; }
      .ag-btn { background: rgba(56,189,248,0.16); color: #38bdf8;
                border: 1px solid rgba(56,189,248,0.32);
                padding: 8px 14px; border-radius: 8px; cursor: pointer;
                font-size: 0.86rem; font-family: inherit; }
      .ag-btn:hover { background: rgba(56,189,248,0.26); }
      .ag-btn.ghost { background: transparent; }
      .ag-btn.danger { background: rgba(239,68,68,0.14); color: #f87171;
                       border-color: rgba(239,68,68,0.32); }
      .ag-btn.danger:hover { background: rgba(239,68,68,0.24); }
      .ag-btn.primary { background: linear-gradient(135deg, #7c3aed, #6366f1);
                        color: #fff; border-color: rgba(124,58,237,0.5);
                        box-shadow: 0 8px 24px -10px rgba(124,58,237,0.7); }

      .ag-drop { border: 2px dashed rgba(148,163,184,0.3); border-radius: 10px;
                 padding: 10px 14px; cursor: pointer; transition: all .18s;
                 display: grid; grid-template-columns: auto 1fr; column-gap: 12px;
                 grid-template-rows: auto auto; align-items: center;
                 background: rgba(15,23,42,0.4); }
      .ag-drop:hover, .ag-drop.over { background: rgba(56,189,248,0.08);
                                       border-color: #38bdf8;
                                       box-shadow: 0 0 22px -10px #38bdf8; }
      .ag-drop-icon { font-size: 1.6rem; grid-row: 1 / span 2; }
      .ag-drop-strong { font-size: 0.9rem; color: #e2e8f0; }
      .ag-drop-hint { font-size: 0.72rem; color: #64748b; }
      .ag-msg { font-size: 0.85rem; min-height: 1.2em; padding: 0 4px 12px; }
      .ag-msg.ok { color: #4ade80; } .ag-msg.err { color: #f87171; }

      /* ── Grid kartu agent ── */
      .ag-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
                 gap: 18px; }
      .ag-empty { grid-column: 1 / -1; text-align: center; padding: 60px 12px;
                  border: 1px dashed rgba(148,163,184,0.18); border-radius: 14px;
                  color: #64748b; font-size: 0.95rem; }
      .ag-card { position: relative; border-radius: 16px; padding: 20px;
                 background: linear-gradient(170deg, rgba(30,41,59,0.95), rgba(15,23,42,0.95));
                 border: 1px solid rgba(148,163,184,0.16);
                 transition: transform .15s, border-color .15s, box-shadow .15s;
                 overflow: hidden; }
      .ag-card::before { content: ''; position: absolute; inset: -1px;
                          background: linear-gradient(135deg, transparent, var(--accent, transparent) 50%, transparent);
                          border-radius: inherit; opacity: 0; transition: opacity .2s;
                          pointer-events: none; }
      .ag-card.ready::before { opacity: 0.55; }
      .ag-card:hover { transform: translateY(-3px);
                       border-color: rgba(56,189,248,0.4);
                       box-shadow: 0 14px 36px -16px rgba(56,189,248,0.4); }
      .ag-card.failed { opacity: 0.7; }
      .ag-card.off { opacity: 0.55; filter: grayscale(0.55); }
      .ag-card.off .ag-avatar-ring { animation: none; opacity: 0.2; }

      /* Switch enable/disable */
      .ag-switch { position: relative; display: inline-block;
                   width: 42px; height: 22px; cursor: pointer; }
      .ag-switch input { opacity: 0; width: 0; height: 0; }
      .ag-switch-slider { position: absolute; inset: 0;
                          background: rgba(148,163,184,0.3);
                          border: 1px solid rgba(148,163,184,0.4);
                          border-radius: 22px; transition: .2s;
                          cursor: pointer; }
      .ag-switch-slider::before { content: '';
                                  position: absolute; left: 2px; top: 1px;
                                  width: 16px; height: 16px;
                                  background: #cbd5e1; border-radius: 50%;
                                  transition: .2s; }
      .ag-switch input:checked + .ag-switch-slider {
        background: linear-gradient(135deg, #7c3aed, #6366f1);
        border-color: rgba(124,58,237,0.6); }
      .ag-switch input:checked + .ag-switch-slider::before {
        transform: translateX(20px); background: #fff;
        box-shadow: 0 0 6px rgba(255,255,255,0.5); }

      .ag-card-head { display: grid; grid-template-columns: auto 1fr auto;
                      align-items: center; gap: 12px; margin-bottom: 12px; }
      .ag-avatar { position: relative; border-radius: 16px;
                   display: grid; place-items: center; flex-shrink: 0; }
      .ag-avatar-face { line-height: 1; filter: drop-shadow(0 2px 6px rgba(0,0,0,0.3)); }
      .ag-avatar-ring { position: absolute; inset: -3px;
                        border: 2px solid; border-radius: inherit;
                        opacity: 0.55; animation: ag-spin 22s linear infinite; }
      @keyframes ag-spin { to { transform: rotate(360deg); } }

      .ag-card-name { margin: 0; font-size: 1.05rem; color: #f1f5f9;
                      display: flex; align-items: center; gap: 6px; }
      .ag-card-id   { font-size: 0.72rem; color: #64748b;
                      font-family: ui-monospace, monospace; }

      .ag-card-dot { width: 8px; height: 8px; border-radius: 50%;
                     background: #64748b; box-shadow: 0 0 0 4px rgba(100,116,139,0.18); }
      .ag-card.ready  .ag-card-dot { background: #4ade80; box-shadow: 0 0 0 4px rgba(74,222,128,0.18); animation: ag-blink 2.4s ease-in-out infinite; }
      .ag-card.failed .ag-card-dot { background: #f87171; box-shadow: 0 0 0 4px rgba(239,68,68,0.18); }
      @keyframes ag-blink { 50% { opacity: 0.45; } }

      .ag-meta { font-size: 0.78rem; color: #94a3b8; line-height: 1.5;
                 display: flex; flex-wrap: wrap; gap: 6px 12px; margin-bottom: 8px; }
      .ag-meta code { color: #c4b5fd; background: rgba(124,58,237,0.12);
                      padding: 1px 5px; border-radius: 4px; font-size: 0.72rem; }
      .ag-tags { display: flex; flex-wrap: wrap; gap: 4px; min-height: 24px;
                 margin-bottom: 12px; }
      .ag-cap { font-size: 0.7rem; padding: 2px 7px;
                background: rgba(56,189,248,0.12); color: #7dd3fc;
                border-radius: 12px; border: 1px solid rgba(56,189,248,0.22); }
      .ag-reject { color: #f87171; font-size: 0.75rem;
                   background: rgba(239,68,68,0.08); padding: 6px 8px;
                   border-radius: 6px; margin: 4px 0 8px;
                   border-left: 3px solid #f87171; }
      .ag-card-actions { display: flex; gap: 6px; }
      .ag-card-actions .ag-btn { flex: 1; font-size: 0.78rem; padding: 6px 8px; }

      /* ── Modal: full-screen overlay ── */
      .ag-modal-bg { position: fixed; inset: 0;
                     background: linear-gradient(135deg, rgba(15,23,42,0.92), rgba(2,6,23,0.96));
                     display: block; z-index: 999;
                     backdrop-filter: blur(10px) saturate(120%);
                     animation: ag-fade-in .18s ease-out; }
      @keyframes ag-fade-in { from { opacity: 0; } to { opacity: 1; } }
      .ag-modal { width: 100vw; height: 100vh; max-height: none; overflow-y: auto;
                  background: transparent; border: 0; box-shadow: none;
                  padding: 0; display: grid;
                  grid-template-rows: auto 1fr auto; }

      .ag-modal-head { position: sticky; top: 0; z-index: 10;
                       display: grid; grid-template-columns: auto 1fr auto;
                       align-items: center; gap: 18px;
                       padding: 18px 32px;
                       background: linear-gradient(180deg, rgba(15,23,42,0.95), rgba(15,23,42,0.85));
                       border-bottom: 1px solid rgba(148,163,184,0.16);
                       backdrop-filter: blur(8px); }
      .ag-modal-head h3 { margin: 0; font-size: 1.4rem; font-weight: 700;
                          background: linear-gradient(90deg, #c4b5fd, #67e8f9);
                          -webkit-background-clip: text; background-clip: text;
                          color: transparent; }
      .ag-modal-head .meta { font-size: 0.84rem; color: #64748b;
                              font-family: ui-monospace, monospace; margin-top: 2px; }
      .ag-modal-head .head-actions { display: flex; gap: 8px; }

      .ag-modal-body { padding: 28px 36px; width: 100%;
                       display: flex; flex-direction: column; gap: 18px; }

      .ag-section { padding: 22px 26px; border-radius: 16px;
                    background: linear-gradient(170deg, rgba(30,41,59,0.7), rgba(15,23,42,0.7));
                    border: 1px solid rgba(148,163,184,0.16);
                    transition: border-color .15s, box-shadow .15s; }
      .ag-section:hover { border-color: rgba(124,58,237,0.32);
                          box-shadow: 0 10px 30px -16px rgba(124,58,237,0.4); }
      .ag-section h4 { margin: 0 0 12px; font-size: 1.05rem; color: #a78bfa;
                       font-weight: 600;
                       display: flex; align-items: center; gap: 8px; }
      .ag-section h4.sub { font-size: 0.9rem; color: #94a3b8; margin-top: 18px; }

      .ag-field { display: flex; flex-direction: column; gap: 5px; margin-bottom: 12px; }
      .ag-field label { font-size: 0.82rem; color: #cbd5e1; font-weight: 500; }
      .ag-field input, .ag-field textarea, .ag-field select {
        background: rgba(15,23,42,0.7); color: inherit;
        border: 1px solid rgba(148,163,184,0.22);
        border-radius: 8px; padding: 10px 12px; font: inherit; font-size: 0.92rem;
      }
      .ag-field input[readonly] { background: rgba(15,23,42,0.4); color: #94a3b8; cursor: text; }
      .ag-field input:focus, .ag-field textarea:focus {
        outline: none; border-color: #7c3aed;
        box-shadow: 0 0 0 3px rgba(124,58,237,0.22);
      }
      .ag-field textarea { min-height: 180px; resize: vertical; line-height: 1.6;
                           font-family: ui-monospace, "Cascadia Code", monospace; font-size: 0.88rem; }

      .ag-tools-grid { display: grid; grid-template-columns: 1fr 1fr;
                       gap: 6px 14px; }
      .ag-tools-grid label { font-size: 0.92rem; cursor: pointer;
                              display: flex; align-items: center; gap: 10px;
                              padding: 10px 12px; border-radius: 8px;
                              border: 1px solid rgba(148,163,184,0.14);
                              background: rgba(15,23,42,0.4);
                              transition: all .15s; }
      .ag-tools-grid label:hover { background: rgba(124,58,237,0.12);
                                    border-color: rgba(124,58,237,0.32); }
      .ag-tools-grid input[type="checkbox"] { width: 16px; height: 16px; accent-color: #7c3aed; }

      .ag-sched-row { display: grid;
                      grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 2fr) auto;
                      gap: 8px; margin-bottom: 8px; align-items: center; }
      .ag-cred-row { display: grid;
                     grid-template-columns: minmax(0, 1fr) minmax(0, 2fr) auto auto;
                     gap: 8px; margin-bottom: 8px; align-items: center; }
      .ag-sched-row input,
      .ag-cred-row input { min-width: 0; }   /* allow shrink, no overflow */
      .ag-cred-row input { font-family: ui-monospace, monospace; font-size: 0.85rem; }
      .ag-field-help { display: block; font-size: 0.75rem; color: #64748b;
                       margin-top: 4px; line-height: 1.4; }
      .ag-cb-row { display: flex; align-items: center; gap: 10px;
                   padding: 8px 10px; border-radius: 6px;
                   border: 1px solid rgba(148,163,184,0.16);
                   background: rgba(15,23,42,0.4); cursor: pointer; }
      .ag-cb-row:hover { background: rgba(124,58,237,0.08); }
      .ag-cb-row input { width: 16px; height: 16px; accent-color: #7c3aed; }
      .ag-json { font-family: ui-monospace, "Cascadia Code", monospace;
                 font-size: 0.85rem; min-height: 120px; }

      .ag-modal-foot { position: sticky; bottom: 0; z-index: 10;
                       padding: 16px 32px;
                       display: flex; gap: 10px; justify-content: flex-end;
                       align-items: center;
                       background: linear-gradient(0deg, rgba(15,23,42,0.96), rgba(15,23,42,0.85));
                       border-top: 1px solid rgba(148,163,184,0.16);
                       backdrop-filter: blur(8px); }
      .ag-modal-foot .ag-msg-modal { margin-right: auto; }
      .ag-modal-foot .ag-btn { padding: 10px 22px; font-size: 0.92rem; }

      .ag-msg-modal { font-size: 0.88rem; min-height: 1.2em; }
      .ag-msg-modal.ok { color: #4ade80; } .ag-msg-modal.err { color: #f87171; }
    </style>
  `;
}

// ── upload ─────────────────────────────────────────────────────────────────

function wireUpload(root) {
  const drop = root.querySelector('#ag-drop');
  const file = root.querySelector('#ag-file');
  const msg  = root.querySelector('#ag-upload-msg');

  ['dragenter', 'dragover'].forEach((ev) => drop.addEventListener(ev, (e) => {
    e.preventDefault(); drop.classList.add('over');
  }));
  ['dragleave', 'drop'].forEach((ev) => drop.addEventListener(ev, (e) => {
    e.preventDefault(); drop.classList.remove('over');
  }));
  drop.addEventListener('drop', (e) => {
    if (e.dataTransfer.files?.[0]) upload(e.dataTransfer.files[0]);
  });
  file.onchange = () => { if (file.files?.[0]) upload(file.files[0]); };

  async function upload(f) {
    msg.className = 'ag-msg'; msg.textContent = `${t('menu.tab.agents.uploading')} ${f.name}…`;
    const fd = new FormData();
    fd.append('file', f);
    try {
      const r = await fetchJSON(API_UPLOAD, { method: 'POST', body: fd });
      msg.className = 'ag-msg ok';
      msg.textContent = t('menu.tab.agents.upload_ok')
        .replace('{id}', r.agent_id).replace('{files}', r.files_written);
      setTimeout(() => refreshList(root), 1500);
    } catch (err) {
      msg.className = 'ag-msg err'; msg.textContent = err.message;
    }
  }
}

// ── list / grid ────────────────────────────────────────────────────────────

async function refreshList(root) {
  const grid = root.querySelector('#ag-grid');
  grid.innerHTML = `<div class="ag-empty">⏳</div>`;
  try {
    const data = await fetchJSON(API_LIST);
    const items = (data.plugins || data.agents || []).filter((x) => x.id);
    root.querySelector('#ag-stat-total').textContent = items.length;
    root.querySelector('#ag-stat-ready').textContent = items.filter((x) => x.state === 'ready').length;
    if (!items.length) {
      grid.innerHTML = `<div class="ag-empty">${esc(t('menu.tab.agents.list_empty'))}</div>`;
      return;
    }
    grid.innerHTML = items.map(renderCard).join('');
    items.forEach((a) => wireCard(root, a));
  } catch (err) {
    grid.innerHTML = `<div class="ag-empty err">${esc(err.message)}</div>`;
  }
}

function renderCard(a) {
  const enabled = a.enabled !== false;  // default true when field absent (back-compat)
  const stateCls = !enabled ? 'off'
    : a.state === 'ready'  ? 'ready'
    : a.state === 'failed' ? 'failed' : '';
  const { palette } = avatarFor(a.id);
  const caps = (a.capabilities_required || []).slice(0, 4)
    .map((c) => `<span class="ag-cap">${esc(c.split(':')[0])}</span>`).join('');
  const moreCaps = (a.capabilities_required || []).length > 4
    ? `<span class="ag-cap">+${(a.capabilities_required.length - 4)}</span>` : '';
  const statusLabel = !enabled ? t('menu.tab.agents.card_disabled') : (a.state || '?');
  return `
    <article class="ag-card ${stateCls}" data-id="${escAttr(a.id)}"
             style="--accent:${palette[0]}66">
      <div class="ag-card-head">
        ${avatarHTML(a.id, 56)}
        <div>
          <h4 class="ag-card-name">${esc(a.display_name || a.id)}</h4>
          <div class="ag-card-id">@${esc(a.id)} · v${esc(a.version || '?')}</div>
        </div>
        <label class="ag-switch" title="${enabled ? esc(t('menu.tab.agents.card_enabled')) : esc(t('menu.tab.agents.card_disabled'))}">
          <input type="checkbox" data-action="toggle" ${enabled ? 'checked' : ''}>
          <span class="ag-switch-slider"></span>
        </label>
      </div>
      <div class="ag-meta">
        <span>🧬 <code>${esc(a.kind || '?')}</code></span>
        <span>⚙️ ${esc(statusLabel)}</span>
      </div>
      <div class="ag-tags">${caps}${moreCaps}</div>
      ${a.reject_reason ? `<div class="ag-reject">${esc(a.reject_reason)}</div>` : ''}
      <div class="ag-card-actions">
        <button class="ag-btn primary" data-action="setting">${esc(t('menu.tab.agents.btn_setting'))}</button>
        <button class="ag-btn"         data-action="diagnostics" title="Diagnostics">📊</button>
        <button class="ag-btn"         data-action="doktrin" title="Educational Errors (Doktrin) — this agent's own store">📚</button>
        <button class="ag-btn"         data-action="duplicate" title="Duplicate / copy this agent">⧉</button>
        <button class="ag-btn"         data-action="slash" title="${escAttr(t('menu.tab.agents.btn_slash_title') || 'Quick slash command')}">/</button>
        <button class="ag-btn"         data-action="download" title="${escAttr(t('menu.tab.agents.btn_download'))}">⬇</button>
        <button class="ag-btn"         data-action="compact" title="Compact Now — digest pengalaman ke brain + trim konteks (anti-halu)">🧠</button>
        <button class="ag-btn danger"  data-action="remove" title="${escAttr(t('menu.tab.agents.btn_remove'))}">🗑</button>
      </div>
    </article>
  `;
}

function wireCard(root, a) {
  const card = root.querySelector(`.ag-card[data-id="${a.id}"]`);
  if (!card) return;
  card.querySelector('[data-action="setting"]').onclick = () => openSettingModal(root, a);
  // Per-agent diagnostics (moved out of the global sidebar into the agent itself).
  card.querySelector('[data-action="diagnostics"]').onclick = () => openDiagnostics(root, a);
  // Per-agent educational errors (Doktrin), moved out of the global sidebar.
  card.querySelector('[data-action="doktrin"]').onclick = () => openDoktrin(root, a);
  // Duplicate / copy this agent (the "copas" recipe via GUI).
  card.querySelector('[data-action="duplicate"]').onclick = () => duplicateAgent(root, a);
  // Section 17 phase 2: tombol slash quick input.
  card.querySelector('[data-action="slash"]').onclick = () => openSlashModal(a.id);
  card.querySelector('[data-action="remove"]').onclick = async () => {
    const ok = confirm(t('menu.tab.agents.remove_confirm_full').replace('{id}', a.id));
    if (!ok) return;
    try {
      await fetchJSON(`${API_REMOVE}?id=${encodeURIComponent(a.id)}`, { method: 'DELETE' });
      refreshList(root);
    } catch (err) { alert(err.message); }
  };
  card.querySelector('[data-action="download"]').onclick = () => {
    // Direct browser download — server kirim Content-Disposition.
    window.location.href = `/api/agents/download?id=${encodeURIComponent(a.id)}`;
  };
  // Compact Now (owner 2026-06-20, pindah dari menu cognitive): digest pengalaman agent ini ke
  // brain + trim konteks (force=1 = abaikan ambang, owner mau sekarang). Anti-halu konteks panjang.
  card.querySelector('[data-action="compact"]').onclick = (e) => compactOneAgent(a.id, e.currentTarget);
  card.querySelector('[data-action="toggle"]').onchange = async (e) => {
    const wantDisabled = !e.target.checked;
    const meta = card.querySelector('.ag-meta span:last-child');
    const prev = meta.textContent;
    meta.textContent = wantDisabled
      ? t('menu.tab.agents.card_disabling')
      : t('menu.tab.agents.card_enabling');
    try {
      await fetchJSON(
        `/api/agents/toggle?id=${encodeURIComponent(a.id)}&disabled=${wantDisabled ? 1 : 0}`,
        { method: 'POST' });
      setTimeout(() => refreshList(root), 800);
    } catch (err) {
      meta.textContent = prev;
      e.target.checked = !wantDisabled;  // revert visual
      alert(err.message);
    }
  };
}

// ── Setting popup ──────────────────────────────────────────────────────────

// openDiagnostics — per-agent diagnostics opened from the agent card. The global
// "Diagnostics" sidebar tab was removed; diagnostics.js is now rendered scoped to
// this agent's id inside a modal.
async function openDiagnostics(root, a) {
  const host = root.querySelector('#ag-modal-root');
  host.innerHTML = `<div class="ag-modal-bg"><div class="ag-modal">
    <div class="ag-modal-head">${avatarHTML(a.id, 64)}
      <div><h3>Diagnostics — ${esc(a.display_name || a.id)}</h3>
           <div class="meta">@${esc(a.id)}</div></div>
      <div class="head-actions"><button class="ag-btn ghost" id="ag-dg-close">✕ ${esc(t('common.btn.close'))}</button></div>
    </div>
    <div class="ag-modal-body"><div id="ag-dg-mount"><p class="ag-msg-modal">⏳</p></div></div>
  </div></div>`;
  host.querySelector('#ag-dg-close').onclick = () => (host.innerHTML = '');
  try {
    const { render } = await import('./diagnostics.js');
    await render(host.querySelector('#ag-dg-mount'), a.id);
  } catch (e) {
    const m = host.querySelector('#ag-dg-mount');
    if (m) m.innerHTML = `<p class="ag-msg-modal">Diagnostics failed to load: ${esc(String(e))}</p>`;
  }
}

// openDoktrin — per-agent Educational Errors (Doktrin) opened from the agent card.
// The global "Doktrin" sidebar tab was removed; doktrin_edukasi.js is now rendered
// scoped to this agent's id, reading its OWN edu-errors store (§C).
async function openDoktrin(root, a) {
  const host = root.querySelector('#ag-modal-root');
  host.innerHTML = `<div class="ag-modal-bg"><div class="ag-modal">
    <div class="ag-modal-head">${avatarHTML(a.id, 64)}
      <div><h3>Educational Errors — ${esc(a.display_name || a.id)}</h3>
           <div class="meta">@${esc(a.id)}</div></div>
      <div class="head-actions"><button class="ag-btn ghost" id="ag-de-close">✕ ${esc(t('common.btn.close'))}</button></div>
    </div>
    <div class="ag-modal-body"><div id="ag-de-mount"><p class="ag-msg-modal">⏳</p></div></div>
  </div></div>`;
  host.querySelector('#ag-de-close').onclick = () => (host.innerHTML = '');
  try {
    const { render } = await import('./doktrin_edukasi.js');
    await render(host.querySelector('#ag-de-mount'), a.id);
  } catch (e) {
    const m = host.querySelector('#ag-de-mount');
    if (m) m.innerHTML = `<p class="ag-msg-modal">Doktrin failed to load: ${esc(String(e))}</p>`;
  }
}

// duplicateAgent — copy this agent (wasm + manifest + config, fresh brain) under a
// new id. The "copas" recipe as a button: duplicate, then tweak persona/tools.
async function duplicateAgent(root, a) {
  const suggested = (a.id + '-copy').toLowerCase();
  const newId = (prompt(`Duplicate "${a.id}" as a new agent id (a-z, 0-9, -):`, suggested) || '').trim();
  if (!newId) return;
  if (!/^[a-z][a-z0-9-]{1,63}$/.test(newId)) {
    alert('Invalid id — use a-z, 0-9, - and start with a letter.');
    return;
  }
  try {
    const r = await fetch(`/api/agents/duplicate?id=${encodeURIComponent(a.id)}&new_id=${encodeURIComponent(newId)}`, { method: 'POST' });
    const j = await r.json().catch(() => ({}));
    if (!r.ok || j.error) { alert('Duplicate failed: ' + (j.error || r.status)); return; }
    alert(`Duplicated → ${j.new_id}. Open its Setting to change the persona/tools.`);
    setTimeout(() => refreshList(root), 900); // give the hot-reload watcher time
  } catch (e) {
    alert('Duplicate error: ' + String(e));
  }
}

async function openSettingModal(root, a) {
  const host = root.querySelector('#ag-modal-root');
  host.innerHTML = `<div class="ag-modal-bg"><div class="ag-modal">
    <div class="ag-modal-head">${avatarHTML(a.id, 64)}
      <div><h3>${esc(t('menu.tab.agents.setting_title').replace('{id}', a.display_name || a.id))}</h3>
           <div class="meta">@${esc(a.id)} · v${esc(a.version || '?')}</div></div>
      <div class="head-actions"><button class="ag-btn ghost" id="ag-close-x">✕ ${esc(t('common.btn.close'))}</button></div>
    </div>
    <div class="ag-modal-body"><p class="ag-msg-modal">⏳</p></div>
  </div></div>`;
  host.querySelector('#ag-close-x').onclick = () => (host.innerHTML = '');
  document.addEventListener('keydown', escClose);

  let cfg = {};
  let schema = { sections: [] };
  try {
    const [cfgBody, schemaBody] = await Promise.all([
      fetchJSON(`${API_CFG}?id=${encodeURIComponent(a.id)}`),
      fetchJSON(`${API_SCHEMA}?id=${encodeURIComponent(a.id)}`).catch(() => ({ sections: [] })),
    ]);
    cfg = cfgBody.config || {};
    if (Array.isArray(schemaBody.sections)) schema = schemaBody;
  } catch (err) {
    host.querySelector('.ag-modal-body').innerHTML = modalErrorBodyHTML(err.message);
    return;
  }

  const router = cfg.router || {};
  // Tools default centang SEMUA kalau agent fresh (belum pernah save).
  // Backend Load omit "tools" key kalau meta.config_initialized absent →
  // cfg.tools undefined → default ALL. Setelah save sekali, backend
  // selalu kirim "tools" (even []) → respect pilihan user.
  const tools  = new Set(
    Array.isArray(cfg.tools) ? cfg.tools : TOOL_FLAGS.slice(),
  );
  const sched  = Array.isArray(cfg.schedule) ? cfg.schedule.slice() : [];
  const skills = Array.isArray(cfg.skills)   ? cfg.skills.slice()   : [];
  // Kredensial fleksibel — KEY → value (Telegram token, Google API key, dst).
  // Sumber: cfg.secrets {KEY: val}; back-compat read cfg.telegram juga.
  const credsObj = (cfg.secrets && typeof cfg.secrets === 'object') ? cfg.secrets : {};
  // Filter: skip key yang udah di-declare di schema (storage=secrets) supaya
  // ngga muncul dua kali — sekali di section Settings credentials, sekali
  // di section schema custom. Schema field punya label + validation; itu
  // primary. Creds list = escape hatch buat key extra yang ngga declared.
  const schemaSecretKeys = new Set(
    (schema.sections || []).flatMap((sec) =>
      (sec.fields || [])
        .filter((f) => (f.storage || 'secrets').toLowerCase() === 'secrets')
        .map((f) => f.key)
    )
  );
  const creds = Object.entries(credsObj)
    .filter(([k]) => !schemaSecretKeys.has(k))
    .map(([key, value]) => ({ key, value: String(value ?? '') }));

  const toolsHTML = TOOL_FLAGS.map((f) => `
    <label><input type="checkbox" data-tool="${f}" ${tools.has(f) ? 'checked' : ''}>
      <span>${esc(t('menu.tab.agents.tool_' + f))}</span></label>
  `).join('');

  host.querySelector('.ag-modal-body').innerHTML = `
    <section class="ag-section">
      <h4>📝 1. ${esc(t('menu.tab.agents.section_prompt'))}</h4>
      <div class="ag-field">
        <label>${esc(t('menu.tab.agents.prompt_lbl'))}</label>
        <textarea id="cf-prompt" placeholder="${escAttr(t('menu.tab.agents.prompt_ph'))}">${esc(cfg.prompt || '')}</textarea>
      </div>
    </section>

    <section class="ag-section">
      <h4>⏰ 2. ${esc(t('menu.tab.agents.section_schedule'))}</h4>
      <p class="ag-msg-modal" style="color:#94a3b8">${esc(t('menu.tab.agents.schedule_sub'))}</p>
      <div id="cf-sched-list"></div>
      <button class="ag-btn" id="cf-sched-add" type="button" style="margin-top:8px">${esc(t('menu.tab.agents.schedule_add'))}</button>
    </section>

    <section class="ag-section">
      <h4>🧰 3. ${esc(t('menu.tab.agents.section_tools'))}</h4>
      <p class="ag-msg-modal" style="color:#94a3b8">${esc(t('menu.tab.agents.tools_sub'))}</p>
      <div style="margin-bottom:10px;padding:8px 10px;background:#0b1020;border:1px solid #1e293b;border-radius:7px;font-size:.74rem;line-height:1.7">
        <span style="color:#34d399">● Active capabilities (manifest, read-only)</span>
        <div style="margin-top:5px;display:flex;flex-wrap:wrap;gap:5px">${
          (a.capabilities_required || []).length
            ? (a.capabilities_required).map((c) => `<span class="ag-cap" style="word-break:break-all">${esc(c)}</span>`).join('')
            : '<span style="color:#64748b">none declared</span>'
        }</div>
      </div>
      <p class="ag-msg-modal" style="color:#64748b;font-size:.72rem;margin-bottom:6px">Coarse override (config.tools) — biarin kosong = pakai capability manifest di atas:</p>
      <div class="ag-tools-grid">${toolsHTML}</div>
      <details style="margin-top:12px">
        <summary style="cursor:pointer;color:#94a3b8;font-size:13px">${esc(t('menu.tab.agents.tools_catalog_h') || '📚 Browse all registered tools (Section 13)')}</summary>
        <div id="cf-tools-catalog" data-agent-id="${escAttr(a.id)}" style="margin-top:8px"></div>
      </details>
      <details style="margin-top:10px">
        <summary style="cursor:pointer;color:#94a3b8;font-size:13px">🔗 ${esc(t('menu.tab.agents.mcp_h') || 'MCP servers — uncheck to hide from this agent')}</summary>
        <div id="cf-mcp-list" data-agent-id="${escAttr(a.id)}" style="margin-top:8px;font-size:13px;color:#cbd5e1"></div>
      </details>

      <h4 class="sub">🧩 ${esc(t('menu.tab.agents.apps_h') || 'Aplikasi yang boleh dipakai')}</h4>
      <p class="ag-msg-modal" style="color:#94a3b8;font-size:.74rem;margin-bottom:6px">${esc(t('menu.tab.agents.apps_sub') || 'Centang = agent boleh pakai app ini (langsung berlaku, tanpa restart). Daftar ikut app terinstall — app baru muncul otomatis, app dihapus ilang otomatis.')}</p>
      <div id="cf-apps-list" data-agent-id="${escAttr(a.id)}" style="font-size:13px;color:#cbd5e1">…</div>
    </section>

    <section class="ag-section">
      <h4>🪄 4. ${esc(t('menu.tab.agents.section_skills'))}</h4>
      <p class="ag-msg-modal" style="color:#94a3b8">${esc(t('menu.tab.agents.skills_sub'))}</p>
      <div id="cf-skills-list"></div>
      <div style="display:flex;gap:8px;margin-top:8px;flex-wrap:wrap">
        <button class="ag-btn" id="cf-skills-add" type="button">${esc(t('menu.tab.agents.skills_add'))}</button>
        <button class="ag-btn" id="cf-skills-browse-router" type="button" data-agent-id="${escAttr(a.id)}">${esc(t('menu.tab.agents.skills_browse_router'))}</button>
      </div>
    </section>

    <section class="ag-section">
      <h4>⚙️ 5. ${esc(t('menu.tab.agents.section_settings'))}</h4>
      <p class="ag-msg-modal" style="color:#94a3b8">${esc(t('menu.tab.agents.settings_sub'))}</p>

      <h4 class="sub">${esc(t('menu.tab.agents.settings_router_h'))}</h4>
      <p class="ag-msg-modal" style="color:#94a3b8">${esc(t('menu.tab.agents.settings_router_sub'))}</p>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:12px">
        <div class="ag-field">
          <label>${esc(t('menu.tab.agents.router_url_lbl'))}</label>
          <input id="cf-router-url" value="${escAttr(router.url || '')}" placeholder="${escAttr(t('menu.tab.agents.router_url_ph'))}">
        </div>
        <div class="ag-field">
          <label>${esc(t('menu.tab.agents.router_model_lbl'))}</label>
          <input id="cf-router-model" value="${escAttr(router.model || '')}" placeholder="${escAttr(t('menu.tab.agents.router_model_ph'))}">
        </div>
      </div>

      <h4 class="sub">${esc(t('menu.tab.agents.settings_creds_h'))}</h4>
      <p class="ag-msg-modal" style="color:#94a3b8">${esc(t('menu.tab.agents.settings_creds_sub'))}</p>
      <div id="cf-creds-list"></div>
      <button class="ag-btn" id="cf-creds-add" type="button" style="margin-top:8px">${esc(t('menu.tab.agents.settings_creds_add'))}</button>
    </section>

    ${renderSchemaSections(schema, cfg, 6)}
  `;

  // Sticky footer (di luar body biar fixed di bawah viewport).
  host.querySelector('.ag-modal').insertAdjacentHTML('beforeend', `
    <div class="ag-modal-foot">
      <div class="ag-msg-modal" id="cf-msg"></div>
      <button class="ag-btn ghost" id="ag-close">${esc(t('common.btn.close'))}</button>
      <button class="ag-btn primary" id="ag-save">${esc(t('menu.tab.agents.btn_save'))}</button>
    </div>
  `);

  const list = host.querySelector('#cf-sched-list');
  const renderSched = () => {
    if (!sched.length) {
      list.innerHTML = `<p class="ag-msg-modal" style="color:#64748b">${esc(t('menu.tab.agents.schedule_empty'))}</p>`;
      return;
    }
    list.innerHTML = sched.map((s, i) => `
      <div class="ag-sched-row" data-idx="${i}">
        <input data-k="id"   value="${escAttr(s.id || '')}"   placeholder="${escAttr(t('menu.tab.agents.schedule_id_ph'))}">
        <input data-k="cron" value="${escAttr(s.cron || '')}" placeholder="${escAttr(t('menu.tab.agents.schedule_cron_ph'))}">
        <input data-k="task" value="${escAttr(s.task || '')}" placeholder="${escAttr(t('menu.tab.agents.schedule_task_ph'))}">
        <button class="ag-btn danger" type="button" data-rm="${i}">${esc(t('menu.tab.agents.schedule_remove'))}</button>
      </div>
    `).join('');
    list.querySelectorAll('input').forEach((inp) => inp.oninput = (e) => {
      const row = e.target.closest('.ag-sched-row');
      const idx = +row.dataset.idx;
      sched[idx][e.target.dataset.k] = e.target.value;
    });
    list.querySelectorAll('[data-rm]').forEach((b) => b.onclick = () => {
      sched.splice(+b.dataset.rm, 1); renderSched();
    });
  };
  renderSched();
  host.querySelector('#cf-sched-add').onclick = () => {
    sched.push({ id: '', cron: '', task: '' }); renderSched();
  };

  // Skills list: id | trigger | instruksi | hapus.
  const skillsBox = host.querySelector('#cf-skills-list');
  const renderSkills = () => {
    if (!skills.length) {
      skillsBox.innerHTML = `<p class="ag-msg-modal" style="color:#64748b">${esc(t('menu.tab.agents.skills_empty'))}</p>`;
      return;
    }
    skillsBox.innerHTML = skills.map((s, i) => `
      <div class="ag-sched-row" data-idx="${i}">
        <input data-k="id"           value="${escAttr(s.id || '')}"           placeholder="${escAttr(t('menu.tab.agents.skill_id_ph'))}">
        <input data-k="trigger"      value="${escAttr(s.trigger || '')}"      placeholder="${escAttr(t('menu.tab.agents.skill_trigger_ph'))}">
        <input data-k="instructions" value="${escAttr(s.instructions || '')}" placeholder="${escAttr(t('menu.tab.agents.skill_instr_ph'))}">
        <button class="ag-btn danger" type="button" data-rm="${i}">${esc(t('menu.tab.agents.skills_remove'))}</button>
      </div>
    `).join('');
    skillsBox.querySelectorAll('input').forEach((inp) => inp.oninput = (e) => {
      const row = e.target.closest('.ag-sched-row');
      const idx = +row.dataset.idx;
      skills[idx][e.target.dataset.k] = e.target.value;
    });
    skillsBox.querySelectorAll('[data-rm]').forEach((b) => b.onclick = () => {
      skills.splice(+b.dataset.rm, 1); renderSkills();
    });
  };
  renderSkills();
  host.querySelector('#cf-skills-add').onclick = () => {
    skills.push({ id: '', trigger: '', instructions: '' }); renderSkills();
  };
  // Section 13 phase 3: lazy render tool catalog when <details> toggled open.
  const catalogHost = host.querySelector('#cf-tools-catalog');
  if (catalogHost) {
    const details = catalogHost.closest('details');
    let loaded = false;
    details.addEventListener('toggle', () => {
      if (details.open && !loaded) {
        loaded = true;
        renderToolCatalog(catalogHost, a.id);
      }
    });
  }
  // MCP servers checklist (per-agent opt-out): lazy render when toggled open.
  const mcpHost = host.querySelector('#cf-mcp-list');
  if (mcpHost) {
    const md = mcpHost.closest('details');
    let mLoaded = false;
    md.addEventListener('toggle', () => { if (md.open && !mLoaded) { mLoaded = true; renderMCPChecklist(mcpHost, a.id); } });
  }
  // Izin-app per-agent (GUI=truth): render langsung (penting + kelihatan), bukan lazy.
  const appsHost = host.querySelector('#cf-apps-list');
  if (appsHost) renderAppGrants(appsHost, a.id);
  // Section 7 phase 2: Browse Router Catalog — open modal yang fetch dari
  // /api/agents/router-skills/list, user pilih, Use → push ke skills[].
  host.querySelector('#cf-skills-browse-router').onclick = () => {
    openRouterSkillBrowser(a.id, (chosen) => {
      // chosen = { name, description, body } dari Router GET endpoint.
      skills.push({
        id:           chosen.name,
        trigger:      '/' + chosen.name,
        instructions: chosen.body || chosen.description || '',
      });
      renderSkills();
    });
  };

  // Credential list — KEY → value, dengan reveal toggle.
  const credsBox = host.querySelector('#cf-creds-list');
  const renderCreds = () => {
    if (!creds.length) {
      credsBox.innerHTML = `<p class="ag-msg-modal" style="color:#64748b">${esc(t('menu.tab.agents.settings_creds_empty'))}</p>`;
      return;
    }
    credsBox.innerHTML = creds.map((c, i) => `
      <div class="ag-cred-row" data-idx="${i}">
        <input data-k="key"   value="${escAttr(c.key || '')}"   placeholder="${escAttr(t('menu.tab.agents.settings_creds_key_ph'))}">
        <input data-k="value" value="${escAttr(c.value || '')}" placeholder="${escAttr(t('menu.tab.agents.settings_creds_val_ph'))}" type="password">
        <button class="ag-btn ghost" type="button" data-reveal="${i}" title="show/hide">${esc(t('menu.tab.agents.settings_creds_reveal'))}</button>
        <button class="ag-btn danger" type="button" data-rm="${i}">${esc(t('menu.tab.agents.settings_creds_rm'))}</button>
      </div>
    `).join('');
    credsBox.querySelectorAll('input').forEach((inp) => inp.oninput = (e) => {
      const row = e.target.closest('.ag-cred-row');
      const idx = +row.dataset.idx;
      creds[idx][e.target.dataset.k] = e.target.value;
    });
    credsBox.querySelectorAll('[data-reveal]').forEach((b) => b.onclick = () => {
      const row = credsBox.querySelector(`.ag-cred-row[data-idx="${b.dataset.reveal}"]`);
      const valInput = row.querySelector('[data-k="value"]');
      valInput.type = valInput.type === 'password' ? 'text' : 'password';
    });
    credsBox.querySelectorAll('[data-rm]').forEach((b) => b.onclick = () => {
      creds.splice(+b.dataset.rm, 1); renderCreds();
    });
  };
  renderCreds();
  host.querySelector('#cf-creds-add').onclick = () => {
    creds.push({ key: '', value: '' }); renderCreds();
  };

  const closeModal = () => { host.innerHTML = ''; document.removeEventListener('keydown', escClose); };
  host.querySelector('#ag-close').onclick = closeModal;
  host.querySelector('#ag-close-x').onclick = closeModal;
  host.querySelector('#ag-save').onclick = async () => {
    const msg = host.querySelector('#cf-msg');
    const secretsOut = {};
    for (const c of creds) {
      const k = (c.key || '').trim();
      if (k) secretsOut[k] = c.value;
    }
    const newCfg = {
      prompt: host.querySelector('#cf-prompt').value,
      schedule: sched.filter((s) => s.cron && s.task),
      tools: Array.from(host.querySelectorAll('[data-tool]'))
        .filter((c) => c.checked).map((c) => c.dataset.tool),
      skills: skills.filter((s) => s.id && s.instructions),
      router: {
        url:   host.querySelector('#cf-router-url').value.trim(),
        model: host.querySelector('#cf-router-model').value.trim(),
      },
      secrets: secretsOut,
    };
    // Schema-driven fields: merge ke newCfg.secrets/kv/meta. Field key
    // dari schema override / nambah ke key yang dari Credentials list
    // (Credentials list = power-user mode, schema = guided form).
    collectSchemaValues(host, schema, newCfg);
    try {
      await fetchJSON(`${API_CFG}?id=${encodeURIComponent(a.id)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newCfg),
      });
      msg.className = 'ag-msg-modal ok';
      msg.textContent = t('menu.tab.agents.save_ok');
      setTimeout(() => (host.innerHTML = ''), 1200);
    } catch (err) {
      msg.className = 'ag-msg-modal err';
      msg.textContent = `${t('menu.tab.agents.save_err')} ${err.message}`;
    }
  };
}

// ── Schema-driven field renderer ───────────────────────────────────────────
//
// Per UISchema section: bikin section card + render tiap field sesuai
// type. Storage destination ("secrets" | "kv" | "meta") nentuin row di
// cfg mana yang dibaca + di mana newCfg simpan. Default "secrets".

function getSchemaValue(cfg, field) {
  const storage = (field.storage || 'secrets').toLowerCase();
  if (storage === 'secrets') return (cfg.secrets || {})[field.key];
  if (storage === 'kv')      return (cfg.kv      || {})[field.key];
  if (storage === 'meta')    return (cfg.meta    || {})[field.key];
  return undefined;
}

// renderSchemaField — return innerHTML buat satu field. Markup pakai
// id `sf-<sectionId>-<key>` supaya save handler bisa querySelector.
function renderSchemaField(sectionId, field, cfg) {
  const v = getSchemaValue(cfg, field);
  const val = (v === undefined || v === null) ? (field.default ?? '') : v;
  const id = `sf-${sectionId}-${field.key}`;
  const ph = esc(field.placeholder || '');
  const help = field.help ? `<small class="ag-field-help">${esc(field.help)}</small>` : '';
  const labelHTML = `<label for="${id}">${esc(field.label || field.key)}${field.required ? ' *' : ''}</label>`;

  const type = (field.type || 'text').toLowerCase();
  let widget = '';
  switch (type) {
    case 'password':
      widget = `<input id="${id}" type="password" value="${escAttr(val)}" placeholder="${ph}">`;
      break;
    case 'textarea':
      widget = `<textarea id="${id}" placeholder="${ph}">${esc(val)}</textarea>`;
      break;
    case 'number':
      widget = `<input id="${id}" type="number" value="${escAttr(val)}" placeholder="${ph}">`;
      break;
    case 'checkbox': {
      const checked = (val === true || val === '1' || val === 'true') ? 'checked' : '';
      widget = `<label class="ag-cb-row"><input id="${id}" type="checkbox" ${checked}> <span>${esc(field.label || field.key)}</span></label>`;
      // Checkbox label included in widget; skip outer label.
      return `<div class="ag-field">${widget}${help}</div>`;
    }
    case 'select': {
      const opts = (field.options || []).map((o) =>
        `<option value="${escAttr(o.value)}" ${o.value === String(val) ? 'selected' : ''}>${esc(o.label || o.value)}</option>`
      ).join('');
      widget = `<select id="${id}">${opts}</select>`;
      break;
    }
    case 'json':
      widget = `<textarea id="${id}" class="ag-json" placeholder="${ph}">${esc(val)}</textarea>`;
      break;
    case 'text':
    default:
      widget = `<input id="${id}" type="text" value="${escAttr(val)}" placeholder="${ph}">`;
  }
  return `<div class="ag-field">${labelHTML}${widget}${help}</div>`;
}

function renderSchemaSections(schema, cfg, startIdx) {
  if (!schema.sections || !schema.sections.length) return '';
  return schema.sections.map((sec, i) => {
    const icon = sec.icon || '🧩';
    const num = startIdx + i;
    const fields = (sec.fields || []).map((f) => renderSchemaField(sec.id, f, cfg)).join('');
    const desc = sec.description ? `<p class="ag-msg-modal" style="color:#94a3b8">${esc(sec.description)}</p>` : '';
    return `
      <section class="ag-section" data-schema-section="${escAttr(sec.id)}">
        <h4>${icon} ${num}. ${esc(sec.title || sec.id)}</h4>
        ${desc}
        ${fields}
      </section>
    `;
  }).join('');
}

// Collect schema field values back into newCfg shape (secrets/kv/meta).
function collectSchemaValues(host, schema, newCfg) {
  newCfg.secrets = newCfg.secrets || {};
  newCfg.kv      = newCfg.kv      || {};
  newCfg.meta    = newCfg.meta    || {};
  for (const sec of schema.sections || []) {
    for (const field of sec.fields || []) {
      const id = `sf-${sec.id}-${field.key}`;
      const el = host.querySelector(`#${CSS.escape(id)}`);
      if (!el) continue;
      let val;
      const type = (field.type || 'text').toLowerCase();
      if (type === 'checkbox') val = el.checked;
      else if (type === 'number') val = el.value === '' ? null : Number(el.value);
      else val = el.value;
      const storage = (field.storage || 'secrets').toLowerCase();
      // String empty → skip (jangan simpan key kosong).
      if (val === '' || val === null || val === undefined) continue;
      const bucket = storage === 'kv' ? newCfg.kv : (storage === 'meta' ? newCfg.meta : newCfg.secrets);
      bucket[field.key] = (typeof val === 'boolean' || typeof val === 'number') ? val : String(val);
    }
  }
}

function modalErrorBodyHTML(errMsg) {
  return `
    <section class="ag-section">
      <p class="ag-msg-modal err">${esc(t('menu.tab.agents.load_err'))} ${esc(errMsg)}</p>
    </section>
  `;
}

// Close modal on ESC. Listener self-removes when modal closes.
function escClose(e) {
  if (e.key !== 'Escape') return;
  const host = document.querySelector('#ag-modal-root');
  if (host && host.innerHTML) {
    host.innerHTML = '';
    document.removeEventListener('keydown', escClose);
  }
}

// renderMCPChecklist — per-agent MCP opt-out: a checkbox per installed MCP connector
// (checked = this agent can use its tools). Unchecking hides that connector's tools
// from THIS agent's tool_search. Backed by /api/agents/mcp.
async function renderMCPChecklist(host, agentID) {
  host.innerHTML = '<span style="opacity:.6">loading…</span>';
  let data;
  try { data = await fetchJSON(`/api/agents/mcp?id=${encodeURIComponent(agentID)}`); }
  catch (e) { host.innerHTML = `<span style="color:#f87171">${esc(String(e))}</span>`; return; }
  const conns = (data && data.connectors) || [];
  if (!conns.length) {
    host.innerHTML = '<span style="opacity:.6">No MCP servers installed. Add one in Connections → MCP.</span>';
    return;
  }
  host.innerHTML = conns.map((c) =>
    `<label style="display:flex;align-items:center;gap:8px;padding:4px 0;cursor:pointer">
      <input type="checkbox" data-mcp-id="${escAttr(c.id)}" ${c.enabled ? 'checked' : ''}>
      <span>${esc(c.id)} <span style="opacity:.5">(${esc(String(c.tools))} tools)</span></span>
    </label>`).join('') +
    '<div style="margin-top:6px;font-size:11px;opacity:.5">Unchecked = hidden from this agent\'s tool search.</div>';
  const save = async () => {
    const excluded = [...host.querySelectorAll('input[data-mcp-id]')]
      .filter((i) => !i.checked).map((i) => i.getAttribute('data-mcp-id'));
    try { await fetchJSON(`/api/agents/mcp?id=${encodeURIComponent(agentID)}`, { method: 'POST', body: JSON.stringify({ excluded }) }); }
    catch (e) { alert('save failed: ' + e); }
  };
  host.querySelectorAll('input[data-mcp-id]').forEach((i) => i.addEventListener('change', save));
}

// renderAppGrants — izin-app per-agent (GUI=truth, owner 2026-06-20). List
// digerakin app TERINSTALL (/api/agents/apps → apps.Manager, BUKAN json): app
// baru muncul otomatis, app dihapus ilang otomatis. Centang = POST allow:true →
// cap app:<id> di-grant LIVE → agent langsung bisa. Uncentang = allow:false →
// dicabut. Tiap toggle langsung simpan (no save-button) — kebenaran instan.
async function renderAppGrants(host, agentID) {
  host.innerHTML = '<span style="opacity:.6">loading…</span>';
  let data;
  try { data = await fetchJSON(`/api/agents/apps?id=${encodeURIComponent(agentID)}`); }
  catch (e) { host.innerHTML = `<span style="color:#f87171">${esc(String(e))}</span>`; return; }
  const apps = (data && data.apps) || [];
  if (!apps.length) {
    host.innerHTML = '<span style="opacity:.6">Belum ada app terinstall. Pasang app dulu di tab Apps → otomatis muncul di sini.</span>';
    return;
  }
  host.innerHTML = apps.map((ap) => {
    const conn = (ap.connectors | 0);
    const tag = conn > 0
      ? `<span style="opacity:.5">(${esc(ap.id)} · ${conn} tool)</span>`
      : `<span style="color:#f59e0b">(${esc(ap.id)} · user-only, ga ada konektor agent)</span>`;
    return `<label style="display:flex;align-items:center;gap:8px;padding:4px 0;cursor:pointer">
      <input type="checkbox" data-app-id="${escAttr(ap.id)}" ${ap.permitted ? 'checked' : ''} ${conn > 0 ? '' : 'disabled'}>
      <span>${esc(ap.name || ap.id)} ${tag}</span>
    </label>`;
  }).join('') +
    '<div style="margin-top:6px;font-size:11px;opacity:.5">Uncentang = capability app:&lt;id&gt; dicabut, agent ga bisa pakai app itu. App tanpa konektor (0 tool) ga bisa dipakai agent — cuma buat user di tab Apps.</div>';
  host.querySelectorAll('input[data-app-id]').forEach((i) => i.addEventListener('change', async () => {
    const appID = i.getAttribute('data-app-id');
    i.disabled = true;
    try {
      await fetchJSON(`/api/agents/apps?id=${encodeURIComponent(agentID)}`, {
        method: 'POST', body: JSON.stringify({ app_id: appID, allow: i.checked }),
      });
    } catch (e) { alert('toggle gagal: ' + e); i.checked = !i.checked; }
    finally { i.disabled = false; }
  }));
}
