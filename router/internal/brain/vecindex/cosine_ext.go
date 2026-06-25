// === FROZEN 2026-06-26 · Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS ===
// ⚠️ FROZEN — jangan edit. Stabil+teruji. Extend lewat sibling baru, BUKAN file ini. lock/semantic.md
//
// cosine_ext.go — pulihin COSINE ABSOLUT dari skor int8-dot mentah hasil Search.
//
// AKAR masalah search "ngak valid": semantic.go (FROZEN) menormalisasi skor ke hit
// teratas (sn.Score = h.Score/top) → hasil #1 SELALU 1.0 walau query-nya sampah.
// Ga ada LANTAI relevansi absolut. Untuk pasang lantai, kita butuh skor absolut.
//
// Matematika (vektor bge-m3 UNIT-normalized, kuantisasi simetris skala global `scale`):
//   code_i  = round(v_i * qLevels/scale)
//   dot(a,b)= Σ code_a·code_b ≈ (qLevels/scale)² · Σ a_i·b_i = (qLevels/scale)² · cosine
//   ⇒ cosine ≈ rawScore · (scale/qLevels)²
// Order-equivalent dgn cosine (sudah dibuktikan recall@1=1.0); ini cuma rescale absolut.
// NON-frozen, additif: ga ngubah Search/ranking yg udah beku.

package vecindex

// Cosine mengembalikan perkiraan cosine absolut [-1,1] dari skor int8-dot mentah
// (field Hit.Score) milik index INI. nil/scale 0 → 0 (aman, fails-soft).
func (ix *Index) Cosine(rawScore float32) float32 {
	if ix == nil || ix.scale == 0 {
		return 0
	}
	r := ix.scale / float32(qLevels)
	c := rawScore * r * r
	if c > 1 {
		c = 1
	} else if c < -1 {
		c = -1
	}
	return c
}
