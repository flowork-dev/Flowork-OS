package builtins

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/tools"
)

func TestGraphRecallTool(t *testing.T) {
	s, err := agentdb.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// seed a tiny twin graph
	if _, err := s.UpsertNode(agentdb.CogNode{ID: "a/person/aola", Label: "Aola", Type: "person", Status: "active", Confidence: 0.9}); err != nil {
		t.Fatal(err)
	}
	_, _ = s.UpsertNode(agentdb.CogNode{ID: "a/pref/direct", Label: "direct answers", Type: "preference", Status: "active", Confidence: 0.9})
	_ = s.UpsertEdge(agentdb.CogEdge{FromID: "a/person/aola", ToID: "a/pref/direct", RelationType: "prefers", Status: "active", Confidence: 0.9})

	ctx := tools.WithStore(context.Background(), s)
	res, err := (graphRecallTool{}).Run(ctx, map[string]any{"query": "direct answers"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	out := res.Output.(map[string]any)
	sheet, _ := out["fact_sheet"].(string)
	if !strings.Contains(sheet, "prefers") {
		t.Fatalf("fact_sheet missing grounding:\n%s", sheet)
	}

	// missing store → error
	if _, err := (graphRecallTool{}).Run(context.Background(), map[string]any{"query": "x"}); err == nil {
		t.Fatal("expected error without store in context")
	}
	// empty query → error
	if _, err := (graphRecallTool{}).Run(ctx, map[string]any{"query": " "}); err == nil {
		t.Fatal("expected error on empty query")
	}
}
