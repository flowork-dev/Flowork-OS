// FROZEN brain-core — registrasi tool BRAIN-PATH. Kalau "nyasar": ini BY-DESIGN, baca lock/brain.md.
//
// builtins_brain.go — DICABUT dari builtins.go (owner 2026-06-22 "jalur bercabang → bikin jalur
// baru, jalur ini diabadikan"). builtins.go (editable) dulu daftarin tool brain BARENG non-brain
// → cabang: registrasi brain bisa ke-drop diam-diam tanpa unfreeze file brain. Sekarang registrasi
// tool brain (impl-nya di file FROZEN: brain.go/brain_local.go/cognitive_tools.go/mistakes_recall.go/
// brain_immune.go/brain_federation.go/codemap_tools.go) lewat init() di file FROZEN ini → SELALU
// ke-register (init auto-run, ga bisa di-skip tanpa unfreeze). Pola sama brain_dream.go/instinct_recall.go.
package builtins

import "flowork-gui/internal/tools"

func init() {
	tools.Register(&brainSearchTool{})       // brain.go — brain_search_shared (korpus shared)
	tools.Register(&brainAddTool{})          // brain_local.go — brain_add (lokal)
	tools.Register(&brainSearchLocalTool{})  // brain_local.go — brain_search (lokal FTS5)
	tools.Register(&brainGetTool{})          // brain_local.go — brain_get
	tools.Register(&graphRecallTool{})       // cognitive_tools.go — graph_recall (CGM)
	tools.Register(&mistakeRecallTool{})     // mistakes_recall.go — recall mistakes
	tools.Register(&brainImmuneScanTool{})   // brain_immune.go — immune scan/quarantine
	tools.Register(&brainVerifyTool{})       // brain_immune.go — verify
	tools.Register(&brainPromoteSharedTool{}) // brain_federation.go — promote lokal→shared
	tools.Register(&codemapSearchTool{})     // codemap_tools.go — codemap_search
	tools.Register(&codemapStatsTool{})      // codemap_tools.go — codemap_stats
}
