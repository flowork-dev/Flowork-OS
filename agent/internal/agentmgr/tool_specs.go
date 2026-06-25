// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-02
// Reason: Fase 0 — endpoint OpenAI function-schema (core ~13 + manual subs,
//   cap 25, BUKAN 106 = anti over-prompt). E2E verified (Mr.Flow file_write+read).
//
// MODIFIED 2026-06-20 (owner-approved, re-locked; header-lock, BUKAN hash-frozen):
//   coreExposedTools +graph_recall +instinct_recall +brain_search_shared = PIPA
//   genom. Tiap agent (termasuk hasil AI Studio) lahir nyolok ke instinct + graph
//   + otak-kolektif (referensi SHARED brain, bukan copy). 12→15 core (cap 50, aman).
//   Alasan: directive owner "agent baru kewarisan roadmap (instinct/graph/tools)".
//
// MODIFIED 2026-06-21 (owner-approved buka-lock, re-locked; header-lock): D15 —
//   +codemap_search ke primaryExtraTools + cap 50→51. Akar: codemap KE-INDEX (336
//   file) tapi GA ke-expose ke spec → tool_search bisa DISCOVER tapi GA bisa CALL
//   (spec di-fetch SEKALI per-turn = statik, ga ada meta-runner). Sekarang mr-flow
//   (primary) bisa query struktur kode-nya sendiri (semantic). Additive: ants
//   (core-only) ga kena; cap+1 = codemap masuk TANPA drop subscription. Host
//   rebuilt. Re-locked.
//
// MODIFIED 2026-06-25 (owner-approved, #2C tool-as-instinct): `const maxExposedTools`
//   → `func maxExposedToolsLimit()` ENV-tunable `FLOWORK_MAX_EXPOSED_TOOLS` (4..200).
//   SWITCH (Rule 7): dial token↔kapabilitas TANPA rebuild; revert instan (set balik 56).
//   DEFAULT TETAP 56 (ADDITIVE — auto-update ga maksa-ubah agent user lain; owner opt-in
//   per-mesin via flowork.local.env). AKAR: tool 56-schema = ~55% prompt (~8.76k tok).
//   Pasangan: injeksi insting-tool proaktif (`router/internal/router/instinctenrich.go`)
//   + room `instinct_tool` (capability-instinct WHEN→THEN tiap tool) → tool yg DI-DROP
//   dari expose TETAP ke-RECALL + ke-discover via `tool_search` (1-hop). Owner lokal set
//   16 (core-only): 56→16 tool, ukur ~10.7k→~2.0k byte schema (~8.7k tok hemat). NON-chattr
//   (file tuning, header-lock honor-system). Arsitektur: `lock/FLoworkInstincts.md` §0.5.
//
// MODIFIED 2026-06-25 (owner-approved, #2C deferred-tools PROTOTIPE — emulasi prompt-space ala
//   Claude Code). AKAR studi: `defer_loading`/`tool_reference` Claude Code itu API-native (server
//   Anthropic yg nyaring schema) → model LOKAL (llama.cpp OpenAI-compat) GA punya → wajib ditiru
//   di PROMPT. SWITCH ENV `FLOWORK_DEFER_TOOLS` (default OFF = byte-identik perilaku lama, additive).
//   ON: alwaysLoad (core + primaryVital) kirim schema PENUH; ekor (sub/sidecar) cuma diumumin
//   NAMA+hint di KATALOG yg disisipin ke deskripsi `tool_search` (channel yg main.go FROZEN tetap
//   forward). Model ambil param via `tool_lookup{name}` (di-core-kan SAAT defer) → panggil tool
//   (ToolRunHandler lookup REGISTRY PENUH → non-exposed TETAP callable). Reversible instan (unset
//   ENV, tanpa rebuild). Ga ada perubahan ke file FROZEN. Ukur before/after byte schema.
//
// tool_specs.go — Fase 0 (tool-calling loop): endpoint yang balikin tools yang
// di-EXPOSE ke LLM dalam format OpenAI function-schema. Host yang bangun schema
// (punya registry + subscription); WASM agent tinggal fetch + forward ke LLM.
//
// ANTI OVER-PROMPT (akar refactor 11×): yang di-expose CUMA core set (~13),
// BUKAN 106. Sisanya tetep di registry, dipanggil via `tool_search` on-demand.
// Fase 2: per-agent exposed selection (sekarang core + tool yang di-subscribe
// MANUAL via popup).

package agentmgr

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"flowork-gui/internal/httpx"
	"flowork-gui/internal/tools"
	"flowork-gui/internal/toolsidecar"
)

// coreExposedTools — SELALU di-expose ke LLM (cover kebutuhan umum). Kecil =
// prompt kecil. Sisanya via tool_search.
// Note: exec is NOT in the always-on core — an agent that needs a shell subscribes
// to `shell` (the hardened, semantics-classified exec tool, P1). The old `bash`
// (substring denylist) stays registered for back-compat but is opt-in only, so a
// capable agent gets the safer one and the ants (no subscriptions) get no shell.
var coreExposedTools = []string{
	"file_read", "file_write", "file_list", "grep", "glob",
	"webfetch", "brain_search", "memory_get", "memory_set",
	"telegram_send", "tool_search", "now",
	// PIPA roadmap (2026-06-20): tiap agent (termasuk hasil AI Studio) lahir
	// nyolok ke instinct + graph + otak-kolektif. graph_recall (state:read,
	// universal aman) = recall dari cognitive graph sendiri (twin/relasi);
	// instinct_recall + brain_search_shared (rpc:router:brain) = insting
	// coding/security + pengetahuan kolektif (859K) di SHARED brain. Di-REFERENSI
	// (bukan di-copy) → update shared sekali, semua agent lihat. Cap-denied =
	// graceful (tool balikin error, LLM lanjut).
	"graph_recall", "instinct_recall", "brain_search_shared",
	// 2026-06-23 (owner: "semua agent bisa bikin tools — PALING penting"): tool_create selalu
	// ke-expose ke SEMUA agent → tiap agent SADAR bisa bikin tool sendiri (self-evolving, roadmap §15).
	"tool_create",
}

// maxExposedTools caps how many tool schemas an agent offers its LLM at once. A
// capable agent (mr-flow holds ~40 first-class tools via subscriptions) needs the
// higher ceiling; ants stay tiny because they have no subscriptions (core set only),
// so raising the ceiling never bloats them.
// 2026-06-23 (owner "kejar hemat token"): 66→56. Ukur live: ekor cap mr-flow (posisi 57-66) = 10 tool
// app_flowalpha_* GRANULAR (bot_add/list/remove/step/toggle, compute/custom_indicator, compare_strategies,
// backtest_history, alert_remove) = trading-ops yg JARANG jadi langkah pertama → drop dari always-on, tetap
// ke-discover via tool_search (1-hop, recoverable). Hemat ~1170 tok/turn. Yg DIPERTAHANIN: SEMUA core +
// primaryExtra (browser/web_search/task_*/cognitive_*/system_power) + sidecar + app_flowalpha AI-level
// (ai_analyze/ai_team) + alert_add/check/list. Guaranteed-set (core+primaryExtra+sidecar) ga kesentuh —
// cap cuma ngegerus EKOR subscription. Naikin lagi kalau mr-flow sering kerja trading.
const maxExposedToolsDefault = 56 // sebelumnya 66 (50→51 codemap; 52 system_power; 53 web_search; 55 task_list/run; 64 browser_*; 66 cognitive_tensions/resolve)

// maxExposedToolsLimit — SWITCH (Rule 7): cap jumlah schema tool di-expose, dial
// lewat ENV FLOWORK_MAX_EXPOSED_TOOLS (4..200) TANPA rebuild. Default TETAP 56
// (additive — auto-update ga maksa-ubah agent user lain; owner opt-in per-mesin via
// flowork.local.env). #2C tool-as-instinct: turunin (mis. 16 = core-only) → tool
// sisanya TETAP ke-RECALL via insting-tool (room=instinct_tool, injeksi proaktif
// instinctenrich.go) + tool_search 1-hop. Naikin balik = revert instan tanpa rebuild.
func maxExposedToolsLimit() int {
	if v := strings.TrimSpace(os.Getenv("FLOWORK_MAX_EXPOSED_TOOLS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 4 && n <= 200 {
			return n
		}
	}
	return maxExposedToolsDefault
}

// #2C deferred-tools + all-tools: MEKANISME-nya DI-EKSTRAK ke `tool_specs_defer.go` (FROZEN).
// Switch: ENV FLOWORK_DEFER_TOOLS/EXPOSE_ALL_TOOLS atau RegisterDeferPolicy (per-agent, no unfreeze).

// primaryExtraTools — surface-vocabulary tools exposed ONLY to the primary
// orchestrator (mr-flow), not to ants. These cover shell/task-lifecycle/schedule/
// structured-output/orchestration that a coordinator needs. Kept off the ants'
// core set so ant prompts stay tiny (the over-prompt guard that drove the refactor).
var primaryExtraTools = []string{
	"PowerShell", "TaskCreate", "TaskUpdate", "TaskStop", "TaskOutput",
	"ScheduleWakeup", "Monitor", "SendUserFile", "StructuredOutput", "Workflow",
	// 2026-06-22: system_power — operator-essential (owner pakai "matiin pc malam").
	// Di primaryExtra (BUKAN subscription) biar GARANSI ke-expose (mr-flow 182 subs >
	// cap → subscription ke-drop). Cap-gated exec:power (cuma mr-flow punya) → primary
	// lain yg ga punya cap = graceful denial. ARM switch (FLOWORK_POWER_ARMED) tetap jaga.
	"system_power",
	// D15 (2026-06-21): codemap_search — primary bisa query struktur kode-nya sendiri
	// (semantic, 336 file ke-index). Ditambah SEBELUM subscriptions → pasti masuk
	// (cap dinaikin ke 51 biar ga nyenggol subscription flowalpha). Coordinator-only:
	// ants (core set) ga ikut, prompt mereka tetap kecil (anti over-prompt utuh).
	"codemap_search",
	// 2026-06-23: web_search — LIVE internet (Google News/trending/berita real-time). mr-flow
	// defaultnya NYASAR ke brain_search_shared (data internal CVE/threat-intel) pas disuruh
	// "cari berita trending" → web_search di primaryExtra biar GARANSI ke-expose (ga ke-drop
	// cap kayak subscription). webfetch udah di coreExposedTools (baca hasil). cap=net:fetch:*.
	"web_search",
	// 2026-06-23 (owner-directive "stabilkan group biar mr-flow SADAR ada group + tau tugas
	// ke group mana"): task_list + task_run = TOOL ROUTER orchestrator. AKAR nyasar: persona
	// mr-flow nyuruh route via task_list, TAPI dua tool ini GA ke-expose (cuma via tool_search
	// = 1 hop ekstra yg sering di-skip LLM) → mr-flow jatuh ke brain_search_shared/jawab langsung
	// = NYASAR (kasus screenshot owner). Ditaruh di primaryExtra (BUKAN subscription) biar GARANSI
	// ke-expose — mr-flow 182 subs > cap → subscription task_list/task_run KE-DROP (persis web_search).
	// task_list = liat daftar Category/Group team; task_run = delegasi tugas ke crew→synth. Primary-
	// only: ants ga ikut (prompt tetap kecil). Cap 53→55 = masuk TANPA drop subscription.
	"task_list", "task_run",
	// 2026-06-23 (owner: "buka manifest, akses penuh"): SEMUA 9 tool browser asli (chromium
	// via go-rod). mr-flow bisa kontrol browser PENUH — buka situs JS/login/berat yg webfetch
	// ga bisa, baca localhost, dst. cap=browser:control (ditambah ke manifest mr-flow → di-grant
	// boot). Persona arahin: web_search/webfetch DULU, browser buat yg berat (chromium mahal).
	// Resource dijaga: browser_close + idle-reaper 30mnt (lihat lock/browser.md).
	"browser_navigate", "browser_snapshot", "browser_click", "browser_type",
	"browser_upload", "browser_screenshot", "browser_set_cookies", "browser_eval", "browser_close",
	// 2026-06-23 (owner: "mr-flow tahu Open contradictions → minta klarifikasi"): cognitive_tensions
	// (lihat kontradiksi data nunggu keputusan owner) + cognitive_resolve (apply keputusan owner →
	// graph makin akurat). GARANSI ke-expose biar mr-flow bisa proaktif klarifikasi. cap state:read/write.
	"cognitive_tensions", "cognitive_resolve",
}

// ToolSpecsHandler — GET /api/agents/tools/specs?id=<agent>
// Return {tools: [<openai function schema>...], count}. Loopback-only (dipanggil
// WASM agent sendiri).
func ToolSpecsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		id = defaultAgentID
	}

	isPrimary := IsPrimaryAgent(id)
	// kebijakan defer/all-tools dari resolveDeferPolicy (tool_specs_defer.go, FROZEN): ENV
	// scoped-primary by-default, ATAU hook RegisterDeferPolicy (per-agent GUI/kv, no unfreeze).
	deferOn, exposeAll := resolveDeferPolicy(id, isPrimary)
	limit := maxExposedToolsLimit() // SWITCH ENV (Rule 7) — baca sekali per-request.
	if deferOn {
		// #2C: schema cuma buat alwaysLoad (~17) → cap-56 ga relevan lagi. Naikin batas
		// biar SEMUA tool ke-subscribe ke-ANNOUNCE namanya (bukan kepotong cap = buta).
		limit = deferAnnounceMax
	}
	exposed := map[string]bool{}
	ordered := []string{}
	add := func(n string) {
		if exposed[n] || len(ordered) >= limit {
			return
		}
		// Tier gate: tool primary-only (brain 5jt shared) ga di-expose ke
		// extension — brain-nya folder sendiri (brain_search lokal).
		if IsPrimaryOnlyTool(n) && !isPrimary {
			return
		}
		if _, ok := tools.Lookup(n); ok {
			exposed[n] = true
			ordered = append(ordered, n)
		}
	}
	// 1. core set (selalu).
	for _, n := range coreExposedTools {
		add(n)
	}
	// 1b. (owner 2026-06-23 "GUI = KEBENARAN UTAMA, hardcode HARAM") — primaryExtra HARDCODE-force DIBUANG.
	// Tool primary (browser/task/sleep-wait/brain-extra) sekarang murni dari SUBSCRIPTION GUI (di-seed
	// default sekali via SeedPrimaryDefaults pas provision; owner curate/uncheck per-agent di Tools-catalog).
	// Exposure = core (engine wajib) + sidecar + subscription. Cap = window-safety (16K), bukan kontrol fitur.
	// 1c. SIDECAR TOOLS — SHARED ke SEMUA agent + PRIVAT cuma ke pembuatnya (owner 2026-06-23:
	// self-evolving — tool buatan-agent lahir privat sampai lolos Dewan → shared). NamesForAgent
	// nyaring: shared (semua) + private-owned-by-id. Ditaruh SEBELUM subscription = prioritas.
	for _, n := range toolsidecar.NamesForAgent(id) {
		add(n)
	}
	// 2. tool yang di-subscribe MANUAL (owner pilih di popup) — di luar default seed.
	//    ATAU (arah owner "buang subscription"): mode all-tools (defer+primary+switch) →
	//    expose SEMUA tool registry, bukan cuma subscription.
	if deferOn && exposeAll {
		// #2C + all-tools: buang gating subscription → mr-flow raih tool APAPUN (nama murah
		// via katalog). CAP-GATE pas run yg jaga; doktrin/insting jadi kemudi. `add()` tetep
		// hormati IsPrimaryOnlyTool / IsPrivate / cap-limit (deferAnnounceMax).
		for _, s := range tools.ListSummaries() {
			add(s.Name)
		}
	} else if store, err := openAgentStore(id); err == nil {
		if subs, serr := store.ListSubscriptions(); serr == nil {
			for _, s := range subs {
				if s.Source != "" && !strings.EqualFold(s.Source, "default") {
					add(s.ToolName)
				}
			}
		}
		store.Close()
	}

	// #2C: kalau defer ON, cuma alwaysLoad (core + tool_lookup + primaryVital utk primary)
	// yang dikirim schema PENUH; sisanya diumumin NAMA+hint di katalog.
	always := map[string]bool{}
	if deferOn {
		for _, n := range coreExposedTools {
			always[n] = true
		}
		always[deferFetchTool] = true // primitif ambil-schema
		if isPrimary {
			for _, n := range primaryVitalTools {
				always[n] = true
			}
		}
	}

	specs := make([]map[string]any, 0, len(ordered))
	deferredLines := make([]string, 0)
	emitted := map[string]bool{}
	for _, n := range ordered {
		t, ok := tools.Lookup(n)
		if !ok {
			continue
		}
		if !deferOn || always[n] || isActiveDeferred(id, n) {
			// alwaysLoad ATAU udah di-lookup model (active) → schema penuh (callable).
			specs = append(specs, toOpenAIToolSchema(t))
			emitted[n] = true
		} else {
			deferredLines = append(deferredLines, deferCatalogLine(t))
		}
	}

	if deferOn {
		// pastikan primitif tool_lookup ke-expose (mungkin gak ke-subscribe).
		if !emitted[deferFetchTool] {
			if t, ok := tools.Lookup(deferFetchTool); ok {
				specs = append(specs, toOpenAIToolSchema(t))
			}
		}
		if len(deferredLines) > 0 {
			injectDeferredCatalog(specs, deferredLines)
		}
	}

	resp := map[string]any{"tools": specs, "count": len(specs)}
	if deferOn {
		resp["deferred_count"] = len(deferredLines)
	}
	httpx.WriteJSON(w, resp)
}

// toOpenAIToolSchema — konversi tools.Schema → OpenAI function-calling schema.
func toOpenAIToolSchema(t tools.Tool) map[string]any {
	sc := t.Schema()
	props := map[string]any{}
	required := []string{}
	for _, p := range sc.Params {
		props[p.Name] = map[string]any{
			"type":        jsonSchemaType(p.Type),
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        tools.DisplayName(t.Name()),
			"description": sc.Description,
			"parameters": map[string]any{
				"type":       "object",
				"properties": props,
				"required":   required,
			},
		},
	}
}

func jsonSchemaType(pt tools.ParamType) string {
	switch pt {
	case tools.ParamInt:
		return "integer"
	case tools.ParamFloat:
		return "number"
	case tools.ParamBool:
		return "boolean"
	case tools.ParamArray:
		return "array"
	case tools.ParamObject:
		return "object"
	default:
		return "string"
	}
}
