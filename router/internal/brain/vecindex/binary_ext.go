// binary_ext.go — GROWTH-POINT (NON-frozen). #5 binary-vector recall (scale lever).
//
// AKAR: di korpus JUTAAN drawer, int8 full-scan dot (Search) jadi bottleneck. SOLUSI 2-tahap:
//   (1) COARSE biner: sign-bit tiap dim (int8>0 → 1) → query XNOR vektor → popcount = agreement
//       (≈ cosine, sign-LSH). Cepat banget (128 byte/vektor vs 1024). Ambil top-M kandidat.
//   (2) RERANK int8: SearchSubset(int8 dot) cuma di M kandidat → akurasi int8 BALIK (final score exact).
// Recall tinggi (rerank exact) + jauh lebih cepat di korpus gede.
//
// AUTO-DETECT (owner 2026-06-26): default "auto" → AKTIF cuma kalau korpus >= 1 JUTA drawer.
// Di bawah 1jt → int8 full-scan biasa (ZERO perubahan). Switch FLOWORK_BINARY_VECTOR (auto|on|off),
// threshold FLOWORK_BINARY_VECTOR_MIN (default 1000000). GUI Switch Fitur (prefix FLOWORK_).
//
// CATATAN: ini optimasi VECTOR-SEARCH drawer. Konstitusi/insting injection = jalur LAIN (DB/
// constitution table) → ga kesentuh. Konstitusi default tetep selalu nempel.
package vecindex

import (
	"math/bits"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type binData struct {
	words int
	sigs  []uint64 // n*words, sign-bit per dim
}

var binCache sync.Map // *Index -> *binData (lazy, per-index)

// binaryMinN — threshold auto-aktif (default 1 juta). Env FLOWORK_BINARY_VECTOR_MIN override.
func binaryMinN() int {
	if v := strings.TrimSpace(os.Getenv("FLOWORK_BINARY_VECTOR_MIN")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 1_000_000
}

// useBinary — auto (default): aktif kalau Len>=threshold. on/off: paksa. Anti-degenerate: butuh dim>0.
func (ix *Index) useBinary() bool {
	if ix.dim <= 0 || ix.Len() < 256 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FLOWORK_BINARY_VECTOR"))) {
	case "off", "0", "false":
		return false
	case "on", "1", "true", "force":
		return true
	}
	return ix.Len() >= binaryMinN() // auto
}

func (ix *Index) binSigs() *binData {
	if v, ok := binCache.Load(ix); ok {
		return v.(*binData)
	}
	words := (ix.dim + 63) / 64
	n := ix.Len()
	sigs := make([]uint64, n*words)
	workers := runtime.NumCPU()
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		lo, hi := w*chunk, w*chunk+chunk
		if hi > n {
			hi = n
		}
		if lo >= hi {
			break
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			for i := lo; i < hi; i++ {
				row := ix.codes[i*ix.dim : (i+1)*ix.dim]
				base := i * words
				for j, c := range row {
					if c > 0 {
						sigs[base+j/64] |= 1 << uint(j%64)
					}
				}
			}
		}(lo, hi)
	}
	wg.Wait()
	bd := &binData{words: words, sigs: sigs}
	binCache.Store(ix, bd)
	return bd
}

func signQuery(q []int8, words int) []uint64 {
	out := make([]uint64, words)
	for j, c := range q {
		if c > 0 {
			out[j/64] |= 1 << uint(j%64)
		}
	}
	return out
}

// searchBinary — coarse popcount (top-M) → rerank int8 (top-k exact). Dipanggil dari Search.
func (ix *Index) searchBinary(query []float32, k int) []Hit {
	bd := ix.binSigs()
	q := make([]int8, ix.dim)
	quantizeInto(query, ix.scale, q)
	qs := signQuery(q, bd.words)

	// M = overfetch generous biar true top-k masuk kandidat (rerank exact → recall tinggi).
	M := k * 64
	if M < 2000 {
		M = 2000
	}
	if M > ix.Len() {
		M = ix.Len()
	}
	n := ix.Len()
	workers := runtime.NumCPU()
	if workers > n {
		workers = n
	}
	partial := make([][]scored, workers)
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		lo, hi := w*chunk, w*chunk+chunk
		if hi > n {
			hi = n
		}
		if lo >= hi {
			break
		}
		wg.Add(1)
		go func(w, lo, hi int) {
			defer wg.Done()
			top := make([]scored, 0, M)
			for i := lo; i < hi; i++ {
				base := i * bd.words
				var agree int32
				for wi := 0; wi < bd.words; wi++ {
					// bit sama (NOT XOR) = agreement; makin tinggi makin mirip.
					agree += int32(bits.OnesCount64(^(qs[wi] ^ bd.sigs[base+wi])))
				}
				top = pushTopK(top, M, scored{i, agree})
			}
			partial[w] = top
		}(w, lo, hi)
	}
	wg.Wait()
	cand := make([]int, 0, workers*M)
	for _, p := range partial {
		for _, s := range p {
			cand = append(cand, s.idx)
		}
	}
	// RERANK int8 exact di kandidat → top-k final.
	hits := ix.SearchSubset(query, cand, k)
	sort.Slice(hits, func(a, b int) bool { return hits[a].Score > hits[b].Score })
	return hits
}
