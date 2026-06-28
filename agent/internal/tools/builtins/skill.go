// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"flowork-gui/internal/routerclient"
	"flowork-gui/internal/tools"
)

type skillTool struct{}

func (skillTool) Name() string       { return "skill" }
func (skillTool) Capability() string { return "rpc:router:skill" }
func (skillTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Retrieve full skill (name+description+body markdown) dari Router brain catalog. Caller treat body sebagai system-prompt-style instruction. Timeout 10s. Body cap 256KB.",
		Params: []tools.Param{
			{Name: "name", Type: tools.ParamString, Description: "skill name (case-sensitive)", Required: true},
		},
		Returns: "{name, description, body}",
	}
}

func (skillTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return tools.Result{}, fmt.Errorf("name required")
	}
	client := routerClientFromCtx()
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var doc routerclient.SkillDoc
	rerr := routerclient.WithRetry(runCtx, routerclient.DefaultRetry(),
		func(ctx context.Context) error {
			var ierr error
			doc, ierr = client.GetSkill(ctx, name)
			return ierr
		})
	if rerr != nil {
		return tools.Result{}, fmt.Errorf("get skill: %w", rerr)
	}
	return tools.Result{Output: map[string]any{
		"name":        doc.Name,
		"description": doc.Description,
		"body":        doc.Body,
	}}, nil
}

type skillSearchTool struct{}

func (skillSearchTool) Name() string       { return "skill_search" }
func (skillSearchTool) Capability() string { return "rpc:router:skill" }
func (skillSearchTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Search skill catalog di Router brain. Search optional (kosong=top of catalog). Returns summary (name + description) cap 10 per call (Router anti over-prompt).",
		Params: []tools.Param{
			{Name: "search", Type: tools.ParamString, Description: "optional search query"},
			{Name: "limit", Type: tools.ParamInt, Description: "default 10, max 10"},
		},
		Returns: "{items: [{name, description}], count, total}",
	}
}

func (skillSearchTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	search, _ := args["search"].(string)
	limit := 10
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	if vs, ok := args["limit"].(int); ok && vs > 0 {
		limit = vs
	}
	if limit > 10 {
		limit = 10
	}
	client := routerClientFromCtx()
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var resp routerclient.SkillListResp
	rerr := routerclient.WithRetry(runCtx, routerclient.DefaultRetry(),
		func(ctx context.Context) error {
			var ierr error
			resp, ierr = client.ListSkills(ctx, search, limit)
			return ierr
		})
	if rerr != nil {
		return tools.Result{}, fmt.Errorf("list skills: %w", rerr)
	}
	items := make([]map[string]any, 0, len(resp.Items))
	for _, s := range resp.Items {
		items = append(items, map[string]any{
			"name":        s.Name,
			"description": s.Description,
		})
	}
	return tools.Result{Output: map[string]any{
		"items": items,
		"count": resp.Count,
		"total": resp.Total,
	}}, nil
}

func routerClientFromCtx() *routerclient.Client {

	return routerclient.NewFromAgentURL("")
}

// SELF-REGISTER plug-in (📄 lock/tool-manager.md) — edit/hapus/tambah BEBAS tanpa unfreeze.
func init() {
	tools.Register(&skillTool{})
	tools.Register(&skillSearchTool{})
}
