// === LOCKED FILE (soft) ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Locked at: 2026-06-15 (owner-approved autonomous sprint)
// Reason: Flowork Architect — group/team creator. VERIFIED E2E: POST /api/architect/build
//   {"prompt":"team peramal …"} → designed "Tim Peramal Nasib" → ONE pack (3 specialists +
//   1 lead synth, ALL group-prefixed "peramal-nasib-*") → installed → created group +
//   SyncToOrchestrator → coordinator loaded → /api/chat returned a real synthesized fortune.
//   2026-06-15 BUG-1 FIX: was assembling worker+synth per specialist (orphan synths polluted
//   EVERY group's member pool). Now ONE pack, every crew member used, agent ids group-prefixed
//   so the Groups GUI auto-claims them → no pollution (mirrors bundled investment/thinking).
//   One LLM call (design) + fast local assembly. Loopback-only, owner trust = /api/coder/*.
//
// architect.go — FLOWORK ARCHITECT: stand up a whole TEAM (group) from one natural
// prompt. "buatin team peramal" → ONE structured design call (Opus, forced tool)
// returns the full roster (every specialist's persona/directive + a lead) → the Go
// engine deterministically assembles + installs each agent with the SAME proven
// machinery that built the saham/crypto/primbon crews (coderAssemblePack →
// installPluginPack) → groupsapi.CreateGroup wires them into a coordinator group.
// Result: the team shows up in the GUI Group tab, is chattable via
// POST /api/chat {"agent":"<group_id>"} (the coordinator fans out to members and
// synthesizes), and its Telegram slash command auto-registers.
//
//	POST /api/architect/build {prompt|task, model?}  → design team → build agents → create group
//
// ONE LLM call (not N+2): a multi-agent team used to need a design call per member,
// and with a rate-limited upstream each call stalls ~90s on 429-retries → the whole
// build timed out. Designing the entire team in a single forced-tool call keeps the
// build to one upstream round-trip; everything after is fast local Go. Principle
// "agent bodoh, engine pinter": the LLM only fills the creative SPEC. Loopback-only,
// owner-gated (install auto-approves caps because this endpoint is reachable only
// from the trusted loopback — same trust model as /api/coder/*).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/kernelhost"
)

// idReGroup is tighter than groupsapi's idRe (2-40): a group_id here also becomes
// the lead category "<group_id>-lead", which must satisfy coderCatRe (max 31). Cap
// the group_id at 26 chars so "<group_id>-lead" never overflows.
var idReGroup = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,25}$`)

// teamWorker — one specialist (worker) in the team, fully specified by the design
// call. Only worker-side fields; the synth-side fields of its AgentSpec are filled
// with defaults at assembly (each specialist contributes its -worker to the group).
type teamWorker struct {
	CategoryID string `json:"category_id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	Role       string `json:"role"`
	Persona    string `json:"persona"`
	Directive  string `json:"directive"`
}

// teamLead — the synthesizer/lead that combines the workers' outputs.
type teamLead struct {
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Persona   string `json:"persona"`
	Directive string `json:"directive"`
}

// teamPlan — the Architect's complete design (one forced-tool call fills all of it).
type teamPlan struct {
	GroupID     string       `json:"group_id"`
	DisplayName string       `json:"display_name"`
	Task        string       `json:"task"`
	Specialists []teamWorker `json:"specialists"`
	Lead        teamLead     `json:"lead"`
}

// teamPlanSchema — the JSON-Schema for a full team design (group + specialists + lead),
// shared by the design_team forced-tool (one-shot endpoint) and the build_team chat tool
// (conversational brain) so the two never drift.
func teamPlanSchema() map[string]any {
	workerItem := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category_id": map[string]any{"type": "string", "description": "id slug unik specialist, lowercase-dash, 2-31 char (mis. 'primbon-jawa', 'zodiak')."},
			"name":        map[string]any{"type": "string", "description": "nama specialist human-readable (mis. 'Ahli Primbon Jawa')."},
			"icon":        map[string]any{"type": "string", "description": "1 emoji yang cocok."},
			"role":        map[string]any{"type": "string", "description": "label peran singkat (mis. 'penafsir weton')."},
			"persona":     map[string]any{"type": "string", "description": "persona/system-prompt specialist ini (keahlian + gaya). RINGKAS (1 keahlian fokus)."},
			"directive":   map[string]any{"type": "string", "description": "cara kerja specialist. Kalau KREATIF/tradisi (ga butuh data real) bilang itu tugasnya; kalau ANALISIS suruh cari data."},
		},
		"required": []string{"category_id", "name", "icon", "role", "persona", "directive"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"group_id":     map[string]any{"type": "string", "description": "id slug unik group, lowercase-dash, 2-26 char (mis. 'peramal', 'tim-kuliner')."},
			"display_name": map[string]any{"type": "string", "description": "nama tim human-readable (mis. 'Tim Peramal')."},
			"task":         map[string]any{"type": "string", "description": "instruksi kerja BERSAMA tim: apa yg tim hasilkan + cara koordinasi. SINGKAT."},
			"specialists":  map[string]any{"type": "array", "description": "2-4 specialist (worker) yg saling melengkapi.", "minItems": 2, "maxItems": 4, "items": workerItem},
			"lead": map[string]any{
				"type":        "object",
				"description": "lead/synthesizer yg gabungin output para specialist jadi 1 jawaban final.",
				"properties": map[string]any{
					"name":      map[string]any{"type": "string", "description": "nama lead (mis. 'Peramal Utama')."},
					"icon":      map[string]any{"type": "string", "description": "1 emoji."},
					"persona":   map[string]any{"type": "string", "description": "persona/system-prompt lead (perakit jawaban final). RINGKAS."},
					"directive": map[string]any{"type": "string", "description": "format output final: struktur + gaya. SINGKAT."},
				},
				"required": []string{"name", "icon", "persona", "directive"},
			},
		},
		"required": []string{"group_id", "display_name", "task", "specialists", "lead"},
	}
}

// architectDesignTeam — one Opus forced-tool call → the full team design. tool_choice
// is forced (no free-text hallucination; same pattern as coderDesignSpec).
func architectDesignTeam(ctx context.Context, prompt, model string) (teamPlan, error) {
	var plan teamPlan
	tool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "design_team",
			"description": "Rancang 1 TIM (group) Flowork LENGKAP dari permintaan user: 2-4 specialist (worker) yg saling melengkapi + 1 lead (synthesizer). Isi SEMUA field sekali jalan. WAJIB dipanggil sekali.",
			"parameters":  teamPlanSchema(),
		},
	}
	args, err := routerForcedTool(ctx, model,
		"Lo arsitek TIM Flowork. Dari permintaan user, rancang group LENGKAP sekali jalan: pecah jadi 2-4 specialist (worker) yg saling melengkapi + 1 lead yg gabungin jadi 1 jawaban. Persona & directive sesuai domain. Bahasa Indonesia. RINGKAS (anti over-prompt).",
		"Bikin tim buat: "+prompt, tool, "design_team", 2500)
	if err != nil {
		return plan, err
	}
	if err := json.Unmarshal(args, &plan); err != nil {
		return plan, fmt.Errorf("decode team plan: %w", err)
	}
	plan.GroupID = strings.ToLower(strings.TrimSpace(plan.GroupID))
	plan.DisplayName = strings.TrimSpace(plan.DisplayName)
	return plan, nil
}

// nonEmpty returns v trimmed, or def if v is blank — fills AgentSpec fields the
// design call legitimately leaves out (a specialist has no synth role, etc.) so
// AgentSpec.validate() passes without burdening the LLM with throwaway text.
func nonEmpty(v, def string) string {
	if s := strings.TrimSpace(v); s != "" {
		return s
	}
	return def
}

// swapManifest — clone a template agent manifest, swap id + display_name. Caps stay
// the template's (proven). Shared by the team assembler for every crew member.
func swapManifest(tmpl []byte, id, display string) ([]byte, error) {
	m := map[string]any{} // non-nil: Unmarshal("null") is a no-op → write below won't panic
	if e := json.Unmarshal(tmpl, &m); e != nil {
		return nil, e
	}
	m["id"] = id
	m["display_name"] = display
	return json.MarshalIndent(m, "", "  ")
}

// architectAssembleTeamPack — build ONE .fwpack for the WHOLE team: every specialist
// as a worker + the lead as the single synth (installPluginPack requires exactly one
// synth per pack). Agent ids are GROUP-PREFIXED ("<group>-<slug>", "<group>-synth") so
// the Groups GUI auto-claims them to this group (a.id.startsWith(group+'-')) → they
// never pollute other groups' member pools. One pack, every crew member used → NO
// orphan agents (the Bug 1 fix). Mirrors how the bundled investment/thinking groups
// are structured. Returns (pack, memberIDs, synthID).
func architectAssembleTeamPack(plan teamPlan) ([]byte, []string, string, error) {
	workerWasm, workerMan, err := coderTemplate("worker")
	if err != nil {
		return nil, nil, "", err
	}
	synthWasm, synthMan, err := coderTemplate("synth")
	if err != nil {
		return nil, nil, "", err
	}
	files := map[string][]byte{}
	crew := []pluginCrewMember{}
	members := []string{}
	seen := map[string]bool{}
	for _, sp := range plan.Specialists {
		slug := strings.ToLower(strings.TrimSpace(sp.CategoryID))
		slug = strings.TrimPrefix(slug, plan.GroupID+"-") // avoid double prefix if the LLM already prefixed
		aid := plan.GroupID + "-" + slug
		if len(aid) > 63 {
			aid = aid[:63]
		}
		if slug == "" || !pluginIDRe.MatchString(aid) || seen[aid] {
			continue
		}
		seen[aid] = true
		man, merr := swapManifest(workerMan, aid, nonEmpty(sp.Name, slug))
		if merr != nil {
			return nil, nil, "", fmt.Errorf("worker manifest %s: %w", aid, merr)
		}
		files["agents/"+aid+"/agent.wasm"] = workerWasm
		files["agents/"+aid+"/manifest.json"] = man
		crew = append(crew, pluginCrewMember{
			AgentID: aid, RoleLabel: nonEmpty(sp.Role, "specialist"), Kind: "worker",
			Persona: nonEmpty(sp.Persona, "Specialist "+plan.DisplayName+" — fokus 1 keahlian, ringkas."),
		})
		members = append(members, aid)
	}
	if len(members) == 0 {
		return nil, nil, "", fmt.Errorf("no valid specialists in plan")
	}
	synthID := plan.GroupID + "-synth"
	sman, merr := swapManifest(synthMan, synthID, plan.DisplayName+" — lead")
	if merr != nil {
		return nil, nil, "", fmt.Errorf("synth manifest: %w", merr)
	}
	files["agents/"+synthID+"/agent.wasm"] = synthWasm
	files["agents/"+synthID+"/manifest.json"] = sman
	crew = append(crew, pluginCrewMember{
		AgentID: synthID, RoleLabel: "lead", Kind: "synth",
		Persona: nonEmpty(plan.Lead.Persona, "Lead tim "+plan.DisplayName+" — gabungkan jawaban anggota jadi 1 jawaban final yg jelas."),
	})

	man := pluginManifest{ID: plan.GroupID + "-crew", Name: plan.DisplayName, Version: "1.0.0", Author: "flowork-architect"}
	man.Category.ID = plan.GroupID
	man.Category.Name = plan.DisplayName
	man.Category.Icon = nonEmpty(plan.Lead.Icon, "🧩")
	man.Category.TriggerHint = "tim " + plan.DisplayName
	man.Category.SynthDirective = nonEmpty(plan.Lead.Directive, "Rangkai jadi 1 jawaban final yg jelas + rapi.")
	man.Category.WorkerDirective = "Kerjakan bagianmu sesuai keahlian, ringkas (anti over-prompt)."
	man.Crew = crew
	pluginJSON, e := json.MarshalIndent(man, "", "  ")
	if e != nil {
		return nil, nil, "", e
	}
	files["plugin.json"] = pluginJSON

	pack, e := zipPack(files)
	if e != nil {
		return nil, nil, "", e
	}
	return pack, members, synthID, nil
}

// architectBuildFromPlan — build a team from an ALREADY-DECIDED plan (no design LLM
// call): assemble the whole team into ONE pack → install → create the coordinator
// group. This is what the chat brain calls on the build_team tool, so the team built
// is EXACTLY the one discussed (not a re-design). Re-callable: same group_id rebuilds.
func architectBuildFromPlan(ctx context.Context, host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler, plan teamPlan) (map[string]any, error) {
	plan.GroupID = strings.ToLower(strings.TrimSpace(plan.GroupID))
	plan.DisplayName = strings.TrimSpace(plan.DisplayName)
	if !idReGroup.MatchString(plan.GroupID) {
		return nil, fmt.Errorf("group_id invalid/too long (2-26 lowercase-dash): %q", plan.GroupID)
	}
	if plan.DisplayName == "" {
		plan.DisplayName = plan.GroupID
	}
	if len(plan.Specialists) == 0 {
		return nil, fmt.Errorf("plan has no specialists")
	}
	pack, members, synthesizer, aerr := architectAssembleTeamPack(plan)
	if aerr != nil {
		return nil, fmt.Errorf("assemble team: %w", aerr)
	}
	if res := installPluginPack(host, store, pack, true); res.status != 0 {
		return nil, fmt.Errorf("install team failed: %v", res.body)
	}
	// Wire the coordinator group (folder + roster + orchestrator sync). Live now.
	if cerr := groups.CreateGroup(plan.GroupID, plan.DisplayName, members, synthesizer, plan.Task); cerr != nil {
		return nil, fmt.Errorf("create group: %w", cerr)
	}
	return map[string]any{
		"ok":           true,
		"group_id":     plan.GroupID,
		"display_name": plan.DisplayName,
		"task":         plan.Task,
		"members":      members,
		"synthesizer":  synthesizer,
		"chat":         fmt.Sprintf("POST /api/chat {\"agent\":%q,\"text\":\"...\"}", plan.GroupID),
		"next":         "Team is live in the Group tab + Telegram slash menu. Chat it via the group id above.",
	}, nil
}

// architectBuild — one-shot: design a team from a prompt (one LLM call) then build it.
// Used by POST /api/architect/build. The conversational chat brain instead designs
// through dialogue and calls architectBuildFromPlan directly.
func architectBuild(ctx context.Context, host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler, prompt, model string) (map[string]any, error) {
	plan, err := architectDesignTeam(ctx, prompt, model)
	if err != nil {
		return nil, fmt.Errorf("design team: %w", err)
	}
	return architectBuildFromPlan(ctx, host, store, groups, plan)
}

// architectBuildHandler — POST /api/architect/build {prompt|task, model?}.
func architectBuildHandler(host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			tfWriteJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST only"})
			return
		}
		var body struct {
			Prompt string `json:"prompt"`
			Task   string `json:"task"` // alias for prompt
			Model  string `json:"model"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
			return
		}
		prompt := strings.TrimSpace(body.Prompt)
		if prompt == "" {
			prompt = strings.TrimSpace(body.Task)
		}
		if prompt == "" {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "prompt required"})
			return
		}
		// One design call (may stall ~90s if upstream rate-limits before failover) +
		// fast local assembly → generous but bounded timeout.
		ctx, cancel := context.WithTimeout(r.Context(), 280*time.Second)
		defer cancel()
		res, err := architectBuild(ctx, host, store, groups, prompt, coderModel(body.Model))
		if err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		tfWriteJSON(w, 0, res)
	}
}
