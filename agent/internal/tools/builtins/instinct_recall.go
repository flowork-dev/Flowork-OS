// instinct_recall.go — Phase 3B (§4.10, D7): retrieve coding/security INSTINCT
// sebelum agent nulis code. File BARU (register via init(), pola agent_run.go —
// JANGAN modify builtins.go locked).
//
// Beda dari brain_search_shared (korpus umum) & graph_recall (twin/relasi lokal):
// ini KHUSUS narik POLA INSTINCT distilasi (room coding_instinct / security_instinct)
// dari shared brain → fact-sheet RINGKAS budget-capped → agent sadar pola+celah
// SEBELUM ngoding. Anti muntah prompt: cuma top-k relevan, di-cap.
//
// CAPABILITY: rpc:router:brain (sama brain_search_shared).
//
// Retrieval = FTS shared brain (routerclient.SearchBrain), filter client-side ke
// room instinct (endpoint search-drawers gak punya filter room → over-fetch+filter).
// Embedding semantic = bonus kalau brain udah di-reindex; FTS udah cukup buat keyword.

package builtins

import (
	"context"
	"fmt"
	"strings"

	"flowork-gui/internal/routerclient"
	"flowork-gui/internal/tools"
)

func init() { tools.Register(&instinctRecallTool{}) }

const (
	instinctDefaultK  = 6
	instinctMaxChars  = 1400 // budget-cap (anti muntah prompt, pola §4.8)
	instinctOverFetch = 20   // over-fetch lalu filter room (endpoint gak filter room)
)

// room instinct yang valid (distilasi, bukan korpus mentah).
var instinctRooms = map[string]bool{"coding_instinct": true, "security_instinct": true}

type instinctRecallTool struct{}

func (instinctRecallTool) Name() string       { return "instinct_recall" }
func (instinctRecallTool) Capability() string { return "rpc:router:brain" }
func (instinctRecallTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Tarik POLA INSTINCT coding+security (distilasi dari model kuat) yang relevan SEBELUM nulis/review code. Return fact-sheet ringkas 'WHEN trigger -> rule'. Pakai pas mulai task coding (apalagi yg nyentuh input/auth/network/crypto). Beda dari brain_search_shared (umum) & graph_recall (twin lokal).",
		Params: []tools.Param{
			{Name: "query", Type: tools.ParamString, Description: "deskripsi task coding / area kode (mis. 'parse user input ke SQL query')", Required: true},
			{Name: "k", Type: tools.ParamInt, Description: "max insting (default 6)", Required: false, Default: instinctDefaultK},
		},
		Returns: "{instincts: [\"WHEN ... -> ...\"], count, fact_sheet}",
	}
}

func (instinctRecallTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	query, _ := args["query"].(string)
	if strings.TrimSpace(query) == "" {
		return tools.Result{}, fmt.Errorf("query required")
	}
	k := instinctDefaultK
	switch v := args["k"].(type) {
	case float64:
		k = int(v)
	case int:
		k = v
	}
	if k <= 0 {
		k = instinctDefaultK
	}

	routerURL := routerclient.DefaultRouterURL
	if cfg, lerr := store.Load(); lerr == nil {
		if u, ok := cfg["router_url"].(string); ok && u != "" {
			routerURL = u
		}
	}

	// over-fetch lalu filter ke room instinct (endpoint gak punya filter room).
	resp, err := routerclient.New(routerURL).SearchBrain(ctx, query, instinctOverFetch)
	if err != nil {
		return tools.Result{}, fmt.Errorf("instinct recall: %w", err)
	}

	var instincts []string
	seen := map[string]bool{}
	var sb strings.Builder
	sb.WriteString("# Coding/security instincts — apply BEFORE writing code\n")
	for _, h := range resp.Hits {
		if !instinctRooms[h.Room] {
			continue // bukan room instinct (skip korpus umum)
		}
		line := strings.TrimSpace(strings.ReplaceAll(h.Content, "\n", " "))
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		entry := "- " + line + "\n"
		if sb.Len()+len(entry) > instinctMaxChars {
			break
		}
		sb.WriteString(entry)
		instincts = append(instincts, line)
		if len(instincts) >= k {
			break
		}
	}

	sheet := ""
	if len(instincts) > 0 {
		sheet = sb.String()
	}
	return tools.Result{Output: map[string]any{
		"instincts":  instincts,
		"count":      len(instincts),
		"fact_sheet": sheet,
	}}, nil
}
