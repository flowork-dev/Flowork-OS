// permission_policy.go — read-only classification for the approval gate (P2).
//
// This is the NON-FROZEN side of the plug-and-play permission hook: it registers
// tools.ReadOnlyClassifier so the (locked) sandbox knows which calls only read/observe
// and can exempt them from owner approval — without ever editing the locked file. To
// add or change permission policy in the future, edit THIS file (or add another that
// sets tools.ExtraGatePolicy); do NOT unlock sandbox_v3.go.
package builtins

import "flowork-gui/internal/tools"

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
}
