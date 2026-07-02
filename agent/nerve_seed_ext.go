// 📄 Dok: FLowork_os/lock/peta-saraf.md
//
// nerve_seed_ext.go — SIBLING (BISA DIHAPUS) yg nyolok katalog saraf ke papan RegisterNerve.
// Data DI-GENERATE dari lock/peta-saraf.md (SATU sumber kebenaran — anti-rot). Hapus file ini →
// papan kosong → inti TETEP jalan (default aman, POLA-A). Update saraf: edit peta-saraf.md lalu
// regen tabel di sini. Plus mount GET /api/evolve/nerves (feature seam) biar AI/GUI bisa tanya
// "saklar apa yang gue punya?". Format baris: NAME|DEFAULT|DESC (switch) · NAME|DESC (registry).
package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

const nerveSwitchTable = `
FLOWORK_AGENT_CONFIG|store|JSON config agent dari GUI
FLOWORK_AGENT_DB|/workspace/state.db|Path state.db agent
FLOWORK_AGENT_ID|manifest|ID agent (inject host)
FLOWORK_AGENTS_DIR|~/.flowork/agents|Folder agents
FLOWORK_AGENT_WORKSPACE|/workspace|Path workspace agent
FLOWORK_BASE|127.0.0.1:1987|Base URL flowork-gui (CLI)
FLOWORK_BINARY_VECTOR|auto|Binary-vector recall (jutaan drawer)
FLOWORK_BINARY_VECTOR_MIN|1000000|Drawer minimum auto-aktif
FLOWORK_BRAIN_EXTERNAL_SCOPE|off|Caller eksternal ga dapat insting-tool
FLOWORK_BRAIN_GGUF|auto|Path model GGUF brain
FLOWORK_BRAIN_VINDEX|auto|Path vector index brain
FLOWORK_BROWSER_FLAGS|—|Flag browser tambahan
FLOWORK_BROWSER_HEADLESS|on|Browser headless
FLOWORK_BROWSER_IDLE_MIN|15|Idle timeout browser (menit)
FLOWORK_BROWSER_URL|—|URL Chrome remote-debug
FLOWORK_BUSY_ALERT|on|Reflex beban-tinggi (notif owner)
FLOWORK_BUSY_PCT|90|Threshold load CPU (%) busy alert
FLOWORK_CACHE_REUSE|0|KV cache-reuse prefix statik
FLOWORK_CGM_AUTODIGEST|off|Deep digest percakapan ke graph
FLOWORK_CGM_CODEMAP|on|Proyeksi codemap ke graph agent
FLOWORK_CGM_DEADLETTER|on|Proyeksi task gagal ke graph
FLOWORK_CGM_EDGE_LIMIT|6000|Batas edge graph sebelum prune
FLOWORK_CGM_NODE_LIMIT|3000|Batas node graph sebelum prune
FLOWORK_CGM_ORPHAN_BACKFILL|on|Link node orphan ke hub brain
FLOWORK_CHROME_BIN|auto|Path binary Chrome
FLOWORK_CLOAK_DECOYS|—|Daftar tool decoy
FLOWORK_CLOAK_SUFFIX|_cc|Suffix nama tool saat cloaking
FLOWORK_CLOAK_VERSION|2.1.92|Versi klien disamarkan
FLOWORK_CODEMAP_AUTOENRICH_MIN|30|Interval auto-enrich (menit)
FLOWORK_CODEMAP_AUTOENRICH|on|Auto-enrich codemap tiap interval
FLOWORK_CODEMAP_CANONICAL_AGENT|—|Agent referensi tool codemap_files
FLOWORK_CODEMAP_ROOT|—|Root path codemap scanning
FLOWORK_CODER_MODEL|—|Model coder/codegen
FLOWORK_CODESCAN_ROOT|PROJECT_ROOT|Root codemap scanning
FLOWORK_CONNECT_CONFIG|—|Config flowork-connect CLI
FLOWORK_CPU_MOE|off|CPU mixture-of-experts
FLOWORK_CTX|—|Context window llama-cpp
FLOWORK_DATA_DIR|~/.flowork|Data directory portable
FLOWORK_DEADAIR_MIN|60|Ambang diem deadair (menit)
FLOWORK_DEADAIR|on|Deteksi anomali semua-beku
FLOWORK_DEFER_TOOLS|off|Skema tool on-demand (hemat prompt)
FLOWORK_DREAMGRAPH_AUTOSYNC|on|Auto-populate Knowledge Graph router
FLOWORK_DREAMGRAPH_INSTINCTS|on|Proyeksi instinct ke DreamGraph
FLOWORK_DREAMGRAPH_KNOWLEDGE|on|Proyeksi korpus knowledge jadi hub
FLOWORK_DREAMGRAPH_MESH|on|Proyeksi pengetahuan mesh ke graph
FLOWORK_DREAMGRAPH_SYNC_MIN|5|Interval refresh DreamGraph (menit)
FLOWORK_DYNAMIC_TOOLS_MINSCORE|0.30|Cosine min tool dianggap relevan
FLOWORK_DYNAMIC_TOOLS|off|Intent-gated tools (prune schema)
FLOWORK_DYNAMIC_TOOLS_TOPK|12|Max tool relevan dikirim
FLOWORK_EDITION|public|Edisi: dev (evolusi penuh) / public
FLOWORK_EMBED_MODEL|bge-m3|Model embedding (no-hardcode bge-m3)
FLOWORK_ENGINE_DIR|auto|Folder engine
FLOWORK_EXPOSE_ALL_TOOLS|off|Semua agent akses semua tool
FLOWORK_FANOUT_BUDGET|—|Budget paralel task fanout
FLOWORK_FRESH_MEMTYPES|—|Override daftar mem_type default
FLOWORK_GITHUB_TOKEN|—|GitHub token API
FLOWORK_GROUP_SLASH|off|Group slash commands
FLOWORK_GUARDIAN_AUTO|on|Guardian arm otomatis saat boot
FLOWORK_GUARDIAN_INTERVAL_SEC|—|Interval guardian sentinel (detik)
FLOWORK_HOME|~|Home directory guardian vault
FLOWORK_INSTINCT_INJECT_MAX|3|Cap jumlah insting disuntik
FLOWORK_INSTINCT_INJECT|on|Suntik insting relevan ke prompt
FLOWORK_INSTINCT_SCOPED|off|Agent cuma dapat insting domain-nya
FLOWORK_INSTINCT_SCOPE_MAP|—|Peta domain per-agent
FLOWORK_INSTINCT_SEMANTIC|on|Pilih insting by-makna (vektor) vs overlap
FLOWORK_INTEGRITY_GATE|on|Tolak belajar mesh kalau frozen-core berubah
FLOWORK_JOURNAL|on|Jurnal pengalaman
FLOWORK_KERNEL_MANIFEST|auto|Path manifest kernel (integrity gate)
FLOWORK_KV_TYPE|—|KV cache type llama-cpp
FLOWORK_LEGACY_DREAM|off|Legacy dream cycle (deprecated)
FLOWORK_LLAMA_BIN|auto|Path binary llama-cpp-server
FLOWORK_LLM_IDLE_DEBUG|off|Debug log idle-sleep
FLOWORK_LLM_IDLE_SLEEP|on|Idle-sleep LLM lokal
FLOWORK_LLM_IDLE_SLEEP_SEC|60|Detik sebelum idle-sleep (min 10)
FLOWORK_LLM_MODEL|—|Model LLM default
FLOWORK_LOCALAI_AUTOSTART|off|Auto-start LocalAI saat boot
FLOWORK_LOCAL_EMBED_MODEL|—|Model local embedding
FLOWORK_LOCAL_EMBED_URL|—|URL local embedding service
FLOWORK_LOGIN_FAIL_WINDOW|60|Window hitung gagal login (menit)
FLOWORK_LOGIN_MAX_FAILS|5|Gagal login sebelum lockout
FLOWORK_LOKET_CALL_TIMEOUT|240|Cap timeout bus.request loket (detik)
FLOWORK_LOOPBACK_SECRET|random|Secret loopback intra-process
FLOWORK_MANDOR|off|Agent Mandor (supervisor idle)
FLOWORK_MAX_CONCURRENT_TASKS|2|Batas background-task barengan (boot)
FLOWORK_MAX_EXPOSED_TOOLS|∞|Batas tool per-agent
FLOWORK_MCP_AGENT|—|Agent untuk MCP server
FLOWORK_MESH_APPROVE|manual|Mode approve masuk: manual / auto
FLOWORK_MESH_SHARE|on|Share & terima pengetahuan mesh
FLOWORK_NGL|—|GPU layers llama-cpp
FLOWORK_ORCHESTRATOR|mr-flow|Agent orkestrator default
FLOWORK_OS|auto|Penanda OS host
FLOWORK_PARALLEL_SLOTS|0|Parallel slots server (-np N)
FLOWORK_POWER_ARMED|off|Enable power control (shutdown/reboot)
FLOWORK_POWER_REQUIRE_APPROVAL|on|Wajib approval sebelum power action
FLOWORK_PRIVILEGED_AGENTS|—|Daftar agent ID privileged
FLOWORK_PROJECT_ROOT|cwd|Root project
FLOWORK_PUBLIC_URL|—|Public URL sistem
FLOWORK_REAP_ERRRATE|0.40|Error-rate ambang vonis Reaper (0..1)
FLOWORK_REAP_MIN_SAMPLES|5|Min run selesai sebelum Reaper vonis
FLOWORK_REASONING|—|Mode reasoning llama-cpp
FLOWORK_RESILIENCE_OFF|off|Matiin agent resilience
FLOWORK_RESOLVE_MINSCORE|0.86|Ambang entity-resolution dedup graph
FLOWORK_RL_MAX_RETRY|6|Retry saat 429 sebelum fallback
FLOWORK_ROUTER_HTTP_TIMEOUT|300|Timeout request upstream LLM (boot)
FLOWORK_ROUTER_RETRY|5|Jumlah attempt maksimal retry router transient (angka; 1=tanpa retry)
FLOWORK_ROUTER_TIMEOUT|30|Timeout HTTP router client (detik)
FLOWORK_ROUTER_URL|settings|Router URL routerclient
FLOWORK_SCANNER_AUTOSCAN|on|Auto-scan kode saat berubah
FLOWORK_SCHED_EXEC_TIMEOUT_SEC|90|Timeout eksekusi per-job scheduler
FLOWORK_SCHEME|http|Scheme HTTP (https/http)
FLOWORK_SEARCH_MINSCORE|0.45|Lantai relevansi cosine search
FLOWORK_SELF_HANDLE_PHRASES|—|Custom phrase self-response
FLOWORK_SELF_URL|—|Self-URL MCP & TUI
FLOWORK_SHARED_WORKSPACE|/shared|Path workspace bersama
FLOWORK_SIDECAR|~/.flowork|Data home sidecar
FLOWORK_SKILL_AUTOSYNC_MIN|30|Interval auto-sync skill (menit)
FLOWORK_SKILL_AUTOSYNC|on|Auto-sync skill dari Catalog
FLOWORK_SKILL_REGISTRY|flowork-os/flowork-skills|Repo skill registry
FLOWORK_SLOT_SAVE_PATH|—|Persist KV slot ke disk
FLOWORK_SYS_STATUS|on|Sisipin kondisi PC ke prompt
FLOWORK_TG_ALLOWED_CHATS|—|Chat ID diizinkan
FLOWORK_TG_BOT_TOKEN|—|Bot token Telegram
FLOWORK_TG_CHUNK|on|Chunking pesan
FLOWORK_TG_FORMAT|html|Format pesan: html / plain
FLOWORK_TG_MEDIA|on|Enable media (on-demand)
FLOWORK_TOOLCALL_RECOVER|on|Parse <tool_call> bocor model lokal
FLOWORK_TOOL_GC_IDLE_DAYS|90|Hari idle sebelum prune tool
FLOWORK_TOOL_GC_MAXERR|5|Error threshold prune tool
FLOWORK_TOOL_GC_OFF|off|Matiin GC tools otomatis
FLOWORK_TOOLS_DIR|auto|Folder sidecar-tools
FLOWORK_TRIGGER_TIMEOUT_SEC|300|Timeout invoke action trigger
FLOWORK_TRUST_PROXY|off|Trust X-Forwarded-* headers
FLOWORK_TZ_LABEL|WIB|Label zona waktu
FLOWORK_TZ_OFFSET_HOURS|7|Offset jam zona waktu
FLOWORK_URGENCY|normal|Mode urgensi cost-of-thought
FLOWORK_WORKLOG|on|Papan kerja bersama lintas-agent
FLOWORK_WORKLOG_STALE_MIN|60|Ambang task nyangkut (menit)
`

const nerveRegistryTable = `
RegisterDetector|Runtime adopt custom (Python/Node/Go/Rust+)
RegisterScanRule|Pola berbahaya custom scanner
RegisterCapabilityVerifier|Pemeriksa jenis kapabilitas baru (AI Studio gate)
RegisterFeature|Fitur self-contained per fase (Wire/Route/Seed)
RegisterGraphProjection|Sumber proyeksi ke Cognitive Graph
RegisterDeathObserver|Reaksi pada kematian kemampuan (DeathLetter)
RegisterLLMLifecycleObserver|Callback wake/sleep LLM idle
RegisterDynamic|Tool plugin runtime register/update
RegisterInterceptor|Interceptor pre-execution tool
RegisterHook|Slash hook (before/after)
RegisterExecStrategy|Strategi eksekusi crew (parallel/debate)
RegisterSkillProvider|Provider skill ([]SkillDoc)
RegisterInstinctSelector|Selector insting custom
RegisterInstinctSelectorCtx|Selector insting ctx-aware (scoped)
RegisterDeliverer|Pengirim balasan trigger (telegram/chat)
RegisterGroupSyncHook|Hook sinkronisasi roster group
RegisterPrimitive|Primitive capability kernel (browser/custom)
RegisterBuiltins|Provider builtin kernel (store/time/fs/exec/…)
RegisterDeferPolicy|Kebijakan defer/expose-tool per-agent
RegisterMeshFilter|Filter paket mesh (reject/pass/quarantine)
RegisterMigration|Migration SQL (additive)
RegisterProxyDeployTarget|Target deploy proxy
RegisterTunnelProvider|Provider tunnel (ngrok/Cloudflare/SSH)
RegisterExtraRoute|Route HTTP tambahan
RegisterCLITool / RegisterCustomCLITool|CLI tool extra (code + DB-driven)
RegisterToolcallExtractor|Extractor format tool-call muntah
(provider) Register × LLM/embedding/image/stt/tts/search/fetch/quota/mcp/translator/rtk|Provider plug per-jenis
(tools/slashcmd/triggers) Register|Builtin tool / slash cmd / jenis trigger
`

// nerveSeedFromTable — parse tabel embedded → colok tiap baris ke papan via RegisterNerve.
func nerveSeedFromTable() {
	for _, ln := range strings.Split(strings.TrimSpace(nerveSwitchTable), "\n") {
		p := strings.SplitN(strings.TrimSpace(ln), "|", 3)
		if len(p) < 3 || p[0] == "" {
			continue
		}
		RegisterNerve(Nerve{Name: p[0], Kind: "switch", Default: p[1], Desc: p[2]})
	}
	for _, ln := range strings.Split(strings.TrimSpace(nerveRegistryTable), "\n") {
		p := strings.SplitN(strings.TrimSpace(ln), "|", 2)
		if len(p) < 2 || p[0] == "" {
			continue
		}
		RegisterNerve(Nerve{Name: p[0], Kind: "registry", Default: "kosong", Desc: p[1]})
	}
}

func init() {
	nerveSeedFromTable()
	RegisterFeature(Feature{Name: "nerve-map", Phase: PhaseRoute, Apply: func(d *Deps) {
		d.Mux.HandleFunc("/api/evolve/nerves", nervesHandler())
	}})
}

// nervesHandler — GET /api/evolve/nerves. AI/GUI tanya daftar saraf yg boleh dipakai.
func nervesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ns := Nerves()
		byKind := map[string]int{}
		for _, n := range ns {
			byKind[n.Kind]++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"channels": NerveChannels(),
			"count":    len(ns),
			"by_kind":  byKind,
			"nerves":   ns,
		})
	}
}
