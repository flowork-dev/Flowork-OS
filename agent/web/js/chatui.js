// === LOCKED FILE (soft) === Status: STABLE (owner-approved 2026-06-15). ChatGPT-style
// chat component (sessions, full-context, typing animation) used by AI Studio. Tested.
// DO NOT MODIFY without owner approval.
// 2026-06-21 (owner-approved): BUANG menu model (dropdown .cu-model). Alasan owner: "model sudah
//   ada di agent" — model dipilih PER-AGENT (ai-studio buat architect, model group buat group) lewat
//   Settings agent, bukan dropdown global di chat. barValues kirim model:'' → backend pakai model
//   per-target. RE-LOCKED.
// 2026-07-02 (roadmap owner, multimodal paste): Ctrl+V screenshot → chip preview →
//   kirim images[] (data URL base64) ke /api/chat/send → LLM vision. Render thumbnail
//   di bubble user + history. 📄 Dok: FLowork_os/lock/chat-vision.md
//
// chatui.js — SHARED ChatGPT-style chat component, reused by the Group, Schedule and
// Trigger tabs. Self-contained: own CSS (cu-* classes, no collision), own i18n
// ('chatui' domain), per-mount state via closures (each tab gets an independent
// instance). Talks to the persistent chat backend (/api/chat/sessions* + /api/chat/send):
// a session targets either the ARCHITECT (design/build/schedule via chat) or a GROUP
// (talk to a team). renderChatUI(host) returns { startGroupChat(id) } for deep-links.
import { esc, escAttr, fetchJSON, loadStyle } from '/js/utils.js';
import { t } from '/js/i18n.js';

const L = new Proxy({}, { get: (_, k) => t('chatui.' + String(k)) });

const CSS = `
.cu-wrap { display:grid; grid-template-columns:266px 1fr; gap:16px; height:calc(100vh - 300px); min-height:440px; color:#e2e8f0; transition:grid-template-columns .28s cubic-bezier(.4,0,.2,1); }
.cu-wrap.cu-hide { grid-template-columns:0 1fr; }
.cu-side { display:flex; flex-direction:column; gap:10px; min-height:0; overflow:hidden; transition:opacity .2s ease; }
.cu-wrap.cu-hide .cu-side { opacity:0; pointer-events:none; }
.cu-side-top { display:flex; gap:8px; }
.cu-new { flex:1; padding:10px 14px; border-radius:12px; font:inherit; font-size:0.86rem; font-weight:700; cursor:pointer;
  border:1px solid var(--glass-border-hover); background:linear-gradient(165deg,#8b5cf6,#0ea5e9); color:#fff;
  box-shadow:0 8px 20px rgba(124,58,237,.34), inset 0 1px 0 rgba(255,255,255,.2); transition:transform .12s, filter .15s; }
.cu-new:hover { filter:brightness(1.13); transform:translateY(-1px); }
.cu-prune { width:44px; flex-shrink:0; border-radius:12px; font-size:1rem; cursor:pointer; color:var(--text-muted);
  background:var(--bg-panel); border:1px solid var(--glass-border); box-shadow:0 4px 12px rgba(0,0,0,.22), inset 0 1px 0 rgba(255,255,255,.05); transition:.15s; }
.cu-prune:hover { color:var(--text-main); border-color:var(--glass-border-hover); transform:translateY(-1px); }
.cu-toggle { width:40px; height:40px; flex-shrink:0; border-radius:11px; font-size:1.15rem; cursor:pointer; color:var(--text-muted);
  background:var(--bg-panel); border:1px solid var(--glass-border); box-shadow:0 4px 12px rgba(0,0,0,.22); transition:.15s; }
.cu-toggle:hover { color:var(--text-main); border-color:var(--glass-border-hover); }
.cu-sessions { flex:1; overflow-y:auto; display:flex; flex-direction:column; gap:3px; padding-right:2px; }
.cu-sess { display:flex; align-items:center; gap:6px; padding:9px 11px; border-radius:12px; cursor:pointer; border:1px solid var(--glass-border);
  background:var(--bg-panel); box-shadow:inset 0 1px 0 rgba(255,255,255,.04); transition:transform .12s, border-color .15s, background .15s; }
.cu-sess:hover { background:var(--bg-panel-hover); border-color:var(--glass-border-hover); transform:translateX(2px); }
.cu-sess.on { background:linear-gradient(165deg,rgba(124,58,237,.24),rgba(124,58,237,.07)); border-color:var(--glass-border-hover); box-shadow:0 6px 16px rgba(124,58,237,.26); }
.cu-sess-t { flex:1; min-width:0; font-size:0.85rem; color:#cbd5e1; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.cu-sess-act { display:flex; gap:1px; opacity:0; transition:opacity .15s; }
.cu-sess:hover .cu-sess-act { opacity:1; }
.cu-sess-act button { background:transparent; border:none; color:#64748b; cursor:pointer; font-size:0.78rem; padding:2px 5px; border-radius:5px; }
.cu-sess-act button:hover { color:#e2e8f0; background:rgba(148,163,184,0.15); }
.cu-main { display:flex; flex-direction:column; min-height:0;
  background:linear-gradient(180deg, rgba(22,25,43,.55), rgba(13,15,26,.5));
  border:1px solid var(--glass-border); border-radius:18px; overflow:hidden;
  box-shadow:0 22px 60px rgba(0,0,0,.42), inset 0 1px 0 rgba(255,255,255,.06); }
.cu-bar { display:flex; gap:10px; align-items:center; padding:12px 14px; border-bottom:1px solid var(--glass-border); flex-wrap:wrap; }
.cu-sel { background:rgba(2,6,18,0.55); border:1px solid rgba(148,163,184,0.2); border-radius:9px; color:#e2e8f0; padding:9px 12px; font:inherit; font-size:0.9rem; }
.cu-sel:focus { outline:none; border-color:#a78bfa; }
.cu-target { flex:1; min-width:180px; }
.cu-log { flex:1; overflow-y:auto; padding:18px; display:flex; flex-direction:column; gap:14px; }
.cu-empty { color:#64748b; font-size:0.86rem; padding:10px 0; }
.cu-attach { display:none; gap:8px; padding:10px 14px 0; flex-wrap:wrap; border-top:1px solid rgba(148,163,184,0.16); }
.cu-attach.has { display:flex; }
.cu-chip { position:relative; width:56px; height:56px; border-radius:9px; overflow:hidden; border:1px solid var(--glass-border-hover); box-shadow:0 4px 10px rgba(0,0,0,.3); }
.cu-chip img { width:100%; height:100%; object-fit:cover; display:block; }
.cu-chip button { position:absolute; top:1px; right:1px; width:18px; height:18px; line-height:1; padding:0; border:none; border-radius:50%;
  background:rgba(2,6,18,.75); color:#e2e8f0; font-size:0.7rem; cursor:pointer; }
.cu-chip button:hover { background:#ef4444; }
.cu-bubble .cu-img { max-width:230px; max-height:180px; border-radius:9px; display:block; margin-top:6px; border:1px solid rgba(255,255,255,.25); }
.cu-input-row { display:flex; gap:10px; padding:12px 14px; border-top:1px solid rgba(148,163,184,0.16); }
.cu-attach.has + .cu-input-row { border-top:none; }
.cu-input { flex:1; resize:none; box-sizing:border-box; background:rgba(2,6,18,0.55); border:1px solid rgba(148,163,184,0.2);
  border-radius:9px; color:#e2e8f0; padding:9px 12px; font:inherit; font-size:0.9rem; }
.cu-input:focus { outline:none; border-color:#a78bfa; }
.cu-send { padding:9px 16px; border-radius:9px; font:inherit; font-size:0.84rem; font-weight:600; cursor:pointer;
  border:1px solid transparent; background:linear-gradient(90deg,#7c3aed,#0ea5e9); color:#fff; }
.cu-send:hover { filter:brightness(1.12); } .cu-send:disabled { opacity:.5; cursor:default; }
.cu-bubble { max-width:90%; padding:11px 14px; border-radius:13px; font-size:0.9rem; line-height:1.55; word-wrap:break-word; }
.cu-bubble.me { align-self:flex-end; background:linear-gradient(135deg,#8b5cf6,#0ea5e9); color:#fff; border-bottom-right-radius:4px; white-space:pre-wrap; box-shadow:0 8px 22px rgba(124,58,237,.34), inset 0 1px 0 rgba(255,255,255,.18); }
.cu-bubble.them { align-self:flex-start; background:linear-gradient(165deg,rgba(30,34,56,.92),rgba(15,18,30,.88)); border:1px solid var(--glass-border); color:#e2e8f0; border-bottom-left-radius:4px; box-shadow:0 8px 22px rgba(0,0,0,.32), inset 0 1px 0 rgba(255,255,255,.05); }
.cu-bubble.them h2,.cu-bubble.them h3,.cu-bubble.them h4 { margin:.5em 0 .3em; color:#c4b5fd; line-height:1.25; }
.cu-bubble.them h2 { font-size:1.05rem; } .cu-bubble.them h3 { font-size:0.97rem; } .cu-bubble.them h4 { font-size:0.9rem; }
.cu-bubble.them hr { border:none; border-top:1px solid rgba(148,163,184,0.2); margin:.7em 0; }
.cu-bubble.them b { color:#f1f5f9; }
.cu-bubble.them code { font-family:ui-monospace,monospace; background:rgba(2,6,18,0.6); padding:1px 5px; border-radius:5px; font-size:0.86em; }
.cu-bubble.pending { color:#94a3b8; font-style:italic; }
.cu-spin { display:inline-block; width:13px; height:13px; border:2px solid rgba(167,139,250,0.35); border-top-color:#a78bfa;
  border-radius:50%; animation:cu-spin .7s linear infinite; vertical-align:-2px; margin-right:7px; }
@keyframes cu-spin { to { transform:rotate(360deg); } }
/* typing indicator — 3 bouncing dots ("animasi ngetik") */
.cu-typing { display:inline-flex; gap:5px; align-items:center; padding:2px 0; }
.cu-typing span { width:8px; height:8px; border-radius:50%; background:#a78bfa; opacity:.45; animation:cu-bounce 1.2s infinite ease-in-out; }
.cu-typing span:nth-child(2) { animation-delay:.18s; } .cu-typing span:nth-child(3) { animation-delay:.36s; }
@keyframes cu-bounce { 0%,60%,100% { transform:translateY(0); opacity:.4; } 30% { transform:translateY(-6px); opacity:1; } }
.cu-bubble.them { animation:cu-fade .22s ease; }
@keyframes cu-fade { from { opacity:0; transform:translateY(4px); } to { opacity:1; transform:none; } }
.cu-caret::after { content:'▋'; color:#a78bfa; animation:cu-blink 1s steps(1) infinite; margin-left:1px; }
@keyframes cu-blink { 50% { opacity:0; } }
@media (max-width:760px) { .cu-wrap { grid-template-columns:1fr; height:auto; } .cu-side { max-height:220px; } }
`;

// mdLite — XSS-safe markdown → HTML (esc first, then only fixed tags from markers).
export function mdLite(raw) {
  let s = esc(String(raw == null ? '' : raw));
  s = s.replace(/^### (.+)$/gm, '<h4>$1</h4>')
    .replace(/^## (.+)$/gm, '<h3>$1</h3>')
    .replace(/^# (.+)$/gm, '<h2>$1</h2>')
    .replace(/^---+\s*$/gm, '<hr>');
  s = s.replace(/\*\*([^*]+)\*\*/g, '<b>$1</b>').replace(/`([^`]+)`/g, '<code>$1</code>');
  s = s.replace(/\n/g, '<br>').replace(/(<\/h[234]>|<hr>)<br>/g, '$1');
  return s;
}

// typeReveal — typewriter animation: type the raw text out (blinking caret), then
// swap to rendered markdown. Long replies are capped so it never drags.
function typeReveal(bubble, text) {
  const log = bubble.parentElement;
  bubble.classList.add('cu-caret');
  bubble.textContent = '';
  const total = text.length;
  const dur = Math.min(1600, Math.max(350, total * 4));
  const per = Math.max(1, Math.ceil(total / Math.max(1, dur / 16)));
  let shown = 0;
  const iv = setInterval(() => {
    shown += per;
    bubble.textContent = text.slice(0, shown);
    if (log) log.scrollTop = log.scrollHeight;
    if (shown >= total) {
      clearInterval(iv);
      bubble.classList.remove('cu-caret');
      bubble.innerHTML = mdLite(text);
      if (log) log.scrollTop = log.scrollHeight;
    }
  }, 16);
}

// renderChatUI — mount the chat into host. Independent per-mount state. Returns an API.
export function renderChatUI(host) {
  loadStyle('chatui', CSS);
  const S = { el: host, sessionId: null, sessions: [], groups: [] };
  host.innerHTML = `
    <div class="cu-wrap">
      <aside class="cu-side">
        <div class="cu-side-top">
          <button class="cu-new">+ ${esc(L.new)}</button>
          <button class="cu-prune" title="${escAttr(L.prune_title)}">🧹</button>
        </div>
        <div class="cu-sessions"><div class="cu-empty">${esc(L.loading)}</div></div>
      </aside>
      <section class="cu-main">
        <div class="cu-bar">
          <button class="cu-toggle" title="${escAttr(L.new)}">☰</button>
          <select class="cu-sel cu-target"></select>
        </div>
        <div class="cu-log"><div class="cu-empty cu-intro">${esc(L.pick)}</div></div>
        <div class="cu-attach"></div>
        <div class="cu-input-row">
          <textarea class="cu-input" rows="2" placeholder="${escAttr(L.input_ph)}"></textarea>
          <button class="cu-send">${esc(L.send)}</button>
        </div>
      </section>
    </div>`;

  S.pendingImages = []; // lampiran gambar (data URL) yang nunggu dikirim (multimodal paste)
  const $ = (sel) => host.querySelector(sel);
  const CU_MAX_IMG = 4;
  // userHTML — bubble user: teks (escaped) + thumbnail lampiran (data URL aman di-embed).
  const userHTML = (text, imgs) => esc(text || '')
    + ((imgs || []).map((u) => `<img class="cu-img" src="${escAttr(u)}" alt="lampiran">`).join(''));
  function renderAttach() {
    const row = $('.cu-attach'); row.innerHTML = '';
    row.classList.toggle('has', S.pendingImages.length > 0);
    S.pendingImages.forEach((u, i) => {
      const chip = document.createElement('div'); chip.className = 'cu-chip';
      const img = document.createElement('img'); img.src = u; chip.appendChild(img);
      const x = document.createElement('button'); x.type = 'button'; x.textContent = '×'; x.title = 'hapus';
      x.addEventListener('click', () => { S.pendingImages.splice(i, 1); renderAttach(); });
      chip.appendChild(x); row.appendChild(chip);
    });
  }
  function addImageFiles(files) {
    for (const f of files) {
      if (!f.type || !f.type.startsWith('image/')) continue;
      if (S.pendingImages.length >= CU_MAX_IMG) { alert(`Maks ${CU_MAX_IMG} gambar per pesan`); return; }
      const rd = new FileReader();
      rd.onload = () => { S.pendingImages.push(String(rd.result)); renderAttach(); };
      rd.readAsDataURL(f);
    }
  }
  const barValues = () => {
    const target = $('.cu-target').value;
    // model dropdown DIHAPUS (owner 2026-06-21: "model sudah ada di agent") → model:'' = backend
    // pakai model PER-TARGET (ai-studio buat architect, model group buat group), bukan pilihan global.
    if (target.startsWith('group:')) return { mode: 'group', target_id: target.slice(6), model: '' };
    return { mode: 'architect', target_id: '', model: '' };
  };
  const bubble = (cls, html) => {
    const log = $('.cu-log'); const intro = log.querySelector('.cu-intro'); if (intro) intro.remove();
    const b = document.createElement('div'); b.className = 'cu-bubble ' + cls; b.innerHTML = html;
    log.appendChild(b); log.scrollTop = log.scrollHeight; return b;
  };

  async function loadGroups() {
    try { const d = await fetchJSON('/api/groups'); S.groups = d.groups || []; } catch { S.groups = []; }
    $('.cu-target').innerHTML = `<option value="architect">${esc(L.target_architect)}</option>`
      + S.groups.map((g) => `<option value="group:${escAttr(g.id)}">${esc(L.target_group_prefix)}${esc(g.display_name || g.id)}</option>`).join('');
  }
  async function loadSessions() {
    const box = $('.cu-sessions');
    let d;
    try { d = await fetchJSON('/api/chat/sessions'); } catch (e) { box.innerHTML = `<div class="cu-empty">${esc(String(e.message || e))}</div>`; return; }
    S.sessions = d.sessions || [];
    if (!S.sessions.length) { box.innerHTML = `<div class="cu-empty">${esc(L.sessions_empty)}</div>`; return; }
    box.innerHTML = '';
    for (const s of S.sessions) {
      const row = document.createElement('div');
      row.className = 'cu-sess' + (s.id === S.sessionId ? ' on' : '');
      row.innerHTML = `<span class="cu-sess-t">${esc(s.title || L.new)}</span>
        <span class="cu-sess-act"><button class="cu-ren" title="rename">✎</button><button class="cu-del" title="delete">🗑</button></span>`;
      row.querySelector('.cu-sess-t').addEventListener('click', () => open(s.id));
      row.querySelector('.cu-ren').addEventListener('click', (e) => { e.stopPropagation(); rename(s.id); });
      row.querySelector('.cu-del').addEventListener('click', (e) => { e.stopPropagation(); del(s.id); });
      box.appendChild(row);
    }
  }
  async function newChat() {
    try {
      const r = await fetchJSON('/api/chat/sessions', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(barValues()) });
      await loadSessions(); await open(r.session.id); $('.cu-input').focus();
    } catch (e) { alert(L.fail + (e.message || e)); }
  }
  async function startGroupChat(groupId) {
    try {
      const r = await fetchJSON('/api/chat/sessions', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ mode: 'group', target_id: groupId, model: '' }) });
      await loadSessions(); await open(r.session.id);
    } catch (e) { alert(L.fail + (e.message || e)); }
  }
  async function open(id) {
    S.sessionId = id;
    const sess = S.sessions.find((s) => s.id === id);
    $('.cu-target').value = sess && sess.mode === 'group' ? 'group:' + sess.target_id : 'architect';
    const log = $('.cu-log'); log.innerHTML = `<div class="cu-empty">${esc(L.loading)}</div>`;
    try {
      const d = await fetchJSON(`/api/chat/sessions/messages?id=${encodeURIComponent(id)}`);
      const msgs = d.messages || [];
      log.innerHTML = '';
      if (!msgs.length) {
        const intro = sess && sess.mode === 'group' ? L.intro_group : L.intro_architect;
        log.innerHTML = `<div class="cu-empty cu-intro">${esc(intro)}</div>`;
      } else {
        for (const m of msgs) bubble(m.role === 'user' ? 'me' : 'them', m.role === 'user' ? userHTML(m.content, m.images) : mdLite(m.content));
      }
    } catch (e) { log.innerHTML = `<div class="cu-empty">${esc(String(e.message || e))}</div>`; }
    await loadSessions();
  }
  async function send() {
    if (!S.sessionId) { await newChat(); if (!S.sessionId) return; }
    const input = $('.cu-input'); const text = input.value.trim();
    const images = S.pendingImages.slice();
    if (!text && !images.length) return;
    input.value = '';
    S.pendingImages = []; renderAttach();
    bubble('me', userHTML(text, images));
    const btn = $('.cu-send'); btn.disabled = true; input.disabled = true;
    const pending = bubble('them pending', `<span class="cu-typing"><span></span><span></span><span></span></span>`);
    try {
      const r = await fetchJSON('/api/chat/send', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ session_id: S.sessionId, text, images }) });
      pending.classList.remove('pending');
      typeReveal(pending, r.reply || r.error || '(no reply)');
      loadSessions();
    } catch (e) { pending.classList.remove('pending'); pending.style.color = '#f87171'; pending.textContent = L.fail + (e.message || e); }
    finally { btn.disabled = false; input.disabled = false; input.focus(); }
  }
  async function saveMeta() {
    if (!S.sessionId) return;
    try { await fetchJSON(`/api/chat/sessions/meta?id=${encodeURIComponent(S.sessionId)}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(barValues()) }); } catch (e) { /* best-effort */ }
  }
  async function rename(id) {
    const s = S.sessions.find((x) => x.id === id);
    const title = prompt(L.rename_prompt, s ? s.title : ''); if (title === null) return;
    try { await fetchJSON(`/api/chat/sessions/rename?id=${encodeURIComponent(id)}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ title: title.trim() }) }); await loadSessions(); } catch (e) { alert(e.message || e); }
  }
  async function del(id) {
    if (!confirm(L.delete_confirm)) return;
    try {
      await fetchJSON(`/api/chat/sessions/delete?id=${encodeURIComponent(id)}`, { method: 'POST' });
      if (S.sessionId === id) { S.sessionId = null; $('.cu-log').innerHTML = `<div class="cu-empty cu-intro">${esc(L.pick)}</div>`; }
      await loadSessions();
    } catch (e) { alert(e.message || e); }
  }

  $('.cu-new').addEventListener('click', newChat);
  // toggle hide/show history sidebar (kayak side-bar collapsible).
  $('.cu-toggle').addEventListener('click', () => $('.cu-wrap').classList.toggle('cu-hide'));
  $('.cu-prune').addEventListener('click', async () => {
    if (!confirm(L.prune_confirm)) return;
    try {
      const r = await fetchJSON('/api/chat/sessions/prune', { method: 'POST' });
      if (S.sessionId && !S.sessions.find((x) => x.id === S.sessionId)) S.sessionId = null;
      await loadSessions();
      alert(L.prune_done.replace('{deleted}', r.deleted || 0).replace('{kept}', r.kept || 0));
    } catch (e) { alert(L.prune_fail + (e.message || e)); }
  });
  $('.cu-target').addEventListener('change', saveMeta);
  $('.cu-send').addEventListener('click', send);
  $('.cu-input').addEventListener('keydown', (e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); } });
  // Multimodal paste: Ctrl+V screenshot/gambar dari clipboard → chip preview → ikut kekirim.
  $('.cu-input').addEventListener('paste', (e) => {
    const items = (e.clipboardData && e.clipboardData.items) || [];
    const files = [];
    for (const it of items) {
      if (it.kind === 'file') { const f = it.getAsFile(); if (f && f.type && f.type.startsWith('image/')) files.push(f); }
    }
    if (files.length) { e.preventDefault(); addImageFiles(files); }
  });
  loadGroups();
  loadSessions();

  return { startGroupChat };
}
