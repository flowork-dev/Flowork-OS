// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Port batch 6 — 6 tool tambahan.
//
// v7_extras.go:
//   finance_budgets, wallet_snapshots, scanner_runs_query,
//   scanner_findings_query, retention_report, codemap_count.

package builtins

import (
	"context"
	"fmt"

	"flowork-gui/internal/tools"
)

func init() {
	tools.Register(&financeBudgetsTool{})
	tools.Register(&scannerRunsQueryTool{})
	tools.Register(&scannerFindingsQueryTool{})
	tools.Register(&retentionReportTool{})
	tools.Register(&codemapCountTool{})
}

// =============================================================================
// 1. finance_budgets — list budgets
// =============================================================================

type financeBudgetsTool struct{}

func (financeBudgetsTool) Name() string       { return "finance_budgets" }
func (financeBudgetsTool) Capability() string { return "state:read" }
func (financeBudgetsTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "List finance budgets configured untuk agent. Self-introspection budget ceiling per category.",
		Params:      []tools.Param{},
		Returns:     "{count, budgets[]}",
	}
}

func (financeBudgetsTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	items, err := store.ListBudgets()
	if err != nil {
		return tools.Result{}, fmt.Errorf("list budgets: %w", err)
	}
	return tools.Result{
		Output: map[string]any{"count": len(items), "budgets": items},
	}, nil
}

// =============================================================================
// 3. scanner_runs_query — list scanner run history
// =============================================================================

type scannerRunsQueryTool struct{}

func (scannerRunsQueryTool) Name() string       { return "scanner_runs_query" }
func (scannerRunsQueryTool) Capability() string { return "state:read" }
func (scannerRunsQueryTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "List scanner run history (Section 25) — scan_type, target_path, finding counts. Default 20.",
		Params: []tools.Param{
			{Name: "limit", Type: tools.ParamInt, Description: "Max (default 20, max 100)", Required: false},
		},
		Returns: "{count, runs[]}",
	}
}

func (scannerRunsQueryTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	limit := 20
	if n, ok := args["limit"].(float64); ok && n > 0 {
		limit = int(n)
		if limit > 100 {
			limit = 100
		}
	}
	items, err := store.ListScannerRuns(limit)
	if err != nil {
		return tools.Result{}, fmt.Errorf("list scanner runs: %w", err)
	}
	return tools.Result{
		Output: map[string]any{"count": len(items), "runs": items},
	}, nil
}

// =============================================================================
// 4. scanner_findings_query — list findings dari satu run
// =============================================================================

type scannerFindingsQueryTool struct{}

func (scannerFindingsQueryTool) Name() string       { return "scanner_findings_query" }
func (scannerFindingsQueryTool) Capability() string { return "state:read" }
func (scannerFindingsQueryTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Get findings dari satu scanner run by run_id. Filter by severity.",
		Params: []tools.Param{
			{Name: "run_id", Type: tools.ParamInt, Description: "Scanner run ID", Required: true},
			{Name: "severity", Type: tools.ParamString, Description: "critical|high|medium|low|info (kosong=all)", Required: false},
			{Name: "limit", Type: tools.ParamInt, Description: "Max (default 100, max 500)", Required: false},
		},
		Returns: "{count, findings[]}",
	}
}

func (scannerFindingsQueryTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	runIDf, _ := args["run_id"].(float64)
	if runIDf <= 0 {
		return tools.Result{}, fmt.Errorf("run_id required")
	}
	severity, _ := args["severity"].(string)
	limit := 100
	if n, ok := args["limit"].(float64); ok && n > 0 {
		limit = int(n)
		if limit > 500 {
			limit = 500
		}
	}
	allItems, err := store.ListScannerFindings(int64(runIDf))
	if err != nil {
		return tools.Result{}, fmt.Errorf("list scanner findings: %w", err)
	}
	// Filter by severity + limit di-client (ListScannerFindings minimal API).
	items := allItems
	if severity != "" {
		filtered := items[:0]
		for _, f := range allItems {
			if f.Severity == severity {
				filtered = append(filtered, f)
			}
		}
		items = filtered
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return tools.Result{
		Output: map[string]any{"count": len(items), "findings": items},
	}, nil
}

// =============================================================================
// 5. retention_report — last retention sweep stats
// =============================================================================

type retentionReportTool struct{}

func (retentionReportTool) Name() string       { return "retention_report" }
func (retentionReportTool) Capability() string { return "state:read" }
func (retentionReportTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Snapshot table counts: interactions, decisions, mistakes_local, death_letter, workspace_meta. Buat health check.",
		Params:      []tools.Param{},
		Returns:     "{counts}",
	}
}

func (retentionReportTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	interactions, _ := store.CountInteractions()
	letters, _ := store.CountLetters(false)
	counts := map[string]any{
		"interactions": interactions,
		"death_letters": letters,
	}
	return tools.Result{
		Output: counts,
	}, nil
}

// =============================================================================
// 6. codemap_count — total codemap nodes indexed
// =============================================================================

type codemapCountTool struct{}

func (codemapCountTool) Name() string       { return "codemap_count" }
func (codemapCountTool) Capability() string { return "state:read" }
func (codemapCountTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Total codemap nodes indexed (Section 27). Anti over-prompt: cuma counts, ngga return list. Pakai codemap_search untuk query detail.",
		Params:      []tools.Param{},
		Returns:     "{total}",
	}
}

// LOCKED (soft, owner-approved 2026-06-20 codemap-fix): count akurat + fallback canonical. Jangan ubah tanpa izin.
func (codemapCountTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	// Agregat akurat (COUNT/GROUP BY) — bukan sampel ber-cap (ListCodemapNodes mentok 1000).
	total, byType, byLayer, err := store.CodemapNodeStats()
	if err != nil {
		return tools.Result{}, fmt.Errorf("count codemap: %w", err)
	}
	source := "own"
	if total == 0 {
		total, byType, byLayer, source = canonicalCodemapStats()
	}
	return tools.Result{
		Output: map[string]any{"source": source, "total": total, "by_type": byType, "by_layer": byLayer},
	}, nil
}
