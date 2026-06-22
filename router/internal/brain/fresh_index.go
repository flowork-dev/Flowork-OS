// fresh_index.go — F5 (D32-INC4 enabler): FRESH-RECALL untuk knowledge yang BARU di-promote
// ke shared-brain. Akar masalah: index utama `brain.vindex` di-build MANUAL + di-cache, jadi
// drawer baru (mis. recovery-instinct dari INC-4) KESIMPAN tapi belum ke-recall sampe rebuild
// (~jam-an) + restart. Tanpa fresh-recall, imunitas-kolektif INC-4 mati.
//
// SOLUSI (cabut-akar, VECTOR-ONLY, hormatin "jangan hybrid"): index VECTOR KEDUA yang KECIL &
// in-memory, isinya cuma drawer federation (mem_type='recovery_instinct'), di-rebuild periodik
// dari DB (change-detect → hemat). Di-query BARENG index utama di SemanticRetrieve lalu di-MERGE.
// AMAN by-construction: index utama (859k) GAK DISENTUH; fresh-index kosong/error → SemanticRetrieve
// persis perilaku lama (0 regresi). Begitu index utama rebuild (nyerap yg baru), fresh-index reset.

package brain

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"

	"github.com/flowork-os/flowork_Router/internal/brain/vecindex"
)

// freshMemTypes — mem_type yang masuk fresh-index (knowledge fresh hasil federation/promote).
// Sengaja sempit (murah + relevan); lebarin di sini kalau mau fresh-recall lebih luas.
var freshMemTypes = []string{"recovery_instinct"}

const freshMaxDrawers = 2000

var (
	freshMu  sync.Mutex
	freshIdx *vecindex.Index
	freshSig string // signature set terakhir (count|maxfiled) → skip rebuild kalau sama
)

// freshWhereIn — "?,?,..." + args buat freshMemTypes.
func freshWhereIn() (string, []any) {
	ph := ""
	args := make([]any, 0, len(freshMemTypes))
	for i, t := range freshMemTypes {
		if i > 0 {
			ph += ","
		}
		ph += "?"
		args = append(args, t)
	}
	return ph, args
}

// RebuildFreshIndex — rebuild fresh vector-index dari drawer federation aktif. Change-detect:
// skip kalau set ga berubah (count|max(filed_at)). Di-panggil ticker boot router + periodik.
// Resilient: error apa pun → fresh-index lama dipertahankan (atau nil), SemanticRetrieve aman.
func RebuildFreshIndex(ctx context.Context) (int, error) {
	if !Available() {
		return 0, nil
	}
	db, err := Open()
	if err != nil {
		return 0, err
	}
	inPH, inArgs := freshWhereIn()

	// signature (change-detection).
	var cnt int
	var maxFiled sql.NullString
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*), MAX(filed_at) FROM drawers WHERE deleted_at IS NULL AND mem_type IN ("+inPH+")",
		inArgs...).Scan(&cnt, &maxFiled); err != nil {
		return 0, err
	}
	sig := fmt.Sprintf("%d|%s", cnt, maxFiled.String)
	freshMu.Lock()
	same := sig == freshSig
	freshMu.Unlock()
	if same {
		return cnt, nil // ga berubah → skip embed (hemat)
	}
	if cnt == 0 {
		freshMu.Lock()
		freshIdx, freshSig = nil, sig
		freshMu.Unlock()
		return 0, nil
	}

	// ambil konten → embed (engine SAMA dgn index utama → vektor align).
	rows, err := db.QueryContext(ctx,
		"SELECT id, content FROM drawers WHERE deleted_at IS NULL AND mem_type IN ("+inPH+") ORDER BY filed_at DESC LIMIT ?",
		append(append([]any{}, inArgs...), freshMaxDrawers)...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var ids []string
	var vecs [][]float32
	for rows.Next() {
		var id, content string
		if rows.Scan(&id, &content) != nil {
			continue
		}
		v, eerr := embedQueryLocal(ctx, content)
		if eerr != nil || len(v) == 0 {
			continue // skip yg gagal embed; sisanya tetep ke-index
		}
		ids = append(ids, id)
		vecs = append(vecs, v)
	}
	if len(ids) == 0 {
		return 0, nil // semua gagal embed → jangan timpa index lama
	}
	idx, berr := vecindex.Build(ids, vecs)
	if berr != nil {
		return 0, berr
	}
	freshMu.Lock()
	freshIdx, freshSig = idx, sig
	freshMu.Unlock()
	return len(ids), nil
}

// freshRetrieve — query fresh-index by-makna (vector). nil kalau fresh-index kosong → caller
// (SemanticRetrieve) jatuh ke perilaku lama. Skor di-normalisasi (0,1] mirror vectorRetrieve.
func freshRetrieve(ctx context.Context, db *sql.DB, query string, limit, maxLen int, wings []string) []Snippet {
	freshMu.Lock()
	idx := freshIdx
	freshMu.Unlock()
	if idx == nil || db == nil {
		return nil
	}
	qv, err := embedQueryLocal(ctx, query)
	if err != nil || len(qv) != idx.Dim() {
		return nil
	}
	hits := idx.Search(qv, limit*4)
	if len(hits) == 0 {
		return nil
	}
	ph := make([]string, len(hits))
	args := make([]any, 0, len(hits)+len(wings))
	for i, h := range hits {
		ph[i] = "?"
		args = append(args, h.ID)
	}
	q := "SELECT id, wing, room, content FROM drawers WHERE id IN (" + joinComma(ph) + ") AND deleted_at IS NULL"
	if len(wings) > 0 {
		wp := make([]string, len(wings))
		for i, w := range wings {
			wp[i] = "?"
			args = append(args, w)
		}
		q += " AND wing IN (" + joinComma(wp) + ")"
	}
	rows, qerr := db.QueryContext(ctx, q, args...)
	if qerr != nil {
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

// mergeFresh — gabung hasil index-utama + fresh, dedup by DrawerID, urut skor desc, cap limit.
// Dipakai SemanticRetrieve (additive — main tetep prioritas via skor-nya sendiri).
func mergeFresh(main, fresh []Snippet, limit int) []Snippet {
	if len(fresh) == 0 {
		return main
	}
	seen := map[string]bool{}
	all := make([]Snippet, 0, len(main)+len(fresh))
	for _, s := range main {
		if !seen[s.DrawerID] {
			seen[s.DrawerID] = true
			all = append(all, s)
		}
	}
	for _, s := range fresh {
		if !seen[s.DrawerID] {
			seen[s.DrawerID] = true
			all = append(all, s)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Score > all[j].Score })
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}
