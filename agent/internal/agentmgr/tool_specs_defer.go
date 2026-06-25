// === LOCKED FILE ===
// Status: STABLE — FROZEN (chattr +i + hash KERNEL_FREEZE.md). JANGAN edit tanpa unfreeze owner.
// Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS
// Arsitektur & alasan: lock/tools.md §7.5–7.6 (deferred-tools + all-tools).
//
// tool_specs_defer.go — MEKANISME #2C deferred-tools + all-tools, DI-EKSTRAK dari tool_specs.go
// (2026-06-25, pola nano-modular: mekanisme STABIL di-FREEZE; daftar-tool + handler EDITABLE
// tetap di tool_specs.go). Tujuan freeze: Flowork ber-EVOLUSI sendiri (auto-edit *.go) → lindungi
// mekanisme ini biar GAK ke-rusak diam-diam.
//
// ⭐ SWITCH / EXTENSION (Rule 7 — nambah fitur TANPA buka file frozen ini):
//   - ENV `FLOWORK_DEFER_TOOLS` / `FLOWORK_EXPOSE_ALL_TOOLS` (default OFF) — kontrol global.
//   - `RegisterDeferPolicy(fn)` — DAFTARIN sumber kebijakan per-agent (mis. toggle GUI/kv per-agent,
//     doktrin "GUI=kebenaran-utama") TANPA unfreeze; hook menang atas ENV.
//   - Daftar tool (coreExposedTools/primaryExtra) + handler ToolSpecsHandler = di tool_specs.go (EDITABLE).
package agentmgr

import (
	"os"
	"strings"
	"sync"

	"flowork-gui/internal/tools"
)

// deferPolicyHook — extension point (Rule 7): kalau di-register (mis. per-agent GUI/kv), dia
// jadi sumber kebijakan defer/all-tools, OVERRIDE ENV. Default nil → pakai ENV (scoped-primary).
var deferPolicyHook func(agentID string, isPrimary bool) (deferOn, exposeAll bool)

// RegisterDeferPolicy — DAFTARIN kebijakan defer/all-tools per-agent TANPA buka file frozen ini.
// Dipanggil sekali (mis. dari feature_*.go editable) buat wiring sumber GUI/kv. Nil = default ENV.
func RegisterDeferPolicy(fn func(agentID string, isPrimary bool) (deferOn, exposeAll bool)) {
	deferPolicyHook = fn
}

// resolveDeferPolicy — SUMBER TUNGGAL kebijakan. Hook (kalau ada) menang; else ENV
// (FLOWORK_DEFER_TOOLS scoped-primary + FLOWORK_EXPOSE_ALL_TOOLS). Handler & ToolRunHandler
// WAJIB lewat sini (jangan baca ENV langsung) → future per-agent control = register hook, no unfreeze.
func resolveDeferPolicy(agentID string, isPrimary bool) (deferOn, exposeAll bool) {
	if deferPolicyHook != nil {
		return deferPolicyHook(agentID, isPrimary)
	}
	return deferToolsEnabled() && isPrimary, exposeAllTools()
}

// deferToolsEnabled — ENV switch global (default OFF = byte-identik perilaku lama).
func deferToolsEnabled() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv("FLOWORK_DEFER_TOOLS"))) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// exposeAllTools — ENV switch (arah owner "buang subscription"): ON = expose SEMUA tool
// ke-registry (bukan cuma core+sidecar+subscription) → agent raih tool APAPUN; CAP-GATE pas
// RUN yg jaga (validated 2026-06-25), doktrin/insting jadi KEMUDI. Efektif bareng defer + primary.
func exposeAllTools() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv("FLOWORK_EXPOSE_ALL_TOOLS"))) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// activeDeferred — set tool yg udah di-LOOKUP model (per-agent) → ToolSpecsHandler kirim schema
// PENUH-nya (masuk array `tools` → grammar llama bisa manggil). Ini "meta-runner" yg dulu ilang:
// tool deferred jadi callable SETELAH model discover via tool_lookup. Session-scoped (in-memory,
// reset saat restart) = ala Claude Code. main.go re-fetch specs abis tool_lookup → tool nongol next iter.
var (
	activeDeferredMu sync.Mutex
	activeDeferred   = map[string]map[string]bool{} // agentID -> set nama-kanonik tool aktif
)

func activateDeferred(agentID, canonicalName string) {
	agentID = strings.TrimSpace(agentID)
	canonicalName = strings.TrimSpace(canonicalName)
	if agentID == "" || canonicalName == "" {
		return
	}
	activeDeferredMu.Lock()
	defer activeDeferredMu.Unlock()
	s := activeDeferred[agentID]
	if s == nil {
		s = map[string]bool{}
		activeDeferred[agentID] = s
	}
	s[canonicalName] = true
}

func isActiveDeferred(agentID, canonicalName string) bool {
	activeDeferredMu.Lock()
	defer activeDeferredMu.Unlock()
	return activeDeferred[agentID][canonicalName]
}

// primaryVitalTools — tool primary yg TERBUKTI bikin mr-flow NYASAR kalau di-drop
// (web_search/task-routing/system_power/codemap). Tetap full-schema (alwaysLoad) walau defer ON.
var primaryVitalTools = []string{
	"web_search", "task_list", "task_run", "system_power", "codemap_search",
}

// deferFetchTool — primitif "ambil schema" (analog ToolSearch Claude Code). Di-core-kan SAAT defer ON.
const deferFetchTool = "tool_lookup"

// deferAnnounceMax — batas jumlah tool yg di-ANNOUNCE saat defer ON (schema+nama). Saat defer,
// nama murah → batas dinaikin biar semua tool keliatan (cabut "buta krn cap"). >batas = di-truncate.
const deferAnnounceMax = 256 // muat SEMUA tool registry (~202) + headroom (mode all-tools)

// deferCatalogLine — satu baris katalog "nama — hint" buat tool yg di-defer.
func deferCatalogLine(t tools.Tool) string {
	return tools.DisplayName(t.Name()) + " — " + firstSentence(t.Schema().Description)
}

// firstSentence — potong deskripsi jadi hint pendek (≤80 char / kalimat pertama).
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, ".\n"); i > 0 && i <= 80 {
		return strings.TrimSpace(s[:i])
	}
	if len(s) > 80 {
		return strings.TrimSpace(s[:80])
	}
	return s
}

// injectDeferredCatalog — sisipin katalog deferred-tools ke deskripsi `tool_search` (channel
// always-on yg main.go FROZEN tetap forward ke LLM). Cache-stable: list statik per-agent.
func injectDeferredCatalog(specs []map[string]any, lines []string) {
	catalog := "\n\n[DEFERRED TOOLS — ADA tapi schema BELUM dimuat (hemat token)]. Cara pakai: panggil `" +
		deferFetchTool + "` dgn {name} buat ambil parameter tool, LALU panggil tool itu langsung " +
		"(host tetap bisa jalanin walau gak di-load awal). Daftar tersedia:\n- " + strings.Join(lines, "\n- ")
	for _, sp := range specs {
		fn, ok := sp["function"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := fn["name"].(string)
		if name == "tool_search" || name == tools.DisplayName("tool_search") {
			if d, ok := fn["description"].(string); ok {
				fn["description"] = d + catalog
			}
			return
		}
	}
}
