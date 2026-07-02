// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-31
// Reason: Auth HTTP handlers + middleware. Audit pass — MaxBytesReader body
//
//	cap, status code eksplisit, whitelist exact-path (anti future-handler
//	bypass) + asset prefix, /api/* → 401 JSON, HTML → redirect. E2E verified.
//
// 2026-06-07 pre-freeze hardening: drive-by-localhost defense — cross-site
// browser request (Sec-Fetch-Site) dicabut dari bypass loopback, biar web jahat
// yang dibuka owner ga bisa memicu exec/install via fetch ke 127.0.0.1.
//
// 2026-06-15 (owner-approved autonomous sprint): added "/api/architect/build" + the
// "/api/chat/sessions*" + "/api/chat/send" family to the loopback-gated allowlist —
// same trust model as the existing /api/coder/* + /api/chat cases (owner-local,
// loopback-only, behind isLocalRequest + the cross-site-browser cut). No new attack
// surface; re-locked. (Pending owner ratification on wake.)
//
// handlers.go — HTTP endpoint + middleware untuk floworkauth.
//
// Endpoint (shape dicocokkan dengan web/login.html + web/register.html):
//
//	POST /api/auth/register        {name,password}      → 201 {ok,name,role}
//	POST /api/auth/login           {password}           → 200 {name,role} + cookie
//	GET  /api/auth/me                                   → {name,role,authenticated} | {setup_required}
//	POST /api/auth/logout                               → {ok}
//	POST /api/auth/change-password {old,new}            → {ok}   (butuh sesi)
package floworkauth

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// writeJSON — JSON response dengan status code eksplisit.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// RegisterHandler — POST /api/auth/register. Single-owner: register pertama
// jadi owner; register berikutnya → 409.
func (m *Manager) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST only"})
		return
	}
	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	if err := m.Register(body.Name, body.Password); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "sudah terdaftar") {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"ok":   true,
		"name": m.OwnerName(),
		"role": RoleOwner,
	})
}

// LoginHandler — POST /api/auth/login.
func (m *Manager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST only"})
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	token, name, err := m.Login(body.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if err == ErrNoOwner {
			status = http.StatusNotFound // frontend bisa arahkan ke /register
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	m.SetCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "role": RoleOwner})
}

// MeHandler — GET /api/auth/me.
func (m *Manager) MeHandler(w http.ResponseWriter, r *http.Request) {
	if !m.IsSetup() {
		writeJSON(w, http.StatusOK, map[string]any{"setup_required": true})
		return
	}
	name, ok := m.SessionFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not logged in"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":          name,
		"role":          RoleOwner,
		"authenticated": true,
	})
}

// LogoutHandler — POST /api/auth/logout. Idempotent.
func (m *Manager) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		m.Logout(c.Value)
	}
	m.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ChangePasswordHandler — POST /api/auth/change-password {old,new}. Butuh sesi.
func (m *Manager) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST only"})
		return
	}
	if _, ok := m.SessionFromRequest(r); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not logged in"})
		return
	}
	var body struct {
		Old string `json:"old"`
		New string `json:"new"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	if err := m.ChangePassword(body.Old, body.New); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Middleware — gate semua route kecuali whitelist. Pasang PALING LUAR.
//
//   - Cookie valid → lanjut.
//   - Cookie invalid + /api/* → 401 JSON.
//   - Cookie invalid + HTML  → 302 ke /register (kalau belum setup) atau
//     /login?next=<path>.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r) {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := m.SessionFromRequest(r); ok {
			next.ServeHTTP(w, r)
			return
		}
		// Not authenticated.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not logged in"})
			return
		}
		dest := "/login?next=" + r.URL.RequestURI()
		if !m.IsSetup() {
			dest = "/register"
		}
		http.Redirect(w, r, dest, http.StatusFound)
	})
}

// isPublicPath — request yang lewat tanpa cek sesi.
//
// Exact: halaman login/register + endpoint auth + liveness probe.
// Prefix: asset statik yang dibutuhkan halaman login/register (i18n, js, css,
// vendor). /tabs/ TIDAK public — cuma kebuka setelah login.
// Loopback-only: endpoint internal yang dipanggil WASM agent ke API-nya
// sendiri via hostNetFetch (fetchHistory + fetchSelfPrompt). Server bind
// 127.0.0.1, jadi loopback-only aman dari attacker remote.
func isPublicPath(r *http.Request) bool {
	path := r.URL.Path
	switch path {
	case "/login", "/login.html", "/register", "/register.html", "/favicon.ico":
		return true
	case "/api/auth/login", "/api/auth/register", "/api/auth/logout", "/api/auth/me":
		return true
	case "/api/system/health":
		return true
	}
	for _, p := range []string{"/js/", "/css/", "/i18n/", "/vendor/"} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	// Generic inbound webhook (§8.H): an EXTERNAL caller POSTs here. The path is
	// public, but WebhookHandler refuses it unless the per-module "webhook_secret"
	// matches — a module opts in, so this is never an open trigger.
	if r.Method == http.MethodPost && (strings.HasPrefix(path, "/api/kernel/webhook/") || strings.HasPrefix(path, "/api/triggers/hook/")) {
		return true
	}
	// DRIVE-BY-LOCALHOST DEFENSE (pre-freeze hardening 2026-06-07): semua di BAWAH
	// ini = endpoint sensitif loopback-gated (exec/install/coder/kernel-call) yang
	// lewat TANPA sesi asal request dari 127.0.0.1. Tapi web jahat yang dibuka owner
	// JUGA datang dari 127.0.0.1 → bisa fetch() endpoint ini & memicu exec tanpa
	// sesi. Sec-Fetch-Site di-set browser (forbidden header, JS halaman ga bisa
	// palsu); caller non-browser (agent WASM bawa loopback-secret / curl headless)
	// ga ngirimnya → ga kena. Cross-site browser → cabut bypass loopback.
	if isCrossSiteBrowser(r) {
		return false
	}
	// Agent self-call (loopback only): daemon WASM fetch konteks/self-prompt/
	// tool-specs buat di-inject ke LLM + EXEC tool via tools/run. Tanpa ini,
	// agent ga bisa inget history / pake tools (kena 401). Server bind 127.0.0.1
	// jadi loopback-only aman dari remote.
	switch path {
	case "/api/agents/interactions", "/api/agents/self-prompt/render", "/api/agents/tools/specs":
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/agents/tools/run":
		// POST — eksekusi tool. Tetep aman: SandboxRunV3 enforce capability +
		// rate + approval di dalam handler-nya.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/kernel/call":
		// POST — the "papan kosong" microkernel loket: module call(cap,args).
		// Loopback-only; caller id is kernel-stamped via the loopback secret,
		// and the dispatcher enforces per-module capability grants.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/kernel/rpc":
		// POST — invoke a loaded module's exported function. Loopback-only
		// (owner-local kernel control / module-to-module), consistent with
		// tools/run; server binds 127.0.0.1 so it is safe from remote.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/notify":
		// POST — owner-notify hub: a local sandboxed app/scanner pushes an owner
		// Telegram alert. Loopback-only (server binds 127.0.0.1; non-browser caller),
		// reuses notifyOwnerTelegram. No token leaves the kernel.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/taskflow/run", "/api/taskflow/category", "/api/taskflow/category/delete":
		// POST/GET — FASE 4/5 Category Task trigger + CRUD. Loopback-only
		// (Mr.Flow/scheduler/owner-local). Server bind 127.0.0.1 → aman remote.
		return isLocalRequest(r)
	case "/api/taskflow/categories", "/api/taskflow/runs", "/api/taskflow/run-detail",
		"/api/taskflow/schedules":
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/taskflow/schedule", "/api/taskflow/schedule/delete":
		return isLocalRequest(r)
	case "/api/plugins/install", "/api/plugins/uninstall", "/api/plugins/verify":
		// Install/uninstall/verify task pack — loopback-only (owner-local / CLI /
		// Coder). Server bind 127.0.0.1 → aman remote. Caps-consent = Phase 4.
		// verify = VERIFIER dry-run gate (roadmap 2.3), no side-effect.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/plugins/export":
		// Export kategori jadi .fwpack (download) — loopback-only owner-local.
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/coder/generate", "/api/coder/approve", "/api/coder/reject":
		// CODER (AI Utama 2.2) generate/approve/reject agent baru — loopback-only
		// (owner-local / CLI). Gerbang: generate→verify→OWNER-approve→install.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/coder/pending":
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/architect/build":
		// ARCHITECT (group/team creator): bikin TIM utuh dari 1 prompt → design (Opus)
		// → generate+install tiap specialist (mesin coder yg sama) → bikin group.
		// SAME trust model as /api/coder/* — owner-local, loopback-only POST, gated by
		// isLocalRequest + the cross-site-browser cut above. No new attack surface.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/reaper/reap":
		// REAPER (AI Utama 2.4) apoptosis reap — loopback-only owner-gated.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/reaper/candidates":
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/studio/deathletters":
		// AI STUDIO siklus-hidup: surat kematian (read-only) — loopback owner-local,
		// same trust model as /api/reaper/candidates + /api/coder/pending.
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/chat":
		// Channel HTTP/CLI ke mr-flow (roadmap Channels) — loopback-only owner-local.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/chat/sessions", "/api/chat/sessions/rename", "/api/chat/sessions/delete",
		"/api/chat/sessions/meta", "/api/chat/sessions/messages", "/api/chat/sessions/prune", "/api/chat/send":
		// CHATGPT-STYLE CHAT TAB (Group): persistent sessions + full-context chat with a
		// group or the Architect. Owner-local, loopback-only (server binds 127.0.0.1;
		// behind isLocalRequest + the cross-site-browser cut above). Same trust model as
		// /api/chat + /api/architect/build.
		return isLocalRequest(r)
	case "/api/tools/install", "/api/tools/uninstall":
		// TOOL-PACK plug-and-play (multi-KIND) install/uninstall — loopback owner-local.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/tools/installed":
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/slash/install", "/api/slash/uninstall":
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/slash/installed":
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/scanner/allowlist", "/api/scanner/allowlist/delete", "/api/scanner/allowlist/check":
		// Allowlist scanner — owner-editable (GET/POST) loopback. Agent ga punya
		// akses (gerbang scope/exec di tangan owner).
		return isLocalRequest(r)
	case "/api/scanner/run":
		// GATED-EXEC scan tool (P2) — owner-only loopback. Blocklist+allowlist+no-shell.
		// Agent GA punya cap akses ini (gerbang exec di tangan owner).
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/scanner/runs", "/api/scanner/findings":
		// Audit history + finding terstruktur — owner-only loopback (laporan).
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/scanner/findings/verify":
		// Owner konfirmasi finding (reproducible_ok) — owner-only loopback POST.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/scanner/findings/triage":
		// RAG triage 5jt drawer (P1.3) — owner-only loopback GET.
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/scanner/findings/push":
		// Sync finding → tracker brain (immune_system/pentest_karma) — owner-only POST.
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/scanner/trackers":
		// Dashboard laporan tracker (read-only proxy) — owner-only loopback GET.
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/scanner/registry", "/api/scanner/packs/installed":
		// Katalog arsenal + daftar scanner-pack keinstall — owner-only loopback GET.
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/scanner/registry/toggle":
		// Install/uninstall pack nuclei — owner-only loopback POST (gerbang di tangan owner).
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/scanner/checks/add", "/api/scanner/checks/delete", "/api/scanner/distill", "/api/scanner/distill/corpus", "/api/scanner/bodyscan", "/api/scanner/efficacy", "/api/scanner/packs/install", "/api/scanner/packs/uninstall":
		// Ingest/hapus/GENERATE check privat (distill 5jt / komunitas) — owner-only
		// loopback POST. Gerbang nuclei -validate; nuclei jalan TANPA -code (inert).
		return r.Method == http.MethodPost && isLocalRequest(r)
	case "/api/mcp/config":
		return r.Method == http.MethodGet && isLocalRequest(r)
	case "/api/agents/skills", "/api/agents/skills/curate":
		// FASE 8 Curator — list/grade/curate skill per-agent (owner-local).
		return isLocalRequest(r)
	}
	// PAPAN COLOKAN (Rule #7, 2026-07-02): endpoint loopback baru dicolok dari file
	// NON-FROZEN via RegisterLoopbackPublic (allow_seam.go) — file beku ini GA PERLU
	// dibuka lagi. Seam MAKSA loopback + cross-site-cut terpusat: ext ga akan bisa
	// buka celah remote walau kodenya salah.
	return loopbackAllowExt(path, r)
}

// isCrossSiteBrowser — true kalau request kemungkinan besar dipicu DARI origin
// lain di dalam BROWSER (drive-by). Sec-Fetch-Site di-set browser & ga bisa
// di-override JS halaman (forbidden header):
//   same-origin / none  = GUI sendiri / navigasi langsung → AMAN
//   same-site / cross-site = origin lain yang mulai → BAHAYA
// Fallback browser lama: header Origin ada & host-nya beda dari Host request.
// Caller non-browser (agent WASM / curl / script) ga kirim Sec-Fetch maupun
// Origin → false → tetap boleh (mereka bukan vektor drive-by).
func isCrossSiteBrowser(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-site", "cross-site":
		return true
	}
	if o := r.Header.Get("Origin"); o != "" {
		if u, err := url.Parse(o); err == nil && u.Host != "" && !strings.EqualFold(u.Host, r.Host) {
			return true
		}
	}
	return false
}

// isLocalRequest — true kalau request datang dari loopback (127.0.0.1/::1).
// RemoteAddr = TCP peer level, ngga bisa di-spoof client. JANGAN trust
// X-Forwarded-For di sini.
func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
