// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"context"
	"fmt"

	"flowork-gui/internal/tools"
)

type skillSuggestTool struct{}

func (skillSuggestTool) Name() string       { return "skill_suggest" }
func (skillSuggestTool) Capability() string { return "state:read" }
func (skillSuggestTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Lihat usulan SKILL dari pola tool yang sering lo pakai SUKSES. Pakai buat refleksi: tool apa yang udah jadi kebiasaan sukses → bisa diformalin jadi skill/alur. Balik kandidat urut paling sering.",
		Params: []tools.Param{
			{Name: "min_count", Type: tools.ParamInt, Description: "minimal jumlah sukses biar jadi kandidat (default 2)"},
			{Name: "limit", Type: tools.ParamInt, Description: "max kandidat (default 10, max 50)"},
		},
		Returns: "{count, candidates:[{tool_name, success_count, last_used, suggestion}]}",
	}
}

func (skillSuggestTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	minCount := 0
	limit := 0
	switch v := args["min_count"].(type) {
	case float64:
		minCount = int(v)
	case int:
		minCount = v
	}
	switch v := args["limit"].(type) {
	case float64:
		limit = int(v)
	case int:
		limit = v
	}
	cands, err := store.SuggestSkillCandidates(minCount, limit)
	if err != nil {
		return tools.Result{}, fmt.Errorf("skill_suggest: %w", err)
	}
	return tools.Result{Output: map[string]any{
		"count":      len(cands),
		"candidates": cands,
	}}, nil
}

// SELF-REGISTER plug-in (📄 lock/tool-manager.md) — edit/hapus/tambah BEBAS tanpa unfreeze.
func init() {
	tools.Register(&skillSuggestTool{})
}
