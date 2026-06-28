// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"flowork-gui/internal/tools"
)

func Init() {
	// Primitive sederhana — tetep di anchor beku ini.
	tools.Register(&echoTool{})
	tools.Register(&nowTool{})
	tools.Register(&memGetTool{})
	tools.Register(&memSetTool{})
	tools.Register(&memDelTool{})

	// File BEKU yg nyimpen seam/helper dipakai frozen-core → tool-nya daftar di sini:
	//   agent_command.go (InvokeAgentFunc), shell.go (capWriter/scrubEnv/shellDenyPatterns),
	//   file.go (validateCategoryAndName), web.go (isBlockedIP), telegram.go (telegramAPIBase).
	tools.Register(&agentCommandTool{})
	tools.Register(&bashTool{})
	tools.Register(&fileReadTool{})
	tools.Register(&fileWriteTool{})
	tools.Register(&fileListTool{})
	tools.Register(&webFetchTool{})
	tools.Register(&telegramSendTool{})

	// Tool PLUG-IN (NON-frozen) SELF-REGISTER via init() di file masing2 — edit/hapus/tambah
	// TANPA buka freeze: file_advanced.go, skill.go, skill_suggest.go, skill_author.go,
	//   taskflow_tools.go, orchestration.go, web_research.go, git.go, system_power.go, app_open.go.
	// 📄 lock/tool-manager.md
}

type echoTool struct{}

func (echoTool) Name() string       { return "echo" }
func (echoTool) Capability() string { return "" }
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

type nowTool struct{}

func (nowTool) Name() string       { return "now" }
func (nowTool) Capability() string { return "time:read" }
func (nowTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Waktu sekarang LIVE: UTC (rfc3339) + waktu lokal default WIB (UTC+7). Pakai 'local' buat tanggal/jam terkini (anti berita-basi).",
		Params:      nil,
		Returns:     "{rfc3339: '<UTC>', unix_ms: <int>, local: 'YYYY-MM-DD HH:MM:SS', tz_label: 'WIB', tz_offset_hours: 7}",
	}
}

func tzOffsetHoursEnv() int {
	if v := strings.TrimSpace(os.Getenv("FLOWORK_TZ_OFFSET_HOURS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= -12 && n <= 14 {
			return n
		}
	}
	return 7
}
func tzLabelEnv() string {
	if v := strings.TrimSpace(os.Getenv("FLOWORK_TZ_LABEL")); v != "" {
		return v
	}
	return "WIB"
}
func (nowTool) Run(_ context.Context, _ map[string]any) (tools.Result, error) {
	t := time.Now().UTC()
	off := tzOffsetHoursEnv()
	local := t.Add(time.Duration(off) * time.Hour)
	return tools.Result{
		Output: map[string]any{
			"rfc3339":         t.Format(time.RFC3339),
			"unix_ms":         t.UnixMilli(),
			"local":           local.Format("2006-01-02 15:04:05"),
			"tz_label":        tzLabelEnv(),
			"tz_offset_hours": off,
		},
	}, nil
}

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
