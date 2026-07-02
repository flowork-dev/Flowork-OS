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

func init() {
	tools.Register(&karmaSetTool{})
	tools.Register(&kvGetTool{})
	tools.Register(&kvSetTool{})
	tools.Register(&manifestInspectTool{})
	tools.Register(&toolLookupTool{})
	tools.Register(&toolSearchTool{})
}

type karmaSetTool struct{}

func (karmaSetTool) Name() string       { return "karma_set" }
func (karmaSetTool) Capability() string { return "state:write" }
func (karmaSetTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Update karma metric: op=increment (delta) atau op=average (sample). Counter atau moving average per Section 5.",
		Params: []tools.Param{
			{Name: "key", Type: tools.ParamString, Description: "Metric key (mis. success_count, avg_response_ms)", Required: true},
			{Name: "op", Type: tools.ParamString, Description: "increment|average", Required: true},
			{Name: "value", Type: tools.ParamFloat, Description: "Delta for increment, sample for average", Required: true},
		},
		Returns: "{ok, current}",
	}
}

func (karmaSetTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	key, _ := args["key"].(string)
	op, _ := args["op"].(string)
	value, _ := args["value"].(float64)
	if key == "" || op == "" {
		return tools.Result{}, fmt.Errorf("key + op required")
	}
	var current float64
	var err error
	switch op {
	case "increment":
		current, err = store.IncrementKarma(key, value)
	case "average":
		current, err = store.AverageUpdateKarma(key, value)
	default:
		return tools.Result{}, fmt.Errorf("op must be increment|average")
	}
	if err != nil {
		return tools.Result{}, fmt.Errorf("update karma: %w", err)
	}
	return tools.Result{
		Output: map[string]any{"ok": true, "current": current},
	}, nil
}

type kvGetTool struct{}

func (kvGetTool) Name() string       { return "kv_get" }
func (kvGetTool) Capability() string { return "state:read" }
func (kvGetTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Read kv table value by key. Diferentiate dari tool_memory (key dedicated namespace).",
		Params: []tools.Param{
			{Name: "key", Type: tools.ParamString, Description: "KV key", Required: true},
		},
		Returns: "{key, value, found}",
	}
}

func (kvGetTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	key, _ := args["key"].(string)
	if key == "" {
		return tools.Result{}, fmt.Errorf("key required")
	}
	cfg, err := store.Load()
	if err != nil {
		return tools.Result{}, fmt.Errorf("load: %w", err)
	}
	kv, _ := cfg["kv"].(map[string]any)
	val, found := kv[key]
	return tools.Result{
		Output: map[string]any{"key": key, "value": val, "found": found},
	}, nil
}

type kvSetTool struct{}

func (kvSetTool) Name() string       { return "kv_set" }
func (kvSetTool) Capability() string { return "state:write" }
func (kvSetTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Write single kv key. Reserved keys (prompt/router_url/router_model) skip — pakai cara dedicated.",
		Params: []tools.Param{
			{Name: "key", Type: tools.ParamString, Description: "KV key (non-reserved)", Required: true},
			{Name: "value", Type: tools.ParamString, Description: "Value string", Required: true},
		},
		Returns: "{ok}",
	}
}

func (kvSetTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)
	if key == "" {
		return tools.Result{}, fmt.Errorf("key required")
	}
	if key == "prompt" || key == "router_url" || key == "router_model" {
		return tools.Result{}, fmt.Errorf("reserved key %q (pakai cara dedicated)", key)
	}

	cfg := map[string]any{"kv": map[string]any{key: value}}
	if err := store.Save(cfg); err != nil {
		return tools.Result{}, fmt.Errorf("save kv: %w", err)
	}
	return tools.Result{
		Output: map[string]any{"ok": true, "key": key},
	}, nil
}

type manifestInspectTool struct{}

func (manifestInspectTool) Name() string       { return "manifest_inspect" }
func (manifestInspectTool) Capability() string { return "state:read" }
func (manifestInspectTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Inspect own manifest config snapshot — list KV entries summary, total schedule count, total skills count, secret keys (no values).",
		Params:      []tools.Param{},
		Returns:     "{kv_keys, schedule_count, skills_count, secret_keys[], meta}",
	}
}

func (manifestInspectTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	cfg, err := store.Load()
	if err != nil {
		return tools.Result{}, fmt.Errorf("load cfg: %w", err)
	}
	kv, _ := cfg["kv"].(map[string]any)
	kvKeys := make([]string, 0, len(kv))
	for k := range kv {
		kvKeys = append(kvKeys, k)
	}
	schedule, _ := cfg["schedule"].([]any)
	skills, _ := cfg["skills"].([]any)
	secrets, _ := cfg["secrets"].(map[string]any)
	secretKeys := make([]string, 0, len(secrets))
	for k := range secrets {
		secretKeys = append(secretKeys, k)
	}
	meta, _ := cfg["meta"].(map[string]any)
	return tools.Result{
		Output: map[string]any{
			"kv_keys":        kvKeys,
			"schedule_count": len(schedule),
			"skills_count":   len(skills),
			"secret_keys":    secretKeys,
			"meta":           meta,
		},
	}, nil
}

type toolLookupTool struct{}

func (toolLookupTool) Name() string       { return "tool_lookup" }
func (toolLookupTool) Capability() string { return "state:read" }
func (toolLookupTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Lookup single tool by name. Return Name + Capability + Schema (description+params+returns). Anti over-prompt: ngga return full catalog.",
		Params: []tools.Param{
			{Name: "name", Type: tools.ParamString, Description: "Tool name", Required: true},
		},
		Returns: "{found, name, capability, schema}",
	}
}

func (toolLookupTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{}, fmt.Errorf("name required")
	}
	tool, ok := tools.Lookup(name)
	if !ok {
		return tools.Result{
			Output: map[string]any{"found": false, "name": name},
		}, nil
	}
	return tools.Result{
		Output: map[string]any{
			"found":      true,
			"name":       tool.Name(),
			"capability": tool.Capability(),
			"schema":     tool.Schema(),
			// Fix defer #2C (2026-07-02): tanpa arahan eksplisit, model abis lookup malah
			// ngerakit tool baru (tool_create) padahal tool-nya UDAH aktif → boros iterasi.
			"note": "Tool ini AKTIF & siap — panggil `" + tools.DisplayName(tool.Name()) +
				"` LANGSUNG di langkah berikutnya pakai parameter schema di atas. " +
				"JANGAN bikin tool baru (tool_create) untuk kebutuhan ini.",
		},
	}, nil
}

type toolSearchTool struct{}

func (toolSearchTool) Name() string       { return "tool_search" }
func (toolSearchTool) Capability() string { return "state:read" }
func (toolSearchTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Search tool registry by substring di name/capability/description. Anti over-prompt: cap 10 hit. Pakai tool_lookup untuk detail spesifik.",
		Params: []tools.Param{
			{Name: "query", Type: tools.ParamString, Description: "Substring (case-insensitive)", Required: true},
		},
		Returns: "{count, hits[]}",
	}
}

func (toolSearchTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	query, _ := args["query"].(string)
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return tools.Result{}, fmt.Errorf("query required")
	}
	summaries := tools.ListSummaries()
	hits := []map[string]any{}
	for _, s := range summaries {
		if strings.Contains(strings.ToLower(s.Name), query) ||
			strings.Contains(strings.ToLower(s.Capability), query) ||
			strings.Contains(strings.ToLower(s.Description), query) {
			hits = append(hits, map[string]any{
				"name":        s.Name,
				"capability":  s.Capability,
				"description": s.Description,
			})
			if len(hits) >= 10 {
				break
			}
		}
	}
	return tools.Result{
		Output: map[string]any{"count": len(hits), "hits": hits},
	}, nil
}
