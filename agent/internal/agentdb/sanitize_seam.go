// sanitize_seam.go — SEAM (Pola B): scrub rahasia SEBELUM row masuk sink DB.
// 📄 Dok: FLowork_os/lock/secscrub.md
//
// AKAR chokepoint (2026-07-02): call-site LogInteraction/LogDecision/AddMistake
// nyebar (kernelhost, agentmgr, builtins, slashcmd, runtime host) — scrub di
// caller = tambal per-lubang. Di sini = SEMUA jalur ke-cover sekali.
// Default no-op (perilaku lama persis); diisi dari file non-frozen
// (agent/secscrub_ext.go → internal/secscrub). agentdb TIDAK import secscrub
// (dependency-free — colokan doang).
package agentdb

var (
	// SanitizeText — dipanggil di content/rationale/title sebelum INSERT.
	SanitizeText = func(s string) string { return s }
	// SanitizeMeta — dipanggil di metadata/inputs (map) sebelum INSERT.
	SanitizeMeta = func(m map[string]any) map[string]any { return m }
)
