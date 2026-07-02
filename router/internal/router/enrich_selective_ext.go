// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// enrich_selective_ext.go — sibling NON-frozen (Rule 7): ENRICHMENT SELEKTIF (roadmap F-A1).
//
// AKAR yang dicabut: maybeEnrichBrain (frozen) pakai brain.SemanticRetrieve yang
// MENORMALISASI skor ke hit teratas → hit #1 selalu 1.0 walau query-nya ga nyambung
// → brain nyuntik top-K snippet TIAP call (boros token + noise bikin model "tenggelam").
// Fix: lewat seam `enrichRetrieve` (POLA B di brainenrich.go), ganti sumber ke
// SemanticRetrieveScored (cosine ABSOLUT + lantai) — query ga relevan → 0 snippet →
// maybeEnrichBrain skip suntik (selektif beneran, bukan cap doang).
//
// SWITCH (GUI = kebenaran): FLOWORK_ENRICH_MINSCORE (fwswitch registry, live tanpa restart).
//   0 / kosong = OFF → perilaku lama byte-identik. Saran 0.30–0.45.
// Fail-open: index vektor belum siap ATAU retrieve error → fallback perilaku lama.
// File ini DIHAPUS → seam balik default (perilaku lama) → inti tetep jalan (delete-test).

package router

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/brain"
)

// enrichMinScore — lantai cosine absolut dari switch GUI. Dibaca per-request
// (os.Getenv) → nilai GUI hidup lewat fwswitch watcher tanpa restart.
func enrichMinScore() float64 {
	v := strings.TrimSpace(os.Getenv("FLOWORK_ENRICH_MINSCORE"))
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 || f >= 1 {
		return 0 // invalid → off (fail-open, jangan matiin enrichment gara-gara typo)
	}
	return f
}

func init() {
	legacy := enrichRetrieve
	enrichRetrieve = func(ctx context.Context, db *sql.DB, query string, opts brain.RetrieveOpts) ([]brain.Snippet, error) {
		min := enrichMinScore()
		if min <= 0 || !brain.VectorIndexReady() {
			return legacy(ctx, db, query, opts) // switch off / index belum siap → perilaku lama
		}
		snips, err := brain.SemanticRetrieveScored(ctx, db, query, opts, min)
		if err != nil {
			return legacy(ctx, db, query, opts) // fail-open: error → perilaku lama
		}
		if len(snips) == 0 {
			log.Printf("flow_router enrich-selective: 0 snippet ≥ lantai %.2f (query=%.48q) → skip suntik knowledge (hemat)", min, query)
		}
		return snips, nil
	}
}
