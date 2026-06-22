// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-16 (LOCKED ≠ FREEZE).
// re-edit 2026-06-17 (owner-approved): vectorRetrieve filter deleted_at IS NULL + over-fetch selalu
//
//	— drawer tombstoned JANGAN di-retrieve (paritas dgn FTS retrieve.go). Re-LOCK.
//
// AI lain: JANGAN otak-atik tanpa izin owner. Teruji (TestSemanticLive: vector murni by-makna).
// 2026-06-22 (owner-approved, F5/D32-INC4): SemanticRetrieve MERGE fresh-recall (freshRetrieve di
//   fresh_index.go) — ADDITIF, vector-only, fresh kosong → 0 regresi. Re-LOCK.
//
// semantic.go — #6 ARSITEK BARU: pencarian brain by-MAKNA (Quantum Recall, vector murni).
//
// TUJUAN (buat AI lain): agent cari memori by-MAKNA (semantic), BUKAN keyword. Owner 2026-06-16:
// "fokus arsitek baru saja, jangan hybrid" — JADI INI BUKAN fusion FTS+vector (yg bikin bingung).
// SATU JALUR: kalau index semantik (brain.vindex, dari brain-buildindex atas 5jt drawer) udah ada
// -> pakai VECTOR murni. Kalau belum ada (re-embed masih jalan ~jam-an) -> fallback FTS SEMENTARA
// biar brain gak mati pas transisi. Begitu index jadi, otomatis full semantic.
//
// Query di-embed bge-m3 LOKAL (provider 'local' — engine SAMA dgn index -> vektor align). Konten
// hit diambil dari tabel `drawers` (PK lookup) di brain 5jt yg sama. retrieve.go (LOCKED, FTS)
// gak disentuh — fallback manggil Retrieve apa adanya.
package brain

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/flowork-os/flowork_Router/internal/brain/vecindex"
	"github.com/flowork-os/flowork_Router/internal/providers/embedding"
	"github.com/flowork-os/flowork_Router/internal/sidecar"
)

const vindexEnv = "FLOWORK_BRAIN_VINDEX"

var (
	vMu  sync.Mutex
	vIdx *vecindex.Index
)

// vindexPath — lokasi index. roadmap_sidecar Fase 0/3: dipindah ke paket sidecar
// (sumber path tunggal). Legacy-default = chain lama PERSIS (FLOWORK_BRAIN_VINDEX >
// <exe_dir>/brain/brain.vindex > cwd brain/).
func vindexPath() string {
	return sidecar.Vindex()
}

// loadVIndex — load index sekali (cache). nil kalau file belum ada (vector OFF -> FTS sementara).
// Di-retry tiap call sampai file muncul (index baru jadi setelah re-embed selesai).
func loadVIndex() *vecindex.Index {
	vMu.Lock()
	defer vMu.Unlock()
	if vIdx != nil {
		return vIdx
	}
	p := vindexPath()
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	idx, err := vecindex.Load(p)
	if err != nil {
		return nil
	}
	vIdx = idx
	return vIdx
}

// VectorReady — true kalau index semantik udah ke-load (buat /status & debug).
func VectorReady() bool { return loadVIndex() != nil }

// embedQueryLocal — embed query pakai bge-m3 LOKAL (provider 'local'), balik unit []float32.
func embedQueryLocal(ctx context.Context, query string) ([]float32, error) {
	p := embedding.Get("local")
	if p == nil {
		return nil, fmt.Errorf("embedder 'local' belum ke-register")
	}
	res, err := p.Embed(ctx, embedding.Request{Input: []string{query}})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 || len(res.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding kosong")
	}
	raw := res.Data[0].Embedding
	v := make([]float32, len(raw))
	var ss float64
	for i, x := range raw {
		v[i] = float32(x)
		ss += x * x
	}
	if n := float32(math.Sqrt(ss)); n > 0 {
		for i := range v {
			v[i] /= n
		}
	}
	return v, nil
}

// vectorRetrieve — semantic murni: embed -> vecindex top-k -> konten dari drawers. nil kalau off.
// wings (opsional): batasi ke wing tertentu (over-fetch lalu filter SQL — jaga paritas dgn FTS).
func vectorRetrieve(ctx context.Context, db *sql.DB, query string, limit, maxLen int, wings []string) []Snippet {
	idx := loadVIndex()
	if idx == nil || db == nil {
		return nil
	}
	qv, err := embedQueryLocal(ctx, query)
	if err != nil || len(qv) != idx.Dim() {
		return nil
	}
	// over-fetch SELALU: sebagian hit bakal kebuang filter wing ATAU deleted_at
	// (drawer tombstoned skip — 2026-06-17). Index live-only abis re-embed-fix, tapi
	// over-fetch jaga transisi (index lama bisa masih punya tombstoned).
	searchK := limit * 6
	hits := idx.Search(qv, searchK)
	if len(hits) == 0 {
		return nil
	}
	ph := make([]string, len(hits))
	args := make([]any, 0, len(hits)+len(wings))
	for i, h := range hits {
		ph[i] = "?"
		args = append(args, h.ID)
	}
	// deleted_at IS NULL: drawer SOFT-DELETED JANGAN di-retrieve (paritas dgn FTS).
	q := "SELECT id, wing, room, content FROM drawers WHERE id IN (" + strings.Join(ph, ",") + ") AND deleted_at IS NULL"
	if len(wings) > 0 {
		wp := make([]string, len(wings))
		for i, w := range wings {
			wp[i] = "?"
			args = append(args, w)
		}
		q += " AND wing IN (" + strings.Join(wp, ",") + ")"
	}
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	byID := map[string]Snippet{}
	for rows.Next() {
		var id, wing, room, content string
		if rows.Scan(&id, &wing, &room, &content) == nil {
			if maxLen > 0 {
				content = truncateRunes(content, maxLen)
			}
			byID[id] = Snippet{DrawerID: id, Wing: wing, Room: room, Content: content}
		}
	}
	// jaga URUTAN vector (relevansi makna); skor = dot ter-normalisasi (0,1]. Cap di limit.
	out := make([]Snippet, 0, limit)
	var top float64 = 1
	if hits[0].Score > 0 {
		top = float64(hits[0].Score)
	}
	for _, h := range hits {
		if len(out) >= limit {
			break
		}
		if sn, ok := byID[h.ID]; ok {
			sn.Score = float64(h.Score) / top
			out = append(out, sn)
		}
	}
	return out
}

// SemanticRetrieve — ARSITEK BARU (by-makna). Vector murni kalau index siap; FTS SEMENTARA kalau
// belum (transisi). BUKAN hybrid/fusion. Drop-in pengganti Retrieve di SEMUA jalur search agent
// (search-drawers, enrichment auto-context, handler search). Hormatin opts.Wings.
func SemanticRetrieve(ctx context.Context, db *sql.DB, query string, opts RetrieveOpts) ([]Snippet, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 6
	}
	vec := vectorRetrieve(ctx, db, query, limit, opts.MaxContentLen, opts.Wings)
	// F5 (2026-06-22, owner-approved): MERGE fresh-recall — drawer federation BARU (mis.
	// recovery-instinct INC-4) yg belum masuk index utama (di-build manual + cached). Tetap
	// VECTOR-ONLY (bukan hybrid FTS); ADDITIF: fresh kosong → mergeFresh(vec,nil)==vec → 0
	// regresi. Lihat fresh_index.go.
	fresh := freshRetrieve(ctx, db, query, limit, opts.MaxContentLen, opts.Wings)
	if len(vec) > 0 || len(fresh) > 0 {
		return mergeFresh(vec, fresh, limit), nil // semantic murni (utama + fresh)
	}
	return Retrieve(ctx, db, query, opts) // fallback SEMENTARA (index belum jadi)
}
