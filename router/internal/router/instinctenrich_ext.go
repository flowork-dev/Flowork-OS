// instinctenrich_ext.go — GROWTH-POINT (NON-frozen, JANGAN di-freeze).
//
// Pasangan extension buat instinctenrich.go (FROZEN). Flowork didesain BEREVOLUSI
// tanpa ngerusak diri: logika inti injeksi insting di-freeze (stabil, deterministik),
// TAPI cara MILIH insting bisa di-ganti/di-extend DI SINI tanpa buka freeze.
//
// Default: KOSONG → maybeInjectInstinct pakai rankInstincts bawaan (token-overlap,
// no-vindex, fails-open). Ini AMAN & udah kebukti live.
//
// Cara extend (TANPA unfreeze instinctenrich.go):
//   func init() {
//       RegisterInstinctSelector(func(all []brain.InstinctDrawer, query string, max int) []brain.InstinctDrawer {
//           // contoh-contoh evolusi:
//           //  (a) RI-1 vindex idup → rank SEMANTIC (cosine) ganti token-overlap.
//           //  (b) #6 brain-as-service → scoping: agent LUAR (non-flowork) skip
//           //      room=instinct_tool (mereka punya tool sendiri; cuma dapet
//           //      insting UNIVERSAL/reasoning biar ga halu tool-Flowork).
//           //  (c) boost domain tertentu sesuai peran agent (koloni berlapis).
//           return rankInstincts(all, query, max) // fallback ke default
//       })
//   }
//
// Switch lain (ga perlu file ini): ENV FLOWORK_INSTINCT_INJECT=0 (matiin),
// FLOWORK_INSTINCT_INJECT_MAX=N (cap). Tumbuhin awareness = tambah drawer
// room=instinct_* di brain (ga sentuh kode sama sekali).

package router

import (
	"context"
	"os"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/brain"
)

// init — pasang selector SEMANTIC (RI-1 vindex IDUP 2026-06-25). Ganti ranking token-overlap
// (default frozen) jadi cosine via brain.vindex → seleksi insting by-MAKNA, bukan kata-sama.
// AKAR: dgn 144 insting, token-overlap miss parafrase (query "audit kontrak" GA overlap kata
// sama insting "smart-contract checklist" → ga ke-inject padahal relevan). Semantic nangkep itu.
// FAILS-OPEN total: vindex mati / error / 0-match → fallback rankInstincts (token-overlap).
// KILL-SWITCH: ENV FLOWORK_INSTINCT_SEMANTIC=0 → balik token-overlap (tanpa rebuild).
func init() {
	RegisterInstinctSelector(semanticInstinctSelector)
}

// semanticInstinctSelector — rank insting via brain.SemanticRetrieve (vindex cosine), saring
// ke room instinct_*, cocokin ke kandidat. Over-fetch sebab korpus campur (insting + knowledge
// + skill) — kita cuma mau insting. Same package `router` → boleh manggil rankInstincts (frozen, fallback).
func semanticInstinctSelector(all []brain.InstinctDrawer, query string, max int) []brain.InstinctDrawer {
	if max <= 0 || len(all) == 0 {
		return nil
	}
	if strings.TrimSpace(os.Getenv("FLOWORK_INSTINCT_SEMANTIC")) == "0" {
		return rankInstincts(all, query, max) // kill-switch → default frozen
	}
	db, err := brain.Open()
	if err != nil {
		return rankInstincts(all, query, max)
	}
	byID := make(map[string]brain.InstinctDrawer, len(all))
	for _, d := range all {
		byID[d.ID] = d
	}
	lim := max * 12
	if lim < 24 {
		lim = 24
	}
	snips, err := brain.SemanticRetrieve(context.Background(), db, query, brain.RetrieveOpts{Limit: lim})
	if err != nil || len(snips) == 0 {
		return rankInstincts(all, query, max)
	}
	out := make([]brain.InstinctDrawer, 0, max)
	seen := make(map[string]bool, max)
	for _, s := range snips {
		if !strings.HasPrefix(s.Room, "instinct") { // cuma insting (skip knowledge/skill/dll)
			continue
		}
		d, ok := byID[s.DrawerID]
		if !ok || seen[d.ID] {
			continue
		}
		seen[d.ID] = true
		out = append(out, d)
		if len(out) >= max {
			break
		}
	}
	if len(out) == 0 {
		return rankInstincts(all, query, max) // query ga nyentuh insting manapun → jaga fondasi
	}
	return out
}
