package agentdb

import (
	"path/filepath"
	"testing"
)

// TestSeedSelfKnowledge — kartu fitur ke-seed ke brain lokal + idempotent (run ke-2 = 0).
func TestSeedSelfKnowledge(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	n1, err := s.SeedSelfKnowledge()
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n1 < 50 {
		t.Fatalf("expected >=50 self-knowledge cards seeded, got %d", n1)
	}

	// Idempotent: room udah keisi → run ke-2 no-op.
	n2, err := s.SeedSelfKnowledge()
	if err != nil {
		t.Fatalf("reseed: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("expected idempotent 0 on second seed, got %d", n2)
	}

	// Recall smoke: kartu bisa ditemukan lewat FTS brain lokal.
	hits, err := s.SearchLocalBrain("flowork browser", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	found := false
	for _, h := range hits {
		if h.Room == SelfKnowledgeRoom {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("self-knowledge card not recalled via brain_search for 'flowork browser'")
	}
}
