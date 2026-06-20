// ⚠️ EDITING A GROUP? READ doc/handbook/menu-group.md FIRST — the group list is auto-derived (any module with kv group=1); slash menu, ask_group, schedules and Mr.Flow all read the SAME synced list; on/off cascades to all members; reset restores from the repo. Do NOT hardcode the roster or kv "groups".
// === LOCKED FILE (soft) ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Locked at: 2026-06-06
// Reason: Groups audit (GUI→logic). ConfigHandler error-swallow fixed — the old
//
//	`if err := KVSet(...); err == nil { … }` dropped a failure on the FIRST write
//	and still returned {ok:true} (lied to the GUI); now every KVSet chains one err.
//	id validated by idRe on Config/Delete/Create; Delete refuses non-group modules.
//	Tested (groupsapi_test.go, 4/4 PASS).
//
// 2026-06-14 (owner-approved, gateway audit): GATEWAY BUG FIXED — DeleteHandler used
//
//	to only remove the group's coordinator folder, leaving its MEMBER agents loaded +
//	directly invokable (scheduler/triggers/kernel-rpc could still reach them even
//	though the group — their entry door — was deleted). Now Delete disables (unloads)
//	every member BEFORE removing the group, mirroring the ToggleHandler OFF cascade.
//	Verified live: after delete, members return "plugin not loaded"; mr-flow unaffected.
//
// 2026-06-12 (owner-approved): ConfigHandler now also mirrors the roster to a
//
//	secret-free group.json (see seed.go) so a group's membership is committable;
//	SeedFromJSON() restores it into the loket store on a fresh install.
//
// Package groupsapi serves the GUI "Groups" tab (§F2): list the GROUP modules,
// show each group's roster (members + synthesizer + task), and edit it. A GROUP is
// just a loket-native module marked by kv "group"=="1" in its OWN loket store; its
// roster lives in that same store (members/synthesizer/task), read LIVE by the
// group wasm — so an edit here applies without a restart. The handler never
// reaches into another module's folder; it only opens loket stores by the same
// per-module path mapping the kernel uses, keeping isolation intact.
package groupsapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"flowork-gui/internal/httpx"
	"flowork-gui/internal/loket"
)

// idRe is the allowed shape for a new group id — it becomes a folder name and a
// module id, so keep it to safe lowercase slug characters.
var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,39}$`)

// Deps are the host-backed lookups the handler needs (no globals).
type Deps struct {
	AgentIDs       func() []string                      // all loaded module ids
	LoketStorePath func(module string) (string, error)  // → that module's loket.db
	AgentsDir      string                               // where <id>.fwagent folders live
	GroupWasmPath  string                               // template group wasm to copy on create
	Toggle         func(id string, disabled bool) error // enable/disable one agent (= agentmgr.ToggleAgent)
	TelegramToken  func() string                        // bot token (floworkdb secret) → push slash menu Telegram; nil/"" = skip
}

type Handler struct{ d Deps }

func New(d Deps) *Handler { return &Handler{d: d} }

// writeErr writes a JSON error with an explicit status (httpx.WriteJSON is 200-only).
func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": msg})
}

type groupView struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"display_name"`
	Members      []string `json:"members"`
	Synthesizer  string   `json:"synthesizer"`
	Task         string   `json:"task"`
	Mode         string   `json:"mode"`          // A1: parallel|debate
	DebateRounds string   `json:"debate_rounds"` // A1: total ronde debat
	Enabled      bool     `json:"enabled"`       // false = group toggled OFF (group_off=1)
	// Claims = every agent this group uses across ALL its roster roles (members,
	// synthesizer, and auxiliary roles like questioner/how/caster), so the picker
	// can hide an organ that already belongs to another group — not just its
	// "members". Display still uses Members; Claims is only for scoping the pool.
	Claims []string `json:"claims"`
}

type agentRef struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// displayName reads a module's manifest.json display_name (falls back to id).
func (h *Handler) displayName(id string) string {
	raw, err := os.ReadFile(filepath.Join(h.d.AgentsDir, id+".fwagent", "manifest.json"))
	if err != nil {
		return id
	}
	var m struct {
		DisplayName string `json:"display_name"`
	}
	if json.Unmarshal(raw, &m) == nil && strings.TrimSpace(m.DisplayName) != "" {
		return m.DisplayName
	}
	return id
}

// manifestKind reads a module's manifest.json kind ("" if unreadable).
func (h *Handler) manifestKind(id string) string {
	raw, err := os.ReadFile(filepath.Join(h.d.AgentsDir, id+".fwagent", "manifest.json"))
	if err != nil {
		return ""
	}
	var m struct {
		Kind string `json:"kind"`
	}
	if json.Unmarshal(raw, &m) == nil {
		return strings.TrimSpace(m.Kind)
	}
	return ""
}

// eligibleMember decides whether a module may appear in the member pool at all.
// A group's roster is analyst-type AGENTS; channels (telegram/discord/…), the
// Mr.Flow router/orchestrator, scanners and services are NOT members — listing
// them was just clutter ("pajangan"), so they are filtered out here.
func (h *Handler) eligibleMember(id string) bool {
	if strings.HasPrefix(id, "mr-flow") {
		return false
	}
	// Operator/control agents (operator-*) hold privileged caps like exec:power — they
	// can power the host OFF. They are NEVER group members: if one were accidentally
	// ticked into a roster, a group fan-out could shut the machine down. Hard-exclude.
	if strings.HasPrefix(id, "operator-") {
		return false
	}
	// Channels are deployed as kind "agent" (dumb pipes) but are I/O endpoints, not
	// analyst members — exclude by the "-channel" id convention they all follow.
	if strings.HasSuffix(id, "-channel") {
		return false
	}
	switch h.manifestKind(id) {
	case "channel", "group", "scanner", "service":
		return false
	}
	return true
}

func splitCSV(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ListHandler — GET /api/groups → {groups:[…], available_agents:[…]}.
// groups = modules marked kv group=1; available_agents = every other module (the
// pool the owner can drop into a group as a member/synthesizer).
func (h *Handler) ListHandler(w http.ResponseWriter, _ *http.Request) {
	ids := h.d.AgentIDs()
	sort.Strings(ids)
	groups := []groupView{}
	avail := []agentRef{}
	for _, id := range ids {
		path, err := h.d.LoketStorePath(id)
		if err != nil {
			continue
		}
		st, err := loket.OpenStore(path)
		if err != nil {
			// No loket store yet → not a group; a candidate member if eligible.
			if h.eligibleMember(id) {
				avail = append(avail, agentRef{ID: id, DisplayName: h.displayName(id)})
			}
			continue
		}
		marker, _, _ := st.KVGet("group")
		if strings.TrimSpace(marker) == "1" {
			members, _, _ := st.KVGet("members")
			synth, _, _ := st.KVGet("synthesizer")
			task, _, _ := st.KVGet("task")
			// Display name: an owner-set kv override wins over the manifest, so a
			// group can be renamed without touching its manifest.json.
			name := h.displayName(id)
			if dn, ok, _ := st.KVGet("display_name"); ok && strings.TrimSpace(dn) != "" {
				name = strings.TrimSpace(dn)
			}
			// Claims span every roster role, not just "members": members + synthesizer
			// + auxiliary roles a pipeline group uses (questioner/how/caster). An organ
			// referenced by ANY of these belongs to this group and is hidden elsewhere.
			claims := splitCSV(members)
			for _, role := range []string{"synthesizer", "questioner", "how_agent", "caster"} {
				if v, _, _ := st.KVGet(role); strings.TrimSpace(v) != "" {
					claims = append(claims, strings.TrimSpace(v))
				}
			}
			off, _, _ := st.KVGet("group_off")
			mode, _, _ := st.KVGet("mode")
			drounds, _, _ := st.KVGet("debate_rounds")
			groups = append(groups, groupView{
				ID: id, DisplayName: name,
				Members: splitCSV(members), Synthesizer: strings.TrimSpace(synth), Task: strings.TrimSpace(task),
				Mode: strings.TrimSpace(mode), DebateRounds: strings.TrimSpace(drounds),
				Claims:  claims,
				Enabled: strings.TrimSpace(off) != "1",
			})
		} else if h.eligibleMember(id) {
			avail = append(avail, agentRef{ID: id, DisplayName: h.displayName(id)})
		}
		_ = st.Close()
	}
	httpx.WriteJSON(w, map[string]any{"groups": groups, "available_agents": avail})
}

// ConfigHandler — POST /api/groups/config?id=<group> body {members[],synthesizer,task}.
// Writes the roster into the group's loket store; the group wasm reads it live.
func (h *Handler) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeErr(w, http.StatusBadRequest, "id required")
		return
	}
	// id becomes a path component (AgentsDir/<id>.fwagent) — validate the exact shape
	// (same as create) so it can never escape the agents dir or create odd stores.
	if !idRe.MatchString(id) {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Members      []string `json:"members"`
		Synthesizer  string   `json:"synthesizer"`
		Task         string   `json:"task"`
		DisplayName  string   `json:"display_name"`
		Mode         string   `json:"mode"`          // A1: "parallel" (default) | "debate"
		DebateRounds string   `json:"debate_rounds"` // A1: total ronde debat (2-4); kosong = default
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad body: "+err.Error())
		return
	}
	path, err := h.d.LoketStorePath(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	st, err := loket.OpenStore(path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "open store: "+err.Error())
		return
	}
	defer st.Close()
	clean := []string{}
	for _, m := range body.Members {
		if m = strings.TrimSpace(m); m != "" && m != id {
			clean = append(clean, m)
		}
	}
	// Write the roster. EVERY KVSet (including the first) must gate the result —
	// the old `if err := KVSet(...); err == nil { … }` swallowed a failure on the
	// FIRST write and still reported {ok:true}, lying to the GUI ("✓ saved") while
	// nothing persisted. Chain through a single err so any failure surfaces.
	err = st.KVSet("group", "1")
	if err == nil {
		err = st.KVSet("members", strings.Join(clean, ","))
	}
	if err == nil {
		err = st.KVSet("synthesizer", strings.TrimSpace(body.Synthesizer))
	}
	if err == nil {
		err = st.KVSet("task", strings.TrimSpace(body.Task))
	}
	if err == nil && strings.TrimSpace(body.DisplayName) != "" {
		err = st.KVSet("display_name", strings.TrimSpace(body.DisplayName))
	}
	// A1 debate config: mode (parallel|debate) + ronde. Validasi: mode selain 'debate' → 'parallel'.
	if err == nil {
		mode := strings.ToLower(strings.TrimSpace(body.Mode))
		if mode != "debate" {
			mode = "parallel"
		}
		err = st.KVSet("mode", mode)
	}
	if err == nil {
		err = st.KVSet("debate_rounds", strings.TrimSpace(body.DebateRounds))
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Mirror the roster to a secret-free group.json so the group's membership is
	// versionable + portable (the loket store above is .db-ignored). Best-effort.
	writeGroupSeed(h.d.AgentsDir, id, clean, body.Synthesizer, body.Task, body.DisplayName)
	// Plug-and-play: re-sync the group list to the orchestrator so a renamed group's
	// slash-menu description stays current (see orchestrator.go).
	h.SyncToOrchestrator()
	httpx.WriteJSON(w, map[string]any{"ok": true, "id": id, "members": clean})
}

// DeleteHandler — POST /api/groups/delete?id=<group>. Removes the group's folder
// (the hot-reload watcher unloads it). Guarded: only deletes a module that is
// actually marked group=1, so this endpoint can never remove a real agent.
func (h *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !idRe.MatchString(id) {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	// Confirm it's a GROUP before deleting anything — never delete a plain agent.
	path, err := h.d.LoketStorePath(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	st, err := loket.OpenStore(path)
	if err != nil {
		writeErr(w, http.StatusNotFound, "group not found")
		return
	}
	marker, _, _ := st.KVGet("group")
	members, _, _ := st.KVGet("members") // read roster BEFORE close — needed to disable them
	_ = st.Close()
	if strings.TrimSpace(marker) != "1" {
		writeErr(w, http.StatusForbidden, "bukan group — ditolak (cuma group yang bisa dihapus di sini)")
		return
	}
	// GATEWAY ENFORCEMENT (fix 2026-06-14): a DELETED group's agents MUST become
	// uncallable — the group is the entry door. Disable (unload) every member BEFORE
	// removing the group; otherwise the members stay loaded and the scheduler /
	// triggers / kernel-rpc could still invoke them directly even though their group
	// is gone. Mirrors the ToggleHandler OFF cascade. (Bug: delete used to only
	// remove the coordinator folder, leaving members live + invokable.)
	disabledMembers := []string{}
	if h.d.Toggle != nil {
		for _, m := range splitCSV(members) {
			if m != "" && m != id {
				if err := h.d.Toggle(m, true); err == nil {
					disabledMembers = append(disabledMembers, m)
				}
			}
		}
	}
	dir := filepath.Join(h.d.AgentsDir, id+".fwagent")
	if err := os.RemoveAll(dir); err != nil {
		writeErr(w, http.StatusInternalServerError, "hapus folder: "+err.Error())
		return
	}
	// Plug-and-play: re-sync so the deleted group drops out of the slash menu + the
	// ask_group tool automatically (see orchestrator.go).
	h.SyncToOrchestrator()
	httpx.WriteJSON(w, map[string]any{"ok": true, "id": id, "disabled_members": disabledMembers})
}

// groupManifest is the manifest.json for a freshly created GROUP module. It runs
// the shared group wasm; its roster lives in its loket store (read live).
func groupManifest(id, display string) []byte {
	m := map[string]any{
		"id": id, "version": "1.0.0", "kind": "agent",
		"display_name": display, "min_kernel_version": "0.1.0",
		"description": "GROUP (koloni semut): sebar tugas ke anggota lewat bus, synthesizer gabungin jadi 1 jawaban. Roster di-set di tab Group.",
		"abi_version": 1, "author": "@flowork-os", "license": "MIT",
		"entry": "agent.wasm", "memory_max_mb": 16, "timeout_call_ms": 120000,
		"capabilities_required": []string{
			"net:fetch:http://127.0.0.1:1987/api/kernel/call", "state:read", "state:write", "time:read",
		},
		"exposes_rpc": []map[string]any{{
			"name": "handle_message", "description": "Run the group task: fan out to members, synthesize.",
			"input_schema": map[string]any{"type": "object", "properties": map[string]any{}},
		}},
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return b
}

// CreateHandler — POST /api/groups/create body {id, display_name}. Deploys a new
// GROUP module: a fresh <id>.fwagent folder with the shared group wasm + manifest,
// marked group=1 in its loket store. The hot-reload watcher loads it; the owner
// then fills the roster via ConfigHandler. No code per group — pure plug-and-play.
func (h *Handler) CreateHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad body: "+err.Error())
		return
	}
	id := strings.ToLower(strings.TrimSpace(body.ID))
	if !idRe.MatchString(id) {
		writeErr(w, http.StatusBadRequest, "id harus huruf kecil/angka/dash, 2-40 char")
		return
	}
	display := strings.TrimSpace(body.DisplayName)
	if display == "" {
		display = id
	}
	dir := filepath.Join(h.d.AgentsDir, id+".fwagent")
	if _, err := os.Stat(dir); err == nil {
		writeErr(w, http.StatusConflict, "id sudah dipakai")
		return
	}
	// Need the template group wasm to copy.
	wasm, err := os.ReadFile(h.d.GroupWasmPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "template group wasm ga ketemu: "+err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Join(dir, "workspace"), 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "mkdir: "+err.Error())
		return
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), groupManifest(id, display), 0o644); err != nil {
		cleanup()
		writeErr(w, http.StatusInternalServerError, "write manifest: "+err.Error())
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "agent.wasm"), wasm, 0o644); err != nil {
		cleanup()
		writeErr(w, http.StatusInternalServerError, "write wasm: "+err.Error())
		return
	}
	_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("agent.wasm\nworkspace/*.db\nworkspace/*.db-*\n"), 0o644)
	// Mark it a group in its loket store so it shows up immediately (empty roster).
	if path, perr := h.d.LoketStorePath(id); perr == nil {
		if st, serr := loket.OpenStore(path); serr == nil {
			_ = st.KVSet("group", "1")
			_ = st.KVSet("members", "")
			_ = st.KVSet("synthesizer", "")
			_ = st.KVSet("task", "")
			_ = st.Close()
		}
	}
	// Plug-and-play: the moment a group is created it auto-registers with the
	// orchestrator → appears in the Telegram slash menu + becomes runnable via
	// Mr.Flow, with no kv editing and no code change (see orchestrator.go).
	h.SyncToOrchestrator()
	httpx.WriteJSON(w, map[string]any{"ok": true, "id": id, "display_name": display})
}
