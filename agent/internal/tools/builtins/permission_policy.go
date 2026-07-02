// permission_policy.go — read-only classification for the approval gate (P2).
//
// This is the NON-FROZEN side of the plug-and-play permission hook: it registers
// tools.ReadOnlyClassifier so the (locked) sandbox knows which calls only read/observe
// and can exempt them from owner approval — without ever editing the locked file. To
// add or change permission policy in the future, edit THIS file (or add another that
// sets tools.ExtraGatePolicy); do NOT unlock sandbox_v3.go.
package builtins

import (
	"context"
	"fmt"
	"os"
	"strings"

	"flowork-gui/internal/tools"
)

// readOnlyTools — tools whose EVERY invocation only reads/observes (no state change,
// no irreversible action). Conservative: anything not listed is treated as mutating.
var readOnlyTools = map[string]bool{
	"file_read": true, "file_list": true, "glob": true, "grep": true, "codemap_search": true,
	"codemap_search_advanced": true, "codemap_stats": true,
	"brain_search": true, "brain_get": true, "brain_search_shared": true,
	"memory_get": true, "kv_get": true, "kv_list": true, "fact_recall": true,
	"plan_read": true, "scheduler_list": true, "schedule_runs_query": true, "scheduler_next": true,
	"capabilities_list": true, "tool_search": true, "tool_lookup": true, "now": true,
	"system_health": true, "manifest_inspect": true, "persona_get": true,
	"web_search": true, "web_archive": true, "webfetch": true, "html_extract": true, "pdf_read": true,
	"market_quote": true, "scanner_findings_query": true, "scanner_runs_query": true,
	"skill_search": true, "decision_search": true, "mistake_search": true, "audit_search": true,
	"stat_summary": true, "interaction_recall": true, "edu_error_lookup": true, "karma_query": true,
	// surface-vocabulary read-only tools (claude_tools.go)
	"TaskOutput": true, "StructuredOutput": true,
	// F-G checkpointing (file_checkpoint.go)
	"file_checkpoints": true,
}

func init() {
	tools.ReadOnlyClassifier = func(name string, args map[string]any) bool {
		switch name {
		case "shell", "bash":
			// per-call: read-only iff the command structure is observe-only (cmdsem).
			if cmd, ok := args["command"].(string); ok {
				_, _, ro := classifyCommand(cmd)
				return ro
			}
			return false
		}
		return readOnlyTools[name]
	}
	tools.ExtraGatePolicy = approvalGatePolicy
	tools.RegisterInterceptor(approvalModePerAgent{})
}

// ── F-B: gerbang approval interaktif (mode ala Claude Code) ─────────────────
//
// Mode GLOBAL via switch GUI FLOWORK_APPROVAL_MODE (fwswitch → os.Setenv host,
// live). requiresApproval (sandbox_v3, frozen) udah exempt read-only + gate
// sensitive-args/tools DULUAN; policy ini nambah lapisan mode di atasnya:
//
//	default     — tanya (approval queue) SEBELUM aksi destruktif: shell yang
//	              MUTASI, termasuk `git push`/`commit` (cmdsem sadar-subcommand).
//	              File-tool workspace TIDAK di-gate (udah disandbox per-agent).
//	acceptEdits — alias default di Flowork (edit file emang udah auto-allow).
//	plan        — SEMUA call non-read-only butuh approval owner (read-only tetap jalan).
//	bypass      — (DEFAULT) tanpa gerbang interaktif; keamanan mandiri tetap aktif
//	              (protector/cmdsem/caps/sandbox/ARM).
//
// system_power SENGAJA ga diurus di sini — udah punya gerbang sendiri
// (cap exec:power + ARM switch + FLOWORK_POWER_REQUIRE_APPROVAL).
// Approval yang di-enqueue diputus owner via GUI/endpoint:
// GET /api/agents/protector/approval/queue · POST .../approve_pending · .../reject_pending
// (approved match by tool+args_hash, berlaku 1 jam).
// FALLBACK = "bypass" (owner 2026-07-02: "Flowork sebebas mungkin — dia harus
// mandiri termasuk keamanan"): evolusi ga boleh nunggu manusia; keamanan mandiri
// dipegang lapisan DETERMINISTIK yang tetap aktif (protector baseline immutable,
// cmdsem structural block, caps, workspace sandbox, ARM power). Mode interaktif
// (default/plan) = OPT-IN buat fase yang owner mau awasi ketat.
func approvalMode() string {
	switch m := strings.ToLower(strings.TrimSpace(os.Getenv("FLOWORK_APPROVAL_MODE"))); m {
	case "plan", "acceptedits", "default":
		return m
	default:
		return "bypass"
	}
}

func approvalGatePolicy(name string, args map[string]any) bool {
	switch approvalMode() {
	case "bypass":
		return false
	case "plan":
		// read-only udah di-exempt duluan di requiresApproval → sisanya = mutasi.
		return true
	default: // "default" / "acceptedits": gate aksi destruktif (shell mutasi + git push)
		if name != "bash" && name != "shell" {
			return false
		}
		cmd, _ := args["command"].(string)
		blocked, _, ro := classifyCommand(cmd)
		// blocked bakal ditolak classifier-nya sendiri — jangan buang klik owner.
		return !blocked && !ro
	}
}

// approvalModePerAgent — mode per-agent via config agent (kv `approval_mode`,
// diedit dari panel Agent Brain GUI). HANYA bisa MEMPERKETAT (nilai "plan" =
// agent read-only); relaksasi per-agent SENGAJA ga ada (downgrade keamanan
// per-agent = celah). Interceptor jalan di SandboxRunV2 → ctx punya store.
type approvalModePerAgent struct{}

func (approvalModePerAgent) Name() string { return "approval-mode-agent" }

func (approvalModePerAgent) Before(ctx context.Context, t tools.Tool, args map[string]any) error {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return nil
	}
	cfg, err := store.Load()
	if err != nil {
		return nil
	}
	mode, _ := cfg["approval_mode"].(string)
	if !strings.EqualFold(strings.TrimSpace(mode), "plan") {
		return nil
	}
	if tools.ReadOnlyClassifier != nil && tools.ReadOnlyClassifier(t.Name(), args) {
		return nil
	}
	return fmt.Errorf("[PETUNJUK, bukan salahmu] agent ini lagi MODE PLAN (read-only) — aksi yang MENGUBAH ditahan, ini aturan dari owner. Susun rencana + laporkan hasil analisamu; kalau eksekusi beneran perlu, minta owner ubah approval_mode agent ini di GUI")
}
