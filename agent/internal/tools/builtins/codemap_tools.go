// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Section 28 phase 1 codemap warga query tools. Wrap Section 27
//   codemap_nodes accessor as builtin tools. Anti over-prompt: result
//   summary form (top-N + count), bukan full dump. Phase 2 (callgraph
//   tools setelah Section 27 edges siap) → tambah file baru.
//
// codemap_tools.go — Section 28 phase 1: codemap_search + codemap_get.

package builtins

import (
	"context"
	"fmt"
	"strings"

	"flowork-gui/internal/tools"
)

// codemapSearchTool — query agent's codemap nodes.
type codemapSearchTool struct{}

func (codemapSearchTool) Name() string       { return "codemap_search" }
func (codemapSearchTool) Capability() string { return "state:read" }
func (codemapSearchTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Search codemap nodes by name substring + optional node_type/layer filter. Cap 10 (anti over-prompt). Return summary fields (name, type, file, lines).",
		Params: []tools.Param{
			{Name: "search", Type: tools.ParamString, Description: "name substring (required)", Required: true},
			{Name: "node_type", Type: tools.ParamString, Description: "func | type | method | var"},
			{Name: "layer", Type: tools.ParamString, Description: "agent | tool | gui | kernel | brain"},
		},
		Returns: "{items: [{name, type, file, lines, size_loc}], count, truncated}",
	}
}

func (codemapSearchTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	search, _ := args["search"].(string)
	search = strings.TrimSpace(search)
	if search == "" {
		return tools.Result{}, fmt.Errorf("search required")
	}
	nodeType, _ := args["node_type"].(string)
	layer, _ := args["layer"].(string)
	rows, err := store.ListCodemapNodes(nodeType, layer, search, 50)
	if err != nil {
		return tools.Result{}, err
	}
	if len(rows) == 0 {
		rows = canonicalCodemapNodes(nodeType, layer, search, 50)
	}
	const cap = 10
	truncated := len(rows) > cap
	if truncated {
		rows = rows[:cap]
	}
	items := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]any{
			"name":     r.Name,
			"type":     r.NodeType,
			"file":     r.FilePath,
			"lines":    fmt.Sprintf("%d-%d", r.LineStart, r.LineEnd),
			"size_loc": r.SizeLOC,
		})
	}
	return tools.Result{Output: map[string]any{
		"items":     items,
		"count":     len(items),
		"truncated": truncated,
	}}, nil
}

// codemapStatsTool — counts per node_type + layer untuk overview.
type codemapStatsTool struct{}

func (codemapStatsTool) Name() string       { return "codemap_stats" }
func (codemapStatsTool) Capability() string { return "state:read" }
func (codemapStatsTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Codemap overview — total nodes indexed. Anti over-prompt: ngga return list, cuma counts.",
		Returns:     "{total_nodes, by_type: {func: N, type: N, method: N}, ...}",
	}
}

// LOCKED (soft, owner-approved 2026-06-20 codemap-fix): agregat akurat + fallback canonical. Jangan ubah tanpa izin.
func (codemapStatsTool) Run(ctx context.Context, _ map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	total, byType, byLayer, err := store.CodemapNodeStats()
	if err != nil {
		return tools.Result{}, err
	}
	source := "own"
	if total == 0 {
		total, byType, byLayer, source = canonicalCodemapStats()
	}
	if _, ok := byLayer[""]; ok {
		byLayer["(unspecified)"] = byLayer[""]
		delete(byLayer, "")
	}
	return tools.Result{Output: map[string]any{
		"source":      source,
		"total_nodes": total,
		"by_type":     byType,
		"by_layer":    byLayer,
	}}, nil
}
