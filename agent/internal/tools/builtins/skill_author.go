// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"flowork-gui/internal/tools"
)

var skillDangerRe = regexp.MustCompile(`(?i)(\brm\s+-rf|\bmkfs\b|:\(\)\s*\{|\bdd\s+if=|\bchmod\s+\+?s\b|\bsetuid\b|/etc/(passwd|shadow)|169\.254\.169\.254|\bcurl\s+[^|]*\|\s*(sh|bash)|\bwget\s+[^|]*\|\s*(sh|bash))`)

var skillInjectRe = regexp.MustCompile(`(?i)(ignore\s+(all\s+)?previous|disregard\s+(all\s+)?(previous\s+)?instructions|reveal\s+(your\s+)?(system\s+)?prompt|abaikan\s+(instruksi|perintah)\s+sebelum|bocorkan\s+system\s+prompt|developer\s+mode|do\s+anything\s+now)`)

type skillAuthorTool struct{}

func (skillAuthorTool) Name() string       { return "skill_author" }
func (skillAuthorTool) Capability() string { return "state:write" }
func (skillAuthorTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Self-author a reusable skill distilled from your own experience. The skill is VETTED (immune + verifier gate) before it can be saved — dangerous or injection-laden skills are BLOCKED, never stored. Use after you solve something worth remembering as a repeatable procedure.",
		Params: []tools.Param{
			{Name: "id", Type: tools.ParamString, Description: "Skill ID (snake_case)", Required: true},
			{Name: "instructions", Type: tools.ParamString, Description: "The repeatable procedure to remember (what to do, step by step)", Required: true},
			{Name: "trigger", Type: tools.ParamString, Description: "When this skill applies (e.g. '#deploy', 'fix flaky test')", Required: false},
			{Name: "experience", Type: tools.ParamString, Description: "Provenance: the experience you distilled this from (for audit)", Required: false},
		},
		Returns: "{ok, gate, id} on save — or {ok:false, blocked:true, reason, flags} when the gate rejects it",
	}
}

func gateSkill(text string) []string {
	var flags []string
	for _, m := range skillDangerRe.FindAllString(text, -1) {
		flags = append(flags, "dangerous: "+strings.TrimSpace(m))
	}
	if m := skillInjectRe.FindString(text); m != "" {
		flags = append(flags, "injection: "+strings.TrimSpace(m))
	}
	return flags
}

func (skillAuthorTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok || store == nil {
		return tools.Result{}, fmt.Errorf("agent store not available")
	}
	id, _ := args["id"].(string)
	instructions, _ := args["instructions"].(string)
	trigger, _ := args["trigger"].(string)
	experience, _ := args["experience"].(string)
	id = strings.TrimSpace(id)
	if id == "" || strings.TrimSpace(instructions) == "" {
		return tools.Result{}, fmt.Errorf("id + instructions required")
	}

	if flags := gateSkill(instructions + "\n" + trigger + "\n" + experience); len(flags) > 0 {
		return tools.Result{
			Output: map[string]any{
				"ok":      false,
				"blocked": true,
				"id":      id,
				"reason":  "skill rejected by immune+verifier gate (can't self-poison)",
				"flags":   flags,
			},
			Note: "skill BLOCKED — not stored",
		}, nil
	}

	cfg, err := store.Load()
	if err != nil {
		return tools.Result{}, fmt.Errorf("load: %w", err)
	}
	skillsAny, _ := cfg["skills"].([]any)
	filtered := make([]any, 0, len(skillsAny)+1)
	for _, s := range skillsAny {
		m, _ := s.(map[string]any)
		if m == nil {
			continue
		}
		if existID, _ := m["id"].(string); existID == id {
			continue
		}
		filtered = append(filtered, s)
	}
	entry := map[string]any{"id": id, "trigger": trigger, "instructions": instructions, "authored": true}
	if experience != "" {
		entry["source"] = experience
	}
	filtered = append(filtered, entry)
	cfg["skills"] = filtered
	if err := store.Save(cfg); err != nil {
		return tools.Result{}, fmt.Errorf("save: %w", err)
	}
	return tools.Result{
		Output: map[string]any{"ok": true, "gate": "passed", "id": id, "total_skills": len(filtered)},
		Note:   "skill vetted + saved (recallable)",
	}, nil
}

// SELF-REGISTER plug-in (📄 lock/tool-manager.md) — edit/hapus/tambah BEBAS tanpa unfreeze.
func init() {
	tools.Register(&skillAuthorTool{})
}
