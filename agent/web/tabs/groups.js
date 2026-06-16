// groups.js — tab "Group": the ant colony.
//
// A GROUP is a team of small single-job agents + a synthesizer that fuses their
// answers. Mr.Flow delegates deep analysis to a group over the loket bus → members
// work in parallel → synthesizer merges → reply. Tiny prompt per ant → a small/local
// model can run it (anti over-prompt).
//
// Look: matches the AI Agent tab — calm glass cards, soft violet→cyan→emerald
// gradient, deterministic agent avatars for each member. No HUD animation: AI-native,
// tidy, professional. All copy via the i18n dictionary (en base + id) — no hardcode.
//
// API: /api/groups (list + available members), /api/groups/{config,create,delete}.
// Roster lives in each group's loket store, read LIVE by the wasm (no restart).

import { esc, escAttr, fetchJSON, loadStyle } from '../js/utils.js';
import { t } from '/js/i18n.js';

const L = new Proxy({}, { get: (_, k) => t('groups.' + String(k)) });
const fmt = (k, vars) => Object.entries(vars || {}).reduce((s, [n, v]) => s.replaceAll('{' + n + '}', v), L[k]);

// ── deterministic agent avatar (same vocabulary as the AI Agent tab) ──────────
const AV_PALETTES = [
  ['#7c3aed', '#a855f7'], ['#0ea5e9', '#22d3ee'], ['#10b981', '#34d399'], ['#f59e0b', '#fbbf24'],
  ['#ec4899', '#f472b6'], ['#6366f1', '#818cf8'], ['#ef4444', '#f87171'], ['#14b8a6', '#5eead4'],
];
const AV_FACES = ['🤖', '🪄', '🦊', '🛸', '👾', '🐙', '🦉', '🐉', '🐺', '🪐', '⚡', '🧊'];
function hashStr(s) {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) { h ^= s.charCodeAt(i); h = (h + ((h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24))) >>> 0; }
  return h >>> 0;
}
function avatarFor(id) {
  const h = hashStr(id || 'unknown');
  return { face: AV_FACES[h % AV_FACES.length], palette: AV_PALETTES[(h >>> 8) % AV_PALETTES.length] };
}
function avatar(id, size) {
  const { face, palette } = avatarFor(id);
  return `<span class="gr-av" style="width:${size}px;height:${size}px;font-size:${Math.round(size * 0.5)}px;
    background:radial-gradient(circle at 30% 25%, ${palette[1]} 0%, ${palette[0]} 72%, #0f172a 115%);
    box-shadow:0 0 0 1px ${palette[1]}55, inset 0 0 10px rgba(255,255,255,0.08)">${face}</span>`;
}

const CSS = `
.gr-tab { padding:24px 32px 60px; color:#e2e8f0; }

/* ── hero (matches AI Agent) ── */
.gr-hero { position:relative; overflow:hidden; padding:30px 36px; border-radius:18px; margin-bottom:24px;
  background:linear-gradient(135deg, rgba(124,58,237,0.20) 0%, rgba(14,165,233,0.16) 52%, rgba(16,185,129,0.15) 100%);
  border:1px solid rgba(148,163,184,0.22); display:grid; grid-template-columns:1fr auto; align-items:center; gap:22px;
  box-shadow:0 18px 60px -30px rgba(124,58,237,0.4); }
.gr-eyebrow { font-size:0.72rem; letter-spacing:0.32em; color:#a78bfa; text-transform:uppercase; font-weight:600; margin-bottom:8px; }
.gr-h1 { margin:0; font-size:2.1rem; line-height:1.1; font-weight:700;
  background:linear-gradient(90deg,#c4b5fd,#67e8f9 55%,#6ee7b7); -webkit-background-clip:text; background-clip:text; color:transparent; }
.gr-hsub { margin:10px 0 0; color:#cbd5e1; max-width:72ch; line-height:1.55; font-size:0.95rem; }
.gr-stat { background:rgba(15,23,42,0.6); border:1px solid rgba(148,163,184,0.2); padding:14px 22px; border-radius:14px; text-align:center; min-width:104px; backdrop-filter:blur(4px); }
.gr-stat b { display:block; font-size:1.9rem; font-weight:700; color:#c4b5fd; line-height:1; }
.gr-stat label { font-size:0.66rem; letter-spacing:0.18em; text-transform:uppercase; color:#94a3b8; margin-top:5px; display:block; }

/* ── create bar ── */
.gr-create-bar { display:flex; gap:10px; flex-wrap:wrap; align-items:center; margin-bottom:24px;
  background:rgba(15,23,42,0.5); border:1px solid rgba(148,163,184,0.16); border-radius:14px; padding:14px 16px; }
.gr-create-bar .gr-in { flex:1; min-width:200px; }

/* ── inputs ── */
.gr-in, .gr-sel, .gr-task { background:rgba(2,6,18,0.55); border:1px solid rgba(148,163,184,0.2); border-radius:9px;
  color:#e2e8f0; padding:9px 12px; font:inherit; font-size:0.9rem; transition:border-color .15s; }
.gr-in:focus, .gr-sel:focus, .gr-task:focus { outline:none; border-color:#a78bfa; }
.gr-in::placeholder, .gr-task::placeholder { color:#64748b; }

/* ── colony cards — responsive GRID (row-major: kartu mengalir kiri→kanan lalu turun,
   kolom rapi sejajar). align-items:start = tiap kartu setinggi isinya, baris gak maksa
   sama-tinggi. Lebih intuitif drpd column-masonry (yg ngurut kolom-dulu = bikin bingung). */
#grList { display:grid; grid-template-columns:repeat(auto-fill, minmax(360px, 1fr)); gap:20px; align-items:start; }
#grList > .gr-empty { grid-column:1 / -1; }
.gr-card { background:rgba(15,23,42,0.6); border:1px solid rgba(148,163,184,0.18); border-radius:16px; padding:20px 22px;
  backdrop-filter:blur(4px); transition:border-color .18s, transform .18s, box-shadow .18s;
  min-width:0; }
.gr-card:hover { border-color:rgba(167,139,250,0.5); transform:translateY(-2px); box-shadow:0 16px 40px -26px rgba(124,58,237,0.5); }

.gr-head { display:flex; align-items:center; gap:13px; }
.gr-av { display:inline-flex; align-items:center; justify-content:center; border-radius:50%; flex:0 0 auto; }
.gr-head-text { flex:1; min-width:0; }
.gr-name { width:100%; font-size:1.05rem; font-weight:600; color:#f1f5f9; background:transparent; border:1px solid transparent; border-radius:7px; padding:4px 7px; margin:-4px -7px 0; }
.gr-name:hover { border-color:rgba(148,163,184,0.2); } .gr-name:focus { background:rgba(2,6,18,0.55); border-color:#a78bfa; outline:none; }
.gr-gid { font-family:ui-monospace,monospace; font-size:0.74rem; color:#64748b; margin-top:3px; padding-left:1px; }
.gr-tag { font-size:0.6rem; letter-spacing:0.16em; font-weight:700; color:#a78bfa; border:1px solid rgba(167,139,250,0.4);
  background:rgba(124,58,237,0.12); border-radius:999px; padding:3px 9px; align-self:flex-start; }

.gr-sec { font-size:0.64rem; letter-spacing:0.18em; text-transform:uppercase; color:#94a3b8; margin:18px 0 8px; }

/* ── member avatar chips ── */
.gr-members { display:flex; flex-wrap:wrap; gap:7px; }
.gr-chip { display:inline-flex; align-items:center; gap:7px; padding:4px 11px 4px 4px; border-radius:999px;
  border:1px solid rgba(148,163,184,0.2); background:rgba(2,6,18,0.4); color:#94a3b8; cursor:pointer; user-select:none;
  font-size:0.82rem; transition:all .15s; }
.gr-chip input { display:none; }
.gr-chip:hover { border-color:rgba(148,163,184,0.4); color:#cbd5e1; }
.gr-chip.on { border-color:rgba(103,232,249,0.55); background:rgba(14,165,233,0.14); color:#e2e8f0; box-shadow:0 0 0 1px rgba(103,232,249,0.25); }
.gr-chip.on .gr-av { box-shadow:0 0 0 2px rgba(103,232,249,0.6); }

.gr-task { width:100%; min-height:52px; resize:vertical; margin-top:2px; box-sizing:border-box; }
.gr-sched { font-size:0.78rem; color:#7dd3fc; margin-top:14px; display:flex; align-items:center; gap:6px; }

/* ── buttons ── */
.gr-foot { display:flex; align-items:center; gap:10px; margin-top:18px; }
.gr-btn { padding:8px 16px; border-radius:9px; font:inherit; font-size:0.84rem; font-weight:600; cursor:pointer;
  border:1px solid rgba(148,163,184,0.25); background:rgba(56,189,248,0.14); color:#7dd3fc; transition:all .15s; }
.gr-btn:hover { background:rgba(56,189,248,0.24); border-color:rgba(125,211,252,0.5); }
.gr-btn:disabled { opacity:.5; cursor:default; }
.gr-btn.primary { background:linear-gradient(90deg,#7c3aed,#0ea5e9); border-color:transparent; color:#fff; }
.gr-btn.primary:hover { filter:brightness(1.12); }
.gr-btn.danger { background:transparent; border-color:rgba(248,113,113,0.4); color:#f87171; }
.gr-btn.danger:hover { background:rgba(248,113,113,0.12); }
.gr-msg { font-size:0.8rem; }
.gr-empty { color:#64748b; font-size:0.86rem; padding:10px 0; }

/* ── architect bar (build a team from a prompt) ── */
.gr-arch { position:relative; overflow:hidden; margin-bottom:24px; padding:18px 20px; border-radius:14px;
  background:linear-gradient(135deg, rgba(124,58,237,0.16), rgba(16,185,129,0.10));
  border:1px solid rgba(167,139,250,0.30); }
.gr-arch-title { font-size:1.05rem; font-weight:700; color:#e9d5ff; margin:2px 0 4px; }
.gr-arch-sub { font-size:0.84rem; color:#cbd5e1; margin:0 0 12px; line-height:1.5; max-width:82ch; }
.gr-arch-row { display:flex; gap:10px; flex-wrap:wrap; align-items:center; }
.gr-arch-row .gr-in { flex:1; min-width:240px; }
.gr-arch-msg { font-size:0.82rem; margin-top:10px; display:none; }
.gr-spin { display:inline-block; width:13px; height:13px; border:2px solid rgba(167,139,250,0.35); border-top-color:#a78bfa;
  border-radius:50%; animation:gr-spin .7s linear infinite; vertical-align:-2px; margin-right:7px; }
@keyframes gr-spin { to { transform:rotate(360deg); } }

/* ── chat modal (talk to a group: coordinator fan-out + synth) ── */
.gr-modal { position:fixed; inset:0; z-index:90; display:flex; align-items:center; justify-content:center;
  background:rgba(2,6,18,0.72); backdrop-filter:blur(3px); padding:24px; }
.gr-modal-box { width:min(720px,100%); max-height:86vh; display:flex; flex-direction:column;
  background:linear-gradient(180deg, rgba(17,24,39,0.98), rgba(2,6,18,0.98)); border:1px solid rgba(148,163,184,0.25);
  border-radius:16px; box-shadow:0 30px 90px -30px rgba(0,0,0,0.8); overflow:hidden; }
.gr-modal-head { display:flex; align-items:center; gap:12px; padding:16px 20px; border-bottom:1px solid rgba(148,163,184,0.16); }
.gr-name2 { font-size:1.0rem; font-weight:700; color:#f1f5f9; }
.gr-gid2 { font-family:ui-monospace,monospace; font-size:0.72rem; color:#64748b; margin-top:2px; }
.gr-modal-x { margin-left:auto; background:transparent; border:1px solid rgba(148,163,184,0.3); color:#94a3b8;
  border-radius:8px; padding:6px 12px; cursor:pointer; font:inherit; font-size:0.8rem; }
.gr-modal-x:hover { color:#e2e8f0; border-color:rgba(148,163,184,0.5); }
.gr-log { flex:1; overflow-y:auto; padding:18px 20px; display:flex; flex-direction:column; gap:14px; }
.gr-bubble { max-width:90%; padding:11px 14px; border-radius:13px; font-size:0.9rem; line-height:1.55; word-wrap:break-word; }
.gr-bubble.me { align-self:flex-end; background:linear-gradient(90deg,#7c3aed,#0ea5e9); color:#fff; border-bottom-right-radius:4px; white-space:pre-wrap; }
.gr-bubble.them { align-self:flex-start; background:rgba(15,23,42,0.85); border:1px solid rgba(148,163,184,0.2); color:#e2e8f0; border-bottom-left-radius:4px; }
.gr-bubble.them h2,.gr-bubble.them h3,.gr-bubble.them h4 { margin:.5em 0 .3em; color:#c4b5fd; line-height:1.25; }
.gr-bubble.them h2 { font-size:1.05rem; } .gr-bubble.them h3 { font-size:0.97rem; } .gr-bubble.them h4 { font-size:0.9rem; }
.gr-bubble.them hr { border:none; border-top:1px solid rgba(148,163,184,0.2); margin:.7em 0; }
.gr-bubble.them b { color:#f1f5f9; }
.gr-bubble.them code { font-family:ui-monospace,monospace; background:rgba(2,6,18,0.6); padding:1px 5px; border-radius:5px; font-size:0.86em; }
.gr-bubble.pending { color:#94a3b8; font-style:italic; }
.gr-chatbar { display:flex; gap:10px; padding:14px 20px; border-top:1px solid rgba(148,163,184,0.16); }
.gr-chatbar .gr-in { flex:1; }
.gr-chat-hint { font-size:0.72rem; color:#64748b; padding:0 20px 12px; }
.gr-btn.gr-chat-open { background:rgba(16,185,129,0.14); color:#6ee7b7; border-color:rgba(110,231,183,0.35); }
.gr-btn.gr-chat-open:hover { background:rgba(16,185,129,0.24); border-color:rgba(110,231,183,0.6); }

/* ── sub-tabs [Teams | Chat] ── */
.gr-subtabs { display:flex; gap:6px; margin-bottom:20px; border-bottom:1px solid rgba(148,163,184,0.16); }
.gr-subtab { background:transparent; border:none; border-bottom:2px solid transparent; color:#94a3b8;
  padding:9px 16px; cursor:pointer; font:inherit; font-size:0.92rem; font-weight:600; margin-bottom:-1px; }
.gr-subtab:hover { color:#cbd5e1; }
.gr-subtab.on { color:#e9d5ff; border-bottom-color:#a78bfa; }

/* ── chat tab (ChatGPT-style) ── */
.gc-wrap { display:grid; grid-template-columns:248px 1fr; gap:16px; height:calc(100vh - 300px); min-height:440px; }
.gc-side { display:flex; flex-direction:column; gap:10px; min-height:0; }
.gc-side .gc-new { width:100%; }
.gc-sessions { flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:3px; padding-right:2px; }
.gc-sess { display:flex; align-items:center; gap:6px; padding:8px 10px; border-radius:9px; cursor:pointer; border:1px solid transparent; }
.gc-sess:hover { background:rgba(15,23,42,0.6); }
.gc-sess.on { background:rgba(124,58,237,0.16); border-color:rgba(167,139,250,0.4); }
.gc-sess-t { flex:1; min-width:0; font-size:0.85rem; color:#cbd5e1; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.gc-sess-act { display:flex; gap:1px; opacity:0; transition:opacity .15s; }
.gc-sess:hover .gc-sess-act { opacity:1; }
.gc-sess-act button { background:transparent; border:none; color:#64748b; cursor:pointer; font-size:0.78rem; padding:2px 5px; border-radius:5px; }
.gc-sess-act button:hover { color:#e2e8f0; background:rgba(148,163,184,0.15); }
.gc-main { display:flex; flex-direction:column; min-height:0; background:rgba(15,23,42,0.4);
  border:1px solid rgba(148,163,184,0.16); border-radius:14px; overflow:hidden; }
.gc-bar { display:flex; gap:10px; padding:12px 14px; border-bottom:1px solid rgba(148,163,184,0.16); flex-wrap:wrap; }
.gc-bar .gc-target { flex:1; min-width:180px; }
.gc-bar .gc-model { width:200px; }
.gc-log { flex:1; overflow-y:auto; padding:18px; display:flex; flex-direction:column; gap:14px; }
.gc-input-row { display:flex; gap:10px; padding:12px 14px; border-top:1px solid rgba(148,163,184,0.16); }
.gc-input-row .gc-input { flex:1; resize:none; box-sizing:border-box; }
@media (max-width:760px) { .gc-wrap { grid-template-columns:1fr; height:auto; } .gc-side { max-height:220px; } }
`;

export async function render(mainEl) {
  loadStyle('groups', CSS);
  mainEl.innerHTML = `
    <section class="gr-tab">
      <header class="gr-hero">
        <div>
          <div class="gr-eyebrow">FLOWORK · ANT COLONY</div>
          <h1 class="gr-h1">${esc(L.title)}</h1>
          <p class="gr-hsub">${esc(L.sub)}</p>
        </div>
        <div class="gr-stat"><b id="grStatN">·</b><label>${esc(L.count_label)}</label></div>
      </header>

      <div class="gr-subtabs">
        <button class="gr-subtab on" data-tab="teams">${esc(L.tab_teams)}</button>
        <button class="gr-subtab" data-tab="chat">💬 ${esc(L.tab_chat)}</button>
      </div>

      <div id="grTeams">
      <div class="gr-arch">
        <div class="gr-eyebrow">${esc(L.arch_eyebrow)}</div>
        <div class="gr-arch-title">🪄 ${esc(L.arch_title)}</div>
        <p class="gr-arch-sub">${esc(L.arch_sub)}</p>
        <div class="gr-arch-row">
          <input class="gr-in gr-arch-in" placeholder="${escAttr(L.arch_ph)}">
          <button class="gr-btn primary gr-arch-build">🪄 ${esc(L.arch_build_btn)}</button>
        </div>
        <div class="gr-arch-msg"></div>
      </div>

      <div class="gr-create-bar">
        <input class="gr-in" id="grNewId" placeholder="${escAttr(L.new_id_ph)}">
        <input class="gr-in" id="grNewName" placeholder="${escAttr(L.new_name_ph)}">
        <button class="gr-btn primary gr-create">+ ${esc(L.create_btn)}</button>
        <button class="gr-btn gr-restore" title="Restore any deleted bundled group to its factory setup">↻ Restore defaults</button>
        <span class="gr-msg" id="grNewMsg" style="display:none"></span>
      </div>

      <div id="grList"><div class="gr-empty">${esc(L.loading)}</div></div>
      </div>

      <div id="grChat" style="display:none"></div>
    </section>
  `;
  // Sub-tab switcher: Teams (colony cards) | Chat (ChatGPT-style).
  const teamsEl = mainEl.querySelector('#grTeams');
  const chatEl = mainEl.querySelector('#grChat');
  let chatInited = false;
  mainEl.querySelectorAll('.gr-subtab').forEach((btn) => {
    btn.addEventListener('click', () => {
      mainEl.querySelectorAll('.gr-subtab').forEach((b) => b.classList.toggle('on', b === btn));
      const isChat = btn.dataset.tab === 'chat';
      teamsEl.style.display = isChat ? 'none' : '';
      chatEl.style.display = isChat ? '' : 'none';
      if (isChat && !chatInited) { chatInited = true; renderChat(chatEl); }
    });
  });
  mainEl.querySelector('.gr-create').addEventListener('click', () => createGroup(mainEl));
  mainEl.querySelector('.gr-arch-build').addEventListener('click', () => architectBuild(mainEl));
  mainEl.querySelector('.gr-arch-in').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') { e.preventDefault(); architectBuild(mainEl); }
  });
  mainEl.querySelector('.gr-restore').addEventListener('click', async () => {
    if (!confirm('Restore any deleted bundled group/agent to its factory setup? Existing ones are left untouched.')) return;
    try {
      const r = await fetchJSON('/api/groups/reset', { method: 'POST' });
      alert(r.count ? ('Restored: ' + (r.restored || []).join(', ')) : 'Nothing to restore — all bundled groups are present.');
      setTimeout(() => load(mainEl), 900);
    } catch (e) { alert('Reset failed: ' + (e.message || e)); }
  });
  await load(mainEl);
}

async function load(mainEl) {
  const list = mainEl.querySelector('#grList');
  let data;
  try {
    data = await fetchJSON('/api/groups');
  } catch (e) {
    list.innerHTML = `<div class="gr-empty">${esc(L.create_fail)}${esc(String(e.message || e))}</div>`;
    return;
  }
  const groups = data.groups || [];
  const avail = data.available_agents || [];
  const statN = mainEl.querySelector('#grStatN');
  if (statN) statN.textContent = groups.length;
  if (!groups.length) {
    list.innerHTML = `<div class="gr-card"><div class="gr-empty">${esc(L.empty)}</div></div>`;
    return;
  }
  // claimedBy maps an agent id → the group that already uses it (member or
  // synthesizer), so a card only offers its own organs + agents no other group
  // has claimed.
  const claimedBy = {};
  for (const g of groups) {
    for (const c of g.claims || g.members || []) claimedBy[c] = g.id;
    if (g.synthesizer) claimedBy[g.synthesizer] = g.id;
  }
  for (const a of avail) {
    if (claimedBy[a.id]) continue;
    for (const g of groups) {
      if (a.id.startsWith(g.id + '-')) { claimedBy[a.id] = g.id; break; }
    }
  }
  list.innerHTML = '';
  for (const g of groups) list.appendChild(card(g, avail, claimedBy, mainEl));
}

function card(g, avail, claimedBy, mainEl) {
  const el = document.createElement('div');
  el.className = 'gr-card';
  const members = new Set(g.members || []);
  const pool = avail.filter((a) => members.has(a.id) || g.synthesizer === a.id || !claimedBy[a.id] || claimedBy[a.id] === g.id);
  const chips = pool.map((a) => `
    <label class="gr-chip ${members.has(a.id) ? 'on' : ''}" data-id="${escAttr(a.id)}">
      ${avatar(a.id, 22)}<span class="gr-chip-name">${esc(a.display_name || a.id)}</span>
      <input type="checkbox" ${members.has(a.id) ? 'checked' : ''}>
    </label>`).join('');
  const synthOpts = [`<option value="">${esc(L.synth_none)}</option>`]
    .concat(pool.map((a) => `<option value="${escAttr(a.id)}" ${g.synthesizer === a.id ? 'selected' : ''}>${esc(a.display_name || a.id)}</option>`))
    .join('');
  el.innerHTML = `
    <div class="gr-head">
      ${avatar(g.id, 46)}
      <div class="gr-head-text">
        <input class="gr-name" value="${escAttr(g.display_name || g.id)}" placeholder="${escAttr(L.name_ph)}">
        <div class="gr-gid">${esc(g.id)}</div>
      </div>
      <span class="gr-tag">GROUP</span>
    </div>

    <div class="gr-sec">${esc(L.members_label)}</div>
    <div class="gr-members">${chips || `<span class="gr-empty">${esc(L.no_agents)}</span>`}</div>

    <div class="gr-sec">${esc(L.synth_label)}</div>
    <select class="gr-sel gr-synth" style="width:100%">${synthOpts}</select>

    <div class="gr-sec">${esc(L.task_label)}</div>
    <textarea class="gr-task" placeholder="${escAttr(L.task_label)}">${esc(g.task || '')}</textarea>

    <div class="gr-sec">${esc(L.mode_label || 'Mode')}</div>
    <div style="display:flex;gap:8px;align-items:center">
      <select class="gr-sel gr-mode" style="flex:1">
        <option value="parallel" ${(g.mode || 'parallel') !== 'debate' ? 'selected' : ''}>${esc(L.mode_parallel || 'Parallel (fast)')}</option>
        <option value="debate" ${g.mode === 'debate' ? 'selected' : ''}>${esc(L.mode_debate || 'Debate (multi-round)')}</option>
      </select>
      <input class="gr-in gr-rounds" type="number" min="2" max="4" style="width:130px;${g.mode === 'debate' ? '' : 'display:none'}" placeholder="${escAttr(L.rounds_ph || 'rounds (2-4)')}" value="${escAttr(g.debate_rounds || '')}">
    </div>

    <div class="gr-foot">
      <button class="gr-btn gr-chat-open" title="Chat with this team">💬 ${esc(L.chat_btn)}</button>
      <button class="gr-btn gr-onoff ${g.enabled === false ? 'danger' : 'primary'}" title="Turn the whole group on/off (coordinator + all members)">${g.enabled === false ? 'OFF' : 'ON'}</button>
      <button class="gr-btn primary gr-do">${esc(L.save_btn)}</button>
      <button class="gr-btn danger gr-del">${esc(L.delete_btn)}</button>
      <span class="gr-msg" style="display:none"></span>
    </div>
  `;
  el.querySelector('.gr-chat-open').addEventListener('click', () => {
    const chatBtn = mainEl.querySelector('.gr-subtab[data-tab="chat"]');
    if (chatBtn) chatBtn.click(); // switches to Chat sub-tab (inits it if first time)
    setTimeout(() => chatStartWithGroup(g.id), 60);
  });
  // Group ON/OFF — disables/enables the coordinator AND every member at once.
  const onoff = el.querySelector('.gr-onoff');
  let grpEnabled = g.enabled !== false;
  onoff.addEventListener('click', async () => {
    const next = !grpEnabled;
    onoff.disabled = true;
    try {
      await fetchJSON(`/api/groups/toggle?id=${encodeURIComponent(g.id)}&disabled=${next ? 0 : 1}`, { method: 'POST' });
      grpEnabled = next;
      onoff.textContent = grpEnabled ? 'ON' : 'OFF';
      onoff.classList.toggle('primary', grpEnabled);
      onoff.classList.toggle('danger', !grpEnabled);
    } catch (e) { alert('Toggle failed: ' + (e.message || e)); }
    onoff.disabled = false;
  });
  el.querySelectorAll('.gr-chip input').forEach((inp) =>
    inp.addEventListener('change', () => inp.closest('.gr-chip').classList.toggle('on', inp.checked)));

  // Rounds input cuma relevan di mode DEBATE — show/hide pas mode berubah (parallel = no rounds,
  // biar kotak "rounds" gak nyasar di samping dropdown pas mode parallel).
  const modeSel = el.querySelector('.gr-mode');
  const roundsIn = el.querySelector('.gr-rounds');
  if (modeSel && roundsIn) {
    modeSel.addEventListener('change', () => {
      roundsIn.style.display = modeSel.value === 'debate' ? '' : 'none';
    });
  }

  el.querySelector('.gr-del').addEventListener('click', async () => {
    if (!confirm(fmt('delete_confirm', { name: g.display_name || g.id }))) return;
    try {
      await fetchJSON(`/api/groups/delete?id=${encodeURIComponent(g.id)}`, { method: 'POST' });
      setTimeout(() => load(mainEl), 600);
    } catch (e) { alert(L.delete_fail + (e.message || e)); }
  });

  const btn = el.querySelector('.gr-do');
  const msg = el.querySelector('.gr-foot .gr-msg');
  btn.addEventListener('click', async () => {
    const chosen = [...el.querySelectorAll('.gr-chip input:checked')].map((i) => i.closest('.gr-chip').dataset.id);
    const payload = {
      members: chosen,
      synthesizer: el.querySelector('.gr-synth').value,
      task: el.querySelector('.gr-task').value,
      display_name: el.querySelector('.gr-name').value.trim(),
      mode: el.querySelector('.gr-mode').value,
      debate_rounds: el.querySelector('.gr-rounds').value.trim(),
    };
    btn.disabled = true; msg.style.display = 'none';
    try {
      await fetchJSON(`/api/groups/config?id=${encodeURIComponent(g.id)}`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload),
      });
      msg.style.color = '#6ee7b7'; msg.textContent = '✓ ' + L.saved; msg.style.display = '';
      setTimeout(() => { msg.style.display = 'none'; }, 2500);
    } catch (e) {
      msg.style.color = '#f87171'; msg.textContent = L.save_fail + (e.message || e); msg.style.display = '';
    } finally {
      btn.disabled = false;
    }
  });
  return el;
}

async function createGroup(mainEl) {
  const idEl = mainEl.querySelector('#grNewId');
  const nameEl = mainEl.querySelector('#grNewName');
  const msg = mainEl.querySelector('#grNewMsg');
  const btn = mainEl.querySelector('.gr-create');
  const id = (idEl.value || '').trim().toLowerCase();
  if (!/^[a-z0-9][a-z0-9-]{1,39}$/.test(id)) {
    msg.style.color = '#f87171'; msg.textContent = L.id_invalid; msg.style.display = '';
    return;
  }
  btn.disabled = true; msg.style.display = 'none';
  try {
    await fetchJSON('/api/groups/create', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, display_name: nameEl.value.trim() }),
    });
    idEl.value = ''; nameEl.value = '';
    msg.style.color = '#6ee7b7'; msg.textContent = '✓ ' + L.create_ok; msg.style.display = '';
    setTimeout(() => load(mainEl), 800);
  } catch (e) {
    msg.style.color = '#f87171'; msg.textContent = L.create_fail + (e.message || e); msg.style.display = '';
  } finally {
    btn.disabled = false;
  }
}

// architectBuild — POST a free-text prompt to /api/architect/build → the Architect
// designs + builds the specialists + lead and wires the group, which then appears in
// the list below. One LLM call server-side, so it can take a moment (longer if the
// upstream model is rate-limited and the router fails over).
async function architectBuild(mainEl) {
  const inEl = mainEl.querySelector('.gr-arch-in');
  const btn = mainEl.querySelector('.gr-arch-build');
  const msg = mainEl.querySelector('.gr-arch-msg');
  const prompt = (inEl.value || '').trim();
  if (prompt.length < 3) {
    msg.style.color = '#f87171'; msg.textContent = L.arch_invalid; msg.style.display = '';
    return;
  }
  btn.disabled = true; inEl.disabled = true;
  msg.style.color = '#a78bfa'; msg.innerHTML = `<span class="gr-spin"></span>${esc(L.arch_building)}`; msg.style.display = '';
  try {
    const r = await fetchJSON('/api/architect/build', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt }),
    });
    inEl.value = '';
    const nm = r.display_name || r.group_id || 'team';
    const n = (r.members || []).length;
    msg.style.color = '#6ee7b7'; msg.textContent = `✓ ${L.arch_built} — ${nm} (${n})`;
    await load(mainEl);
  } catch (e) {
    msg.style.color = '#f87171'; msg.textContent = L.arch_fail + (e.message || e);
  } finally {
    btn.disabled = false; inEl.disabled = false;
  }
}

// mdLite — tiny, XSS-safe markdown → HTML for chat replies. esc() runs FIRST, so only
// our own fixed tags (from markers in the already-escaped text) are ever injected.
function mdLite(raw) {
  let s = esc(String(raw == null ? '' : raw));
  s = s.replace(/^### (.+)$/gm, '<h4>$1</h4>')
    .replace(/^## (.+)$/gm, '<h3>$1</h3>')
    .replace(/^# (.+)$/gm, '<h2>$1</h2>')
    .replace(/^---+\s*$/gm, '<hr>');
  s = s.replace(/\*\*([^*]+)\*\*/g, '<b>$1</b>')
    .replace(/`([^`]+)`/g, '<code>$1</code>');
  s = s.replace(/\n/g, '<br>').replace(/(<\/h[234]>|<hr>)<br>/g, '$1');
  return s;
}

// ── CHAT sub-tab (ChatGPT-style) ──────────────────────────────────────────────
// Persistent sessions (survive shutdown, full-context memory) talking to either a
// GROUP (a team) or the ARCHITECT (brainstorm + build teams). Backed by
// /api/chat/sessions* + /api/chat/send. Module-level state so the Teams cards' 💬
// button can deep-link in via chatStartWithGroup().
const CHAT = { el: null, sessionId: null, sessions: [], groups: [] };

async function renderChat(host) {
  CHAT.el = host;
  host.innerHTML = `
    <div class="gc-wrap">
      <aside class="gc-side">
        <button class="gr-btn primary gc-new">+ ${esc(L.chat_new)}</button>
        <div class="gc-sessions"><div class="gr-empty">${esc(L.loading)}</div></div>
      </aside>
      <section class="gc-main">
        <div class="gc-bar">
          <select class="gr-sel gc-target"></select>
          <select class="gr-sel gc-model" title="${escAttr(L.chat_model_ph)}">
            <option value="">${esc(L.chat_model_default)}</option>
            <option value="claude-opus-4-8">${esc(L.chat_model_opus)}</option>
            <option value="claude-haiku-4-5">${esc(L.chat_model_haiku)}</option>
            <option value="flowork-brain">${esc(L.chat_model_local)}</option>
          </select>
        </div>
        <div class="gc-log"><div class="gr-empty gc-intro">${esc(L.chat_pick)}</div></div>
        <div class="gc-input-row">
          <textarea class="gr-in gc-input" rows="2" placeholder="${escAttr(L.chat_input_ph)}"></textarea>
          <button class="gr-btn primary gc-send">${esc(L.chat_send)}</button>
        </div>
      </section>
    </div>`;
  host.querySelector('.gc-new').addEventListener('click', () => chatNew());
  host.querySelector('.gc-target').addEventListener('change', () => chatSaveMeta());
  host.querySelector('.gc-model').addEventListener('change', () => chatSaveMeta());
  host.querySelector('.gc-send').addEventListener('click', () => chatSend());
  host.querySelector('.gc-input').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); chatSend(); }
  });
  await chatLoadGroups();
  await chatLoadSessions();
}

async function chatLoadGroups() {
  try { const d = await fetchJSON('/api/groups'); CHAT.groups = d.groups || []; } catch { CHAT.groups = []; }
  const sel = CHAT.el.querySelector('.gc-target');
  sel.innerHTML = `<option value="architect">${esc(L.chat_target_architect)}</option>`
    + CHAT.groups.map((g) => `<option value="group:${escAttr(g.id)}">${esc(L.chat_target_group_prefix)}${esc(g.display_name || g.id)}</option>`).join('');
}

async function chatLoadSessions() {
  const box = CHAT.el.querySelector('.gc-sessions');
  let d;
  try { d = await fetchJSON('/api/chat/sessions'); } catch (e) { box.innerHTML = `<div class="gr-empty">${esc(String(e.message || e))}</div>`; return; }
  CHAT.sessions = d.sessions || [];
  if (!CHAT.sessions.length) { box.innerHTML = `<div class="gr-empty">${esc(L.chat_sessions_empty)}</div>`; return; }
  box.innerHTML = '';
  for (const s of CHAT.sessions) {
    const row = document.createElement('div');
    row.className = 'gc-sess' + (s.id === CHAT.sessionId ? ' on' : '');
    row.innerHTML = `<span class="gc-sess-t">${esc(s.title || L.chat_new)}</span>
      <span class="gc-sess-act"><button class="gc-ren" title="rename">✎</button><button class="gc-del" title="delete">🗑</button></span>`;
    row.querySelector('.gc-sess-t').addEventListener('click', () => chatOpen(s.id));
    row.querySelector('.gc-ren').addEventListener('click', (e) => { e.stopPropagation(); chatRename(s.id); });
    row.querySelector('.gc-del').addEventListener('click', (e) => { e.stopPropagation(); chatDelete(s.id); });
    box.appendChild(row);
  }
}

function chatBarValues() {
  const target = CHAT.el.querySelector('.gc-target').value;
  const model = CHAT.el.querySelector('.gc-model').value.trim();
  if (target.startsWith('group:')) return { mode: 'group', target_id: target.slice(6), model };
  return { mode: 'architect', target_id: '', model };
}

async function chatNew() {
  try {
    const r = await fetchJSON('/api/chat/sessions', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(chatBarValues()),
    });
    await chatLoadSessions();
    await chatOpen(r.session.id);
    CHAT.el.querySelector('.gc-input').focus();
  } catch (e) { alert(L.chat_fail + (e.message || e)); }
}

// chatStartWithGroup — deep-link from a Teams card's 💬 button: open a fresh group chat.
async function chatStartWithGroup(groupId) {
  if (!CHAT.el) return;
  try {
    const r = await fetchJSON('/api/chat/sessions', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ mode: 'group', target_id: groupId, model: '' }),
    });
    await chatLoadSessions();
    await chatOpen(r.session.id);
  } catch (e) { alert(L.chat_fail + (e.message || e)); }
}

async function chatOpen(id) {
  CHAT.sessionId = id;
  const sess = CHAT.sessions.find((s) => s.id === id);
  const sel = CHAT.el.querySelector('.gc-target');
  sel.value = sess && sess.mode === 'group' ? 'group:' + sess.target_id : 'architect';
  CHAT.el.querySelector('.gc-model').value = (sess && sess.model) || '';
  // highlight active session without a full reload
  CHAT.el.querySelectorAll('.gc-sess').forEach((el) => el.classList.remove('on'));
  const log = CHAT.el.querySelector('.gc-log');
  log.innerHTML = `<div class="gr-empty">${esc(L.loading)}</div>`;
  try {
    const d = await fetchJSON(`/api/chat/sessions/messages?id=${encodeURIComponent(id)}`);
    const msgs = d.messages || [];
    log.innerHTML = '';
    if (!msgs.length) {
      const intro = sess && sess.mode === 'group' ? L.chat_intro_group : L.chat_intro_architect;
      log.innerHTML = `<div class="gr-empty gc-intro">${esc(intro)}</div>`;
    } else {
      for (const m of msgs) chatBubble(m.role === 'user' ? 'me' : 'them', m.role === 'user' ? esc(m.content) : mdLite(m.content));
    }
  } catch (e) { log.innerHTML = `<div class="gr-empty">${esc(String(e.message || e))}</div>`; }
  await chatLoadSessions();
}

function chatBubble(cls, html) {
  const log = CHAT.el.querySelector('.gc-log');
  const intro = log.querySelector('.gc-intro'); if (intro) intro.remove();
  const b = document.createElement('div'); b.className = 'gr-bubble ' + cls; b.innerHTML = html;
  log.appendChild(b); log.scrollTop = log.scrollHeight; return b;
}

async function chatSend() {
  if (!CHAT.sessionId) { await chatNew(); if (!CHAT.sessionId) return; }
  const input = CHAT.el.querySelector('.gc-input');
  const text = input.value.trim(); if (!text) return;
  input.value = '';
  chatBubble('me', esc(text));
  const sendBtn = CHAT.el.querySelector('.gc-send');
  sendBtn.disabled = true; input.disabled = true;
  const pending = chatBubble('them pending', `<span class="gr-spin"></span>${esc(L.chat_thinking)}`);
  try {
    const r = await fetchJSON('/api/chat/send', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ session_id: CHAT.sessionId, text }),
    });
    pending.classList.remove('pending');
    pending.innerHTML = mdLite(r.reply || r.error || '(no reply)');
    chatLoadSessions(); // title may auto-set; a team may have just been built
  } catch (e) {
    pending.classList.remove('pending'); pending.style.color = '#f87171';
    pending.textContent = L.chat_fail + (e.message || e);
  } finally {
    sendBtn.disabled = false; input.disabled = false; input.focus();
  }
}

async function chatSaveMeta() {
  if (!CHAT.sessionId) return; // no session yet → the bar selection applies on New chat
  try {
    await fetchJSON(`/api/chat/sessions/meta?id=${encodeURIComponent(CHAT.sessionId)}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(chatBarValues()),
    });
  } catch (e) { /* best-effort */ }
}

async function chatRename(id) {
  const s = CHAT.sessions.find((x) => x.id === id);
  const title = prompt(L.chat_rename_prompt, s ? s.title : ''); if (title === null) return;
  try { await fetchJSON(`/api/chat/sessions/rename?id=${encodeURIComponent(id)}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ title: title.trim() }) }); await chatLoadSessions(); } catch (e) { alert(e.message || e); }
}

async function chatDelete(id) {
  if (!confirm(L.chat_delete_confirm)) return;
  try {
    await fetchJSON(`/api/chat/sessions/delete?id=${encodeURIComponent(id)}`, { method: 'POST' });
    if (CHAT.sessionId === id) { CHAT.sessionId = null; CHAT.el.querySelector('.gc-log').innerHTML = `<div class="gr-empty gc-intro">${esc(L.chat_pick)}</div>`; }
    await chatLoadSessions();
  } catch (e) { alert(e.message || e); }
}
