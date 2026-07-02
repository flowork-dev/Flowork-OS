// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// ⚠️ FROZEN — jangan edit file ini. 📄 Dok: lock/tools.md
// Nambah/ubah tool TANPA buka frozen: pakai SEAM tools.Register() di file sibling non-frozen.
// shell_guard.go — `shell`: the hardened exec tool (P1).
//
// Same execution surface as the locked `bash` tool (shell.go) — but danger is judged
// by command SEMANTICS (cmdsem.go), not substring, so the doubled-space / ${IFS} /
// path-prefix bypasses are caught and `echo "rm -rf /"` is NOT false-blocked. It also
// reports `read_only` so a permission tier can auto-allow observe-only commands.
//
// New file (shell.go is owner-LOCKED). Reuses the locked tool's package helpers
// (scrubEnv, capWriter, applyMemLimit) — no duplication of the sensitive bits, no
// modification of the locked file. Same caps + bounds (exec:shell, timeout, 64KB cap,
// rlimit) so it is a drop-in, safer replacement for mr-flow's exec.
package builtins

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"flowork-gui/internal/tools"
)

func init() { tools.Register(&shellGuardTool{}) }

type shellGuardTool struct{}

func (shellGuardTool) Name() string       { return "shell" }
func (shellGuardTool) Capability() string { return "exec:shell" }
func (shellGuardTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Run a shell command in the agent's shared workspace (hardened). Danger is judged by command STRUCTURE, not substring: recursive deletes of system/home roots, power ops (use system_power instead), dd-to-device, mkfs, chmod 777, fork bombs, privilege escalation, and curl|sh are blocked; legit commands and quoted strings are not. Linux/macOS /bin/sh, Windows cmd. Timeout default 20s (cap 60), output cap 64KB.",
		Params: []tools.Param{
			{Name: "command", Type: tools.ParamString, Description: "shell command to execute", Required: true},
			{Name: "working_dir", Type: tools.ParamString, Description: "optional subdir inside shared workspace"},
			{Name: "timeout_seconds", Type: tools.ParamInt, Description: "optional timeout 1..60 (default 20)"},
		},
		Returns: "{stdout, stderr, exit_code, truncated, duration_ms, read_only}",
	}
}

func (shellGuardTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	cmd, _ := args["command"].(string)
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return tools.Result{}, fmt.Errorf("command required")
	}
	// Structural danger classification (vs substring).
	if blocked, reason, _ := classifyCommand(cmd); blocked {
		return tools.Result{}, fmt.Errorf("shell: blocked — %s", reason)
	}
	_, _, readOnly := classifyCommand(cmd)

	shared := tools.FromSharedDir(ctx)
	if shared == "" {
		return tools.Result{}, fmt.Errorf("shell: shared dir not in context")
	}
	workDir := shared
	if wd, _ := args["working_dir"].(string); strings.TrimSpace(wd) != "" {
		wd = strings.TrimSpace(wd)
		if filepath.IsAbs(wd) {
			return tools.Result{}, fmt.Errorf("shell: working_dir must be relative")
		}
		joined := filepath.Join(shared, wd)
		if rel, rerr := filepath.Rel(shared, joined); rerr != nil || strings.HasPrefix(rel, "..") {
			return tools.Result{}, fmt.Errorf("shell: working_dir escapes shared")
		}
		workDir = joined
	}

	timeoutSec := 20
	if v, ok := args["timeout_seconds"].(float64); ok && v > 0 {
		timeoutSec = int(v)
	}
	if vs, ok := args["timeout_seconds"].(int); ok && vs > 0 {
		timeoutSec = vs
	}
	if timeoutSec < 1 {
		timeoutSec = 1
	}
	if timeoutSec > 60 {
		timeoutSec = 60
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	var c *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		c = exec.CommandContext(runCtx, "cmd", "/C", cmd)
	default:
		c = exec.CommandContext(runCtx, "/bin/sh", "-c", cmd)
	}
	c.Dir = workDir
	c.Env = scrubEnv()  // reuse locked tool's secret-scrubbing
	applyMemLimit(c, cmd) // reuse locked tool's rlimit

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
				"stdout": outBuf.String(), "stderr": errBuf.String() + "\n[timeout exceeded]",
				"exit_code": 124, "truncated": outBuf.Len() >= outCap || errBuf.Len() >= outCap,
				"duration_ms": elapsed, "read_only": readOnly,
			}}, nil
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return tools.Result{}, fmt.Errorf("shell exec: %w", runErr)
		}
	}
	return tools.Result{Output: map[string]any{
		"stdout": outBuf.String(), "stderr": errBuf.String(), "exit_code": exitCode,
		"truncated": outBuf.Len() >= outCap || errBuf.Len() >= outCap,
		"duration_ms": elapsed, "read_only": readOnly,
	}}, nil
}
