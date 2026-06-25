package fwswitch

import (
	"os"
	"testing"
)

func reset() {
	mu.Lock()
	managed = map[string]string{}
	mu.Unlock()
}

// WriteValues nulis file → Apply → os.Getenv reflect; "" = hapus (revert ENV).
func TestWriteValues_RoundTrip(t *testing.T) {
	reset()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FLOWORK_SEARCH_MINSCORE", "0.20") // ENV asli
	if err := WriteValues(map[string]string{"FLOWORK_SEARCH_MINSCORE": "0.80"}); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("FLOWORK_SEARCH_MINSCORE"); got != "0.80" {
		t.Errorf("set GUI harus override jadi 0.80, dapet %q", got)
	}
	// hapus (value "") → revert ke ENV asli 0.20
	if err := WriteValues(map[string]string{"FLOWORK_SEARCH_MINSCORE": ""}); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("FLOWORK_SEARCH_MINSCORE"); got != "0.20" {
		t.Errorf("hapus GUI → revert ENV 0.20, dapet %q", got)
	}
}

// Resolve lapor sumber: gui > env > default.
func TestResolve_Source(t *testing.T) {
	reset()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FLOWORK_INSTINCT_SCOPED", "1") // env
	if err := WriteValues(map[string]string{"FLOWORK_SEARCH_MINSCORE": "0.7"}); err != nil {
		t.Fatal(err)
	}
	byKey := map[string]Resolved{}
	for _, r := range Resolve() {
		byKey[r.Key] = r
	}
	if r := byKey["FLOWORK_SEARCH_MINSCORE"]; r.Source != "gui" || r.Value != "0.7" {
		t.Errorf("minscore harus gui/0.7, dapet %s/%s", r.Source, r.Value)
	}
	if r := byKey["FLOWORK_INSTINCT_SCOPED"]; r.Source != "env" || r.Value != "1" {
		t.Errorf("scoped harus env/1, dapet %s/%s", r.Source, r.Value)
	}
	if r := byKey["FLOWORK_TOOLCALL_RECOVER"]; r.Source != "default" || r.Value != "true" {
		t.Errorf("toolcall-recover harus default/true, dapet %s/%s", r.Source, r.Value)
	}
}
