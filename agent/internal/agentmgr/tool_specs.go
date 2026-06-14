// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-02
// Reason: Fase 0 — endpoint OpenAI function-schema (core ~13 + manual subs,
//   cap 25, BUKAN 106 = anti over-prompt). E2E verified (Mr.Flow file_write+read).
//
// tool_specs.go — Fase 0 (tool-calling loop): endpoint yang balikin tools yang
// di-EXPOSE ke LLM dalam format OpenAI function-schema. Host yang bangun schema
// (punya registry + subscription); WASM agent tinggal fetch + forward ke LLM.
//
// ANTI OVER-PROMPT (akar refactor 11×): yang di-expose CUMA core set (~13),
// BUKAN 106. Sisanya tetep di registry, dipanggil via `tool_search` on-demand.
// Fase 2: per-agent exposed selection (sekarang core + tool yang di-subscribe
// MANUAL via popup).

package agentmgr

import (
	"net/http"
	"strings"

	"flowork-gui/internal/httpx"
	"flowork-gui/internal/tools"
)

// coreExposedTools — SELALU di-expose ke LLM (cover kebutuhan umum). Kecil =
// prompt kecil. Sisanya via tool_search.
// Note: exec is NOT in the always-on core — an agent that needs a shell subscribes
// to `shell` (the hardened, semantics-classified exec tool, P1). The old `bash`
// (substring denylist) stays registered for back-compat but is opt-in only, so a
// capable agent gets the safer one and the ants (no subscriptions) get no shell.
var coreExposedTools = []string{
	"file_read", "file_write", "file_list", "grep", "glob",
	"webfetch", "brain_search", "memory_get", "memory_set",
	"telegram_send", "tool_search", "now",
}

// maxExposedTools caps how many tool schemas an agent offers its LLM at once. A
// capable agent (mr-flow holds ~40 first-class tools via subscriptions) needs the
// higher ceiling; ants stay tiny because they have no subscriptions (core set only),
// so raising the ceiling never bloats them.
const maxExposedTools = 50

// primaryExtraTools — surface-vocabulary tools exposed ONLY to the primary
// orchestrator (mr-flow), not to ants. These cover shell/task-lifecycle/schedule/
// structured-output/orchestration that a coordinator needs. Kept off the ants'
// core set so ant prompts stay tiny (the over-prompt guard that drove the refactor).
var primaryExtraTools = []string{
	"PowerShell", "TaskCreate", "TaskUpdate", "TaskStop", "TaskOutput",
	"ScheduleWakeup", "Monitor", "SendUserFile", "StructuredOutput", "Workflow",
}

// ToolSpecsHandler — GET /api/agents/tools/specs?id=<agent>
// Return {tools: [<openai function schema>...], count}. Loopback-only (dipanggil
// WASM agent sendiri).
func ToolSpecsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		id = defaultAgentID
	}

	isPrimary := IsPrimaryAgent(id)
	exposed := map[string]bool{}
	ordered := []string{}
	add := func(n string) {
		if exposed[n] || len(ordered) >= maxExposedTools {
			return
		}
		// Tier gate: tool primary-only (brain 5jt shared) ga di-expose ke
		// extension — brain-nya folder sendiri (brain_search lokal).
		if IsPrimaryOnlyTool(n) && !isPrimary {
			return
		}
		if _, ok := tools.Lookup(n); ok {
			exposed[n] = true
			ordered = append(ordered, n)
		}
	}
	// 1. core set (selalu).
	for _, n := range coreExposedTools {
		add(n)
	}
	// 1b. primary-only surface tools (coordinator vocabulary). Ants skip these.
	if isPrimary {
		for _, n := range primaryExtraTools {
			add(n)
		}
	}
	// 2. tool yang di-subscribe MANUAL (owner pilih di popup) — di luar default seed.
	if store, err := openAgentStore(id); err == nil {
		if subs, serr := store.ListSubscriptions(); serr == nil {
			for _, s := range subs {
				if s.Source != "" && !strings.EqualFold(s.Source, "default") {
					add(s.ToolName)
				}
			}
		}
		store.Close()
	}

	specs := make([]map[string]any, 0, len(ordered))
	for _, n := range ordered {
		if t, ok := tools.Lookup(n); ok {
			specs = append(specs, toOpenAIToolSchema(t))
		}
	}
	httpx.WriteJSON(w, map[string]any{"tools": specs, "count": len(specs)})
}

// toOpenAIToolSchema — konversi tools.Schema → OpenAI function-calling schema.
func toOpenAIToolSchema(t tools.Tool) map[string]any {
	sc := t.Schema()
	props := map[string]any{}
	required := []string{}
	for _, p := range sc.Params {
		props[p.Name] = map[string]any{
			"type":        jsonSchemaType(p.Type),
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        tools.DisplayName(t.Name()),
			"description": sc.Description,
			"parameters": map[string]any{
				"type":       "object",
				"properties": props,
				"required":   required,
			},
		},
	}
}

func jsonSchemaType(pt tools.ParamType) string {
	switch pt {
	case tools.ParamInt:
		return "integer"
	case tools.ParamFloat:
		return "number"
	case tools.ParamBool:
		return "boolean"
	case tools.ParamArray:
		return "array"
	case tools.ParamObject:
		return "object"
	default:
		return "string"
	}
}
