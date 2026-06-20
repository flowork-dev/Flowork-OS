// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Shim adapter — map reference GUI tab paths (dari
//   Pictures/stable_open_router/.../guiapi/static/tabs/) ke endpoint
//   agent-scoped kita. Default agent = mr-flow (single-warga). Phase 2
//   (multi-agent agent_id selector di GUI header) → tambah file baru.
//
// legacy_compat.go — reference GUI tab compat shim.

package agentmgr

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/codemap"
	"flowork-gui/internal/httpx"
	"flowork-gui/internal/protector"
)

// codemapRoot — direktori source yang di-index codemap. Default working dir
// server (repo Flowork). Override via env FLOWORK_CODEMAP_ROOT (mis. untuk test).
func codemapRoot() string {
	if v := strings.TrimSpace(os.Getenv("FLOWORK_CODEMAP_ROOT")); v != "" {
		return v
	}
	return agentdb.ProjectRoot()
}

// defaultAgentID — single-warga shim. Phase 2: read dari X-Agent-ID header.
const defaultAgentID = "mr-flow"

// =============================================================================
// /api/finance/snapshot (reference: finance.js)
// =============================================================================

// FinanceSnapshotCompatHandler — GET /api/finance/snapshot
//
// Mode GABUNGAN: data API-cost REAL dari finance_ledger (Section 23) — total
// 7 hari + per-kategori + budget (dengan % terpakai) + recent calls. Saldo
// wallet personal di-fetch terpisah oleh frontend via /api/settings/wallet/portfolio.
// Shape: {api_cost_total_usd, api_cost_by_category[], budgets[], recent_calls[], updated_at}
func FinanceSnapshotCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -7).Format(time.RFC3339)
	to := now.Format(time.RFC3339)

	summary, _ := store.SummaryLedger(from, to)
	var total float64
	for _, s := range summary {
		total += s.CostUSD
	}
	// Budget + % terpakai (berdasarkan total 7d untuk metric biaya).
	rawBudgets, _ := store.ListBudgets()
	budgets := make([]map[string]any, 0, len(rawBudgets))
	for _, b := range rawBudgets {
		pct := 0.0
		if b.BudgetValue > 0 {
			pct = total / b.BudgetValue * 100
		}
		budgets = append(budgets, map[string]any{
			"metric_key":     b.MetricKey,
			"budget_value":   b.BudgetValue,
			"warning_at_pct": b.WarningAtPct,
			"enabled":        b.Enabled,
			"spent_usd":      total,
			"pct":            pct,
		})
	}
	recent, _ := store.ListLedger("", "", "", 15)

	httpx.WriteJSON(w, map[string]any{
		"api_cost_total_usd":   total,
		"api_cost_by_category": summary,
		"budgets":              budgets,
		"recent_calls":         recent,
		"updated_at":           to,
	})
}

// =============================================================================
// /api/brain/prompt-templates — legacy compat shim. The GUI "Prompt Library" tab
// (web/tabs/prompt.js) that consumed these was removed as a zombie (never wired
// into ACTIVE_TABS / the current agents flow). Routes kept as a stable compat
// surface (loopback+owner-auth, harmless); removing wired routes is guarded by the
// wiring invariant. No live GUI consumer remains.
// =============================================================================

// PromptTemplatesListCompatHandler — GET /api/brain/prompt-templates
// Reference shape: {templates: [{name, description, ...}]}
func PromptTemplatesListCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"templates": []any{}})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	slots, err := store.ListSelfPromptSlots()
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	templates := make([]map[string]any, 0, len(slots))
	for _, s := range slots {
		if s.Body == promptDeletedSentinel { // soft-deleted → hide
			continue
		}
		preview := s.Body
		if len(preview) > 300 {
			preview = preview[:300]
		}
		templates = append(templates, map[string]any{
			"name":         s.Slot,
			"preview":      preview,
			"content_size": len(s.Body),
			"updated_at":   s.UpdatedAt,
			"usage_count":  1, // dipakai oleh agent pemilik slot (mr-flow)
		})
	}
	httpx.WriteJSON(w, map[string]any{"templates": templates, "count": len(templates)})
}

// promptDeletedSentinel — body marker untuk soft-delete slot prompt.
const promptDeletedSentinel = "[deleted]"

// PromptTemplatesDetailCompatHandler — GET /api/brain/prompt-templates/detail?name=
// Reference shape: {name, body, description}
func PromptTemplatesDetailCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		httpx.WriteJSON(w, map[string]any{"error": "name required"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	sp, err := store.GetSelfPrompt(name, 0)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{
		"name":       sp.Slot,
		"content":    sp.Body,
		"used_by":    []map[string]any{{"name": defaultAgentID}},
		"used_count": 1,
		"updated_at": sp.UpdatedAt,
	})
}

// PromptTemplatesUpsertCompatHandler — POST /api/brain/prompt-templates
// + POST /api/brain/prompt-templates/update (same handler — upsert via UNIQUE).
func PromptTemplatesUpsertCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		Name    string `json:"name"`
		Content string `json:"content"`
		Body    string `json:"body"` // fallback (compat lama)
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 128*1024)).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	content := body.Content
	if content == "" {
		content = body.Body
	}
	if body.Name == "" || content == "" {
		httpx.WriteJSON(w, map[string]any{"error": "name + content required"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	id, err := store.SetSelfPrompt(body.Name, content, "", 0)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"ok": true, "id": id})
}

// PromptTemplatesDeleteCompatHandler — POST /api/brain/prompt-templates/delete
// Soft-delete: insert empty body v+1. Phase 2 add real delete.
func PromptTemplatesDeleteCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	// Soft-delete: body sentinel → di-hide oleh list handler.
	_, _ = store.SetSelfPrompt(body.Name, promptDeletedSentinel, "soft-delete", 0)
	httpx.WriteJSON(w, map[string]any{"ok": true})
}

// =============================================================================
// Helpers
// =============================================================================

// =============================================================================
// /api/protector (reference: protector.js)
// =============================================================================

// ProtectorListCompatHandler — GET /api/protector
// Reference shape: {rules: [{id, path, type, action, enabled, source}]}
func ProtectorListCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	custom, _ := store.ListProtectorRulesCat()
	out := []map[string]any{}
	// Baseline (hardcoded immutable): category = rule type (file_path/command/ip/env_var).
	for _, b := range protector.Baseline() {
		out = append(out, map[string]any{
			"path":     b.Pattern,
			"category": b.Type,
			"action":   b.Action,
			"active":   true,
			"source":   "hardcoded",
		})
	}
	// Custom: category label dari kolom category (fallback ke rule type).
	for _, c := range custom {
		cat := c.Category
		if cat == "" {
			cat = c.RuleType
		}
		out = append(out, map[string]any{
			"path":     c.Pattern,
			"category": cat,
			"action":   c.Action,
			"active":   c.Enabled,
			"source":   c.Source,
		})
	}
	httpx.WriteJSON(w, map[string]any{"rules": out, "count": len(out)})
}

// ProtectorAddCompatHandler — POST /api/protector/add
func ProtectorAddCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		Path     string `json:"path"`
		Type     string `json:"type"`     // UI match-style (suffix/basename/custom) — informational
		Category string `json:"category"` // UI label (secrets/core/doktrin/...)
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.Path) == "" {
		httpx.WriteJSON(w, map[string]any{"error": "path required"})
		return
	}
	if body.Category == "" {
		body.Category = "custom"
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	// rule_type "file_path" supaya interceptor file-ops enforce; category = label UI.
	id, err := store.AddProtectorRuleCat("file_path", body.Path, "block", body.Category)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"ok": true, "id": id})
}

// ProtectorRemoveCompatHandler — POST /api/protector/remove {path}
func ProtectorRemoveCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	id, ok := store.FindProtectorRuleIDByPattern(body.Path)
	if !ok {
		httpx.WriteJSON(w, map[string]any{"error": "rule not found (hardcoded baseline tidak bisa dihapus)"})
		return
	}
	if err := store.DeleteProtectorRule(id); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"ok": true})
}

// ProtectorToggleCompatHandler — POST /api/protector/toggle {path, active}
func ProtectorToggleCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		Path   string `json:"path"`
		Active bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	id, ok := store.FindProtectorRuleIDByPattern(body.Path)
	if !ok {
		httpx.WriteJSON(w, map[string]any{"error": "rule not found (hardcoded baseline selalu aktif)"})
		return
	}
	if err := store.ToggleProtectorRule(id, body.Active); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"ok": true})
}

// ProtectorTestCompatHandler — GET /api/protector/test?path=
func ProtectorTestCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		httpx.WriteJSON(w, map[string]any{"error": "path required"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	customRows, _ := store.ListProtectorRules()
	customRules := []protector.BaselineRule{}
	for _, c := range customRows {
		if c.Enabled {
			customRules = append(customRules, protector.BaselineRule{
				Type: c.RuleType, Pattern: c.Pattern, Action: c.Action,
			})
		}
	}
	matched, hit := protector.CheckPattern("file_path", path, customRules)
	httpx.WriteJSON(w, map[string]any{
		"path":      path,
		"protected": hit,
		"matched":   hit,
		"pattern":   matched.Pattern,
		"action":    matched.Action,
	})
}

// =============================================================================
// /api/codemap (reference: codemap.js)
// =============================================================================

// CodemapGraphCompatHandler — GET /api/codemap/graph
// Reference shape: {nodes: [{id, name, path, type}], edges: []}
func CodemapGraphCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	nodes, err := store.ListCodemapFiles()
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	edges, _ := store.ListCodemapFileEdges()
	httpx.WriteJSON(w, map[string]any{
		"nodes":      nodes,
		"edges":      edges,
		"node_count": len(nodes),
		"edge_count": len(edges),
	})
}

// CodemapStatusCompatHandler — GET /api/codemap/status
// Shape: {running, node_count, edge_count}
func CodemapStatusCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	nodes, _ := store.ListCodemapFiles()
	edges, _ := store.ListCodemapFileEdges()
	httpx.WriteJSON(w, map[string]any{
		"running":    false,
		"node_count": len(nodes),
		"edge_count": len(edges),
	})
}

// CodemapZombiesCompatHandler — GET /api/codemap/zombies
func CodemapZombiesCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	// Zombie file-level — HEURISTIK LEMAH (ADVISORY): KANDIDAT buat review manual,
	// BUKAN vonis "hapus". ⚠️ Import Go = level-PAKET (cuma 1 file wakil/paket dapet
	// incoming edge) → naif "dependent=0 && no-outgoing" salah-vonis ratusan file hidup
	// (cth nyata 2026-06-15: agentdb/*, llm.go, walker.go divonis zombie padahal AKTIF).
	// FIX paket-aware: file = kandidat HANYA kalau (a) gak ada incoming, (b) gak outgoing,
	// (c) SELURUH PAKET-nya (dir) gak di-import paket lain, (d) BUKAN package main / dir
	// punya main.go (intra-package call gak ke-track edge), (e) bukan entry/test/GOOS.
	// Owner-flagged: AI/self-evolution DILARANG auto-delete dari sinyal ini.
	files, err := store.ListCodemapFiles()
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	edges, _ := store.ListCodemapFileEdges()
	dirOf := func(p string) string {
		if i := strings.LastIndexByte(p, '/'); i >= 0 {
			return p[:i]
		}
		return ""
	}
	baseOf := func(p string) string {
		if i := strings.LastIndexByte(p, '/'); i >= 0 {
			return p[i+1:]
		}
		return p
	}
	hasOutgoing := map[string]bool{}
	dirImported := map[string]bool{} // paket (dir) yang di-import paket LAIN → kepake
	for _, e := range edges {
		hasOutgoing[e.From] = true
		if dirOf(e.From) != dirOf(e.To) {
			dirImported[dirOf(e.To)] = true
		}
	}
	dirHasMain := map[string]bool{} // dir = package main (executable) → intra-package, skip
	for _, f := range files {
		if p, _ := f["path"].(string); baseOf(p) == "main.go" {
			dirHasMain[dirOf(p)] = true
		}
	}
	skipFile := func(p string) bool {
		b := baseOf(p)
		if b == "main.go" || strings.HasSuffix(b, "_test.go") {
			return true
		}
		for _, suf := range []string{"_linux.go", "_darwin.go", "_windows.go", "_other.go", "_wasm.go", "_js.go", "_unix.go", "_amd64.go", "_arm64.go"} {
			if strings.HasSuffix(b, suf) {
				return true // build-constrained (GOOS/arch) — edge graph gak lengkap
			}
		}
		return false
	}
	zombies := []map[string]any{}
	for _, f := range files {
		path, _ := f["path"].(string)
		dep, _ := f["dependent_count"].(int)
		if dep == 0 && !hasOutgoing[path] && !dirImported[dirOf(path)] && !dirHasMain[dirOf(path)] && !skipFile(path) {
			zombies = append(zombies, map[string]any{
				"path":       path,
				"name":       f["name"],
				"file_type":  f["file_type"],
				"line_count": f["line_count"],
			})
		}
	}
	httpx.WriteJSON(w, map[string]any{
		"zombies": zombies, "count": len(zombies),
		"advisory": true,
		"note":     "HEURISTIK LEMAH (import Go = level-paket): KANDIDAT review manual, JANGAN auto-delete. Verifikasi pakai grep/go build dulu.",
	})
}

// CodemapReindexCompatHandler — POST /api/codemap/reindex
// Walk repo source (codemapRoot) → file node + import edge → replace tabel.
func CodemapReindexCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	files, edges, err := codemap.WalkRepo(codemapRoot())
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	// Convert codemap.FileInfo → agentdb.CodemapFile.
	rows := make([]agentdb.CodemapFile, 0, len(files))
	for _, f := range files {
		rows = append(rows, agentdb.CodemapFile{
			Path: f.Path, Name: f.Name, FileType: f.FileType, LineCount: f.LineCount,
			Layer: f.Layer, HasTests: f.HasTests, HasDocs: f.HasDocs,
			HealthScore: f.HealthScore, RecentlyTouched: f.RecentlyTouched, Issues: f.Issues,
		})
	}
	eRows := make([]agentdb.CodemapFileEdge, 0, len(edges))
	for _, e := range edges {
		eRows = append(eRows, agentdb.CodemapFileEdge{From: e.From, To: e.To})
	}
	if err := store.ReplaceCodemapFiles(rows, eRows); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}

	// ── NODE-LEVEL (fungsi/type/method) ── LOCKED (soft, owner-approved 2026-06-20) ──
	// Fix owner 2026-06-20: codemap_search/stats/count baca tabel codemap_NODES tapi
	// itu KOSONG — selama ini cuma file-level (di atas) yg ke-index; node-index cuma
	// per-file manual (/api/agents/codemap/index). Sekali reindex sekarang ALSO bulk
	// parse tiap .go (ParseGo) → upsert node, jadi tools itu ga blank lagi. Full
	// replace: clear-per-file dulu biar idempotent.
	root := codemapRoot()
	nodeCount := 0
	for _, f := range files {
		if f.FileType != "go" {
			continue
		}
		content, rerr := os.ReadFile(filepath.Join(root, f.Path))
		if rerr != nil {
			continue
		}
		ns, perr := codemap.ParseGo(f.Path, content)
		if perr != nil {
			continue
		}
		_ = store.DeleteCodemapNodesByFile(f.Path)
		now := time.Now().UTC().Format(time.RFC3339)
		for _, n := range ns {
			if _, e := store.UpsertCodemapNode(agentdb.CodemapNode{
				NodeType: n.Type, Name: n.Name, FilePath: f.Path,
				LineStart: n.LineStart, LineEnd: n.LineEnd, Layer: f.Layer,
				Signature: n.Signature, SizeLOC: n.SizeLOC,
				LastModified: now, IndexedAt: now,
			}); e == nil {
				nodeCount++
			}
		}
	}

	httpx.WriteJSON(w, map[string]any{
		"ok":         true,
		"message":    "reindex selesai (file + node level)",
		"file_count": len(rows),
		"node_count": nodeCount,
		"edge_count": len(eRows),
	})
}

// CodemapDocsCompatHandler — GET /api/codemap/docs?path=<rel>
// Return isi file source (text/plain) untuk viewer. Anti path-traversal:
// resolve di dalam codemapRoot, reject keluar root.
func CodemapDocsCompatHandler(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimSpace(r.URL.Query().Get("path"))
	if rel == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}
	root := codemapRoot()
	clean := filepath.Clean(filepath.Join(root, rel))
	// Pastikan masih di dalam root (anti ../ escape).
	if clean != root && !strings.HasPrefix(clean, root+string(os.PathSeparator)) {
		http.Error(w, "path escapes root", http.StatusForbidden)
		return
	}
	data, err := os.ReadFile(clean)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	const maxDocBytes = 256 * 1024
	if len(data) > maxDocBytes {
		data = data[:maxDocBytes]
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// Bungkus ke fenced code block biar mdToHTML render rapi sebagai code.
	lang := strings.TrimPrefix(filepath.Ext(rel), ".")
	_, _ = w.Write([]byte("```" + lang + "\n"))
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n```\n"))
}

// CodemapRootsCompatHandler — GET /api/codemap/roots (return file_path list).
func CodemapRootsCompatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	store, err := openAgentStore(defaultAgentID)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer store.Close()
	all, _ := store.ListCodemapNodes("", "", "", 1000)
	seen := map[string]bool{}
	roots := []string{}
	for _, n := range all {
		if !seen[n.FilePath] {
			seen[n.FilePath] = true
			roots = append(roots, n.FilePath)
		}
	}
	httpx.WriteJSON(w, map[string]any{"roots": roots, "count": len(roots)})
}

// keep agentdb import used.
var _ = agentdb.Decision{}
