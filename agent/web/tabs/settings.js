// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-31 (re-audited 2026-06-07, re-locked 2026-06-11)
// Update 2026-06-11 (owner-approved): API Keys preset chips (per-service, pre-fill
//   the exact env var name, green when set) + new "Router & Model" segment
//   (renderRouterDefault → /api/settings/router-default) for the global default
//   model + router URL. All values esc/escAttr'd, labels via i18n, no inline secrets.
// Reason: Tab Settings (Akun/API-Keys/Notif/YouTube). Audit pass — esc/escAttr
//   semua field, fetchJSON util, API key masked, label via i18n. E2E verified
//   lewat instance isolated. 2026-06-06 (owner-approved): segmen Wallet
//   Personal + Wallet AI DIBUANG (fitur wallet dihapus).
// Update 2026-06-07 (owner-approved audit): the YouTube OAuth poll (setInterval
//   /api/settings/youtube every 2s) was never cancelled when you switched segment —
//   it kept firing in the background and could re-render YouTube over another
//   segment. Now tracked at module scope (ytPoll/stopYtPoll) and cancelled on
//   segment switch, on re-render, and before starting a new poll.
//
// Update 2026-06-16 (owner-approved, re-locked): segment "Wallet" (renderFinance →
//   /api/finance/wallet) — R8 self-finance fase-1: kredensial wallet EVM (address publik +
//   private key masked di secret store + chain/RPC/currency + hard-limit). Transaksi = fase-2.
//
// settings.js — halaman Settings GLOBAL (owner-level).
//
// Section (sub-tab internal): Akun & Keamanan, API Keys, Notifikasi, YouTube, Wallet.
//
// Data owner-level disimpan di flowork.db global (lewat /api/settings/*).
// AI agent TETAP terisolasi (warga punya store + channel sendiri).
//
// Semua label lewat dictionary i18n (t(...)) — no hardcode UI text.

import { t } from '/js/i18n.js';
import { esc, escAttr, fetchJSON, loadStyle } from '../js/utils.js';

const CSS = `
.set-bar { display:flex; gap:4px; margin-bottom:18px; padding:5px; flex-wrap:wrap;
  background:rgba(15,17,26,0.55); border:1px solid var(--glass-border); border-radius:12px; width:fit-content; max-width:100%; }
.set-btn { padding:8px 14px; font-size:0.82rem; font-weight:500; border-radius:8px;
  background:transparent; border:1px solid transparent; color:var(--text-muted); cursor:pointer;
  transition:background .15s,color .15s,border-color .15s; }
.set-btn:hover { background:rgba(139,92,246,0.08); color:#cbd5e1; }
.set-btn.active { background:linear-gradient(135deg,rgba(139,92,246,0.28),rgba(124,58,237,0.12));
  color:#c4b5fd; border-color:rgba(139,92,246,0.45); }
.set-panel { max-width:none; }
.set-card { background:rgba(15,17,26,0.6); border:1px solid var(--glass-border); border-radius:14px;
  padding:18px 20px; margin-bottom:16px; }
.set-card h3 { font-size:0.95rem; margin:0 0 4px; color:#e2e8f0; }
.set-card .sub { font-size:0.8rem; color:var(--text-muted); margin-bottom:14px; line-height:1.5; }
.set-row { display:flex; gap:8px; flex-wrap:wrap; margin-bottom:10px; align-items:center; }
.set-row input, .set-row select { padding:10px 12px; border-radius:8px; background:rgba(30,41,59,0.6);
  border:1px solid rgba(148,163,184,0.2); color:#e2e8f0; font-size:0.88rem; flex:1; min-width:140px; }
.set-row input:focus, .set-row select:focus { outline:none; border-color:#a78bfa; }
.set-btn-primary { padding:10px 16px; background:linear-gradient(135deg,#a78bfa,#7c3aed); color:#fff;
  border:none; border-radius:8px; font-weight:600; font-size:0.85rem; cursor:pointer; }
.set-btn-primary:disabled { opacity:.5; cursor:not-allowed; }
.set-list { list-style:none; padding:0; margin:8px 0 0; }
.set-list li { display:flex; justify-content:space-between; align-items:center; gap:8px;
  padding:9px 12px; border:1px solid var(--glass-border); border-radius:8px; margin-bottom:6px;
  font-size:0.85rem; background:rgba(30,41,59,0.35); }
.set-list .mono { font-family:monospace; font-size:0.8rem; color:#cbd5e1; word-break:break-all; }
.set-list .rm { background:none; border:none; color:#fca5a5; cursor:pointer; font-size:0.85rem; }
.set-msg { margin-top:10px; font-size:0.82rem; min-height:1em; }
.set-msg.ok { color:#86efac; } .set-msg.err { color:#fca5a5; }
.set-empty { color:var(--text-muted); font-size:0.82rem; padding:8px 0; }
.set-tag { font-size:0.7rem; color:#64748b; }
.set-total { font-size:1.4rem; font-weight:700; color:#86efac; margin:6px 0; }
.set-presets { display:flex; gap:6px; flex-wrap:wrap; margin:2px 0 10px; }
.set-preset { padding:5px 11px; font-size:0.76rem; border-radius:999px; cursor:pointer;
  background:rgba(139,92,246,0.10); border:1px solid rgba(139,92,246,0.30); color:#c4b5fd;
  transition:background .15s,border-color .15s; }
.set-preset:hover { background:rgba(139,92,246,0.22); border-color:rgba(139,92,246,0.55); }
.set-preset.set-on { background:rgba(34,197,94,0.14); border-color:rgba(34,197,94,0.4); color:#86efac; }
.set-hint { font-size:0.74rem; color:var(--text-muted); margin:-4px 0 8px; }
/* Guided API-key rows: every provider shown with status + how-to + inline edit. */
.set-list li.set-keyrow { display:block; padding:9px 11px; border:1px solid rgba(148,163,184,0.14);
  border-radius:9px; margin-bottom:7px; background:rgba(148,163,184,0.04); }
.set-keyhead { display:flex; justify-content:space-between; align-items:center; gap:8px; }
.set-keyenv { font-size:0.72rem; opacity:0.65; }
.set-changeme { font-size:0.7rem; font-weight:600; color:#fbbf24; background:rgba(251,191,36,0.12);
  padding:2px 8px; border-radius:999px; white-space:nowrap; }
.set-keyhint { font-size:0.74rem; color:var(--text-muted); margin:5px 0 7px; line-height:1.45; }
.set-keyhint a { color:#a78bfa; text-decoration:none; white-space:nowrap; }
.set-keyhint a:hover { text-decoration:underline; }
.set-keyact { display:flex; gap:6px; align-items:center; }
.set-keyin { flex:1; min-width:0; }
.set-keysave { background:rgba(139,92,246,0.18); border:1px solid rgba(139,92,246,0.4); color:#c4b5fd;
  border-radius:7px; padding:5px 12px; cursor:pointer; font-size:0.8rem; white-space:nowrap; }
.set-keysave:hover { background:rgba(139,92,246,0.3); }
`;

// Known integration keys — friendly platform label → env var name. Clicking a
// chip pre-fills the exact variable name so owners never have to guess it.
// Service names are proper nouns (kept verbatim); instructional text is i18n.
// Known integrations. `hint` + `url` make the keys page self-explanatory for
// non-developers: what the key is for, and exactly where to get it.
const KEY_PRESETS = [
  { label: 'Dev.to', key: 'DEVTO_API_KEY',
    hint: 'Publish articles to Dev.to. Get a key: Dev.to → Settings → Extensions → "Generate API Key".',
    url: 'https://dev.to/settings/extensions' },
  { label: 'X · auth_token', key: 'X_AUTH_TOKEN',
    hint: 'Post to X/Twitter (cookie auth). In a logged-in X tab: DevTools (F12) → Application → Cookies → copy the "auth_token" value.',
    url: 'https://x.com' },
  { label: 'X · ct0', key: 'X_CT0',
    hint: 'X/Twitter CSRF cookie, paired with auth_token. Same place: copy the "ct0" cookie value.',
    url: 'https://x.com' },
  { label: 'Facebook · Page ID', key: 'FB_PAGE_ID',
    hint: 'The numeric ID of a Facebook Page you manage. Open your Page → About → "Page transparency" → Page ID.',
    url: 'https://www.facebook.com' },
  { label: 'Facebook · Page token', key: 'FB_PAGE_TOKEN',
    hint: 'Long-lived Page access token. Get it from the Meta Graph API Explorer with the pages_manage_posts permission.',
    url: 'https://developers.facebook.com/tools/explorer/' },
  { label: 'LinkedIn', key: 'LINKEDIN_COOKIE',
    hint: 'LinkedIn session cookie for posting. In a logged-in tab: DevTools (F12) → Application → Cookies → copy "li_at".',
    url: 'https://www.linkedin.com' },
  { label: 'Telegram bot', key: 'TELEGRAM_BOT_TOKEN',
    hint: 'Token for your own Telegram bot. Chat with @BotFather → send /newbot → copy the token it gives you.',
    url: 'https://t.me/BotFather' },
  { label: 'FLOWORK_OS group id', key: 'FWOS_CHAT_ID',
    hint: 'The chat/group ID where promo posts are sent. Add @userinfobot to the group to read its ID (starts with -100…).',
    url: 'https://t.me/userinfobot' },
  { label: 'FLOWORK_OS bot', key: 'FWOS_BOT_TOKEN',
    hint: 'Bot token used for FLOWORK_OS group posts. Create it via @BotFather the same way as the Telegram bot.',
    url: 'https://t.me/BotFather' },
];

const SEGMENTS = [
  { key: 'account', label: () => t('menu.tab.settings.seg_account'), render: renderAccount },
  { key: 'keys', label: () => t('menu.tab.settings.seg_keys'), render: renderKeys },
  { key: 'switches', label: () => '🎛️ Switch Fitur', render: renderSwitches },
  { key: 'router', label: () => t('menu.tab.settings.seg_router'), render: renderRouterDefault },
  { key: 'notify', label: () => t('menu.tab.settings.seg_notify'), render: renderNotify },
  { key: 'finance', label: () => t('menu.tab.settings.seg_finance'), render: renderFinance },
  { key: 'youtube', label: () => t('menu.tab.settings.seg_youtube'), render: renderYouTube },
  { key: 'guardian', label: () => t('menu.tab.settings.seg_guardian'), render: renderGuardian },
  { key: 'evolve', label: () => '🧬 Auto-Push', render: renderEvolvePush },
  { key: 'compact', label: () => '🧠 Auto-Compact', render: renderCompact },
];

// The YouTube OAuth flow polls /api/settings/youtube every 2s while the owner
// authorizes in another tab. Tracked at module scope so switching segments (or
// re-rendering YouTube) cancels it — otherwise the interval kept firing in the
// background and could re-render YouTube over whatever segment was open.
let ytPoll = null;
function stopYtPoll() { if (ytPoll) { clearInterval(ytPoll); ytPoll = null; } }

export async function render(mainEl) {
  loadStyle('settings', CSS);
  const tk = (k) => t('menu.tab.settings.' + k);
  mainEl.innerHTML = `
    <div class="sgt-header">
      <h2>⚙️ ${esc(tk('title'))}</h2>
      <div class="sub" style="font-size:0.82rem;color:var(--text-muted);margin-top:4px;">${esc(tk('desc'))}</div>
    </div>
    <div class="set-bar" id="setBar">
      ${SEGMENTS.map(s => `<button class="set-btn" data-key="${escAttr(s.key)}">${esc(s.label())}</button>`).join('')}
    </div>
    <div class="set-panel" id="setPanel"></div>
  `;
  const panel = mainEl.querySelector('#setPanel');
  const btns = mainEl.querySelectorAll('.set-btn');
  async function open(key) {
    const seg = SEGMENTS.find(s => s.key === key) || SEGMENTS[0];
    stopYtPoll(); // leaving/switching segment: cancel any in-flight YouTube OAuth poll
    btns.forEach(b => b.classList.toggle('active', b.getAttribute('data-key') === seg.key));
    panel.innerHTML = `<div class="set-empty">${esc(t('common.loading_label').replace('{label}', seg.label()))}</div>`;
    try {
      await seg.render(panel);
    } catch (e) {
      panel.innerHTML = `<div class="set-msg err">${esc(String(e.message || e))}</div>`;
    }
  }
  btns.forEach(b => b.addEventListener('click', () => open(b.getAttribute('data-key'))));
  open('account');
}

// ── Akun & Keamanan ──────────────────────────────────────────────────────
async function renderAccount(panel) {
  const tk = (k) => t('menu.tab.settings.' + k);
  panel.innerHTML = `
    <div class="set-card">
      <h3>${esc(tk('account_h'))}</h3>
      <div class="sub">${esc(tk('account_sub'))}</div>
      <div class="set-row"><input type="password" id="pwOld" placeholder="${escAttr(tk('change_pw_old_ph'))}" autocomplete="current-password"></div>
      <div class="set-row"><input type="password" id="pwNew" placeholder="${escAttr(tk('change_pw_new_ph'))}" autocomplete="new-password"></div>
      <div class="set-row"><button class="set-btn-primary" id="pwBtn">${esc(tk('change_pw_btn'))}</button></div>
      <div class="set-msg" id="pwMsg"></div>
    </div>
    <div class="set-card">
      <h3>${esc(tk('logout_h'))}</h3>
      <div class="sub">${esc(tk('logout_sub'))}</div>
      <button class="set-btn-primary" id="logoutBtn" style="background:linear-gradient(135deg,#ef4444,#b91c1c);">${esc(tk('logout_btn'))}</button>
    </div>
  `;
  const msg = panel.querySelector('#pwMsg');
  panel.querySelector('#pwBtn').addEventListener('click', async () => {
    const oldPw = panel.querySelector('#pwOld').value;
    const newPw = panel.querySelector('#pwNew').value;
    msg.className = 'set-msg';
    msg.textContent = '';
    try {
      await fetchJSON('/api/auth/change-password', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ old: oldPw, new: newPw }),
      });
      msg.className = 'set-msg ok';
      msg.textContent = tk('change_pw_ok');
      panel.querySelector('#pwOld').value = '';
      panel.querySelector('#pwNew').value = '';
    } catch (e) {
      msg.className = 'set-msg err';
      msg.textContent = tk('change_pw_err') + ' ' + cleanErr(e);
    }
  });
  panel.querySelector('#logoutBtn').addEventListener('click', async () => {
    await fetch('/api/auth/logout', { method: 'POST' });
    window.location.href = '/login';
  });
}

// ── Token Crypto / API Keys ────────────────────────────────────────────────
async function renderKeys(panel) {
  const tk = (k) => t('menu.tab.settings.' + k);
  panel.innerHTML = `
    <div class="set-card">
      <h3>${esc(tk('keys_h'))}</h3>
      <div class="sub">${esc(tk('keys_sub'))}</div>
      <div class="set-row">
        <input type="text" id="kKey" placeholder="${escAttr(tk('keys_key_ph'))}" list="kKnown">
        <input type="text" id="kVal" placeholder="${escAttr(tk('keys_val_ph'))}">
        <button class="set-btn-primary" id="kAdd">${esc(t('common.btn.save'))}</button>
      </div>
      <div class="set-hint">${esc(tk('keys_preset_hint'))}</div>
      <div class="set-presets" id="kPresets">
        ${KEY_PRESETS.map(p => `<span class="set-preset" data-key="${escAttr(p.key)}" title="${escAttr(p.key)}">${esc(p.label)}</span>`).join('')}
      </div>
      <datalist id="kKnown">
        ${KEY_PRESETS.map(p => `<option value="${escAttr(p.key)}">`).join('')}
      </datalist>
      <div class="set-msg" id="kMsg"></div>
      <ul class="set-list" id="kList"></ul>
    </div>
  `;
  const list = panel.querySelector('#kList');
  const msg = panel.querySelector('#kMsg');
  // Clicking a preset chip pre-fills the exact env var name → owner just pastes
  // the value. Chips for keys already saved turn green so it's clear what's set.
  panel.querySelectorAll('.set-preset').forEach(c => c.addEventListener('click', () => {
    panel.querySelector('#kKey').value = c.getAttribute('data-key');
    const v = panel.querySelector('#kVal'); v.value = ''; v.focus();
    msg.className = 'set-msg'; msg.textContent = '';
  }));
  function markPresets(items) {
    const set = new Set((items || []).map(it => it.key));
    panel.querySelectorAll('.set-preset').forEach(c =>
      c.classList.toggle('set-on', set.has(c.getAttribute('data-key'))));
  }
  async function reload() {
    const d = await fetchJSON('/api/settings/keys');
    const items = d.items || [];
    markPresets(items);
    const savedMap = new Map(items.map(it => [it.key, it.masked || '••••']));
    const extra = items.filter(it => !KEY_PRESETS.some(p => p.key === it.key));
    // Guided list: EVERY known provider shown, set or not. Unset → "change me"
    // so a non-developer knows exactly what to fill, with a how-to link.
    const rowFor = (p) => {
      const isSet = savedMap.has(p.key);
      const status = isSet
        ? `<span class="set-tag mono set-on">${esc(savedMap.get(p.key))}</span>`
        : `<span class="set-changeme">change me</span>`;
      return `
        <li class="set-keyrow">
          <div class="set-keyhead">
            <span><b>${esc(p.label || p.key)}</b> <span class="mono set-keyenv">${esc(p.key)}</span></span>
            ${status}
          </div>
          ${p.hint ? `<div class="set-keyhint">${esc(p.hint)}${p.url ? ` <a href="${escAttr(p.url)}" target="_blank" rel="noopener">How to get →</a>` : ''}</div>` : ''}
          <div class="set-keyact">
            <input type="text" class="set-keyin" data-key="${escAttr(p.key)}" placeholder="${escAttr(isSet ? 'leave blank to keep' : 'paste token / change me')}">
            <button class="set-keysave" data-key="${escAttr(p.key)}">${esc(t('common.btn.save'))}</button>
            ${isSet ? `<button class="rm" data-key="${escAttr(p.key)}">${esc(t('common.btn.delete'))}</button>` : ''}
          </div>
        </li>`;
    };
    list.innerHTML = KEY_PRESETS.map(rowFor).join('')
      + extra.map(it => rowFor({ key: it.key, label: it.key })).join('');
    list.querySelectorAll('.set-keysave').forEach(b => b.addEventListener('click', async () => {
      const key = b.getAttribute('data-key');
      const input = b.closest('li').querySelector('.set-keyin');
      const value = input ? input.value : '';
      if (!value.trim()) { msg.className = 'set-msg'; msg.textContent = 'Field is empty — nothing to save.'; return; }
      try {
        await fetchJSON('/api/settings/keys', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ key, value }),
        });
        msg.className = 'set-msg ok'; msg.textContent = tk('keys_saved');
        await reload();
      } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
    }));
    list.querySelectorAll('.rm').forEach(b => b.addEventListener('click', async () => {
      const key = b.getAttribute('data-key');
      if (!confirm(tk('keys_del_confirm') + ' "' + key + '"?')) return;
      try {
        await fetchJSON('/api/settings/keys?key=' + encodeURIComponent(key), { method: 'DELETE' });
        await reload();
      } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
    }));
  }
  panel.querySelector('#kAdd').addEventListener('click', async () => {
    const key = panel.querySelector('#kKey').value.trim();
    const value = panel.querySelector('#kVal').value;
    msg.className = 'set-msg'; msg.textContent = '';
    try {
      await fetchJSON('/api/settings/keys', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key, value }),
      });
      msg.className = 'set-msg ok'; msg.textContent = tk('keys_saved');
      panel.querySelector('#kVal').value = '';
      await reload();
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });
  await reload();
}

// ── Switch Fitur (plug-and-play; ganti flowork.local.env) ───────────────────
// Switch perilaku FLOWORK_* yg DULU kebawa di flowork.local.env (invisible buat user fresh).
// Sekarang dari sini → ditulis ~/.flowork/flowork_settings.json (lintas-proses, dibaca router
// :2402 + host :1987). Presedensi: GUI menang > ENV > default. Sumber tiap switch di-badge.
async function renderSwitches(panel) {
  panel.innerHTML = `<div class="set-card"><h3>🎛️ Switch Fitur</h3>
    <div class="sub">Toggle perilaku Flowork — ganti edit <code>flowork.local.env</code> manual.
    Berlaku ke router & host (live ≤3 dtk). Badge: <b>gui</b>=dari sini · <b>env</b>=dari ENV · <b>default</b>=bawaan.</div>
    <div id="swList" class="sub">Loading…</div></div>`;
  const list = panel.querySelector('#swList');
  const badge = (src) => {
    const c = src === 'gui' ? '#22c55e' : src === 'env' ? '#f59e0b' : '#64748b';
    return `<span style="font-size:0.66rem;padding:1px 7px;border-radius:999px;background:${c}22;color:${c};border:1px solid ${c}66">${src}</span>`;
  };
  async function reload() {
    let d;
    try { d = await fetchJSON('/api/settings/switches'); }
    catch (e) { list.innerHTML = `<span class="set-msg err">${esc(cleanErr(e))}</span>`; return; }
    const groups = {};
    (d.switches || []).forEach(s => { (groups[s.category] = groups[s.category] || []).push(s); });
    list.innerHTML = Object.keys(groups).map(cat => `
      <div style="margin-top:14px"><div style="font-size:0.72rem;text-transform:uppercase;letter-spacing:.04em;color:var(--text-muted);margin-bottom:6px">${esc(cat)}</div>
      ${groups[cat].map(s => {
        const boolOn = /^(1|on|true|yes)$/i.test(s.value);
        const ctrl = s.type === 'bool'
          ? `<input type="checkbox" class="sw-in" data-key="${escAttr(s.key)}" data-type="bool" data-orig="${boolOn ? '1' : '0'}" ${boolOn ? 'checked' : ''}>`
          : `<input type="text" class="sw-in" data-key="${escAttr(s.key)}" data-type="${escAttr(s.type)}" data-orig="${escAttr(s.value)}" value="${escAttr(s.value)}" style="width:120px">`;
        return `<div style="display:flex;align-items:center;gap:10px;padding:7px 0;border-top:1px solid var(--border,#ffffff14)">
          <div style="flex:1">
            <div style="font-size:0.86rem">${esc(s.label)} ${badge(s.source)}</div>
            <div style="font-size:0.72rem;color:var(--text-muted)">${esc(s.desc)} <code style="opacity:.6">${esc(s.key)}</code></div>
          </div>${ctrl}</div>`;
      }).join('')}</div>`).join('')
      + `<div style="margin-top:16px;display:flex;gap:10px;align-items:center">
          <button class="set-btn-primary" id="swSave">Simpan</button>
          <span id="swMsg" class="set-msg"></span>
          <span style="flex:1"></span>
          <code style="font-size:0.66rem;color:var(--text-muted)">${esc(d.path || '')}</code>
        </div>`;
    panel.querySelector('#swSave').addEventListener('click', async () => {
      const msg = panel.querySelector('#swMsg');
      const values = {};
      list.querySelectorAll('.sw-in').forEach(el => {
        const cur = el.dataset.type === 'bool' ? (el.checked ? '1' : '0') : el.value.trim();
        if (cur !== (el.dataset.orig || '')) values[el.dataset.key] = cur; // cuma yg BERUBAH → gui-pin
      });
      if (!Object.keys(values).length) { msg.className = 'set-msg'; msg.textContent = 'Ga ada perubahan'; return; }
      msg.className = 'set-msg'; msg.textContent = 'Menyimpan…';
      try {
        await fetchJSON('/api/settings/switches', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ values }),
        });
        msg.className = 'set-msg ok'; msg.textContent = '✓ tersimpan (live ≤3 dtk ke router)';
        await reload();
      } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
    });
  }
  await reload();
}

// ── Default Router & Model ──────────────────────────────────────────────────
// Global fallback for agents that don't pin their own model/router. Per-agent
// config still wins — these values only fill the blank.
async function renderRouterDefault(panel) {
  const tk = (k) => t('menu.tab.settings.' + k);
  panel.innerHTML = `
    <div class="set-card">
      <h3>${esc(tk('router_h'))}</h3>
      <div class="sub">${esc(tk('router_sub'))}</div>
      <div class="set-row">
        <input type="text" id="rdModel" placeholder="${escAttr(tk('router_model_ph'))}">
      </div>
      <div class="set-hint">${esc(tk('router_model_hint'))}</div>
      <div class="set-row">
        <input type="text" id="rdUrl" placeholder="${escAttr(tk('router_url_ph'))}">
      </div>
      <div class="set-hint">${esc(tk('router_url_hint'))}</div>
      <div class="set-row"><button class="set-btn-primary" id="rdSave">${esc(t('common.btn.save'))}</button></div>
      <div class="set-msg" id="rdMsg"></div>
    </div>
  `;
  const msg = panel.querySelector('#rdMsg');
  try {
    const d = await fetchJSON('/api/settings/router-default');
    panel.querySelector('#rdModel').value = d.model || '';
    panel.querySelector('#rdUrl').value = d.router_url || '';
  } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  panel.querySelector('#rdSave').addEventListener('click', async () => {
    const model = panel.querySelector('#rdModel').value.trim();
    const router_url = panel.querySelector('#rdUrl').value.trim();
    msg.className = 'set-msg'; msg.textContent = '';
    try {
      await fetchJSON('/api/settings/router-default', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model, router_url }),
      });
      msg.className = 'set-msg ok'; msg.textContent = tk('router_saved');
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });
}

// LOCKED (soft, owner-approved 2026-06-20): GUI auto-push token. Token write-only, ga ke-commit.
// renderEvolvePush — Settings GLOBAL buat AUTO-PUSH evolusi (owner 2026-06-20: "api github
// untuk push saat selesai evolute taruh di seting global"). Token GitHub disimpan lokal di
// flowork.db (~/.flowork, GA ke-commit), write-only (server ga pernah balikin token — cuma
// has_token bool). Auto-push cuma jalan kalau: enabled + token ada + mode=auto + lolos SEMUA
// gate (karma+model+re-probe). Manual core-apply tetep STAGE (ga ke-push).
async function renderEvolvePush(panel) {
  panel.innerHTML = `
    <div class="set-card">
      <h3>🧬 Evolusi — Auto-Push GitHub</h3>
      <div class="sub">Pas organisme evolusi auto-commit core (mode AUTO + lolos semua gate), hasilnya
        di-push ke GitHub biar Flowork tetep abadi walau owner ga ada. Token disimpan lokal
        (ga ke-commit), ga pernah ditampilin balik. Manual core-apply tetep STAGE (ga ke-push).</div>
      <label class="set-row" style="display:flex;align-items:center;gap:8px;cursor:pointer">
        <input type="checkbox" id="epEnabled" style="width:auto"> <span>Aktifkan auto-push</span>
      </label>
      <div class="set-row">
        <input type="password" id="epToken" placeholder="GitHub token (ghp_… / fine-grained)" autocomplete="new-password">
      </div>
      <div class="set-hint" id="epTokenHint"></div>
      <div class="set-row">
        <input type="text" id="epRemote" placeholder="remote (default: origin)">
      </div>
      <div class="set-row">
        <input type="text" id="epBranch" placeholder="branch (kosong = branch aktif repo)">
      </div>
      <div class="set-hint">⚠️ Push pakai HTTPS http.extraHeader (token ga ke-tulis ke git config). Disaranin
        token fine-grained scope minimal (contents:write) ke repo Flowork aja, jangan token full-akses.</div>
      <div class="set-row"><button class="set-btn-primary" id="epSave">${esc(t('common.btn.save'))}</button></div>
      <div class="set-msg" id="epMsg"></div>
    </div>
  `;
  const msg = panel.querySelector('#epMsg');
  const tokenHint = panel.querySelector('#epTokenHint');
  try {
    const d = await fetchJSON('/api/evolve/push-config');
    panel.querySelector('#epEnabled').checked = !!d.enabled;
    panel.querySelector('#epRemote').value = d.remote || '';
    panel.querySelector('#epBranch').value = d.branch || '';
    tokenHint.textContent = d.has_token
      ? '✅ Token tersimpan. Kosongin field = biarin token lama; isi baru = ganti.'
      : '⚠️ Belum ada token — auto-push ga bakal jalan sampai diisi.';
  } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  panel.querySelector('#epSave').addEventListener('click', async () => {
    const enabled = panel.querySelector('#epEnabled').checked;
    const token = panel.querySelector('#epToken').value;          // JANGAN trim: biar owner bisa paste apa adanya
    const remote = panel.querySelector('#epRemote').value.trim();
    const branch = panel.querySelector('#epBranch').value.trim();
    msg.className = 'set-msg'; msg.textContent = '';
    // Body: token cuma dikirim kalau owner ngetik (field kosong = jangan sentuh token lama).
    const body = { enabled, remote, branch };
    if (token.length > 0) body.token = token;
    try {
      await fetchJSON('/api/evolve/push-config', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      panel.querySelector('#epToken').value = '';
      msg.className = 'set-msg ok'; msg.textContent = 'Tersimpan ✓';
      const d = await fetchJSON('/api/evolve/push-config');
      tokenHint.textContent = d.has_token
        ? '✅ Token tersimpan. Kosongin field = biarin token lama; isi baru = ganti.'
        : '⚠️ Belum ada token — auto-push ga bakal jalan sampai diisi.';
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });
}

// ── Notifikasi (Telegram owner-level, TERPISAH dari agent) ──────────────────
async function renderNotify(panel) {
  const tk = (k) => t('menu.tab.settings.' + k);
  panel.innerHTML = `
    <div class="set-card">
      <h3>${esc(tk('notify_h'))}</h3>
      <div class="sub">${esc(tk('notify_sub'))}</div>
      <div class="set-row"><input type="password" id="ntToken" placeholder="${escAttr(tk('notify_token_ph'))}" autocomplete="off"></div>
      <div class="set-row"><input type="text" id="ntChat" placeholder="${escAttr(tk('notify_chat_ph'))}"></div>
      <div class="set-tag" style="margin-bottom:10px;">${esc(tk('notify_chat_hint'))}</div>
      <div class="set-row">
        <button class="set-btn-primary" id="ntSave">${esc(tk('notify_save'))}</button>
        <button class="set-btn-primary" id="ntTest" style="background:linear-gradient(135deg,#22c55e,#15803d);">${esc(tk('notify_test'))}</button>
      </div>
      <div class="set-msg" id="ntMsg"></div>
    </div>
  `;
  const msg = panel.querySelector('#ntMsg');
  const tokenEl = panel.querySelector('#ntToken');
  const chatEl = panel.querySelector('#ntChat');
  // Prefill: chat id + placeholder masked token kalau udah di-set.
  try {
    const d = await fetchJSON('/api/settings/notify');
    if (d.chat_id) chatEl.value = d.chat_id;
    if (d.set && d.bot_token_masked) tokenEl.setAttribute('placeholder', d.bot_token_masked + ' (saved — leave blank to keep)');
  } catch (e) { /* ignore */ }

  async function save(test) {
    msg.className = 'set-msg'; msg.textContent = '';
    const payload = { chat_id: chatEl.value.trim(), test };
    const tok = tokenEl.value.trim();
    if (tok) payload.bot_token = tok;
    try {
      const r = await fetchJSON('/api/settings/notify', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (test) {
        if (r.test === 'sent') { msg.className = 'set-msg ok'; msg.textContent = tk('notify_test_sent'); }
        else { msg.className = 'set-msg err'; msg.textContent = tk('notify_test_fail') + ' ' + (r.error || r.test || ''); }
      } else {
        msg.className = 'set-msg ok'; msg.textContent = tk('notify_saved');
      }
      tokenEl.value = '';
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  }
  panel.querySelector('#ntSave').addEventListener('click', () => save(false));
  panel.querySelector('#ntTest').addEventListener('click', () => save(true));
}

function cleanErr(e) {
  const m = String(e && e.message || e);
  const j = m.indexOf('{');
  if (j >= 0) { try { return JSON.parse(m.slice(j)).error || m; } catch {} }
  return m;
}

// LOCKED (soft, owner-approved 2026-06-20): GUI ambang auto-compact.
// renderCompact — Settings AUTO-COMPACT (owner 2026-06-20): ambang konteks → agent auto
// digest pengalaman ke brain + trim (anti-halu konteks panjang). Pengalaman ga ilang (pindah
// ke brain), cuma raw interaksi lama di-trim. Trigger by UKURAN, cek tiap 15 menit.
async function renderCompact(panel) {
  panel.innerHTML = `
    <div class="set-card">
      <h3>🧠 Auto-Compact Konteks (anti-halu)</h3>
      <div class="sub">Kalau interaksi agent numpuk, AI bisa halu pas konteks kepanjangan. Tiap 15 menit,
        agent yang interaksinya lewat ambang otomatis: <b>digest pengalaman ke brain</b> (kayak dream) →
        <b>trim</b> interaksi lama (sisain yang terbaru). Pengalaman TIDAK hilang — pindah ke brain,
        bisa di-recall. Aman: cuma yang sudah masuk brain yang di-trim.</div>
      <label class="set-row" style="display:flex;align-items:center;gap:8px;cursor:pointer">
        <input type="checkbox" id="cmpEnabled" style="width:auto"> <span>Aktifkan auto-compact</span>
      </label>
      <div class="set-row">
        <label style="font-size:0.85rem;min-width:180px;display:inline-block">Ambang (jumlah interaksi):</label>
        <input type="number" id="cmpMax" min="20" step="20" style="width:110px">
      </div>
      <div class="set-hint">Agent yang interaksi non-deleted-nya > angka ini bakal di-compact. Default 400.</div>
      <div class="set-row">
        <label style="font-size:0.85rem;min-width:180px;display:inline-block">Sisain terbaru (keep):</label>
        <input type="number" id="cmpKeep" min="0" step="10" style="width:110px">
      </div>
      <div class="set-hint">Berapa interaksi TERBARU yang tetap utuh di konteks (tidak di-trim). Default 60.</div>
      <div class="set-row">
        <label style="font-size:0.85rem;min-width:180px;display:inline-block">Model digest (opsional):</label>
        <input type="text" id="cmpModel" placeholder="kosong = pakai model LOKAL (flowork-brain)" style="flex:1;min-width:240px" autocomplete="off">
      </div>
      <div class="set-hint">Model buat digest pengalaman ke brain saat compact (auto, Compact All, maupun per-agent). <b>Kosongkan = pakai model LOKAL (flowork-brain)</b> — gratis &amp; tetap jalan tanpa langganan. Isi nama model cloud kapabel kalau mau digest lebih andal pas konteks besar (selama masih ada langganan).</div>
      <div class="set-row"><button class="set-btn-primary" id="cmpSave">${esc(t('common.btn.save'))}</button></div>
      <div class="set-msg" id="cmpMsg"></div>
    </div>
  `;
  const msg = panel.querySelector('#cmpMsg');
  try {
    const d = await fetchJSON('/api/compact/config');
    panel.querySelector('#cmpEnabled').checked = d.enabled !== false;
    panel.querySelector('#cmpMax').value = d.max_interactions || 400;
    panel.querySelector('#cmpKeep').value = (d.keep_recent ?? 60);
    panel.querySelector('#cmpModel').value = (d.model || '');
  } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  panel.querySelector('#cmpSave').addEventListener('click', async () => {
    const enabled = panel.querySelector('#cmpEnabled').checked;
    const max_interactions = parseInt(panel.querySelector('#cmpMax').value, 10) || 400;
    const keep_recent = parseInt(panel.querySelector('#cmpKeep').value, 10);
    const model = panel.querySelector('#cmpModel').value.trim();
    msg.className = 'set-msg'; msg.textContent = '';
    if (keep_recent >= max_interactions) {
      msg.className = 'set-msg err'; msg.textContent = 'keep_recent harus < ambang (kalau ga, ga ada yg di-trim)'; return;
    }
    try {
      await fetchJSON('/api/compact/config', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled, max_interactions, keep_recent: isNaN(keep_recent) ? 60 : keep_recent, model }),
      });
      msg.className = 'set-msg ok'; msg.textContent = 'Tersimpan ✓';
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });
}

// ── Self-finance wallet (R8 fase-1) ──────────────────────────────────────────
// Kredensial wallet EVM organisme: address (publik), private key (secret, masked, ga pernah
// ditampilkan), chain/RPC/currency, hard-limit (pagar belanja). Transaksi nyata = fase-2.
async function renderFinance(panel) {
  const tk = (k) => t('menu.tab.settings.' + k);
  panel.innerHTML = `
    <div class="set-card">
      <h3>${esc(tk('fin_h'))}</h3>
      <div class="sub">${esc(tk('fin_sub'))}</div>
      <div class="set-row"><input type="text" id="finAddr" placeholder="${escAttr(tk('fin_addr_ph'))}" autocomplete="off"></div>
      <div class="set-row"><input type="password" id="finPk" placeholder="${escAttr(tk('fin_pk_ph'))}" autocomplete="off"></div>
      <div class="set-tag" style="margin-bottom:10px;color:#fbbf24;">${esc(tk('fin_pk_hint'))}</div>
      <div class="set-row"><input type="text" id="finChain" placeholder="${escAttr(tk('fin_chain_ph'))}"></div>
      <div class="set-row"><input type="text" id="finRpc" placeholder="${escAttr(tk('fin_rpc_ph'))}"></div>
      <div class="set-row" style="display:flex;gap:8px;">
        <input type="text" id="finCur" placeholder="${escAttr(tk('fin_cur_ph'))}" style="flex:1">
        <input type="number" id="finLimit" min="0" step="any" placeholder="${escAttr(tk('fin_limit_ph'))}" style="flex:1">
      </div>
      <div class="set-row"><label style="display:flex;gap:8px;align-items:center;cursor:pointer;font-size:0.86rem"><input type="checkbox" id="finEnabled"> ${esc(tk('fin_enabled'))}</label></div>
      <div class="set-tag" id="finBalance" style="margin-bottom:10px;"></div>
      <div class="set-row">
        <button class="set-btn-primary" id="finSave">${esc(tk('fin_save'))}</button>
        <button class="set-btn-primary" id="finDelete" style="background:linear-gradient(135deg,#ef4444,#b91c1c);">${esc(tk('fin_delete'))}</button>
      </div>
      <div class="set-msg" id="finMsg"></div>
    </div>
  `;
  const msg = panel.querySelector('#finMsg');
  const els = {
    addr: panel.querySelector('#finAddr'), pk: panel.querySelector('#finPk'),
    chain: panel.querySelector('#finChain'), rpc: panel.querySelector('#finRpc'),
    cur: panel.querySelector('#finCur'), limit: panel.querySelector('#finLimit'),
    enabled: panel.querySelector('#finEnabled'),
  };
  try {
    const d = await fetchJSON('/api/finance/wallet');
    if (d.address) els.addr.value = d.address;
    if (d.chain) els.chain.value = d.chain;
    if (d.rpc) els.rpc.value = d.rpc;
    if (d.currency) els.cur.value = d.currency;
    if (d.hard_limit) els.limit.value = d.hard_limit;
    els.enabled.checked = !!d.enabled;
    if (d.has_privkey && d.privkey_masked) els.pk.setAttribute('placeholder', d.privkey_masked + ' (saved — leave blank to keep)');
    panel.querySelector('#finBalance').textContent = d.balance || '';
  } catch (e) { /* ignore */ }

  async function save() {
    msg.className = 'set-msg'; msg.textContent = '';
    const payload = {
      address: els.addr.value.trim(), chain: els.chain.value.trim(), rpc: els.rpc.value.trim(),
      currency: els.cur.value.trim(), hard_limit: els.limit.value.trim(), enabled: els.enabled.checked,
    };
    const pk = els.pk.value.trim();
    if (pk) payload.privkey = pk;
    try {
      await fetchJSON('/api/finance/wallet', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
      msg.className = 'set-msg ok'; msg.textContent = tk('fin_saved');
      els.pk.value = '';
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  }
  panel.querySelector('#finSave').addEventListener('click', save);
  panel.querySelector('#finDelete').addEventListener('click', async () => {
    if (!confirm(tk('fin_delete_confirm'))) return;
    msg.className = 'set-msg'; msg.textContent = '';
    try {
      await fetchJSON('/api/finance/wallet', { method: 'DELETE' });
      msg.className = 'set-msg ok'; msg.textContent = tk('fin_deleted');
      els.pk.value = ''; els.pk.setAttribute('placeholder', tk('fin_pk_ph')); els.enabled.checked = false;
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });
}

// ── YouTube ────────────────────────────────────────────────────────────────
async function renderYouTube(panel) {
  const tk = (k) => t('menu.tab.settings.' + k);
  stopYtPoll(); // any prior OAuth poll is stale once we re-render
  let st;
  try { st = await fetchJSON('/api/settings/youtube'); }
  catch (e) { panel.innerHTML = `<div class="set-msg err">${esc(cleanErr(e))}</div>`; return; }
  const ch = st.channel || null;

  let accountHTML;
  if (st.connected && ch) {
    accountHTML = `
      <div class="set-card">
        <h3>✅ ${esc(tk('yt_connected'))} — ${esc(ch.title || '')}</h3>
        <div class="sub">${esc(ch.handle || '')} · ${esc(ch.video_count || '0')} ${esc(tk('yt_videos'))} · ${esc(ch.sub_count || '0')} ${esc(tk('yt_subs'))}</div>
        <div class="sub">${esc(tk('yt_long_uploads'))}: <span class="mono">${esc(ch.long_uploads_status || '?')}</span></div>
        <div class="set-row" style="margin-top:8px;"><button class="set-btn-primary" id="ytDisc" style="background:linear-gradient(135deg,#ef4444,#b91c1c);">${esc(tk('yt_disconnect_btn'))}</button></div>
      </div>`;
  } else if (st.has_credentials) {
    accountHTML = `
      <div class="set-card">
        <h3>${esc(tk('yt_not_connected'))}</h3>
        <div class="sub">${esc(tk('yt_connect_hint'))}</div>
        <div class="set-row" style="margin-top:8px;"><button class="set-btn-primary" id="ytConnect">${esc(tk('yt_connect_btn'))}</button></div>
        <div class="set-msg" id="ytConnMsg"></div>
      </div>`;
  } else {
    accountHTML = `<div class="set-card"><div class="set-empty">${esc(tk('yt_no_creds'))}</div></div>`;
  }

  panel.innerHTML = `
    <div class="set-card">
      <h3>🎷 ${esc(tk('yt_h'))}</h3>
      <div class="sub">${esc(tk('yt_sub'))}</div>
    </div>
    <details class="set-card">
      <summary style="cursor:pointer;font-weight:600;">${esc(tk('yt_guide_title'))}</summary>
      <div class="sub" style="white-space:pre-line;margin-top:8px;">${esc(tk('yt_guide_body'))}</div>
      <a href="https://console.cloud.google.com/" target="_blank" rel="noopener" class="set-tag" style="display:inline-block;margin-top:8px;">${esc(tk('yt_console_link'))}</a>
    </details>
    <div class="set-card">
      <h3>OAuth Client JSON</h3>
      <div class="set-row"><textarea id="ytJson" rows="4" placeholder="${escAttr(tk('yt_paste_ph'))}" style="width:100%;font-family:monospace;font-size:0.78rem;"></textarea></div>
      <div class="set-row"><button class="set-btn-primary" id="ytSaveCreds">${esc(tk('yt_save_creds'))}</button> ${st.has_credentials ? `<span class="set-tag">${esc(tk('yt_creds_saved'))}</span>` : ''}</div>
      <div class="set-msg" id="ytCredMsg"></div>
    </div>
    ${accountHTML}
    <div class="set-card">
      <h3>⚙️ ${esc(tk('yt_save_config'))}</h3>
      <div class="set-row">
        <label class="sub" style="min-width:170px;">${esc(tk('yt_privacy_label'))}</label>
        <select id="ytPrivacy">
          <option value="private"${st.privacy === 'private' ? ' selected' : ''}>private</option>
          <option value="unlisted"${st.privacy === 'unlisted' ? ' selected' : ''}>unlisted</option>
          <option value="public"${st.privacy === 'public' ? ' selected' : ''}>public</option>
        </select>
      </div>
      <div class="set-row">
        <label class="sub" style="min-width:170px;">${esc(tk('yt_inbox_label'))}</label>
        <input type="text" id="ytInbox" value="${escAttr(st.inbox || '')}" style="flex:1;">
      </div>
      <div class="set-row">
        <label class="sub"><input type="checkbox" id="ytWatcher"${st.watcher_enabled ? ' checked' : ''}> ${esc(tk('yt_watcher_label'))}</label>
      </div>
      <div class="set-row"><button class="set-btn-primary" id="ytSaveCfg">${esc(tk('yt_save_config'))}</button></div>
      <div class="set-msg" id="ytCfgMsg"></div>
    </div>
  `;

  panel.querySelector('#ytSaveCreds').addEventListener('click', async () => {
    const msg = panel.querySelector('#ytCredMsg'); msg.className = 'set-msg';
    const cj = panel.querySelector('#ytJson').value.trim();
    if (!cj) { msg.className = 'set-msg err'; msg.textContent = 'JSON is empty'; return; }
    try {
      await fetchJSON('/api/settings/youtube/credentials', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ client_json: cj }) });
      renderYouTube(panel);
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });

  const connectBtn = panel.querySelector('#ytConnect');
  if (connectBtn) connectBtn.addEventListener('click', async () => {
    const msg = panel.querySelector('#ytConnMsg'); msg.className = 'set-msg';
    try {
      const d = await fetchJSON('/api/settings/youtube/connect', { method: 'POST' });
      window.open(d.auth_url, '_blank');
      msg.innerHTML = '⏳ ' + esc(tk('yt_connect_hint')) + ' <a href="' + escAttr(d.auth_url) + '" target="_blank" rel="noopener">' + esc(tk('yt_open_auth')) + '</a>';
      let tries = 0;
      stopYtPoll(); // never stack two polls (e.g. repeated Connect clicks)
      ytPoll = setInterval(async () => {
        tries++;
        try { const s = await fetchJSON('/api/settings/youtube'); if (s.connected) { stopYtPoll(); renderYouTube(panel); } } catch {}
        if (tries > 90) stopYtPoll();
      }, 2000);
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });

  const discBtn = panel.querySelector('#ytDisc');
  if (discBtn) discBtn.addEventListener('click', async () => {
    if (!confirm(tk('yt_disconnect_btn') + '?')) return;
    try { await fetchJSON('/api/settings/youtube/disconnect', { method: 'POST' }); renderYouTube(panel); } catch {}
  });

  panel.querySelector('#ytSaveCfg').addEventListener('click', async () => {
    const msg = panel.querySelector('#ytCfgMsg'); msg.className = 'set-msg';
    try {
      await fetchJSON('/api/settings/youtube/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({
        privacy: panel.querySelector('#ytPrivacy').value,
        inbox: panel.querySelector('#ytInbox').value.trim(),
        watcher_enabled: panel.querySelector('#ytWatcher').checked,
      }) });
      msg.className = 'set-msg ok'; msg.textContent = '✓';
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });
}

// ── Guardian (integritas kernel) ─────────────────────────────────────────────
async function renderGuardian(panel) {
  const tk = (k) => t('menu.tab.settings.' + k);
  let s;
  try { s = await fetchJSON('/api/guardian/status'); }
  catch (e) { panel.innerHTML = `<div class="set-msg err">${esc(cleanErr(e))}</div>`; return; }
  const badge = s.safe_mode
    ? `<span style="color:#ff476f;font-weight:700">⛔ ${esc(tk('grd_safemode'))}</span>`
    : s.armed
      ? `<span style="color:#22ff88;font-weight:700">🛡️ ${esc(tk('grd_armed'))}</span>`
      : `<span style="color:#ffc24d">○ ${esc(tk('grd_disarmed'))}</span>`;
  panel.innerHTML = `
    <div class="set-card">
      <h3>🛡️ ${esc(tk('grd_title'))}</h3>
      <div class="sub">${esc(tk('grd_desc'))}</div>
      <div class="set-row">${badge} &nbsp;·&nbsp; ${esc(tk('grd_protected'))}: <b>${s.protected || 0}</b>${s.sealed_at ? ` &nbsp;·&nbsp; ${esc(tk('grd_sealed'))}: ${esc(s.sealed_at)}` : ''}</div>
      ${s.armed ? `<div class="set-row" style="font-size:0.8rem">${s.sealed
        ? `🔒 <span style="color:#22ff88">${esc(tk('grd_seal_os'))}</span> <span style="color:#64748b">(${esc(s.seal_method || '')})</span>`
        : `🔓 <span style="color:#ffc24d">${esc(tk('grd_seal_detect'))}</span>`}</div>` : ''}
      <div class="set-row" style="margin-top:12px;display:flex;gap:8px;flex-wrap:wrap">
        ${s.armed
          ? `<input type="password" id="grdPw" placeholder="${escAttr(tk('grd_pw_ph'))}" autocomplete="current-password" style="flex:1;min-width:160px">
             <button class="set-btn-primary" id="grdDisarm">${esc(tk('grd_disarm_btn'))}</button>`
          : `<button class="set-btn-primary" id="grdArm">${esc(tk('grd_arm_btn'))}</button>`}
      </div>
      <div class="set-msg" id="grdMsg"></div>
    </div>`;
  const msg = panel.querySelector('#grdMsg');
  const arm = panel.querySelector('#grdArm');
  if (arm) arm.addEventListener('click', async () => {
    if (!confirm(tk('grd_arm_confirm'))) return;
    msg.className = 'set-msg';
    try { await fetchJSON('/api/guardian/arm', { method: 'POST' }); renderGuardian(panel); }
    catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });
  const dis = panel.querySelector('#grdDisarm');
  if (dis) dis.addEventListener('click', async () => {
    msg.className = 'set-msg';
    try {
      await fetchJSON('/api/guardian/disarm', { method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password: panel.querySelector('#grdPw').value }) });
      renderGuardian(panel);
    } catch (e) { msg.className = 'set-msg err'; msg.textContent = cleanErr(e); }
  });
}
