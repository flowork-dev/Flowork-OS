// ⚠️ EDITING A GROUP? READ doc/handbook/menu-group.md FIRST — the group list is auto-derived (any module with kv group=1); slash menu, ask_group, schedules and Mr.Flow all read the SAME synced list; on/off cascades to all members; reset restores from the repo. Do NOT hardcode the roster or kv "groups".
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-12
// Reason: PLUG-AND-PLAY group discovery. SyncToOrchestrator auto-discovers every
//
//	group (kv group=1) and writes the "id|command|desc" list into the orchestrator's
//	kv "groups" — the single source mr-flow-next reads to build the Telegram slash
//	menu (groupCommandsJSON) and the ask_group tool. So a NEW group appears in the
//	bot's slash commands and becomes runnable via Mr.Flow automatically — no kv
//	editing, no code change per group. Called on boot + on create/config/delete.
//
// orchestrator.go — auto-sync the group list to the orchestrator (plug-and-play).
package groupsapi

import (
	"strings"

	"flowork-gui/internal/loket"
)

// OrchestratorID is the agent that owns the Telegram slash menu + the ask_group
// tool (the agent telegram-channel routes to). It reads its kv "groups" to know
// which groups exist; the host keeps that list in sync automatically.
const OrchestratorID = "mr-flow-next"

// sanitizeDesc strips the delimiters the kv "groups" format uses (";" and "|") and
// newlines so a group's free-text task/display can't corrupt the list.
func sanitizeDesc(s string) string {
	s = strings.NewReplacer(";", ",", "|", "/", "\n", " ", "\r", " ").Replace(strings.TrimSpace(s))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// telegramCommand turns a group id into a VALID, UNIQUE Telegram slash command:
// lowercase, only [a-z0-9_], max 32 chars. Telegram rejects setMyCommands outright
// if any command has a dash, so "promo-devto" → "promo_devto" (keeping the whole id
// keeps it unique — segment-only derivation collides, e.g. promo-x vs promo-devto).
func telegramCommand(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	c := b.String()
	if len(c) > 32 {
		c = c[:32]
	}
	return c
}

// SyncToOrchestrator auto-discovers every group (kv group=1) across all loaded
// modules and writes the "id|command|desc" list (";"-joined) into the orchestrator's
// kv "groups". Idempotent; safe to call often. Returns how many groups it synced.
// This is the plug-and-play core: create a group → it shows up everywhere, no config.
func (h *Handler) SyncToOrchestrator() int {
	if h.d.AgentIDs == nil || h.d.LoketStorePath == nil {
		return 0
	}
	var parts []string
	for _, id := range h.d.AgentIDs() {
		if id == OrchestratorID {
			continue
		}
		path, err := h.d.LoketStorePath(id)
		if err != nil {
			continue
		}
		st, err := loket.OpenStore(path)
		if err != nil {
			continue
		}
		marker, _, _ := st.KVGet("group")
		off, _, _ := st.KVGet("group_off")
		// A group that's turned OFF drops out of the orchestrator list entirely — no
		// slash command, and Mr.Flow won't list it (so the model can't be told about a
		// command that does nothing). It comes back the moment it's turned ON.
		if strings.TrimSpace(marker) == "1" && strings.TrimSpace(off) != "1" {
			desc := h.displayName(id)
			if dn, ok, _ := st.KVGet("display_name"); ok && strings.TrimSpace(dn) != "" {
				desc = strings.TrimSpace(dn)
			} else if task, _, _ := st.KVGet("task"); strings.TrimSpace(task) != "" {
				desc = strings.TrimSpace(task)
			}
			// command = Telegram-valid form of the id (dash→underscore, unique);
			// desc = friendly text shown in the slash menu. mr-flow-next matches the
			// typed command to g.Command and invokes g.ID, so the id keeps its dashes.
			parts = append(parts, id+"|"+telegramCommand(id)+"|"+sanitizeDesc(desc))
		}
		_ = st.Close()
	}

	opath, err := h.d.LoketStorePath(OrchestratorID)
	if err != nil {
		return 0
	}
	ost, err := loket.OpenStore(opath)
	if err != nil {
		return 0
	}
	_ = ost.KVSet("groups", strings.Join(parts, ";"))
	_ = ost.Close()
	// Push daftar group LIVE ke menu slash Telegram (setMyCommands). Group dihapus →
	// menu auto-nyusut; semua dihapus → menu kosong. Fix bug "slash nyangkut group lama"
	// (owner 2026-06-20). Best-effort (telegram_commands.go).
	h.syncTelegramCommands(parts)
	return len(parts)
}
