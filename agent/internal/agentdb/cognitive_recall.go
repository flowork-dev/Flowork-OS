// cognitive_recall.go — Retrieval / Grounding dari cognitive graph (roadmap §4.8, D6, D1b).
//
// Anti muntah prompt: graph GAK PERNAH masuk konteks utuh. Alur: query → cari SEED
// (embedding top-k + label fallback = 3-lapis recall D1b) → ekspansi 1 hop → rank
// confidence×strength → render "fact-sheet" RINGKAS budget-capped. Cuma neighborhood
// relevan yang nyentuh prompt, bukan 863K node.
//
// Layering: Embed di-inject (nil = label-only). Render = pure given graph.

package agentdb

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// RecallDeps — opsi 1 recall.
type RecallDeps struct {
	Embed    EmbedFunc // opsional (nil = label-only seed)
	MaxChars int       // budget fact-sheet (default 1500)
	SeedK    int       // top-k seed (default 5)
}

// ScoredNode — node + skor (cosine / relevance).
type ScoredNode struct {
	ID    string
	Label string
	Type  string
	Score float64
}

// SearchNodesByEmbedding — top-k node active paling mirip queryEmb (cosine). typ
// kosong = semua type.
func (s *Store) SearchNodesByEmbedding(typ string, queryEmb []byte, k int) []ScoredNode {
	if len(queryEmb) == 0 {
		return nil
	}
	if k <= 0 {
		k = 5
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()

	q := `SELECT id, label, type, embedding FROM cognitive_nodes WHERE status='active' AND embedding IS NOT NULL`
	args := []any{}
	if typ != "" {
		q += ` AND type=?`
		args = append(args, typ)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var all []ScoredNode
	for rows.Next() {
		var n ScoredNode
		var emb []byte
		if err := rows.Scan(&n.ID, &n.Label, &n.Type, &emb); err != nil {
			continue
		}
		n.Score = CosineQ(queryEmb, emb)
		all = append(all, n)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Score > all[j].Score })
	if len(all) > k {
		all = all[:k]
	}
	return all
}

// SearchNodesByLabel — fallback keyword: node active yg label-nya cocok token query.
func (s *Store) SearchNodesByLabel(query string, k int) []ScoredNode {
	if k <= 0 {
		k = 5
	}
	toks := tokenize(query)
	if len(toks) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCognitiveGraphSchema()

	seen := map[string]bool{}
	var out []ScoredNode
	for _, tk := range toks {
		rows, err := s.db.Query(
			`SELECT id, label, type FROM cognitive_nodes
			 WHERE status='active' AND LOWER(label) LIKE '%'||?||'%' LIMIT ?`, tk, k)
		if err != nil {
			continue
		}
		for rows.Next() {
			var n ScoredNode
			if err := rows.Scan(&n.ID, &n.Label, &n.Type); err == nil && !seen[n.ID] {
				seen[n.ID] = true
				n.Score = 0.5
				out = append(out, n)
			}
		}
		rows.Close()
	}
	if len(out) > k {
		out = out[:k]
	}
	return out
}

// RecallFactSheet rangkai grounding ringkas buat query. Seed (embedding + label) →
// ekspansi 1 hop → rank → render budget-capped. Return "" kalau ga ada yg relevan.
func (s *Store) RecallFactSheet(ctx context.Context, query string, dep RecallDeps) (string, error) {
	if dep.MaxChars <= 0 {
		dep.MaxChars = 1500
	}
	if dep.SeedK <= 0 {
		dep.SeedK = 5
	}

	// ── seed: embedding (kalau ada) + label fallback (3-lapis D1b) ────────────
	seedSet := map[string]ScoredNode{}
	if dep.Embed != nil {
		if vec, err := dep.Embed(ctx, query); err == nil {
			for _, n := range s.SearchNodesByEmbedding("", Quantize(vec), dep.SeedK) {
				seedSet[n.ID] = n
			}
		}
	}
	for _, n := range s.SearchNodesByLabel(query, dep.SeedK) {
		if _, ok := seedSet[n.ID]; !ok {
			seedSet[n.ID] = n
		}
	}
	if len(seedSet) == 0 {
		return "", nil
	}

	// ── ekspansi 1 hop + kumpulin edge unik ──────────────────────────────────
	type factEdge struct {
		from, rel, to string
		score         float64
	}
	facts := map[string]factEdge{}
	labelCache := map[string]string{}
	labelOf := func(id string) string {
		if l, ok := labelCache[id]; ok {
			return l
		}
		if n, ok, _ := s.GetNode(id); ok {
			labelCache[id] = n.Label
			return n.Label
		}
		labelCache[id] = id
		return id
	}
	for id := range seedSet {
		out, in, err := s.Neighbors(id)
		if err != nil {
			continue
		}
		for _, e := range append(out, in...) {
			key := e.FromID + "|" + e.RelationType + "|" + e.ToID
			if _, ok := facts[key]; !ok {
				facts[key] = factEdge{labelOf(e.FromID), e.RelationType, labelOf(e.ToID), e.Confidence * e.Strength}
			}
		}
	}

	// ── rank + render budget-capped ──────────────────────────────────────────
	ranked := make([]factEdge, 0, len(facts))
	for _, f := range facts {
		ranked = append(ranked, f)
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })

	var b strings.Builder
	b.WriteString("# Relevant memory (grounding)\n")
	for _, f := range ranked {
		line := fmt.Sprintf("- %s —%s→ %s\n", f.from, f.rel, f.to)
		if b.Len()+len(line) > dep.MaxChars {
			break
		}
		b.WriteString(line)
	}
	if b.Len() <= len("# Relevant memory (grounding)\n") {
		// ada seed tapi ga ada edge → minimal sebut node-nya
		for _, n := range seedSet {
			line := fmt.Sprintf("- %s (%s)\n", n.Label, n.Type)
			if b.Len()+len(line) > dep.MaxChars {
				break
			}
			b.WriteString(line)
		}
	}
	return b.String(), nil
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var toks []string
	for _, w := range strings.FieldsFunc(s, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	}) {
		if len(w) >= 3 {
			toks = append(toks, w)
		}
	}
	return toks
}
