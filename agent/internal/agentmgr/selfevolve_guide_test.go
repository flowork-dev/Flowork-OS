package agentmgr

import "testing"

// TestEvolveGuideTheme — SGS Guide: ekstraksi tema (deteksi collapse). reflection-* harus 1 tema.
func TestEvolveGuideTheme(t *testing.T) {
	cases := []struct{ tf, kind, want string }{
		{"NEW:reflection-scheduled", "add-skill", "reflection"},
		{"NEW:reflection-trigger-cadence", "add-skill", "reflection"},
		{"NEW:reflection-orchestrator.fwagent", "add-agent", "reflection"},
		{"NEW:scam-detector", "add-app", "scam"},
		{"new:bounty_hunter", "add-agent", "bounty"},
		{"agent/internal/foo.go", "fix", "foo"},
		{"agent/internal/bar.go", "refactor", "bar"},
		{"", "add-skill", "add-skill"}, // fallback ke kind
	}
	for _, c := range cases {
		if got := evolveGuideTheme(c.tf, c.kind); got != c.want {
			t.Errorf("evolveGuideTheme(%q,%q) = %q, want %q", c.tf, c.kind, got, c.want)
		}
	}
	// 3 reflection-* harus tema SAMA (collapse kedeteksi) — beda dari scam.
	r1 := evolveGuideTheme("NEW:reflection-a", "")
	r2 := evolveGuideTheme("NEW:reflection-b", "")
	s1 := evolveGuideTheme("NEW:scam-x", "")
	if r1 != r2 {
		t.Errorf("reflection-* harus tema sama: %q vs %q", r1, r2)
	}
	if r1 == s1 {
		t.Errorf("reflection vs scam harus beda tema, dua-duanya %q", r1)
	}
}
