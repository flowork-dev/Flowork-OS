// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"flowork-gui/internal/tools"
)

// SELF-REGISTER (plug-in NON-frozen, 📄 lock/tool-manager.md). Edit/hapus bebas tanpa unfreeze.
func init() { tools.Register(&gitTool{}) }

type gitTool struct{}

func (gitTool) Name() string       { return "git" }
func (gitTool) Capability() string { return "exec:git" }
func (gitTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Run git read-only op (status, diff, log, show). Working dir = <shared>/<category>. Default category='tools'. Output cap 64KB. Timeout 15s.",
		Params: []tools.Param{
			{Name: "op", Type: tools.ParamString, Description: "status | diff | log | show", Required: true},
			{Name: "category", Type: tools.ParamString, Description: "tools|job|document|media|cache|log (default: tools)"},
			{Name: "ref", Type: tools.ParamString, Description: "for op=show/log: commit ref (default HEAD)"},
			{Name: "path", Type: tools.ParamString, Description: "for op=diff/show: optional file path filter"},
		},
		Returns: "{stdout, stderr, exit_code, duration_ms}",
	}
}

func (gitTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	op, _ := args["op"].(string)
	op = strings.ToLower(strings.TrimSpace(op))

	allowed := map[string]bool{"status": true, "diff": true, "log": true, "show": true}
	if !allowed[op] {
		return tools.Result{}, fmt.Errorf("op must be one of: status, diff, log, show")
	}

	category, _ := args["category"].(string)
	category = strings.TrimSpace(category)
	if category == "" {
		category = "tools"
	}
	if _, ok := fileCategoryWhitelist[category]; !ok {
		return tools.Result{}, fmt.Errorf("category %q not in whitelist", category)
	}

	ref, _ := args["ref"].(string)
	path, _ := args["path"].(string)

	shared := tools.FromSharedDir(ctx)
	if shared == "" {
		return tools.Result{}, fmt.Errorf("shared workspace not in context")
	}
	workDir := filepath.Join(shared, category)

	gitArgs := []string{}
	switch op {
	case "status":
		gitArgs = []string{"status", "--short"}
	case "diff":
		gitArgs = []string{"diff", "--stat"}
		if path != "" {
			gitArgs = append(gitArgs, "--", path)
		}
	case "log":
		gitArgs = []string{"log", "--oneline", "-n", "20"}
		if ref != "" {
			gitArgs = append(gitArgs, ref)
		}
	case "show":
		gitArgs = []string{"show", "--stat"}
		if ref == "" {
			ref = "HEAD"
		}
		gitArgs = append(gitArgs, ref)
		if path != "" {
			gitArgs = append(gitArgs, "--", path)
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	c := exec.CommandContext(runCtx, "git", gitArgs...)
	c.Dir = workDir

	var outBuf, errBuf bytes.Buffer
	const outCap = 64 * 1024
	c.Stdout = &capWriter{buf: &outBuf, cap: outCap}
	c.Stderr = &capWriter{buf: &errBuf, cap: outCap}

	t0 := time.Now()
	runErr := c.Run()
	elapsed := time.Since(t0).Milliseconds()

	exitCode := 0
	if runErr != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return tools.Result{Output: map[string]any{
				"stdout":      outBuf.String(),
				"stderr":      errBuf.String() + "\n[timeout]",
				"exit_code":   124,
				"duration_ms": elapsed,
			}}, nil
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return tools.Result{}, fmt.Errorf("git exec: %w", runErr)
		}
	}

	return tools.Result{Output: map[string]any{
		"stdout":      outBuf.String(),
		"stderr":      errBuf.String(),
		"exit_code":   exitCode,
		"duration_ms": elapsed,
	}}, nil
}
