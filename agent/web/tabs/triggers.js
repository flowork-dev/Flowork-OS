// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-20 (+ target system:compact-all).
// LOCKED ≠ FREEZE (boleh diedit dgn izin owner).
//
// triggers.js — tab "Trigger" (ROADMAP 3): otomasi event→aksi, ala Google Tag Manager.
// KALAU <event> MAKA suruh <agent/group> dgn <prompt {{payload}}> → kirim Telegram.
// Tema Matrix × Jarvis. Semua label lewat i18n (dictionary), no hardcode.
import { esc, escAttr, fetchJSON, loadStyle } from '../js/utils.js';
import { t } from '/js/i18n.js';

const L = new Proxy({}, { get: (_, k) => t('triggers.' + String(k)) });
const fmt = (k, v) => Object.entries(v || {}).reduce((s, [n, val]) => s.replaceAll('{' + n + '}', val), L[k]);

const CSS = `
.tg{--mx:#22ff88;--cy:#36e6ff;--bad:#ff476f;--warn:#ffc24d;--line:rgba(34,255,136,.20);
  position:relative;padding:6px 2px 50px;color:#bdf5d6;
  font-family:ui-monospace,'JetBrains Mono','Cascadia Code',Consolas,monospace}
.tg::before{content:'';position:absolute;left:0;right:0;top:0;height:90px;z-index:0;pointer-events:none;opacity:.5;
  background:linear-gradient(180deg,rgba(34,255,136,.10),transparent);animation:tgscan 6s linear infinite}
@keyframes tgscan{0%{transform:translateY(-30px);opacity:0}30%{opacity:1}100%{transform:translateY(560px);opacity:0}}
.tg>*{position:relative;z-index:1}
.tg-hud{display:flex;align-items:center;gap:16px;margin:6px 0 20px}
.tg-orb{width:42px;height:42px;flex:0 0 auto;position:relative}
.tg-core{position:absolute;inset:38%;border-radius:50%;background:radial-gradient(circle,#aef6ff,var(--mx) 45%,transparent 72%);
  box-shadow:0 0 12px var(--mx),0 0 26px rgba(34,255,136,.5);animation:tgpulse 2.4s ease-in-out infinite}
.tg-ring{position:absolute;inset:0;border-radius:50%;border:1px solid var(--line);animation:tgspin 7s linear infinite}
@keyframes tgspin{to{transform:rotate(360deg)}}@keyframes tgpulse{0%,100%{transform:scale(1)}50%{transform:scale(1.25);filter:brightness(1.3)}}
.tg-h h2{margin:0;font-size:18px;letter-spacing:4px;color:#eafdff;text-shadow:0 0 12px rgba(34,255,136,.5)}
.tg-h .sub{font-size:12px;color:var(--mx);opacity:.8;margin-top:4px}
.tg-stat{font-size:10px;letter-spacing:2px;color:var(--mx);margin-top:6px;display:flex;align-items:center;gap:7px}
.tg-dot{width:7px;height:7px;border-radius:50%;background:var(--mx);box-shadow:0 0 8px var(--mx);animation:tgblink 1.6s infinite}
@keyframes tgblink{0%,100%{opacity:1}50%{opacity:.25}}
.tg-btn{padding:8px 15px;border-radius:5px;background:rgba(34,255,136,.08);border:1px solid var(--line);color:var(--mx);
  cursor:pointer;font:inherit;font-size:12px;letter-spacing:1px;font-weight:600;transition:.15s}
.tg-btn:hover{border-color:var(--mx);color:#eafdff;background:rgba(34,255,136,.16);box-shadow:0 0 14px rgba(34,255,136,.3)}
.tg-btn.primary{background:linear-gradient(90deg,var(--mx),var(--cy));color:#001a14;font-weight:700;border-color:var(--mx)}
.tg-btn.danger{color:var(--bad);border-color:rgba(255,71,111,.4)}.tg-btn.danger:hover{background:rgba(255,71,111,.12)}
.tg-btn.sm{padding:4px 10px;font-size:11px}
.tg-panel{position:relative;background:rgba(4,16,11,.6);border:1px solid var(--line);border-radius:8px;padding:14px 16px;margin-bottom:14px;backdrop-filter:blur(3px)}
.tg-panel::before,.tg-panel::after{content:'';position:absolute;width:12px;height:12px;border:2px solid var(--mx);opacity:.7;pointer-events:none}
.tg-panel::before{top:-1px;left:-1px;border-right:0;border-bottom:0}.tg-panel::after{bottom:-1px;right:-1px;border-left:0;border-top:0}
.tg-card{display:flex;align-items:center;gap:12px;flex-wrap:wrap}
.tg-name{font-size:15px;color:#eafdff;font-weight:700}
.tg-tag{font-size:10px;letter-spacing:1px;color:var(--cy);border:1px solid var(--line);border-radius:3px;padding:2px 7px;background:rgba(54,230,255,.06)}
.tg-cfg{font-size:11px;color:#7fb9a6;font-family:ui-monospace,monospace}
.tg-grow{flex:1 1 auto}
.tg-state{font-size:10px;letter-spacing:1px;padding:3px 9px;border-radius:3px}
.tg-state.ok{color:var(--mx);border:1px solid rgba(34,255,136,.4);background:rgba(34,255,136,.07)}
.tg-state.error{color:var(--bad);border:1px solid rgba(255,71,111,.35);background:rgba(255,71,111,.06)}
.tg-state.idle{color:var(--warn);border:1px solid rgba(255,194,77,.3)}
.tg-sw{cursor:pointer;font-size:11px;color:var(--mx)}
.tg-row{display:flex;gap:10px;flex-wrap:wrap;align-items:center;margin-top:8px}
.tg-lbl{font-size:10px;letter-spacing:2px;color:var(--mx);text-transform:uppercase;margin:10px 0 4px}
.tg-in,.tg-sel,.tg-ta{width:100%;box-sizing:border-box;background:rgba(2,12,8,.85);border:1px solid var(--line);border-radius:5px;
  color:#dffaf0;padding:9px 11px;font:inherit;font-size:13px;outline:none}
.tg-in:focus,.tg-sel:focus,.tg-ta:focus{border-color:var(--mx);box-shadow:0 0 0 1px var(--mx)}
.tg-ta{min-height:64px;resize:vertical}
.tg-chip{display:inline-block;font-size:11px;color:var(--cy);border:1px solid var(--line);border-radius:3px;padding:2px 7px;margin:3px 4px 0 0;cursor:pointer;background:rgba(54,230,255,.06)}
.tg-msg{font-size:12px;margin-top:8px;min-height:15px}.tg-msg.ok{color:var(--mx)}.tg-msg.err{color:var(--bad)}
.tg-empty{color:var(--mx);opacity:.65;text-align:center;padding:16px;font-size:13px}
.tg-url{font-size:11px;color:#7fb9a6;word-break:break-all;background:rgba(2,12,8,.7);border:1px dashed var(--line);border-radius:4px;padding:6px 9px;margin-top:6px}
.tg-hist{margin-top:8px;font-size:11px;color:#9fd9c4}.tg-hist .r{padding:4px 0;border-bottom:1px solid rgba(34,255,136,.08)}
`;

let agents = [], groups = [], types = [];

export async function render(mainEl) {
  loadStyle('triggers', CSS);
  mainEl.innerHTML = `
    <div class="tg">
      <div class="tg-hud">
        <div class="tg-orb"><div class="tg-ring"></div><div class="tg-core"></div></div>
        <div class="tg-h"><h2>⚡ ${esc(L.title)}</h2><div class="sub">${esc(L.sub)}</div>
          <div class="tg-stat"><span class="tg-dot"></span><span id="tgCount">…</span></div></div>
        <span class="tg-grow"></span>
        <button class="tg-btn primary" id="tgNew">＋ ${esc(L.new_btn)}</button>
      </div>
      <div id="tgForm"></div>
      <div id="tgList"><div class="tg-empty">⟳ ${esc(L.loading)}</div></div>
    </div>`;
  mainEl.querySelector('#tgNew').addEventListener('click', () => openForm(mainEl, null));
  await loadRefs();
  await load(mainEl);
}

async function loadRefs() {
  // Schedule (type "time") punya menu SENDIRI → di sini cuma event PLUGIN (webhook/file-watch/…).
  try { const d = await fetchJSON('/api/triggers/types'); types = (d.types || []).filter(t => t.id !== 'time'); } catch { types = []; }
  try { const d = await fetchJSON('/api/kernel/agents'); agents = (d.plugins || []).filter(a => a.id && a.kind !== 'channel'); } catch { agents = []; }
  try { const d = await fetchJSON('/api/groups'); groups = d.groups || []; } catch { groups = []; }
}

async function load(mainEl) {
  const list = mainEl.querySelector('#tgList');
  let data;
  try { data = await fetchJSON('/api/triggers'); }
  catch (e) { list.innerHTML = `<div class="tg-empty">${esc(String(e.message || e))}</div>`; return; }
  const rules = (data.triggers || []).filter(r => r.type_id !== 'time'); // jadwal waktu → tab Schedule
  mainEl.querySelector('#tgCount').textContent = `${rules.length} ${L.count_label}`;
  if (!rules.length) { list.innerHTML = `<div class="tg-panel"><div class="tg-empty">${esc(L.empty)}</div></div>`; return; }
  list.innerHTML = rules.map(cardHTML).join('');
  rules.forEach(r => wireCard(mainEl, r));
}

function typeOf(id) { return types.find(t => t.id === id) || { id, name: id, mode: 'poll' }; }
function cfgSummary(r) {
  try { const c = JSON.parse(r.config || '{}'); return Object.entries(c).map(([k, v]) => `${k}=${v}`).join(' · '); } catch { return ''; }
}

function cardHTML(r) {
  const st = r.last_status || '';
  const cls = st === 'ok' ? 'ok' : (st === 'error' || st === 'bad_config' || st === 'type_removed') ? 'error' : 'idle';
  const ty = typeOf(r.type_id);
  return `<div class="tg-panel" data-id="${escAttr(r.id)}">
    <div class="tg-card">
      <span class="tg-name">${r.enabled ? '🟢' : '⚪'} ${esc(r.name)}</span>
      <span class="tg-tag">${esc(ty.name)}</span>
      <span class="tg-cfg">${esc(cfgSummary(r))} → ${esc(r.target)}</span>
      <span class="tg-grow"></span>
      <span class="tg-state ${cls}">${esc(st || L.idle)}</span>
      <span class="tg-sw" data-act="toggle">${r.enabled ? esc(L.btn_disable) : esc(L.btn_enable)}</span>
      <button class="tg-btn sm" data-act="run">▷ ${esc(L.btn_run)}</button>
      <button class="tg-btn sm" data-act="edit">✎</button>
      <button class="tg-btn sm" data-act="dup" title="Duplicate (new copy, disabled first)">⧉</button>
      <button class="tg-btn sm" data-act="hist">▸</button>
      <button class="tg-btn sm danger" data-act="del">🗑</button>
    </div>
    ${ty.mode === 'webhook' && r.webhook_secret ? `<div class="tg-url">POST ${location.origin}/api/triggers/hook/${esc(r.id)}?key=${esc(r.webhook_secret)}</div>` : ''}
    <div class="tg-hist" data-hist style="display:none"></div>
  </div>`;
}

function wireCard(mainEl, r) {
  const card = mainEl.querySelector(`.tg-panel[data-id="${r.id}"]`); // id = slug server-validated, aman
  if (!card) return;
  card.querySelector('[data-act="toggle"]').onclick = async () => {
    try { await fetchJSON(`/api/triggers/toggle?id=${encodeURIComponent(r.id)}&enabled=${r.enabled ? 0 : 1}`, { method: 'POST' }); await load(mainEl); } catch (e) { alert(e.message); }
  };
  card.querySelector('[data-act="run"]').onclick = async () => {
    try { await fetchJSON(`/api/triggers/run?id=${encodeURIComponent(r.id)}`, { method: 'POST' }); alert(L.run_ok); } catch (e) { alert(e.message); }
  };
  card.querySelector('[data-act="edit"]').onclick = () => openForm(mainEl, r);
  card.querySelector('[data-act="dup"]').onclick = async () => {
    try { const d = await fetchJSON(`/api/triggers/duplicate?id=${encodeURIComponent(r.id)}`, { method: 'POST' }); alert('Duplicated → ' + d.id + ' (disabled — enable when ready)'); await load(mainEl); } catch (e) { alert(e.message); }
  };
  card.querySelector('[data-act="del"]').onclick = async () => {
    if (!confirm(fmt('del_confirm', { name: r.name }))) return;
    try { await fetchJSON(`/api/triggers/delete?id=${encodeURIComponent(r.id)}`, { method: 'POST' }); await load(mainEl); } catch (e) { alert(e.message); }
  };
  card.querySelector('[data-act="hist"]').onclick = async () => {
    const box = card.querySelector('[data-hist]');
    if (box.style.display === 'block') { box.style.display = 'none'; return; }
    box.style.display = 'block'; box.innerHTML = '…';
    try {
      const d = await fetchJSON(`/api/triggers/runs?id=${encodeURIComponent(r.id)}&limit=15`);
      const runs = d.runs || [];
      box.innerHTML = runs.length ? runs.map(x => `<div class="r">${esc(x.fired_at)} · <b>${esc(x.status)}</b> · ${esc(x.trigger)} · ${esc((x.result_text || x.error_text || '').slice(0, 120))}</div>`).join('') : `<div class="r">${esc(L.no_runs)}</div>`;
    } catch (e) { box.innerHTML = esc(e.message); }
  };
}

function targetOptions(sel) {
  const a = agents.map(x => `<option value="agent:${escAttr(x.id)}" ${sel === 'agent:' + x.id ? 'selected' : ''}>🤖 ${esc(x.display_name || x.id)}</option>`).join('');
  const g = groups.map(x => `<option value="group:${escAttr(x.id)}" ${sel === 'group:' + x.id ? 'selected' : ''}>👥 ${esc(x.display_name || x.id)}</option>`).join('');
  // Aksi SISTEM (owner 2026-06-20 "all compact ke triger"): jadwalin Compact All via trigger.
  const sys = `<option value="system:compact-all" ${sel === 'system:compact-all' ? 'selected' : ''}>🧠 Compact All (semua agent → brain)</option>`;
  return `<optgroup label="Agents">${a}</optgroup>${g ? `<optgroup label="Groups">${g}</optgroup>` : ''}<optgroup label="Sistem">${sys}</optgroup>`;
}

function openForm(mainEl, r) {
  const box = mainEl.querySelector('#tgForm');
  const editing = !!r;
  const cur = r || { id: '', name: '', type_id: types[0] ? types[0].id : 'webhook', config: '{}', target: '', target_kind: 'agent', prompt: '' };
  let cfg = {}; try { cfg = JSON.parse(cur.config || '{}'); } catch {}
  const selTarget = cur.target ? cur.target_kind + ':' + cur.target : '';
  box.innerHTML = `<div class="tg-panel">
    <div class="tg-lbl">${esc(editing ? L.edit_title : L.new_title)}</div>
    <div class="tg-row">
      <div style="flex:1;min-width:160px"><div class="tg-lbl">${esc(L.f_id)}</div>
        <input class="tg-in" id="fId" value="${escAttr(cur.id)}" ${editing ? 'readonly' : ''} placeholder="report-saham"></div>
      <div style="flex:1;min-width:160px"><div class="tg-lbl">${esc(L.f_name)}</div>
        <input class="tg-in" id="fName" value="${escAttr(cur.name)}" placeholder="Report Saham A"></div>
      <div style="flex:1;min-width:140px"><div class="tg-lbl">${esc(L.f_type)}</div>
        <select class="tg-sel" id="fType">${types.map(t => `<option value="${escAttr(t.id)}" ${t.id === cur.type_id ? 'selected' : ''}>${esc(t.name)}</option>`).join('')}</select></div>
    </div>
    <div id="fCfg"></div>
    <div class="tg-lbl">${esc(L.f_target)}</div>
    <select class="tg-sel" id="fTarget">${targetOptions(selTarget)}</select>
    <div class="tg-lbl">${esc(L.f_prompt)}</div>
    <textarea class="tg-ta" id="fPrompt" placeholder="${escAttr(L.prompt_ph)}">${esc(cur.prompt)}</textarea>
    <div id="fChips"></div>
    <div class="tg-row" style="margin-top:12px">
      <button class="tg-btn primary" id="fSave">${esc(L.save)}</button>
      <button class="tg-btn" id="fCancel">${esc(L.cancel)}</button>
      <span class="tg-msg" id="fMsg"></span>
    </div>
  </div>`;
  const renderCfg = () => {
    const ty = typeOf(box.querySelector('#fType').value);
    const fields = ty.config_schema || [];
    box.querySelector('#fCfg').innerHTML = fields.length ? fields.map(f =>
      `<div class="tg-lbl">${esc(f.label || f.key)}${f.required ? ' *' : ''}${f.help ? ` <span style="opacity:.6">— ${esc(f.help)}</span>` : ''}</div>
       <input class="tg-in" data-cfg="${escAttr(f.key)}" value="${escAttr(cfg[f.key] != null ? cfg[f.key] : (f.default || ''))}">` ).join('') : '';
    box.querySelector('#fChips').innerHTML = (ty.payload_keys || []).map(k => `<span class="tg-chip" data-chip="{{${esc(k)}}}">{{${esc(k)}}}</span>`).join('');
    box.querySelectorAll('[data-chip]').forEach(c => c.onclick = () => {
      const ta = box.querySelector('#fPrompt'); ta.value += c.dataset.chip; ta.focus();
    });
  };
  box.querySelector('#fType').onchange = () => { cfg = {}; renderCfg(); };
  renderCfg();
  box.querySelector('#fCancel').onclick = () => { box.innerHTML = ''; };
  box.querySelector('#fSave').onclick = async () => {
    const msg = box.querySelector('#fMsg'); msg.className = 'tg-msg';
    const conf = {};
    box.querySelectorAll('[data-cfg]').forEach(i => { const v = i.value.trim(); if (v) conf[i.getAttribute('data-cfg')] = v; });
    const tv = box.querySelector('#fTarget').value; const ti = tv.indexOf(':');
    const payload = {
      id: box.querySelector('#fId').value.trim().toLowerCase(),
      name: box.querySelector('#fName').value.trim(),
      type_id: box.querySelector('#fType').value,
      config: JSON.stringify(conf),
      target: ti >= 0 ? tv.slice(ti + 1) : tv,
      target_kind: ti >= 0 ? tv.slice(0, ti) : 'agent',
      prompt: box.querySelector('#fPrompt').value,
      enabled: true,
    };
    try {
      await fetchJSON('/api/triggers', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
      box.innerHTML = ''; await load(mainEl);
    } catch (e) { msg.className = 'tg-msg err'; msg.textContent = L.save_fail + ' ' + (e.message || e); }
  };
}
