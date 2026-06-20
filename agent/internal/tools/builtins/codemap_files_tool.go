// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-20 (codemap-works-for-all-agents).
// LOCKED ≠ FREEZE (boleh diedit dgn izin owner).
// Owner: Aola Sahidin (Mr.Dev). Reason: fix codemap blank — tool codemap_files (peta level-FILE) +
// fallback canonical (canonicalCodemapAgent/Nodes/Stats) yg dipake codemap_search/stats/count
// biar SEMUA agent yg di-centang toolnya dapat peta. Live-tested. Jangan ubah tanpa izin.
package builtins

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/tools"
)

// canonicalCodemapAgent — codemap = peta SATU repo yang sama buat semua agent, jadi
// di-index sekali ke store agent kanonik (default mr-flow) lewat /api/codemap/reindex.
// Agent lain (mis. evo-coder) store-nya kosong → baca dari sini. Override via env.
func canonicalCodemapAgent() string {
	if v := strings.TrimSpace(os.Getenv("FLOWORK_CODEMAP_CANONICAL_AGENT")); v != "" {
		return v
	}
	return "mr-flow"
}

// canonicalCodemapNodes — baca node-level dari store kanonik. Dipakai sbg fallback
// oleh codemap_search/stats kalau store agent pemanggil kosong (belum di-reindex),
// biar tool itu ga blank buat agent mana pun (codebase sama buat semua).
func canonicalCodemapNodes(nodeType, layer, search string, limit int) []agentdb.CodemapNode {
	cs, err := agentdb.Open(agentdb.Resolve(canonicalCodemapAgent(), ""))
	if err != nil {
		return nil
	}
	defer cs.Close()
	rows, _ := cs.ListCodemapNodes(nodeType, layer, search, limit)
	return rows
}

// canonicalCodemapStats — agregat akurat dari store kanonik (fallback codemap_stats).
func canonicalCodemapStats() (total int, byType, byLayer map[string]int, source string) {
	byType, byLayer = map[string]int{}, map[string]int{}
	cs, err := agentdb.Open(agentdb.Resolve(canonicalCodemapAgent(), ""))
	if err != nil {
		return 0, byType, byLayer, "empty"
	}
	defer cs.Close()
	t, bt, bl, e := cs.CodemapNodeStats()
	if e != nil || t == 0 {
		return 0, byType, byLayer, "empty"
	}
	return t, bt, bl, "canonical:" + canonicalCodemapAgent()
}

// codemap_files_tool.go — tool MAPPING codebase level-FILE (owner 2026-06-20: codemap
// tools balik blank karena codemap_search/stats baca codemap_NODES yang ga ke-index
// otomatis; sementara codemap_FILES (path+import+dependent+layer+tests) UDAH ke-populate
// reindex tapi GA ADA tool yg baca). Tool ini buka data itu → coder evolusi (+ agent lain)
// bisa beneran mapping codebase: file apa aja, di layer mana, dipakai siapa, ada test ga.
// Anti over-prompt: filter + cap.

func init() { tools.Register(&codemapFilesTool{}) }

type codemapFilesTool struct{}

func (codemapFilesTool) Name() string       { return "codemap_files" }
func (codemapFilesTool) Capability() string { return "state:read" }
func (codemapFilesTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Mapping codebase level-FILE (sumber kebenaran self-map): daftar file + layer + jumlah baris + dependent_count (dipakai brp file) + ada test/doc. Filter optional. Buat ngerti struktur codebase sebelum nulis/ubah kode.",
		Params: []tools.Param{
			{Name: "search", Type: tools.ParamString, Description: "substring path/nama (optional)"},
			{Name: "layer", Type: tools.ParamString, Description: "filter layer (optional, mis. handler/engine/data-store)"},
			{Name: "limit", Type: tools.ParamInt, Description: "max file (default 40, max 200)"},
		},
		Returns: "{total, shown, files:[{path,layer,line_count,dependent_count,has_tests,has_docs}]}",
	}
}

func (codemapFilesTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	source := "own"
	all, err := store.ListCodemapFiles()
	if err != nil {
		return tools.Result{}, err
	}
	// Fallback: store agent ini belum di-index → baca canonical (repo sama buat semua).
	if len(all) == 0 {
		if cs, e := agentdb.Open(agentdb.Resolve(canonicalCodemapAgent(), "")); e == nil {
			if rows, e2 := cs.ListCodemapFiles(); e2 == nil && len(rows) > 0 {
				all = rows
				source = "canonical:" + canonicalCodemapAgent()
			}
			cs.Close()
		}
	}
	search := strings.ToLower(strings.TrimSpace(asString(args["search"])))
	layer := strings.ToLower(strings.TrimSpace(asString(args["layer"])))
	limit := 40
	if v, ok := args["limit"].(float64); ok && int(v) > 0 {
		limit = int(v)
	}
	if v, ok := args["limit"].(int); ok && v > 0 {
		limit = v
	}
	if limit > 200 {
		limit = 200
	}

	out := make([]map[string]any, 0, limit)
	matched := 0
	// urut by dependent_count DESC (file paling kepake = paling penting buat mapping).
	sort.SliceStable(all, func(i, j int) bool {
		return asInt(all[i]["dependent_count"]) > asInt(all[j]["dependent_count"])
	})
	for _, f := range all {
		path := strings.ToLower(asString(f["path"]) + " " + asString(f["name"]))
		if search != "" && !strings.Contains(path, search) {
			continue
		}
		if layer != "" && !strings.Contains(strings.ToLower(asString(f["layer"])), layer) {
			continue
		}
		matched++
		if len(out) < limit {
			out = append(out, map[string]any{
				"path":             f["path"],
				"layer":            f["layer"],
				"line_count":       f["line_count"],
				"dependent_count":  f["dependent_count"],
				"has_tests":        f["has_tests"],
				"has_docs":         f["has_docs"],
			})
		}
	}
	return tools.Result{Output: map[string]any{"source": source, "total": len(all), "matched": matched, "shown": len(out), "files": out}}, nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}
