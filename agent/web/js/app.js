// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Main app entrypoint. Audit pass — route via #hash, localStorage for last tab, dynamic import dengan Date.now() cache buster..

import { ts, loadWargaFromBrain } from './utils.js';
import { loadI18n, applyI18n, t } from './i18n.js';

// Boot the dictionary BEFORE anything renders. Sidebar/header use
// data-t / data-tooltip-key attributes that need t() values populated.
// On miss the loader silently uses the key itself, so worst-case the
// UI shows the key path instead of the localised string — never blanks.
await loadI18n('en');
applyI18n();
window.dispatchEvent(new Event('flowork:i18n-ready'));

// Single source of truth = sidebar buttons di index.html.
// Add tab baru = update ACTIVE_TABS + index.html nav button +
// tabs/<n>.js render() + i18n/<locale>/menu.json + tooltip.json.
const ACTIVE_TABS = new Set(['agents', 'groups', 'connections', 'schedule', 'triggers', 'apps', 'coder', 'codemap', 'cognitive', 'evolution', 'scanner', 'commits', 'document', 'settings']);

// Routing: URL hash wins, then localStorage, then default. Hash format is
// `#top/sub` (e.g. `#tasking/caps`). Sub-tab is read by segmentedTab via
// flowork:route. Putting routing state in the URL means reload, copy-paste,
// and back-button all do the obvious thing.
function parseHash() {
  const raw = (location.hash || '').replace(/^#\/?/, '');
  const parts = raw.split('/').filter(Boolean);
  return { top: parts[0] || '', sub: parts[1] || '' };
}
function pickInitialTab() {
  const fromHash = parseHash().top;
  if (fromHash && ACTIVE_TABS.has(fromHash)) return fromHash;
  const saved = localStorage.getItem('flowork_last_tab');
  if (saved && ACTIVE_TABS.has(saved)) return saved;
  return 'scanner'; // halaman utama: Threat Radar (scanner) — first thing after login
}
let currentTab = pickInitialTab();
// Mirror back into both storages so they're in sync from the first paint on.
localStorage.setItem('flowork_last_tab', currentTab);
if (parseHash().top !== currentTab) {
  // replaceState avoids an extra hashchange event firing during boot.
  history.replaceState(null, '', '#' + currentTab + (parseHash().sub ? '/' + parseHash().sub : ''));
}
// Exposed for segmentedTab + child tabs to call when they switch sub-tabs.
window.flowworkRoute = {
  setSub(subKey) {
    const { top } = parseHash();
    if (!top) return;
    history.replaceState(null, '', '#' + top + (subKey ? '/' + subKey : ''));
  },
  goto(topKey, subKey) {
    if (!ACTIVE_TABS.has(topKey)) return;
    location.hash = '#' + topKey + (subKey ? '/' + subKey : '');
  },
  current: () => parseHash(),
};

// ... (existing code remains intact in chunks below)

// Basic Layout Logic
const navToggle = document.getElementById('navToggle');
if (navToggle) {
  navToggle.addEventListener('click', () => {
    document.body.classList.toggle('nav-collapsed');
    localStorage.setItem('flowork_nav', document.body.classList.contains('nav-collapsed') ? '1' : '0');
  });
  if (localStorage.getItem('flowork_nav') === '1') {
    document.body.classList.add('nav-collapsed');
  }
}

// Theme Toggle Layout
const themeToggle = document.getElementById('themeToggle');
if (themeToggle) {
  themeToggle.addEventListener('click', () => {
    document.documentElement.classList.toggle('theme-light');
    localStorage.setItem('flowork_theme', document.documentElement.classList.contains('theme-light') ? 'light' : 'dark');
  });
  if (localStorage.getItem('flowork_theme') === 'light') {
    document.documentElement.classList.add('theme-light');
  }
}

// Notifications Request
if ('Notification' in window && Notification.permission === 'default') {
  Notification.requestPermission();
}

// Keyboard Shortcuts
window.addEventListener('keydown', (e) => {
  // Ctrl+B: Toggle Sidebar
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'b') {
    e.preventDefault();
    if (navToggle) navToggle.click();
  }
  // Alt+1 to Alt+9: Switch Tabs
  if (e.altKey && e.key >= '1' && e.key <= '9') {
    e.preventDefault();
    const btns = document.querySelectorAll('#nav button');
    const idx = parseInt(e.key, 10) - 1;
    if (btns[idx]) btns[idx].click();
  }
});

// Update clock periodically
setInterval(() => {
  const el = document.getElementById('ts');
  if (el) el.textContent = ts(new Date().toISOString());
}, 1000);

// Tab Router
async function loadTab(tabName) {
  const main = document.getElementById('main');
  main.innerHTML = `<div class="empty">${t('common.loading_label').replace('{label}', tabName)}</div>`;
  try {
    const cb = Date.now();
    const module = await import(`/tabs/${tabName}.js?t=${cb}`);
    if (module && typeof module.render === 'function') {
      await module.render(main);
    } else {
      main.innerHTML = `<div class="err">❌ ${t('common.module_no_render').replace('{name}', tabName)}</div>`;
    }
  } catch (err) {
    console.error(`tab load failed: ${tabName}`, err);
    main.innerHTML = `<div class="err">❌ ${t('common.load_failed_label').replace('{label}', tabName)} ${err.message}</div>`;
  }
}

function highlightNavFor(tabName) {
  document.querySelectorAll('#nav button').forEach((b) => b.classList.toggle('active', b.dataset.tab === tabName));
}

// Internal flag — set when WE are mutating location.hash, so the hashchange
// listener doesn't reload the tab we're already loading.
let _ignoreNextHashChange = false;

function writeHash(top, sub) {
  const target = '#' + top + (sub ? '/' + sub : '');
  if (location.hash === target) return;
  _ignoreNextHashChange = true;
  // Direct assignment (not replaceState) so the browser's URL bar visibly
  // updates and the value is what reload reads back.
  location.hash = target;
}

function navigateTo(tabName, { sub = null } = {}) {
  if (!ACTIVE_TABS.has(tabName)) return;
  currentTab = tabName;
  try { localStorage.setItem('flowork_last_tab', tabName); } catch {}
  const subPart = sub != null ? sub : parseHash().sub;
  writeHash(tabName, subPart);
  highlightNavFor(tabName);
  loadTab(tabName);
}

// Expose writeHash for child code (e.g. segmentedTab) — keep it on
// window.flowworkRoute alongside the existing helpers.
window.flowworkRoute.setSub = (subKey) => {
  const top = parseHash().top || currentTab;
  if (!top) return;
  writeHash(top, subKey);
};

// Nav buttons. Attached at module-eval time; module scripts are deferred so
// the DOM is already parsed at this point.
document.querySelectorAll('#nav button').forEach((btn) => {
  btn.onclick = () => {
    // Clearing the sub portion when the user picks a different top tab gives
    // them the hub's default segment; staying on the same tab keeps the sub.
    const isSameTab = btn.dataset.tab === currentTab;
    navigateTo(btn.dataset.tab, { sub: isSameTab ? undefined : '' });
  };
});

window.addEventListener('hashchange', () => {
  if (_ignoreNextHashChange) { _ignoreNextHashChange = false; return; }
  const top = parseHash().top;
  if (top && ACTIVE_TABS.has(top) && top !== currentTab) {
    navigateTo(top);
  }
});

// Synchronous boot — module scripts are deferred so the DOM exists. We don't
// need DOMContentLoaded; using it caused a confusing race where the listener
// sometimes didn't fire and the initial tab silently stayed at the default.
highlightNavFor(currentTab);
loadTab(currentTab);
// loadWargaFromBrain populates a dropdown elsewhere; failure is non-fatal.
loadWargaFromBrain().catch((e) => console.warn('flowork: brain warga load failed', e));
// initTooltip wires the global tooltip engine; run it once.
window.addEventListener('DOMContentLoaded', () => initTooltip(), { once: true });
if (document.readyState !== 'loading') initTooltip();

// ── Global Tooltip Engine ─────────────────────────────────────────────────
// Trigger: data-tooltip="Title|Deskripsi" atau data-tooltip="Deskripsi saja"
// Optional shortcut suffix: "Title|Desc||Ctrl+K"
function initTooltip() {
  const tt = document.createElement('div');
  tt.id = 'flowork-tooltip';
  document.body.appendChild(tt);

  let showTimer = null;
  let activeTarget = null;

  function showTip(target) {
    const raw = (target.dataset.tooltip || '').trim();
    if (!raw) return;
    const [mainPart, shortcut] = raw.split('||');
    const pipeIdx = mainPart.indexOf('|');
    const title = pipeIdx !== -1 ? mainPart.slice(0, pipeIdx) : null;
    const body = pipeIdx !== -1 ? mainPart.slice(pipeIdx + 1) : mainPart;
    // HTML escape — tooltip text bisa contain `<`, `>`, `"` yang harus
    // ditampilkan literal, bukan di-render sebagai HTML markup. Bug fix
    // 2026-04-27: tooltip Setting bocorin text karena double-escape mismatch.
    const escHTML = (s) => String(s == null ? '' : s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    tt.innerHTML = [
      title ? `<span class="ftt-title">${escHTML(title)}</span>` : '',
      `<span class="ftt-body">${escHTML(body)}</span>`,
      shortcut ? `<span class="ftt-shortcut">${escHTML(shortcut.trim())}</span>` : '',
    ].join('');

    // Position: prefer below, fallback above
    const rect = target.getBoundingClientRect();
    tt.classList.add('ftt-visible');
    const ttH = tt.offsetHeight || 60;
    const ttW = Math.min(300, tt.scrollWidth + 4);
    const spaceBelow = window.innerHeight - rect.bottom;
    const top = spaceBelow > ttH + 14 ? rect.bottom + 8 : rect.top - ttH - 8;
    const left = Math.min(rect.left, window.innerWidth - ttW - 10);
    tt.style.top = `${Math.max(8, top)}px`;
    tt.style.left = `${Math.max(8, left)}px`;
  }

  function hideTip() {
    clearTimeout(showTimer);
    showTimer = null;
    activeTarget = null;
    tt.classList.remove('ftt-visible');
  }

  document.addEventListener('mouseover', (e) => {
    const target = e.target.closest('[data-tooltip]');
    if (!target || target === activeTarget) return;
    clearTimeout(showTimer);
    activeTarget = target;
    showTimer = setTimeout(() => showTip(target), 220);
  });

  document.addEventListener('mouseout', (e) => {
    if (!e.target.closest('[data-tooltip]')) return;
    clearTimeout(showTimer);
    showTimer = null;
    activeTarget = null;
    tt.classList.remove('ftt-visible');
  });

  document.addEventListener('scroll', hideTip, true);
  document.addEventListener('click', hideTip, true);
}

