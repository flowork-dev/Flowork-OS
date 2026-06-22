// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Section 11 phase 1a (5 demo tools) DONE. API stable: Init()
//   register echo, now, memory_get, memory_set, memory_delete.
//   Phase 1b+ add real tools (read_file/write_file/bash_run/web_fetch/
//   brain_search/telegram_send/etc) → tambah file baru di package ini
//   (mis. `file.go`, `web.go`), JANGAN modify ini. Each new tool needs
//   ke-register di Init().
//
// Package builtins — Section 11 phase 1a: 5 demo tools yang prove the
// Tool foundation pattern end-to-end.
//
// Tools:
//   1. echo            — return input message (capability: none)
//   2. now             — return current UTC timestamp (capability: time:read)
//   3. memory_get      — read tool_memory by key (capability: state:read)
//   4. memory_set      — write tool_memory (capability: state:write)
//   5. memory_delete   — delete tool_memory entry (capability: state:write)
//
// Wiring: dispatcher (di agentmgr) panggil tools.WithStore(ctx, store) →
// builtins extract via tools.FromStore.
//
// Source: Flowork_Agent/roadmap.md Section 11 phase 1a.

package builtins

import (
	"context"
	"fmt"
	"time"

	"flowork-gui/internal/tools"
)

// Init — explicit bootstrap call dari main.go. Tidak pakai init() supaya
// register sequence eksplisit (caller pilih kapan tools available).
//
// Idempotent: panic kalau dipanggil 2x karena Registry.Register panics on
// duplicate. Caller wajib panggil exactly once at boot.
func Init() {
	tools.Register(&echoTool{})
	tools.Register(&nowTool{})
	tools.Register(&memGetTool{})
	tools.Register(&memSetTool{})
	tools.Register(&memDelTool{})
	// phase 1b: file ops (file.go)
	tools.Register(&fileReadTool{})
	tools.Register(&fileWriteTool{})
	tools.Register(&fileListTool{})
	// BRAIN-PATH tools (brain_search/add/local/get, graph_recall, mistake_recall, immune
	// scan/verify, federation promote, codemap) DICABUT ke builtins_brain.go (FROZEN, init
	// self-register) — jalur brain diabadikan, ga bisa ke-drop diam-diam. Lihat lock/brain.md.
	// Skill-author tools (impl editable: skill_suggest.go/skill_author.go) tetap di sini:
	tools.Register(&skillSuggestTool{})
	tools.Register(&skillAuthorTool{})
	// phase 1f: comms (telegram.go)
	tools.Register(&telegramSendTool{})
	// phase 1d: web (web.go)
	tools.Register(&webFetchTool{})
	// FASE 3: tools riset (web_research.go) — anti ngarang sumber
	tools.Register(&webSearchTool{})
	tools.Register(&webArchiveTool{})
	tools.Register(&htmlExtractTool{})
	tools.Register(&pdfReadTool{})
	// FASE 6: Mr.Flow jadi router — list + trigger Category Task dari chat.
	tools.Register(&taskListTool{})
	tools.Register(&taskRunTool{})
	// phase 1c: shell (shell.go) — bash exec dengan denylist + timeout
	tools.Register(&bashTool{})
	// P1 file ops (file_advanced.go) — edit + glob + grep
	tools.Register(&editTool{})
	tools.Register(&globTool{})
	tools.Register(&grepTool{})
	// P1 vcs (git.go) — status/diff/log/show
	tools.Register(&gitTool{})
	// P1 skill (skill.go) — Router skill catalog client tool
	tools.Register(&skillTool{})
	tools.Register(&skillSearchTool{})
	// phase 1g: orchestration (orchestration.go) — plan/todo/goal
	tools.Register(&planReadTool{})
	tools.Register(&planWriteTool{})
	tools.Register(&todoTool{})
	tools.Register(&goalDoneTool{})
	// codemap query tools (codemap_tools.go) → builtins_brain.go (FROZEN, init self-register).
	// Operator: host power control (system_power.go) — cap exec:power, ARM-gated.
	tools.Register(&systemPowerTool{})
	// Operator: launch whitelisted desktop apps (app_open.go) — cap exec:app.
	tools.Register(&appOpenTool{})
	// Router delegation (agent_command.go) — cap rpc:agent-invoke, Mr.Flow only.
	tools.Register(&agentCommandTool{})
}

// =============================================================================
// 1. echo — return input message
// =============================================================================

type echoTool struct{}

func (echoTool) Name() string       { return "echo" }
func (echoTool) Capability() string { return "" } // no capability needed
func (echoTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Echo back the input message. Demo tool — verifies dispatcher wiring.",
		Params: []tools.Param{
			{Name: "message", Type: tools.ParamString, Description: "text to echo", Required: true},
		},
		Returns: "{message: <input>}",
	}
}
func (echoTool) Run(_ context.Context, args map[string]any) (tools.Result, error) {
	msg, _ := args["message"].(string)
	if msg == "" {
		return tools.Result{}, fmt.Errorf("message required")
	}
	return tools.Result{Output: map[string]any{"message": msg}}, nil
}

// =============================================================================
// 2. now — current UTC timestamp
// =============================================================================

type nowTool struct{}

func (nowTool) Name() string       { return "now" }
func (nowTool) Capability() string { return "time:read" }
func (nowTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Return current UTC timestamp (RFC3339 + unix ms).",
		Params:      nil, // no params
		Returns:     "{rfc3339: '...', unix_ms: <int>}",
	}
}
func (nowTool) Run(_ context.Context, _ map[string]any) (tools.Result, error) {
	t := time.Now().UTC()
	return tools.Result{
		Output: map[string]any{
			"rfc3339": t.Format(time.RFC3339),
			"unix_ms": t.UnixMilli(),
		},
	}, nil
}

// =============================================================================
// 3. memory_get — read tool_memory by key
// =============================================================================

type memGetTool struct{}

func (memGetTool) Name() string       { return "memory_get" }
func (memGetTool) Capability() string { return "state:read" }
func (memGetTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Read value from tool memory by key. Returns null kalau key ngga ada.",
		Params: []tools.Param{
			{Name: "key", Type: tools.ParamString, Description: "memory key", Required: true},
		},
		Returns: "{key, value, found: bool}",
	}
}
func (memGetTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	key, _ := args["key"].(string)
	if key == "" {
		return tools.Result{}, fmt.Errorf("key required")
	}
	v, found, err := store.GetToolMemory(key)
	if err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: map[string]any{
		"key":   key,
		"value": v,
		"found": found,
	}}, nil
}

// =============================================================================
// 4. memory_set — upsert tool_memory
// =============================================================================

type memSetTool struct{}

func (memSetTool) Name() string       { return "memory_set" }
func (memSetTool) Capability() string { return "state:write" }
func (memSetTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Write or update tool memory by key. Value cap 32KB.",
		Params: []tools.Param{
			{Name: "key", Type: tools.ParamString, Description: "memory key", Required: true},
			{Name: "value", Type: tools.ParamString, Description: "value string", Required: true},
		},
		Returns: "{key, ok: true}",
	}
}
func (memSetTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	key, _ := args["key"].(string)
	val, _ := args["value"].(string)
	if key == "" || val == "" {
		return tools.Result{}, fmt.Errorf("key + value required")
	}
	if err := store.SetToolMemory(key, val); err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: map[string]any{"key": key, "ok": true}}, nil
}

// =============================================================================
// 5. memory_delete — remove tool_memory entry
// =============================================================================

type memDelTool struct{}

func (memDelTool) Name() string       { return "memory_delete" }
func (memDelTool) Capability() string { return "state:write" }
func (memDelTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Delete tool memory entry by key. Return deleted bool.",
		Params: []tools.Param{
			{Name: "key", Type: tools.ParamString, Description: "memory key", Required: true},
		},
		Returns: "{key, deleted: bool}",
	}
}
func (memDelTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("agent store not in context")
	}
	key, _ := args["key"].(string)
	if key == "" {
		return tools.Result{}, fmt.Errorf("key required")
	}
	n, err := store.DelToolMemory(key)
	if err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: map[string]any{"key": key, "deleted": n > 0}}, nil
}
