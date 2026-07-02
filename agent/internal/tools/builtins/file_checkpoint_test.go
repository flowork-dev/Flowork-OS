package builtins

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"flowork-gui/internal/tools"
)

func ckptTestCtx(t *testing.T) (context.Context, string) {
	t.Helper()
	dir := t.TempDir()
	return tools.WithSharedDir(context.Background(), dir), dir
}

func TestCheckpoint_SnapshotUndoRedo(t *testing.T) {
	ctx, dir := ckptTestCtx(t)
	rel := "tools/x.txt"
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}

	// v1 ada di disk → snapshot (kayak interceptor sebelum write v2) → tulis v2.
	if err := os.WriteFile(abs, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ckptSnapshot(ctx, abs, rel); err != nil {
		t.Fatalf("snapshot v1: %v", err)
	}
	if err := os.WriteFile(abs, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	// undo → balik ke v1.
	res, err := (undoFileTool{}).Run(ctx, map[string]any{"file_path": rel})
	if err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got, _ := os.ReadFile(abs); string(got) != "v1" {
		t.Fatalf("undo: mau v1, dapet %q", got)
	}
	out := res.Output.(map[string]any)
	if out["deleted"].(bool) {
		t.Fatal("undo: deleted harusnya false")
	}

	// redo: pre-undo snapshot nyimpen v2 → undo lagi ke checkpoint terbaru = v2.
	if _, err := (undoFileTool{}).Run(ctx, map[string]any{"file_path": rel}); err != nil {
		t.Fatalf("redo: %v", err)
	}
	if got, _ := os.ReadFile(abs); string(got) != "v2" {
		t.Fatalf("redo: mau v2, dapet %q", got)
	}
}

func TestCheckpoint_UndoCreateDeletesFile(t *testing.T) {
	ctx, dir := ckptTestCtx(t)
	rel := "tools/baru.txt"
	abs := filepath.Join(dir, rel)

	// File BELUM ada → snapshot marker absen (kayak interceptor sebelum create).
	if err := ckptSnapshot(ctx, abs, rel); err != nil {
		t.Fatalf("snapshot absen: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("baru"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := (undoFileTool{}).Run(ctx, map[string]any{"file_path": rel})
	if err != nil {
		t.Fatalf("undo create: %v", err)
	}
	if !res.Output.(map[string]any)["deleted"].(bool) {
		t.Fatal("undo create: deleted harusnya true")
	}
	if _, serr := os.Stat(abs); !os.IsNotExist(serr) {
		t.Fatal("undo create: file harusnya kehapus")
	}
}

func TestCheckpoint_ListAndPick(t *testing.T) {
	ctx, dir := ckptTestCtx(t)
	rel := "job/pick.txt"
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"a", "b", "c"} {
		if err := os.WriteFile(abs, []byte(v), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ckptSnapshot(ctx, abs, rel); err != nil {
			t.Fatal(err)
		}
	}
	res, err := (fileCheckpointsTool{}).Run(ctx, map[string]any{"file_path": rel})
	if err != nil {
		t.Fatal(err)
	}
	out := res.Output.(map[string]any)
	items := out["checkpoints"].([]map[string]any)
	if len(items) != 3 {
		t.Fatalf("mau 3 checkpoint, dapet %d", len(items))
	}
	// pilih checkpoint TERTUA (isi "a") via id eksplisit.
	oldest := items[len(items)-1]["id"].(string)
	if _, err := (undoFileTool{}).Run(ctx, map[string]any{"file_path": rel, "checkpoint": oldest}); err != nil {
		t.Fatalf("undo ke id: %v", err)
	}
	if got, _ := os.ReadFile(abs); string(got) != "a" {
		t.Fatalf("mau isi 'a', dapet %q", got)
	}
}

func TestCheckpoint_PruneCap(t *testing.T) {
	ctx, dir := ckptTestCtx(t)
	rel := "cache/n.txt"
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < ckptKeep+15; i++ {
		if err := ckptSnapshot(ctx, abs, rel); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join(dir, ckptDirName)
	ents, _ := os.ReadDir(root)
	if len(ents) > ckptKeep {
		t.Fatalf("prune gagal: %d file > cap %d", len(ents), ckptKeep)
	}
}

func TestCheckpoint_InterceptorNeverBlocks(t *testing.T) {
	// Tanpa shared dir di ctx → snapshot pasti gagal → Before TETEP nil (non-blocking).
	err := (fileCheckpointInterceptor{}).Before(context.Background(),
		&fileWriteTool{}, map[string]any{"file_path": "tools/x.txt", "content": "y"})
	if err != nil {
		t.Fatalf("interceptor harus best-effort (nil), dapet: %v", err)
	}
}
