// fresh_index_ext.go — GROWTH-POINT (NON-frozen sibling fresh_index.go soft-lock).
//
// Owner 2026-06-25 (opsi A — "brain self-maintaining semantic"): LEBARIN fresh-recall index ke
// SEMUA tambahan baru, biar nambah doktrin/insting/knowledge → otomatis ke-recall BY-MAKNA dalam
// ≤2 menit (ticker fresh-rebuild) TANPA `brain-reembed`+`buildindex` manual. Sebelumnya fresh-index
// cuma cakup mem_type federation (recovery_instinct/collective_knowledge); sekarang + project +
// doctrine (= mem_type insting/doktrin/knowledge yang di-add via GUI/tool/seed).
//
// Pola Rule-7: extend var `freshMemTypes` lewat init() sibling (NOL edit file soft-lock), persis
// kayak anchor_noise_ext.go / instinctenrich_ext.go. Switch: ENV FLOWORK_FRESH_MEMTYPES (comma)
// = override TOTAL daftar mem_type (mis. "project" doang, atau balik ke federation aja).
//
// Catatan biaya: fresh-rebuild re-embed set (cap freshMaxDrawers=2000 paling-baru) tiap set BERUBAH
// (change-detect → skip kalau sama). Skala sekarang (~ratusan drawer) = murah. Kalau korpus balik
// jutaan + sering nambah, pertimbangkan embed INCREMENTAL (cuma drawer baru) — refinement nanti.

package brain

import (
	"os"
	"strings"
)

func init() {
	if env := strings.TrimSpace(os.Getenv("FLOWORK_FRESH_MEMTYPES")); env != "" {
		var mt []string
		for _, s := range strings.Split(env, ",") {
			if s = strings.TrimSpace(s); s != "" {
				mt = append(mt, s)
			}
		}
		if len(mt) > 0 {
			freshMemTypes = mt // override total (kill-switch / tuning)
			return
		}
	}
	// DEFAULT (opsi A): warisan federation + SEMUA tipe konten (project/doctrine/basic_instinct/
	// reference) = nambah doktrin/insting/knowledge apapun → auto-semantic dalam ≤2 menit.
	freshMemTypes = appendUnique(freshMemTypes, "project", "doctrine", "basic_instinct", "reference")
}

func appendUnique(base []string, extra ...string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	for _, b := range base {
		seen[b] = true
	}
	for _, e := range extra {
		if !seen[e] {
			base = append(base, e)
			seen[e] = true
		}
	}
	return base
}
