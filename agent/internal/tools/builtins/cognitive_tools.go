// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval (autonomy grant 2026-06-19).
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-19
// Reason: CGM graph_recall tool — built + unit-tested. Extend = new file, jangan modify ini.
//
// cognitive_tools.go — tool yang nyentuh Cognitive Graph (CGM) lokal agent.
//
// graph_recall: tarik subgraph relevan dari twin graph → fact-sheet RINGKAS
// (budget-capped, anti muntah prompt). Beda dari brain_search (FTS teks): ini
// HUBUNGAN/konteks antar-entitas (roadmap §4.8). Pola: tools.FromStore(ctx) +
// store.RecallFactSheet (agentdb/cognitive_recall.go, locked).
//
// Catatan: Embed=nil (seed label-only) buat sekarang — degrade aman tanpa router.
// Wiring embedder (semantic seed via /v1/embeddings) = langkah berikut.

package builtins

import (
	"context"
	"fmt"
	"strings"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/tools"
)

type graphRecallTool struct{}

func (graphRecallTool) Name() string       { return "graph_recall" }
func (graphRecallTool) Capability() string { return "state:read" }
func (graphRecallTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Recall grounding dari Cognitive Graph LOKAL (twin) — tarik subgraph relevan jadi fact-sheet ringkas (budget-capped). Pakai buat 'apa yang gw tau soal X' / 'gimana A nyambung ke B'. Beda dari brain_search (cari teks FTS): ini paham RELASI antar-entitas (Aola→prefers→X, dst).",
		Params: []tools.Param{
			{Name: "query", Type: tools.ParamString, Description: "topik / pertanyaan", Required: true},
			{Name: "max_chars", Type: tools.ParamInt, Description: "budget fact-sheet (default 1500)"},
		},
		Returns: "{query, fact_sheet, chars}",
	}
}

func (graphRecallTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	query, _ := args["query"].(string)
	if strings.TrimSpace(query) == "" {
		return tools.Result{}, fmt.Errorf("query required")
	}
	maxChars := 0
	switch v := args["max_chars"].(type) {
	case float64:
		maxChars = int(v)
	case int:
		maxChars = v
	}
	sheet, err := store.RecallFactSheet(ctx, query, agentdb.RecallDeps{MaxChars: maxChars})
	if err != nil {
		return tools.Result{}, fmt.Errorf("graph_recall: %w", err)
	}
	return tools.Result{Output: map[string]any{
		"query":      query,
		"fact_sheet": sheet,
		"chars":      len(sheet),
	}}, nil
}
