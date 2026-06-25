// === FROZEN 2026-06-26 · Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS ===
// ⚠️ FROZEN — jangan edit. Stabil+teruji. Switch lantai via FLOWORK_SEARCH_MINSCORE. lock/semantic.md
//
// semantic_threshold_ext.go — RETRIEVE dgn LANTAI RELEVANSI ABSOLUT.
//
// Kenapa: SemanticRetrieve (FROZEN, semantic.go) menormalisasi skor ke hit teratas
// → hasil #1 selalu Score 1.0 walau query sampah, dan ga ada cara nyaring hasil
// loosely-related → search GUI "ngak valid". Fungsi ini PARALEL (bukan ganti):
//   • pakai building-block se-package yg sama (loadVIndex, embedQueryLocal, truncateRunes)
//   • skor = cosine ABSOLUT (idx.Cosine) — bukan rasio ke top
//   • buang hit di bawah minCosine → noise ke-saring
// Additif & non-frozen. semantic.go tetep beku. Dipanggil handler search di balik
// switch FLOWORK_SEARCH_MINSCORE (default kalibrasi, 0 = matikan = perilaku lama).

package brain

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

// SemanticRetrieveScored — vector-murni dgn cosine ABSOLUT + lantai relevansi.
// minCosine<=0 → tanpa lantai (cuma skor absolut). Balik snippet ter-urut skor desc,
// di-cap limit. Index belum siap → nil (caller bebas fallback ke SemanticRetrieve).
func SemanticRetrieveScored(ctx context.Context, db *sql.DB, query string, opts RetrieveOpts, minCosine float64) ([]Snippet, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 6
	}
	idx := loadVIndex()
	if idx == nil || db == nil {
		return nil, nil
	}
	qv, err := embedQueryLocal(ctx, query)
	if err != nil || len(qv) != idx.Dim() {
		return nil, nil
	}
	// over-fetch lebih lebar (×8): sebagian hit kebuang filter wing/tombstone/lantai.
	hits := idx.Search(qv, limit*8)
	if len(hits) == 0 {
		return nil, nil
	}
	// skor absolut per-id (dipakai utk filter + sort + tampil).
	cos := make(map[string]float64, len(hits))
	ph := make([]string, 0, len(hits))
	args := make([]any, 0, len(hits)+len(opts.Wings))
	for _, h := range hits {
		c := float64(idx.Cosine(h.Score))
		if minCosine > 0 && c < minCosine {
			continue // di bawah lantai → noise, buang
		}
		if _, dup := cos[h.ID]; dup {
			continue
		}
		cos[h.ID] = c
		ph = append(ph, "?")
		args = append(args, h.ID)
	}
	if len(ph) == 0 {
		return nil, nil // semua di bawah lantai → ga ada yg relevan (lebih jujur dari noise)
	}
	q := "SELECT id, wing, room, content FROM drawers WHERE id IN (" + strings.Join(ph, ",") + ") AND deleted_at IS NULL"
	if len(opts.Wings) > 0 {
		wp := make([]string, len(opts.Wings))
		for i, w := range opts.Wings {
			wp[i] = "?"
			args = append(args, w)
		}
		q += " AND wing IN (" + strings.Join(wp, ",") + ")"
	}
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Snippet, 0, len(cos))
	for rows.Next() {
		var id, wing, room, content string
		if rows.Scan(&id, &wing, &room, &content) != nil {
			continue
		}
		if opts.MaxContentLen > 0 {
			content = truncateRunes(content, opts.MaxContentLen)
		}
		out = append(out, Snippet{DrawerID: id, Wing: wing, Room: room, Content: content, Score: cos[id]})
	}
	// urutkan by cosine absolut desc (urutan SQL IN tak terjamin), cap limit.
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
