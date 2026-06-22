package builtins

// builtins_brain_test.go — QC (owner 2026-06-22): buktiin SEMUA tool brain-path TETAP
// ter-register pasca-cabut registrasi dari builtins.go ke builtins_brain.go (init self-register)
// + yg udah self-register (instinct_recall, brain_dream). init() auto-run pas package load →
// kalau ada yg ke-drop, test ini GAGAL. Anti-regresi cabut-cabang.

import (
	"testing"

	"flowork-gui/internal/tools"
)

func TestBrainToolsRegisteredAfterExtract(t *testing.T) {
	want := []string{
		// dicabut ke builtins_brain.go (init):
		"brain_search_shared", "brain_add", "brain_search", "brain_get",
		"graph_recall", "mistake_recall", "brain_immune_scan", "brain_verify",
		"brain_promote_shared", "codemap_search", "codemap_stats",
		// self-register di file frozen-nya sendiri:
		"instinct_recall", "brain_dream",
	}
	for _, n := range want {
		if _, ok := tools.Lookup(n); !ok {
			t.Errorf("brain tool %q TIDAK ter-register — cabut builtins_brain rusak/ke-drop", n)
		}
	}
}
