package agentdb

import (
	"context"
	"testing"
)

// fake LLM yang balikin triple Aola->prefers->direct answers
func fakeLLMAola(_ context.Context, _ string) (string, error) {
	return `{"nodes":[
		{"label":"Aola","type":"person","source_kind":"user_said","confidence":0.9},
		{"label":"direct answers","type":"preference","confidence":0.8}],
		"edges":[{"from_label":"Aola","to_label":"direct answers","relation_type":"prefers","confidence":0.8}]}`, nil
}

func TestDigestText_Tier2_BuildsActiveGraph(t *testing.T) {
	s := openTestStore(t)
	st, err := s.DigestText(context.Background(), "USER: I prefer direct answers.",
		DigestDeps{LLM: fakeLLMAola, AgentScope: "agent:test", Tier: 2})
	if err != nil {
		t.Fatal(err)
	}
	if st.NodesAdded != 2 || st.EdgesAdded != 1 {
		t.Fatalf("stats = %+v, want 2 nodes 1 edge", st)
	}
	n, ok, _ := s.GetNode("agent:test/person/aola")
	if !ok || n.Status != "active" || n.Label != "Aola" {
		t.Fatalf("aola node: ok=%v status=%q label=%q", ok, n.Status, n.Label)
	}
	out, _, _ := s.Neighbors("agent:test/person/aola")
	if len(out) != 1 || out[0].RelationType != "prefers" {
		t.Fatalf("expected prefers edge, got %+v", out)
	}
}

func TestDigestText_Tier1_Shadow_ThenPromote(t *testing.T) {
	s := openTestStore(t)
	dep := DigestDeps{LLM: fakeLLMAola, AgentScope: "agent:test", Tier: 1}

	// Tier-1 → shadow
	if _, err := s.DigestText(context.Background(), "x", dep); err != nil {
		t.Fatal(err)
	}
	n, _, _ := s.GetNode("agent:test/person/aola")
	if n.Status != "shadow" {
		t.Fatalf("Tier-1 node status = %q, want shadow", n.Status)
	}
	// not yet corroborated → promote does nothing
	if got, _ := s.PromoteShadows(2); got != 0 {
		t.Fatalf("promote before repetition = %d, want 0", got)
	}
	// re-observe (hit_count→2) then promote
	_, _ = s.DigestText(context.Background(), "x again", dep)
	if got, _ := s.PromoteShadows(2); got == 0 {
		t.Fatal("expected promotion after repetition")
	}
	n2, _, _ := s.GetNode("agent:test/person/aola")
	if n2.Status != "active" {
		t.Fatalf("after promote status = %q, want active", n2.Status)
	}
}

func TestDigestPending_NoDelete_Idempotent(t *testing.T) {
	s := openTestStore(t)
	// seed 2 interactions
	for _, c := range []string{"I prefer direct answers", "another message"} {
		if _, err := s.db.Exec(
			`INSERT INTO interactions (channel, direction, actor, content) VALUES ('cli','in','user',?)`, c); err != nil {
			t.Fatal(err)
		}
	}
	var before int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM interactions`).Scan(&before)

	_, n, err := s.DigestPendingInteractions(context.Background(),
		DigestDeps{LLM: fakeLLMAola, AgentScope: "agent:test", Tier: 2}, 100)
	if err != nil || n != 2 {
		t.Fatalf("digest pending: n=%d err=%v, want 2", n, err)
	}

	// REGRESSION (Phase 0): interactions must NOT be deleted
	var after int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM interactions WHERE deleted_at IS NULL`).Scan(&after)
	if after != before {
		t.Fatalf("DATA LOSS: interactions %d -> %d", before, after)
	}
	// digest_log written for both
	var logged int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM cognitive_digest_log`).Scan(&logged)
	if logged != 2 {
		t.Fatalf("digest_log = %d, want 2", logged)
	}
	// idempotent: second run finds nothing pending
	_, n2, _ := s.DigestPendingInteractions(context.Background(),
		DigestDeps{LLM: fakeLLMAola, AgentScope: "agent:test", Tier: 2}, 100)
	if n2 != 0 {
		t.Fatalf("second run pending = %d, want 0 (idempotent)", n2)
	}
}

func TestDigestText_EntityResolutionMerge(t *testing.T) {
	s := openTestStore(t)
	// embedder: "mobil" and "kendaraan" → same vector (similar meaning); others differ
	emb := func(_ context.Context, text string) ([]float32, error) {
		if text == "mobil" || text == "kendaraan" {
			return []float32{1, 0, 0}, nil
		}
		return []float32{0, 1, 0}, nil
	}
	llmKendaraan := func(_ context.Context, _ string) (string, error) {
		return `{"nodes":[{"label":"kendaraan","type":"concept","confidence":0.8}],"edges":[]}`, nil
	}
	llmMobil := func(_ context.Context, _ string) (string, error) {
		return `{"nodes":[{"label":"mobil","type":"concept","confidence":0.8}],"edges":[]}`, nil
	}
	ctx := context.Background()
	_, _ = s.DigestText(ctx, "a", DigestDeps{LLM: llmKendaraan, Embed: emb, AgentScope: "agent:t", Tier: 2})
	_, _ = s.DigestText(ctx, "b", DigestDeps{LLM: llmMobil, Embed: emb, AgentScope: "agent:t", Tier: 2})

	nodes, _ := s.CountCognitiveGraph()
	if nodes != 1 {
		t.Fatalf("expected 1 merged node (mobil↔kendaraan), got %d", nodes)
	}
}
