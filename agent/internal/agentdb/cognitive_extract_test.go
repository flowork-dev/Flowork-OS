package agentdb

import (
	"strings"
	"testing"
)

func TestParseExtraction_ValidatesAndDrops(t *testing.T) {
	raw := "```json\n" + `{
	  "nodes": [
	    {"label":"Aola","type":"person","source_kind":"user_said","confidence":0.9},
	    {"label":"verify before trust","type":"trait","confidence":2.0},
	    {"label":"bad","type":"alien_type"},
	    {"label":"","type":"concept"}
	  ],
	  "edges": [
	    {"from_label":"Aola","to_label":"verify before trust","relation_type":"values","confidence":0.8},
	    {"from_label":"Aola","to_label":"x","relation_type":"ngarang_relasi"},
	    {"from_label":"","to_label":"y","relation_type":"uses"}
	  ]
	}` + "\n```"

	res, err := ParseExtraction(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// 2 valid nodes (Aola, verify...), 2 dropped (alien_type, empty label)
	if len(res.Nodes) != 2 {
		t.Fatalf("nodes = %d, want 2 (%+v)", len(res.Nodes), res.Nodes)
	}
	// 1 valid edge (values), 2 dropped (invalid relation, empty field)
	if len(res.Edges) != 1 {
		t.Fatalf("edges = %d, want 1 (%+v)", len(res.Edges), res.Edges)
	}
	if len(res.Dropped) != 4 {
		t.Fatalf("dropped = %d, want 4 (%v)", len(res.Dropped), res.Dropped)
	}
	// confidence clamp: 2.0 → 1.0
	if res.Nodes[1].Confidence != 1.0 {
		t.Fatalf("confidence clamp failed: %v", res.Nodes[1].Confidence)
	}
	// default source_kind on node without one
	if res.Nodes[1].SourceKind != "agent_inferred" {
		t.Fatalf("default source_kind: %q", res.Nodes[1].SourceKind)
	}
}

func TestParseExtraction_BadJSON(t *testing.T) {
	if _, err := ParseExtraction("not json at all"); err == nil {
		t.Fatal("expected error on bad JSON")
	}
}

func TestBuildExtractPrompt_ConstrainsVocab(t *testing.T) {
	p := BuildExtractPrompt("USER: I prefer direct answers.")
	for _, must := range []string{"values", "prefers", "trait", "STRICT JSON", "user_said"} {
		if !strings.Contains(p, must) {
			t.Fatalf("prompt missing %q", must)
		}
	}
	// the conversation is included
	if !strings.Contains(p, "I prefer direct answers") {
		t.Fatal("prompt missing conversation")
	}
}
