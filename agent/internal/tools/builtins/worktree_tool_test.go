package builtins

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"flowork-gui/internal/tools"
)

func TestGitWorktree_AddListRemove(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git ga ada")
	}
	shared := t.TempDir()
	repo := filepath.Join(shared, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	// init repo + 1 commit (worktree butuh HEAD).
	for _, a := range [][]string{
		{"init"}, {"config", "user.email", "t@t"}, {"config", "user.name", "t"},
	} {
		if out, err := runGit(context.Background(), repo, a...); err != nil {
			t.Fatalf("git %v: %v (%s)", a, err, out)
		}
	}
	_ = os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x"), 0o644)
	runGit(context.Background(), repo, "add", ".")
	if out, err := runGit(context.Background(), repo, "commit", "-m", "init"); err != nil {
		t.Fatalf("commit: %v (%s)", err, out)
	}

	ctx := tools.WithSharedDir(context.Background(), shared)
	tool := gitWorktreeTool{}

	// add
	res, err := tool.Run(ctx, map[string]any{"op": "add", "repo": repo})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	wp := res.Output.(map[string]any)["worktree_path"].(string)
	if _, e := os.Stat(wp); e != nil {
		t.Fatalf("worktree dir ga kebikin: %v", e)
	}

	// list — harus ada wp
	res, err = tool.Run(ctx, map[string]any{"op": "list", "repo": repo})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	wts := res.Output.(map[string]any)["worktrees"].([]string)
	if len(wts) < 2 {
		t.Fatalf("mau >=2 worktree (utama+baru): %v", wts)
	}

	// remove
	if _, err := tool.Run(ctx, map[string]any{"op": "remove", "repo": repo, "path": wp}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, e := os.Stat(wp); !os.IsNotExist(e) {
		t.Fatal("worktree harus kehapus")
	}
}

func TestGitWorktree_RejectNonRepo(t *testing.T) {
	shared := t.TempDir()
	ctx := tools.WithSharedDir(context.Background(), shared)
	if _, err := (gitWorktreeTool{}).Run(ctx, map[string]any{"op": "add", "repo": shared}); err == nil {
		t.Error("non-git repo harus ditolak")
	}
}
