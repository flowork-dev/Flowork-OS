// === LOCKED FILE (soft) ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Locked at: 2026-06-15 (owner-approved autonomous sprint)
// 2026-06-16 (owner-approved): ID-length robustness — generated agent ids capped 63→31
//   (coderCatRe/reID limit, else config API rejects "invalid id" → agent unmanageable) +
//   group_id TRUNCATED via capGroupID instead of rejected (a long LLM name like
//   "autonomy-manifest-governance"=28 no longer fails the whole build). AI Studio never
//   blows up / produces unaddressable ids. RE-LOCKED.
// Reason: Flowork Architect — group/team creator. VERIFIED E2E: POST /api/architect/build
//   {"prompt":"team peramal …"} → designed "Tim Peramal Nasib" → ONE pack (3 specialists +
//   1 lead synth, ALL group-prefixed "peramal-nasib-*") → installed → created group +
//   SyncToOrchestrator → coordinator loaded → /api/chat returned a real synthesized fortune.
//   2026-06-15 BUG-1 FIX: was assembling worker+synth per specialist (orphan synths polluted
//   EVERY group's member pool). Now ONE pack, every crew member used, agent ids group-prefixed
//   so the Groups GUI auto-claims them → no pollution (mirrors bundled investment/thinking).
//   One LLM call (design) + fast local assembly. Loopback-only, owner trust = /api/coder/*.
//   SCOPE NOW (all tested): build_team (group), build_app (REAL HTML App-menu app via
//   designAppUI + fwapps.InstallAppPack), architectBuildFromPlan (chat build), authorSkill
//   (skill_author → ~/.flow_router/skills, brain-injected). Lihat architect_chat.go.
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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	fwapps "flowork-gui/internal/apps"
	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/kernel/loader"
	"flowork-gui/internal/kernelhost"
	"flowork-gui/internal/tools"
)

var appUIIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,40}$`)

// designAppUI — one forced-tool call: design a Flowork app = UI HTML mandiri + KONEKTOR AGEN.
// Owner 2026-06-21 (rule): TIAP app WAJIB bisa dipakai AI agent (≥1 operasi + backend), bukan cuma
// user. Output: html (UI) + operations[] (agent-callable) + core_py (backend python serve operasi,
// protokol stdio jam-digital). Balik appID,name,icon,desc,html,operationsJSON,corePy.
func designAppUI(ctx context.Context, prompt, model string) (appID, name, icon, desc, html, operationsJSON, corePy string, err error) {
	tool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "design_app",
			"description": "Rancang 1 APLIKASI Flowork = UI HTML mandiri + KONEKTOR AGEN. WAJIB dipanggil sekali. STANDAR WAJIB owner: tiap app HARUS punya ≥1 'operations' (kemampuan inti yg bisa dipanggil AI agent) + 'core_py' (backend python yg jalanin operasi). Bukan cuma UI buat user.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"app_id":      map[string]any{"type": "string", "description": "id slug unik lowercase-dash, 2-40 char (mis. 'kalkulator')."},
					"name":        map[string]any{"type": "string", "description": "nama app (mis. 'Kalkulator')."},
					"icon":        map[string]any{"type": "string", "description": "1 emoji."},
					"description": map[string]any{"type": "string", "description": "1 kalimat fungsi app."},
					"html":        map[string]any{"type": "string", "description": "SATU file HTML LENGKAP (<!doctype html>…</html>) CSS+JS embedded, self-contained, TANPA CDN/library eksternal, responsive. UI buat user."},
					"operations": map[string]any{
						"type":        "array",
						"description": "MINIMAL 1 operasi yg bisa dipanggil AI AGENT (konektor). Tiap operasi = kemampuan inti app berguna buat agent (kalkulator→'calculate', converter→'convert', timer→'set_timer'). JANGAN kosong.",
						"minItems":    1,
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name":         map[string]any{"type": "string", "description": "snake_case (mis. 'calculate')."},
								"description":  map[string]any{"type": "string", "description": "buat AGENT: operasi ini ngapain + kapan dipakai + balikin apa."},
								"input_schema": map[string]any{"type": "object", "description": "JSON-schema parameter: {type:'object', properties:{...}, required:[...]}. properties kosong kalau ga butuh input."},
							},
							"required": []string{"name", "description", "input_schema"},
						},
					},
					"core_py": map[string]any{"type": "string", "description": "Backend PYTHON3 yg IMPLEMENT SEMUA operasi. PROTOKOL STDIO WAJIB: loop baca 1 baris JSON {\"op\":..,\"args\":..} dari stdin → fungsi handle(op,args) → tulis 1 baris {\"result\":..} atau {\"error\":..} ke stdout + flush. HANYA Python stdlib (no pip). Pola: `import sys,json\\ndef handle(op,args):\\n  if op=='calculate': return {'result': args['a']+args['b']}\\n  return {'error':'unknown op:'+str(op)}\\nfor line in sys.stdin:\\n  line=line.strip()\\n  if not line: continue\\n  try: req=json.loads(line); out=handle(req.get('op',''), req.get('args') or {})\\n  except Exception as e: out={'error':str(e)}\\n  sys.stdout.write(json.dumps(out)+'\\\\n'); sys.stdout.flush()`."},
				},
				"required": []string{"app_id", "name", "icon", "description", "html", "operations", "core_py"},
			},
		},
	}
	args, e := routerForcedTool(ctx, model,
		"Lo desainer aplikasi Flowork. Tiap app = UI HTML mandiri (CSS+JS embedded, no CDN, offline) + "+
			"KONEKTOR AGEN: ≥1 operasi yg bisa dipanggil AI agent, di-serve backend Python (protokol stdio). "+
			"STANDAR WAJIB owner: app HARUS bisa dipakai agent, bukan cuma user. Rapi & fungsional.",
		"Bikin aplikasi: "+prompt, tool, "design_app", 8000)
	if e != nil {
		return "", "", "", "", "", "", "", e
	}
	var raw struct {
		AppID       string          `json:"app_id"`
		Name        string          `json:"name"`
		Icon        string          `json:"icon"`
		Description string          `json:"description"`
		HTML        string          `json:"html"`
		Operations  json.RawMessage `json:"operations"`
		CorePy      string          `json:"core_py"`
	}
	if e := json.Unmarshal(args, &raw); e != nil {
		return "", "", "", "", "", "", "", fmt.Errorf("decode app spec: %w", e)
	}
	return strings.ToLower(strings.TrimSpace(raw.AppID)), strings.TrimSpace(raw.Name),
		strings.TrimSpace(raw.Icon), strings.TrimSpace(raw.Description), raw.HTML,
		string(raw.Operations), raw.CorePy, nil
}

// architectBuildApp — build a REAL App-menu application (a self-contained HTML/JS
// program) from a prompt and install it so it shows up + runs in the App tab. This is
// the "app" arm of the unified AI Studio chat (vs build_team = a crew of agents). For
// AI-that-answers (pantun, translate), use build_team instead. Loopback owner-trust →
// install with approveExec (GUI-only app, runtime="", no OS process).
func architectBuildApp(ctx context.Context, host *kernelhost.Host, store *floworkdb.Store, prompt, model string) (map[string]any, error) {
	// Design via forced-tool (terstruktur, reliable buat html + operations + core.py). Model =
	// ai-studio per-agent (owner: model dari agent) kalau caller ga override.
	appModel := strings.TrimSpace(model)
	if appModel == "" {
		appModel = aiStudioModel()
	}
	appID, name, icon, desc, html, opsJSON, corePy, err := designAppUI(ctx, prompt, appModel)
	if err != nil {
		return nil, fmt.Errorf("design app: %w", err)
	}
	if !appUIIDRe.MatchString(appID) {
		return nil, fmt.Errorf("app_id invalid (^[a-z0-9][a-z0-9-]{1,40}$): %q", appID)
	}
	if strings.TrimSpace(html) == "" || !strings.Contains(strings.ToLower(html), "<html") {
		return nil, fmt.Errorf("HTML app kosong/invalid")
	}
	// RULE owner 2026-06-21: app WAJIB punya konektor agen (≥1 operasi + backend). Tolak kalau ga ada.
	var ops []map[string]any
	if json.Unmarshal([]byte(opsJSON), &ops) != nil || len(ops) == 0 {
		return nil, fmt.Errorf("app WAJIB punya ≥1 operasi (konektor agen) — desain ga ngasih operasi yg bisa dipanggil agent")
	}
	if strings.TrimSpace(corePy) == "" || !strings.Contains(corePy, "def handle") {
		return nil, fmt.Errorf("core.py backend kosong/invalid (wajib `def handle(op, args)` + loop stdio)")
	}
	if name == "" {
		name = appID
	}
	if icon == "" {
		icon = "🧩"
	}
	// Normalisasi tiap operasi: WAJIB tool:true (= konektor agent) + gui:true; default mutates:false.
	for i := range ops {
		ops[i]["tool"] = true
		ops[i]["gui"] = true
		if _, has := ops[i]["mutates"]; !has {
			ops[i]["mutates"] = false
		}
	}
	manifest := map[string]any{
		"id": appID, "kind": "app", "name": name, "description": desc,
		"icon": "ui/icon.svg", "version": "1.0.0",
		"runtime": "process", "core_entry": "python3 core.py", "gui_entry": "ui/index.html",
		"operations": ops,
	}
	manRaw, _ := json.MarshalIndent(manifest, "", "  ")
	pluginJSON, _ := json.Marshal(map[string]any{"kind": "app", "id": appID, "name": name})
	iconSVG := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="14" fill="#1e293b"/><text x="32" y="44" font-size="34" text-anchor="middle">` + icon + `</text></svg>`
	pack, perr := zipPack(map[string][]byte{
		"plugin.json":                      pluginJSON,
		"apps/" + appID + "/manifest.json": manRaw,
		"apps/" + appID + "/core.py":       []byte(corePy),
		"apps/" + appID + "/ui/index.html": []byte(html),
		"apps/" + appID + "/ui/icon.svg":   []byte(iconSVG),
	})
	if perr != nil {
		return nil, fmt.Errorf("assemble app: %w", perr)
	}
	// approveExec=true: owner-trusted loopback. App ini punya core process (runtime:process) buat
	// serve operasi ke agent — caps-consent di-auto-approve krn loopback (sama pola coder/architect).
	res, status := fwapps.InstallAppPack(pack, true)
	if status != 0 {
		return nil, fmt.Errorf("install app: %v", res)
	}
	_ = host
	_ = store
	opNames := make([]string, 0, len(ops))
	for _, o := range ops {
		if n, _ := o["name"].(string); n != "" {
			opNames = append(opNames, n)
		}
	}
	return map[string]any{
		"ok": true, "app_id": appID, "name": name, "design_model": appModel + " (ai-studio)",
		"operations": opNames,
		"note":       "App '" + name + "' (" + appID + ") LIVE di menu App + " + fmt.Sprintf("%d", len(opNames)) + " operasi konektor agen (agent bisa pakai via tool).",
	}, nil
}

// availableDataTools — enumerasi TOOL DATA NYATA yg bisa di-pakai member tim: operasi app
// (prefix "app_", mis. app_flowalpha_get_price) yg narik data live (harga/pasar/dll). Owner
// 2026-06-21: tim analisa WAJIB pakai data riil, BUKAN ngarang. Balik (hint buat prompt desain,
// set nama tool valid). Cap biar prompt ga bengkak.
func availableDataTools() (hint string, valid map[string]bool) {
	valid = map[string]bool{}
	var b strings.Builder
	n := 0
	for _, name := range tools.ListNames() {
		if !strings.HasPrefix(name, "app_") || n >= 40 {
			continue
		}
		valid[name] = true
		desc := ""
		if t, ok := tools.Lookup(name); ok {
			desc = t.Schema().Description
		}
		b.WriteString("- " + name + ": " + trimStr(desc, 120) + "\n")
		n++
	}
	return b.String(), valid
}

// subscribeMemberTools — subscribe daftar tool ke 1 member tim (state.db-nya), biar tool itu
// ke-EXPOSE ke LLM-nya (app tool subscription-only, ga di core). Cuma tool valid yg di-subscribe.
// Best-effort: gagal subscribe 1 tool GA mecahin build.
func subscribeMemberTools(memberID string, toolNames []string, valid map[string]bool) int {
	if len(toolNames) == 0 {
		return 0
	}
	dir := filepath.Join(loader.AgentsDir(), memberID+".fwagent")
	st, e := agentdb.Open(agentdb.Resolve(memberID, dir))
	if e != nil {
		return 0
	}
	defer st.Close()
	n := 0
	for _, tn := range toolNames {
		tn = strings.TrimSpace(tn)
		if tn == "" || !valid[tn] {
			continue // cuma tool yg beneran ada (anti halu nama tool)
		}
		if st.SubscribeTool(tn, "ai-studio:team-data", "{}") != nil {
			continue
		}
		n++
		// subscribe != capability: app tool butuh GRANT app:<id> biar ke-expose + callable.
		// appID di-extract dari Capability() = "app:<id>" (anti salah-parse nama tool ber-dash).
		if t, ok := tools.Lookup(tn); ok {
			if c := t.Capability(); strings.HasPrefix(c, "app:") {
				_ = st.GrantApp(strings.TrimPrefix(c, "app:"))
			}
		}
	}
	return n
}

// architectSkillsDir — MUST match the router's brain.DynamicSkillsDir() so what the
// architect authors is what the router injects: $FLOW_ROUTER_DATA/skills else
// ~/.flow_router/skills. (Same machine; both default to ~/.flow_router/skills.)
func architectSkillsDir() string {
	if d := strings.TrimSpace(os.Getenv("FLOW_ROUTER_DATA")); d != "" {
		return filepath.Join(d, "skills")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".flow_router", "skills")
}

var skillNameRe = regexp.MustCompile(`[^a-z0-9-]+`)

// authorSkill — write a focused SKILL.md (agent-skills frontmatter format) into the
// shared dynamic-skills dir so the router brain injects it (by keyword) into relevant
// future LLM calls — especially on the LOCAL model (skill_author / ant principle).
// Best-effort: any failure is ignored so it never blocks a build.
func authorSkill(name, description, body string) {
	dir := architectSkillsDir()
	name = strings.Trim(skillNameRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-"), "-")
	if dir == "" || name == "" || strings.TrimSpace(body) == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	desc := strings.ReplaceAll(strings.TrimSpace(description), "\n", " ")
	md := "---\nname: " + name + "\ndescription: " + desc + "\n---\n\n" + strings.TrimSpace(body) + "\n"
	_ = os.WriteFile(filepath.Join(dir, name+".md"), []byte(md), 0o644)
}

// idReGroup is tighter than groupsapi's idRe (2-40): a group_id here also becomes
// the lead category "<group_id>-lead", which must satisfy coderCatRe (max 31). Cap
// the group_id at 26 chars so "<group_id>-lead" never overflows.
var idReGroup = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,25}$`)
var groupSanitizeRe = regexp.MustCompile(`[^a-z0-9-]+`)

// capGroupID — sanitize + TRUNCATE an LLM-picked group_id to a valid ≤26-char slug, instead
// of REJECTING it (a long name like "autonomy-manifest-governance" = 28 char would otherwise
// fail the whole build). Same robustness as the specialist-aid cap: AI Studio must never blow up
// on a long generated id. Returns "" if nothing valid remains (caller errors then).
func capGroupID(s string) string {
	s = groupSanitizeRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-")
	s = strings.Trim(s, "-")
	if len(s) > 26 {
		s = strings.TrimRight(s[:26], "-")
	}
	return s
}

// teamWorker — one specialist (worker) in the team, fully specified by the design
// call. Only worker-side fields; the synth-side fields of its AgentSpec are filled
// with defaults at assembly (each specialist contributes its -worker to the group).
type teamWorker struct {
	CategoryID string   `json:"category_id"`
	Name       string   `json:"name"`
	Icon       string   `json:"icon"`
	Role       string   `json:"role"`
	Persona    string   `json:"persona"`
	Directive  string   `json:"directive"`
	Tools      []string `json:"tools"` // nama tool data yg spesialis ini pakai (mis. app_flowalpha_get_price)
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
			"directive":   map[string]any{"type": "string", "description": "cara kerja specialist. Kalau KREATIF/tradisi (ga butuh data real) bilang itu tugasnya; kalau ANALISIS/butuh DATA NYATA (harga/pasar/fakta) → WAJIB suruh PANGGIL tool dari field 'tools', HARAM ngarang angka."},
			"tools":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "nama tool DATA yg spesialis ini pakai buat data NYATA — PILIH dari DAFTAR TOOL TERSEDIA di instruksi sistem (mis. app_flowalpha_get_price). Kosong [] kalau spesialis ga butuh data eksternal (kreatif/tradisi)."},
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
	toolHint, _ := availableDataTools()
	sysPrompt := "Lo arsitek TIM Flowork. Dari permintaan user, rancang group LENGKAP sekali jalan: " +
		"pecah jadi 2-4 specialist (worker) yg saling melengkapi + 1 lead yg gabungin jadi 1 jawaban. " +
		"Persona & directive sesuai domain. Bahasa Indonesia. RINGKAS (anti over-prompt)."
	if toolHint != "" {
		sysPrompt += "\n\nTOOL DATA NYATA TERSEDIA (buat spesialis yg butuh data live — harga/pasar/dll):\n" +
			toolHint + "\nKalau tim butuh DATA NYATA, ISI field 'tools' tiap spesialis relevan dgn nama tool " +
			"dari daftar ini, dan directive-nya WAJIB suruh PANGGIL tool itu — HARAM ngarang angka. " +
			"Spesialis kreatif/tradisi: tools [] kosong."
	}
	args, err := routerForcedTool(ctx, model, sysPrompt,
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

// stripManifestComments — buang key dokumentasi ber-prefix "_" (mis. "_comment_caps") dari manifest
// yang di-clone dari template. Loader kernel FROZEN (internal/kernel/loader/manifest.go) decode STRICT
// → field tak dikenal = "unknown field" → agent GAGAL hot-load. Template boleh self-dokumentasi pakai
// "_comment", tapi manifest runtime hasil clone WAJIB bersih. Root-fix (bukan tambal): cloner ga pernah
// emit field yg loader tolak. Owner 2026-06-20 (E2E tim peramal: member ga live gara2 "_comment_caps").
func stripManifestComments(m map[string]any) {
	for k := range m {
		if strings.HasPrefix(k, "_") {
			delete(m, k)
		}
	}
}

// swapManifest — clone a template agent manifest, swap id + display_name. Caps stay
// the template's (proven). Shared by the team assembler for every crew member.
func swapManifest(tmpl []byte, id, display string) ([]byte, error) {
	m := map[string]any{} // non-nil: Unmarshal("null") is a no-op → write below won't panic
	if e := json.Unmarshal(tmpl, &m); e != nil {
		return nil, e
	}
	stripManifestComments(m) // loader FROZEN strict — buang "_comment*" biar clone ga ditolak hot-load
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
		// Cap at 31 (coderCatRe/reID limit), NOT 63 (pluginIDRe install limit). An id >31
		// installs fine but is REJECTED by the config API (/api/agents/config reID max 32) →
		// the agent can never be edited / model-switched ("invalid id"). Truncate the slug to
		// fit + trim any trailing dash so the result stays a valid, addressable id.
		if len(aid) > 31 {
			aid = strings.TrimRight(aid[:31], "-")
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
		// skill_author: ship a focused, reusable SKILL.md for this specialist so the
		// brain injects it into relevant future calls (helps esp. on the local model).
		authorSkill(aid,
			nonEmpty(sp.Role, "specialist")+" ("+plan.DisplayName+") — pakai untuk: "+nonEmpty(sp.Name, slug),
			nonEmpty(sp.Persona, "")+"\n\n## Cara kerja\n"+nonEmpty(sp.Directive, "Kerjakan bagianmu fokus + ringkas (anti over-prompt)."))
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
	plan.GroupID = capGroupID(plan.GroupID) // TRUNCATE long/odd ids (don't fail the build)
	plan.DisplayName = strings.TrimSpace(plan.DisplayName)
	if !idReGroup.MatchString(plan.GroupID) {
		return nil, fmt.Errorf("group_id tidak bisa dinormalisasi jadi valid (2-26): %q", plan.GroupID)
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
	// Owner 2026-06-21: subscribe TOOL DATA ke tiap spesialis (post-install) → tim pakai data NYATA,
	// BUKAN ngarang. aid di-re-derive (mirror architectAssembleTeamPack). Best-effort.
	_, validTools := availableDataTools()
	toolsWired, seenSub := 0, map[string]bool{}
	for _, sp := range plan.Specialists {
		if len(sp.Tools) == 0 {
			continue
		}
		slug := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(sp.CategoryID)), plan.GroupID+"-")
		aid := plan.GroupID + "-" + slug
		if len(aid) > 31 {
			aid = strings.TrimRight(aid[:31], "-")
		}
		if slug == "" || seenSub[aid] {
			continue
		}
		seenSub[aid] = true
		toolsWired += subscribeMemberTools(aid, sp.Tools, validTools)
	}
	return map[string]any{
		"ok":           true,
		"group_id":     plan.GroupID,
		"display_name": plan.DisplayName,
		"task":         plan.Task,
		"members":      members,
		"synthesizer":  synthesizer,
		"tools_wired":  toolsWired,
		"chat":         fmt.Sprintf("POST /api/chat {\"agent\":%q,\"text\":\"...\"}", plan.GroupID),
		"next":         "Team is live in the Group tab + Telegram slash menu. Chat it via the group id above.",
	}, nil
}

// architectBuild — one-shot: design a team from a prompt (one LLM call) then build it.
// Used by POST /api/architect/build. The conversational chat brain instead designs
// through dialogue and calls architectBuildFromPlan directly.
func architectBuild(ctx context.Context, host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler, prompt, model string) (map[string]any, error) {
	// Owner 2026-06-20: design TIM pindah ke AGENT ai-studio (model GUI). Coba agent dulu; agent ga
	// ke-load / plan invalid → FALLBACK architectDesignTeam lama (forced-tool routerChat). Assembly TETAP.
	plan, ok := aiStudioDesignTeam(ctx, host, prompt)
	designModel := aiStudioModel() + " (ai-studio)"
	if !ok {
		var err error
		plan, err = architectDesignTeam(ctx, prompt, model)
		if err != nil {
			return nil, fmt.Errorf("design team: %w", err)
		}
		designModel = coderModel(model) + " (fallback)"
	}
	res, err := architectBuildFromPlan(ctx, host, store, groups, plan)
	if err != nil {
		return nil, err
	}
	res["design_model"] = designModel
	return res, nil
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
