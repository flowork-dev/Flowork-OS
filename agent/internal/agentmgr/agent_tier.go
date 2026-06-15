// agent_tier.go — penanda TIER agent (primary vs extension) + tool primary-only.
//
// Konsep (LOCKED design 2026-06-05, SISA_ROADMAP.MD):
//
//   PRIMARY   = agent INTI yang nyatu sama engine (mr-flow, dst). Dapet brain
//               5jt shared (brain_search_shared / korpus router) + tool penuh +
//               protected. coder/verifier = ENGINE (bukan agent folder) → NOT di sini.
//   EXTENSION = agent portable (crypto/music/zodiak/saham/promo/...). Brain-nya
//               DI FOLDER SENDIRI (state.db lokal via brain_search). GA nyolok
//               korpus 5jt shared → bikin portable + anti-halu.
//
// WIRING INVARIANT (Aturan #1 Mr.Dev): perubahan ADDITIVE. Primary's 5jt = jangan
// diputus. Extension's brain-folder = jangan diputus. Gate ini cuma NGATUR siapa
// dapet tool 5jt — bukan motong brain lokal siapa pun.
//
// ⚠️ BY DESIGN — JANGAN "FIX" JADI BUKA AKSES (audit 2026-06-15, owner-confirmed
// "jangan buka aksesnya"). Worker/extension/group SENGAJA gak nyolok brain 5jt shared
// + skip enrich berat (lihat router/enrich_tier.go) — INI BUKAN KABEL PUTUS. Alasan:
// anti-429 (volume call worker gede), fokus tugas (knowledge proyek ga relevan buat
// worker), portable. Sesuai arsitektur baru: "agent bodoh, engine pinter" + tiering
// hemat kuota. Kalau owner/AI berikut ngeliat ini & ngira bug → STOP, ini keputusan sadar.
package agentmgr

import "strings"

// primaryAgents — set agent tier PRIMARY. Extensible (owner bisa nambah lewat
// sini). Lowercase, di-trim pas compare.
var primaryAgents = map[string]bool{
	"mr-flow":      true,
	"mr-flow-next": true, // loket-native incarnation, migrated alongside mr-flow (Phase B)
}

// IsPrimaryAgent — true kalau agent tier primary (inti engine).
func IsPrimaryAgent(id string) bool {
	return primaryAgents[strings.ToLower(strings.TrimSpace(id))]
}

// AgentTier — "primary" | "extension". Default extension (brain folder sendiri).
func AgentTier(id string) string {
	if IsPrimaryAgent(id) {
		return "primary"
	}
	return "extension"
}

// primaryOnlyTools — tool yang CUMA buat tier primary. brain_search_shared =
// korpus 5jt di router; extension pakai brain_search (lokal, folder sendiri).
var primaryOnlyTools = map[string]bool{
	"brain_search_shared": true,
}

// IsPrimaryOnlyTool — true kalau tool ini khusus tier primary (extension ditolak
// di ToolRunHandler, ga di-expose di ToolSpecsHandler).
func IsPrimaryOnlyTool(name string) bool {
	return primaryOnlyTools[strings.TrimSpace(name)]
}
