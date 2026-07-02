// inject_budget_ext_test.go — bukti budget agregat system-inject (F-A3).
package router

import (
	"context"
	"strings"
	"testing"
)

func msgsForBudgetTest() []OpenAIMessage {
	return []OpenAIMessage{
		{Role: "system", Content: "Kamu mr-flow, persona stabil (caller) — JANGAN disentuh."},
		{Role: "system", Content: "## Project doctrine (sacred rules)\nThese rules are immutable..."},
		{Role: "system", Content: "## Relevant knowledge\n" + strings.Repeat("k", 4000)},
		{Role: "system", Content: "## Insting — refleks WHEN→THEN (kapan pakai kapabilitas yg lo punya)\n" + strings.Repeat("i", 500)},
		{Role: "system", Content: "## Antibodi — kesalahan TERBUKTI, JANGAN diulang\n" + strings.Repeat("a", 400)},
		{Role: "user", Content: "halo"},
	}
}

func TestInjectBudget_OffByDefault(t *testing.T) {
	t.Setenv("FLOWORK_INJECT_BUDGET", "")
	req := OpenAIRequest{Messages: msgsForBudgetTest()}
	out := applyInjectShaper(context.Background(), req, nil)
	if len(out.Messages) != 6 {
		t.Fatalf("switch off: pesan harus utuh, dapet %d", len(out.Messages))
	}
}

func TestInjectBudget_DropsLowestClassFirst_KeepsSacred(t *testing.T) {
	// Budget kecil: knowledge (kelas 1, ~4000 char) harus kebuang duluan;
	// insting+antibodi (~900) muat → tinggal. Doctrine + persona WAJIB selamat.
	t.Setenv("FLOWORK_INJECT_BUDGET", "1500")
	req := OpenAIRequest{Messages: msgsForBudgetTest()}
	out := applyInjectShaper(context.Background(), req, nil)
	var hasDoctrine, hasPersona, hasKnowledge, hasInstinct, hasAntibody bool
	for _, m := range out.Messages {
		switch {
		case strings.HasPrefix(m.Content, "## Project doctrine"):
			hasDoctrine = true
		case strings.HasPrefix(m.Content, "Kamu mr-flow"):
			hasPersona = true
		case strings.HasPrefix(m.Content, "## Relevant knowledge"):
			hasKnowledge = true
		case strings.HasPrefix(m.Content, "## Insting —"):
			hasInstinct = true
		case strings.HasPrefix(m.Content, "## Antibodi —"):
			hasAntibody = true
		}
	}
	if !hasDoctrine || !hasPersona {
		t.Fatal("doctrine SACRED / persona caller ikut kebuang — DILARANG")
	}
	if hasKnowledge {
		t.Fatal("knowledge (kelas 1) harusnya kebuang duluan pas lewat budget")
	}
	if !hasInstinct || !hasAntibody {
		t.Fatal("insting/antibodi kebuang padahal sisa total udah di bawah budget")
	}
	if out.Messages[len(out.Messages)-1].Role != "user" {
		t.Fatal("pesan user harus tetap paling akhir")
	}
}

func TestInjectBudget_UnderBudgetUntouched(t *testing.T) {
	t.Setenv("FLOWORK_INJECT_BUDGET", "99999")
	req := OpenAIRequest{Messages: msgsForBudgetTest()}
	out := applyInjectShaper(context.Background(), req, nil)
	if len(out.Messages) != 6 {
		t.Fatalf("di bawah budget: pesan harus utuh, dapet %d", len(out.Messages))
	}
}

func TestClassifyInjected(t *testing.T) {
	cases := []struct {
		content string
		want    int
	}{
		{"## Relevant knowledge\nxx", 1},
		{"You are operating with a shared knowledge brain. ...", 1},
		{"## Applicable skills\nxx", 1},
		{"## Insting — refleks WHEN→THEN (kapan pakai)", 2},
		{"## Antibodi — kesalahan TERBUKTI, JANGAN diulang", 3},
		{"## Project doctrine (sacred rules)", 0},
		{"persona caller biasa", 0},
	}
	for _, c := range cases {
		if got := classifyInjected(c.content); got != c.want {
			t.Fatalf("classifyInjected(%.30q) = %d, want %d", c.content, got, c.want)
		}
	}
}

func TestEnrichMinScoreParse(t *testing.T) {
	cases := []struct {
		v    string
		want float64
	}{
		{"", 0}, {"0", 0}, {"0.35", 0.35}, {"1.5", 0}, {"abc", 0}, {"-0.2", 0},
	}
	for _, c := range cases {
		t.Setenv("FLOWORK_ENRICH_MINSCORE", c.v)
		if got := enrichMinScore(); got != c.want {
			t.Fatalf("enrichMinScore(%q) = %v, want %v", c.v, got, c.want)
		}
	}
}
