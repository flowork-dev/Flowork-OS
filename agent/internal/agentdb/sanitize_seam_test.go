package agentdb

import (
	"path/filepath"
	"strings"
	"testing"
)

// Bukti seam ke-panggil di jalur sink: pasang scrubber (kayak secscrub_ext),
// tulis row mengandung token, baca balik → HARUS ke-redact di DB.
func TestSanitizeSeam_WiredIntoSinks(t *testing.T) {
	origT, origM := SanitizeText, SanitizeMeta
	defer func() { SanitizeText, SanitizeMeta = origT, origM }()
	SanitizeText = func(s string) string {
		return strings.ReplaceAll(s, "ghp_SECRET_TOKEN_1234567890", "[REDACTED]")
	}
	SanitizeMeta = func(m map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = SanitizeText(s)
			} else {
				out[k] = v
			}
		}
		return out
	}

	st, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if _, err := st.LogInteraction("http", "in", "owner",
		"token gw ghp_SECRET_TOKEN_1234567890 ya", map[string]any{"raw": "ghp_SECRET_TOKEN_1234567890"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.LogDecision("test", "pakai ghp_SECRET_TOKEN_1234567890", "ok", nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.AddMistake("cat", "judul ghp_SECRET_TOKEN_1234567890", "isi ghp_SECRET_TOKEN_1234567890", "test"); err != nil {
		t.Fatal(err)
	}

	for _, q := range []string{
		"SELECT content FROM interactions",
		"SELECT rationale FROM decisions",
		"SELECT title || content FROM mistakes_local",
	} {
		rows, err := st.DB().Query(q)
		if err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		for rows.Next() {
			var s string
			_ = rows.Scan(&s)
			if strings.Contains(s, "ghp_SECRET_TOKEN") {
				rows.Close()
				t.Fatalf("token bocor di DB (%s): %q", q, s)
			}
		}
		rows.Close()
	}
}
