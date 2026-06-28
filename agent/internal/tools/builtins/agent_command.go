// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"context"
	"fmt"
	"strings"

	"flowork-gui/internal/tools"
)

// ⚠️ TETAP FROZEN: file ini nyimpen var seam `InvokeAgentFunc` yg DIPAKAI frozen-core (main.go).
// Jadi BUKAN plug-in murni (delete-test gagal kalau dihapus). Tool-nya didaftar di builtins.go Init().
// (Kalau mau jadi plug-in: seam-split — pindah InvokeAgentFunc ke file seam beku, tool ke file lain.)
var InvokeAgentFunc func(ctx context.Context, agentID, text, caller string) (string, error)

type agentCommandTool struct{}

func (agentCommandTool) Name() string       { return "agent_command" }
func (agentCommandTool) Capability() string { return "rpc:agent-invoke" }
func (agentCommandTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Delegate a natural-language command to a specialist agent and get its reply back. Use this to ROUTE a request to an agent that owns the right tools/persona instead of doing it yourself. NOTE: computer power/control (shutdown, restart, sleep, lock, logout) and opening apps are FIRST-CLASS tools now — use system_power / app_open directly, do NOT delegate those. This tool is for genuine specialist delegation. Pass the request through as text; relay the reply to the user verbatim.",
		Params: []tools.Param{
			{Name: "agent_id", Type: tools.ParamString, Description: "target specialist agent id", Required: true},
			{Name: "text", Type: tools.ParamString, Description: "the command / request in natural language", Required: true},
		},
		Returns: "{agent_id, reply}",
	}
}

func (agentCommandTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	target, _ := args["agent_id"].(string)
	target = strings.TrimSpace(target)
	text, _ := args["text"].(string)
	text = strings.TrimSpace(text)
	if target == "" || text == "" {
		return tools.Result{}, fmt.Errorf("agent_command: agent_id and text are required")
	}
	if InvokeAgentFunc == nil {
		return tools.Result{}, fmt.Errorf("agent_command: host invoke hook not wired")
	}

	if from := tools.FromAgent(ctx); from != "" && strings.EqualFold(from, target) {
		return tools.Result{}, fmt.Errorf("agent_command: cannot delegate to self (%s)", target)
	}

	caller := "delegate:" + tools.FromCaller(ctx)
	reply, err := InvokeAgentFunc(ctx, target, text, caller)
	if err != nil {
		return tools.Result{}, fmt.Errorf("agent_command: invoke %q failed: %w", target, err)
	}
	return tools.Result{Output: map[string]any{
		"agent_id": target,
		"reply":    reply,
	}}, nil
}
