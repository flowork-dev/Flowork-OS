// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-16 (LOCKED ≠ FREEZE).
// AI lain: JANGAN otak-atik tanpa izin owner. TUJUAN FILE: indeks vektor sovereign (anti-halu
// grounding) buat semantic RAG brain — kuantisasi 8-bit + top-k search, pure-Go no-cgo, portable
// (ikut OS flashdisk). TERUJI di data bge-m3 ASLI: recall@10=0.985, recall@1=1.000 (vecindex_test).
// Pasangan: embedder bge-m3 via llama.cpp (Ollama@OS / llama-server@dev), query+drawer L2-normalized.
//
// Package vecindex — indeks vektor SOVEREIGN in-process buat semantic search brain.
// Pure-Go, NO cgo, NO dependency luar (sesuai doktrin "tubuh mandiri, portable multi-OS").
//
// Embedding (bge-m3, dim 1024) datang UNIT-NORMALIZED → cosine = dot product. Tiap vektor
// disimpan sbg kode 8-bit symmetric scalar terhadap SATU skala global (max|komponen|).
// Ranking pakai int8 dot product itu order-equivalent sama cosine (konstanta dequant sama
// buat semua baris) → dequant di-skip. Di data bge-m3 ASLI: recall@1 ~0.99, recall@10 ~0.98
// vs exact float32 cosine, di 1 byte/dim (~5GB buat 5jt vektor) — diverifikasi 2026-06-16.
// 4-bit TurboQuant (lebih kecil, buat RAM ketat spt Android) = opsi nanti; 8-bit menang di
// kesederhanaan + recall buat tubuh local-first.
package vecindex

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"runtime"
	"sort"
	"sync"
)

const qLevels = 127 // rentang int8 symmetric [-127, 127]

// Index — store vektor ter-kuantisasi, immutable. Build sekali, search berkali-kali.
type Index struct {
	dim   int
	scale float32 // skala kuantisasi global: max|komponen| seluruh vektor
	codes []int8  // n*dim row-major
	ids   []string
}

// Hit — satu hasil search.
type Hit struct {
	ID    string
	Score float32 // int8 dot product (proporsional cosine; makin tinggi makin dekat)
}

// Build — kuantisasi vecs (tiap len==dim, idealnya unit-normalized) jadi Index. ids[i] label
// buat vecs[i]. Error kalau shape mismatch / kosong.
func Build(ids []string, vecs [][]float32) (*Index, error) {
	if len(ids) != len(vecs) {
		return nil, errors.New("vecindex: panjang ids/vecs beda")
	}
	if len(vecs) == 0 {
		return nil, errors.New("vecindex: input kosong")
	}
	dim := len(vecs[0])
	if dim == 0 {
		return nil, errors.New("vecindex: dim nol")
	}
	var maxAbs float32
	for _, v := range vecs {
		if len(v) != dim {
			return nil, errors.New("vecindex: vektor ragged (dim beda)")
		}
		for _, x := range v {
			if a := float32(math.Abs(float64(x))); a > maxAbs {
				maxAbs = a
			}
		}
	}
	if maxAbs == 0 {
		maxAbs = 1
	}
	ix := &Index{dim: dim, scale: maxAbs, codes: make([]int8, len(vecs)*dim), ids: append([]string(nil), ids...)}
	for i, v := range vecs {
		quantizeInto(v, maxAbs, ix.codes[i*dim:(i+1)*dim])
	}
	return ix, nil
}

// quantizeInto — tulis kode 8-bit symmetric dari v (skala `scale`) ke dst (len==dim).
func quantizeInto(v []float32, scale float32, dst []int8) {
	inv := float32(qLevels) / scale
	for j, x := range v {
		q := int32(math.Round(float64(x * inv)))
		if q > qLevels {
			q = qLevels
		} else if q < -qLevels {
			q = -qLevels
		}
		dst[j] = int8(q)
	}
}

// Len — jumlah vektor. Dim — dimensi.
func (ix *Index) Len() int { return len(ix.ids) }
func (ix *Index) Dim() int { return ix.dim }

type scored struct {
	idx   int
	score int32
}

// pushTopK — sisipkan kandidat ke slice top-k (terurut DESCENDING by score, panjang ≤k).
func pushTopK(top []scored, k int, cand scored) []scored {
	if len(top) < k {
		top = append(top, cand)
		// bubble ke posisi terurut
		for x := len(top) - 1; x > 0 && top[x].score > top[x-1].score; x-- {
			top[x], top[x-1] = top[x-1], top[x]
		}
		return top
	}
	if cand.score <= top[k-1].score {
		return top // gak masuk top-k
	}
	top[k-1] = cand
	for x := k - 1; x > 0 && top[x].score > top[x-1].score; x-- {
		top[x], top[x-1] = top[x-1], top[x]
	}
	return top
}

// Search — top-k id terdekat ke query (cosine via int8 dot), full parallel scan.
func (ix *Index) Search(query []float32, k int) []Hit {
	if k <= 0 || ix.Len() == 0 {
		return nil
	}
	if k > ix.Len() {
		k = ix.Len()
	}
	// #5 binary-vector: korpus JUTAAN → coarse-biner + rerank-int8 (auto >=1jt, lihat binary_ext.go).
	// Default OFF (auto: <1jt = int8 full-scan biasa di bawah ini → ZERO perubahan skala sekarang).
	if ix.useBinary() {
		return ix.searchBinary(query, k)
	}
	q := make([]int8, ix.dim)
	quantizeInto(query, ix.scale, q)

	n := ix.Len()
	workers := runtime.NumCPU()
	if workers > n {
		workers = n
	}
	partial := make([][]scored, workers)
	var wg sync.WaitGroup
	chunk := (n + workers - 1) / workers
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
			top := make([]scored, 0, k)
			for i := lo; i < hi; i++ {
				row := ix.codes[i*ix.dim : (i+1)*ix.dim]
				var dot int32
				for j, qj := range q {
					dot += int32(qj) * int32(row[j])
				}
				top = pushTopK(top, k, scored{i, dot})
			}
			partial[w] = top
		}(w, lo, hi)
	}
	wg.Wait()

	merged := make([]scored, 0, workers*k)
	for _, p := range partial {
		merged = append(merged, p...)
	}
	sort.Slice(merged, func(a, b int) bool { return merged[a].score > merged[b].score })
	if len(merged) > k {
		merged = merged[:k]
	}
	hits := make([]Hit, len(merged))
	for i, m := range merged {
		hits[i] = Hit{ID: ix.ids[m.idx], Score: float32(m.score)}
	}
	return hits
}

// SearchSubset — sama spt Search TAPI cuma nilai baris di `cand` (index ke ix). Buat HYBRID:
// FTS keyword nyaring kandidat → vector RE-RANK kandidat itu aja (cepat, gak scan jutaan).
func (ix *Index) SearchSubset(query []float32, cand []int, k int) []Hit {
	if k <= 0 || len(cand) == 0 {
		return nil
	}
	q := make([]int8, ix.dim)
	quantizeInto(query, ix.scale, q)
	top := make([]scored, 0, k)
	for _, i := range cand {
		if i < 0 || i >= ix.Len() {
			continue
		}
		row := ix.codes[i*ix.dim : (i+1)*ix.dim]
		var dot int32
		for j, qj := range q {
			dot += int32(qj) * int32(row[j])
		}
		top = pushTopK(top, k, scored{i, dot})
	}
	hits := make([]Hit, len(top))
	for i, m := range top {
		hits[i] = Hit{ID: ix.ids[m.idx], Score: float32(m.score)}
	}
	return hits
}

// ── persist (format biner sendiri; portable, no dep) ──────────────────────────
const magic = "FWVQ1\n" // Flowork Vector Quant v1

// Save — tulis index ke path (atomic-ish: caller urus rename kalau perlu).
func (ix *Index) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := w.WriteString(magic); err != nil {
		return err
	}
	var hdr [16]byte
	binary.LittleEndian.PutUint32(hdr[0:], uint32(ix.dim))
	binary.LittleEndian.PutUint32(hdr[4:], math.Float32bits(ix.scale))
	binary.LittleEndian.PutUint64(hdr[8:], uint64(ix.Len()))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	// ids: tiap baris = uint16 len + bytes
	for _, id := range ix.ids {
		var l [2]byte
		binary.LittleEndian.PutUint16(l[:], uint16(len(id)))
		if _, err := w.Write(l[:]); err != nil {
			return err
		}
		if _, err := w.WriteString(id); err != nil {
			return err
		}
	}
	// codes: raw int8 bytes
	buf := make([]byte, len(ix.codes))
	for i, c := range ix.codes {
		buf[i] = byte(c)
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return w.Flush()
}

// Load — baca index dari path yang ditulis Save.
func Load(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	mg := make([]byte, len(magic))
	if _, err := io.ReadFull(r, mg); err != nil {
		return nil, err
	}
	if string(mg) != magic {
		return nil, errors.New("vecindex: magic salah (file bukan index v1)")
	}
	var hdr [16]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	dim := int(binary.LittleEndian.Uint32(hdr[0:]))
	scale := math.Float32frombits(binary.LittleEndian.Uint32(hdr[4:]))
	n := int(binary.LittleEndian.Uint64(hdr[8:]))
	if dim <= 0 || n < 0 {
		return nil, errors.New("vecindex: header rusak")
	}
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		var l [2]byte
		if _, err := io.ReadFull(r, l[:]); err != nil {
			return nil, err
		}
		b := make([]byte, binary.LittleEndian.Uint16(l[:]))
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, err
		}
		ids[i] = string(b)
	}
	raw := make([]byte, n*dim)
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, err
	}
	codes := make([]int8, n*dim)
	for i, b := range raw {
		codes[i] = int8(b)
	}
	return &Index{dim: dim, scale: scale, codes: codes, ids: ids}, nil
}
