// worktree_tool.go — F-G: git worktree isolation (kerja repo besar di salinan
// terpisah, ga ganggu working-tree utama). Pola sama dgn delete-test internal.
// 📄 Dok: FLowork_os/lock/worktree.md
//
// NON-FROZEN sibling (deletable). Tool `git_worktree` op add|list|remove.
// Worktree dibikin di bawah shared workspace agent (terisolasi) → aman, ga
// nyentuh /tmp global. add = detached HEAD (snapshot, ga ganggu branch). Semua
// jalur repo divalidasi = git repo dulu. Timeout wajar, output di-cap.
package builtins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"flowork-gui/internal/tools"
)

func init() { tools.Register(&gitWorktreeTool{}) }

type gitWorktreeTool struct{}

func (gitWorktreeTool) Name() string       { return "git_worktree" }
func (gitWorktreeTool) Capability() string { return "exec:shell" }
func (gitWorktreeTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Git worktree isolation: kerja di SALINAN repo terpisah tanpa ganggu working-tree utama (buat migrasi/refactor gede aman). op: add (bikin worktree detached HEAD di workspace-mu, balikin path) | list | remove. `repo` = path repo git (default: workspace shared).",
		Params: []tools.Param{
			{Name: "op", Type: tools.ParamString, Description: "add | list | remove", Required: true},
			{Name: "repo", Type: tools.ParamString, Description: "path repo git (default shared workspace)"},
			{Name: "path", Type: tools.ParamString, Description: "for op=remove: path worktree yg dihapus"},
			{Name: "ref", Type: tools.ParamString, Description: "for op=add: ref/commit (default HEAD)"},
		},
		Returns: "{op, worktree_path?, worktrees?, output}",
	}
}

func (gitWorktreeTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	op := strings.ToLower(strings.TrimSpace(fmt.Sprint(args["op"])))
	shared := tools.FromSharedDir(ctx)
	repo, _ := args["repo"].(string)
	repo = strings.TrimSpace(repo)
	if repo == "" {
		repo = shared
	}
	if repo == "" {
		return tools.Result{}, fmt.Errorf("repo path kosong (shared workspace ga ke-set)")
	}
	// Validasi: repo harus git repo.
	if _, err := runGit(ctx, repo, "rev-parse", "--is-inside-work-tree"); err != nil {
		return tools.Result{}, fmt.Errorf("%q bukan git repo: %w", repo, err)
	}

	switch op {
	case "add":
		ref := strings.TrimSpace(fmt.Sprint(args["ref"]))
		if ref == "" || ref == "<nil>" {
			ref = "HEAD"
		}
		if shared == "" {
			return tools.Result{}, fmt.Errorf("workspace shared ga ke-set (buat lokasi worktree)")
		}
		wtDir := filepath.Join(shared, ".worktrees", fmt.Sprintf("wt-%d", time.Now().UnixNano()))
		if err := os.MkdirAll(filepath.Dir(wtDir), 0o755); err != nil {
			return tools.Result{}, fmt.Errorf("mkdir: %w", err)
		}
		out, err := runGit(ctx, repo, "worktree", "add", "--detach", wtDir, ref)
		if err != nil {
			return tools.Result{}, fmt.Errorf("worktree add: %w (%s)", err, out)
		}
		return tools.Result{Output: map[string]any{"op": "add", "worktree_path": wtDir, "output": out}}, nil

	case "list":
		out, err := runGit(ctx, repo, "worktree", "list", "--porcelain")
		if err != nil {
			return tools.Result{}, fmt.Errorf("worktree list: %w", err)
		}
		var paths []string
		for _, ln := range strings.Split(out, "\n") {
			if strings.HasPrefix(ln, "worktree ") {
				paths = append(paths, strings.TrimPrefix(ln, "worktree "))
			}
		}
		return tools.Result{Output: map[string]any{"op": "list", "worktrees": paths, "output": out}}, nil

	case "remove":
		wp := strings.TrimSpace(fmt.Sprint(args["path"]))
		if wp == "" || wp == "<nil>" {
			return tools.Result{}, fmt.Errorf("path wajib buat op=remove")
		}
		// Aman: cuma boleh hapus worktree DI DALAM shared workspace (ga bisa hapus sembarang).
		if shared != "" {
			if rel, err := filepath.Rel(shared, wp); err != nil || strings.HasPrefix(rel, "..") {
				return tools.Result{}, fmt.Errorf("cuma boleh remove worktree di dalam workspace-mu")
			}
		}
		out, err := runGit(ctx, repo, "worktree", "remove", "--force", wp)
		if err != nil {
			return tools.Result{}, fmt.Errorf("worktree remove: %w (%s)", err, out)
		}
		return tools.Result{Output: map[string]any{"op": "remove", "output": out}}, nil

	default:
		return tools.Result{}, fmt.Errorf("op tak dikenal %q (add|list|remove)", op)
	}
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	rctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	c := exec.CommandContext(rctx, "git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	s := string(out)
	if len(s) > 8192 {
		s = s[:8192] + "\n…[trunc]"
	}
	return strings.TrimSpace(s), err
}
