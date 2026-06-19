package agentdb

import (
	"context"
	"strings"
	"testing"
)

func seedAolaGraph(t *testing.T, s *Store) {
	t.Helper()
	dep := DigestDeps{LLM: fakeLLMAola, AgentScope: "agent:test", Tier: 2}
	if _, err := s.DigestText(context.Background(), "USER: I prefer direct answers.", dep); err != nil {
		t.Fatal(err)
	}
}

func TestRecallFactSheet_LabelSeed(t *testing.T) {
	s := openTestStore(t)
	seedAolaGraph(t, s)

	sheet, err := s.RecallFactSheet(context.Background(), "what about direct answers", RecallDeps{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sheet, "prefers") || !strings.Contains(sheet, "direct answers") {
		t.Fatalf("fact-sheet missing grounding:\n%s", sheet)
	}
}

func TestRecallFactSheet_BudgetCap(t *testing.T) {
	s := openTestStore(t)
	seedAolaGraph(t, s)

	sheet, err := s.RecallFactSheet(context.Background(), "Aola", RecallDeps{MaxChars: 60})
	if err != nil {
		t.Fatal(err)
	}
	if len(sheet) > 120 { // header + at most one line under a tight cap
		t.Fatalf("budget not respected: %d chars\n%s", len(sheet), sheet)
	}
}

func TestRecallFactSheet_NoMatch(t *testing.T) {
	s := openTestStore(t)
	seedAolaGraph(t, s)
	sheet, err := s.RecallFactSheet(context.Background(), "quantum chromodynamics zzz", RecallDeps{})
	if err != nil {
		t.Fatal(err)
	}
	if sheet != "" {
		t.Fatalf("expected empty for no match, got:\n%s", sheet)
	}
}

func TestSearchNodesByEmbedding_TopK(t *testing.T) {
	s := openTestStore(t)
	_, _ = s.UpsertNode(CogNode{ID: "a/c/x", Label: "x", Type: "concept", Embedding: Quantize([]float32{1, 0, 0})})
	_, _ = s.UpsertNode(CogNode{ID: "a/c/y", Label: "y", Type: "concept", Embedding: Quantize([]float32{0, 1, 0})})

	hits := s.SearchNodesByEmbedding("concept", Quantize([]float32{0.95, 0.05, 0}), 1)
	if len(hits) != 1 || hits[0].ID != "a/c/x" {
		t.Fatalf("top-1 = %+v, want a/c/x", hits)
	}
}
