package vecindex

import (
	"fmt"
	"math/rand"
	"testing"
)

// #5: binary coarse + int8 rerank harus recall TINGGI vs int8 full-scan (rerank exact).
// Default (auto, korpus kecil) = int8 biasa (ZERO perubahan); paksa "on" buat tes.
func TestBinaryVectorRecall(t *testing.T) {
	const n, dim, k = 6000, 128, 10
	rng := rand.New(rand.NewSource(42))
	ids := make([]string, n)
	vecs := make([][]float32, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("d%05d", i)
		v := make([]float32, dim)
		for j := range v {
			v[j] = float32(rng.NormFloat64())
		}
		vecs[i] = unitNorm(v)
	}
	idx, err := Build(ids, vecs)
	if err != nil {
		t.Fatal(err)
	}
	q := unitNorm(vecs[123]) // query mirip salah satu vektor (ada jawaban jelas)

	t.Setenv("FLOWORK_BINARY_VECTOR", "off")
	base := idx.Search(q, k)
	t.Setenv("FLOWORK_BINARY_VECTOR", "on")
	bin := idx.Search(q, k)

	if len(base) != k || len(bin) != k {
		t.Fatalf("len base=%d bin=%d, want %d", len(base), len(bin), k)
	}
	want := map[string]bool{}
	for _, h := range base {
		want[h.ID] = true
	}
	hit := 0
	for _, h := range bin {
		if want[h.ID] {
			hit++
		}
	}
	recall := float64(hit) / float64(k)
	if recall < 0.8 {
		t.Errorf("binary recall@%d = %.2f (<0.8) — coarse kekecilan?", k, recall)
	}
	// top-1 WAJIB sama (query = vektor #123 → jawaban pasti).
	if bin[0].ID != base[0].ID {
		t.Errorf("top-1 beda: bin=%s base=%s", bin[0].ID, base[0].ID)
	}
	t.Logf("binary recall@%d = %.2f, top1=%s", k, recall, bin[0].ID)
}

// auto-gate: korpus kecil + env kosong → useBinary FALSE (int8 biasa, no perubahan).
func TestBinaryAutoGateOffWhenSmall(t *testing.T) {
	t.Setenv("FLOWORK_BINARY_VECTOR", "")
	t.Setenv("FLOWORK_BINARY_VECTOR_MIN", "")
	idx, _ := Build([]string{"a", "b", "c"}, [][]float32{{1, 0}, {0, 1}, {1, 1}})
	if idx.useBinary() {
		t.Error("korpus kecil + auto → harus int8 (useBinary false)")
	}
}
