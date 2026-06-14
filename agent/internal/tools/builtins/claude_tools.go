// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-14
// Reason: 10 surface-vocabulary tools VERIFIED via mr-flow (/api/chat, bukti =
//   tool_invocations + DB): TaskCreate/Output/Update/Stop, ScheduleWakeup,
//   Workflow, StructuredOutput JALAN; PowerShell/Monitor/SendUserFile dispatch
//   kebukti + cap-gated benar (butuh exec:shell / net:fetch:telegram yang mr-flow
//   sengaja ga punya). Nambah tool surface baru → tambah FILE baru + init()
//   sendiri (pola file ini), JANGAN modify ini.
//
// claude_tools.go — Surface-vocabulary tools that the local model
// (distilled) and the external driver model both expect, but Flowork did not yet
// have. Added here in their own file with their own init() so the locked
// builtins.go / registry.go / types.go are never touched.
//
// HONESTY CONTRACT (owner rule: never overclaim "done"):
//   The kernel's invoke model is SYNCHRONOUS — a colony member runs to completion
//   within one Call; there is no shared cross-agent scheduler polling mid-run.
//   So the Task* family and Workflow are a DURABLE LEDGER over the agent's own
//   store (the same agent_runs table agent_run.go uses), NOT fake parallel spawns.
//   Each tool's Schema + Returns says exactly what it does. No tool claims an
//   ability it does not have.
//
// Backing reused (no new infra, no unlock):
//   - agent_runs table (agent_run.go)       → TaskCreate/Update/Stop/Output
//   - wakeups table (created here)           → ScheduleWakeup
//   - shell exec helpers (shell.go)          → PowerShell, Monitor
//   - telegram secrets (telegram.go)         → SendUserFile (sendDocument)

package builtins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"flowork-gui/internal/tools"
)

func init() {
	tools.Register(&powerShellTool{})
	tools.Register(&taskCreateTool{})
	tools.Register(&taskUpdateTool{})
	tools.Register(&taskStopTool{})
	tools.Register(&taskOutputTool{})
	tools.Register(&scheduleWakeupTool{})
	tools.Register(&monitorTool{})
	tools.Register(&sendUserFileTool{})
	tools.Register(&structuredOutputTool{})
	tools.Register(&workflowTool{})
}

// firstAllowedChat — pick the first chat id from TELEGRAM_ALLOWED_CHATS secret.
func firstAllowedChat(secrets map[string]string) (int64, error) {
	raw := strings.TrimSpace(secrets["TELEGRAM_ALLOWED_CHATS"])
	if raw == "" {
		return 0, fmt.Errorf("[PETUNJUK, bukan salahmu] TELEGRAM_ALLOWED_CHATS belum di-set — ga tau harus kirim file ke chat mana. Minta owner isi daftar chat yang diizinkan di Settings (doktrin ERR_SECRET_MISSING)")
	}
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, nil
		}
	}
	return 0, fmt.Errorf("no valid chat id in TELEGRAM_ALLOWED_CHATS")
}

// argStr — read a string arg, accepting either of two keys (e.g. task_id|taskId).
func argStr(args map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := args[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// argInt — read an int arg (JSON number arrives as float64).
func argInt(args map[string]any, key string) (int64, bool) {
	switch v := args[key].(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}

// =============================================================================
// 1. PowerShell — Windows pwsh exec (educational error elsewhere)
// =============================================================================

type powerShellTool struct{}

func (powerShellTool) Name() string       { return "PowerShell" }
func (powerShellTool) Capability() string { return "exec:shell" }
func (powerShellTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Run a PowerShell command via pwsh. Default timeout 20s (cap 60). Output cap 64KB. Requires pwsh in PATH; if absent you get an educational error telling you to use Bash instead.",
		Params: []tools.Param{
			{Name: "command", Type: tools.ParamString, Description: "PowerShell command to run", Required: true},
			{Name: "description", Type: tools.ParamString, Description: "what this does (5-10 words)"},
			{Name: "timeout", Type: tools.ParamInt, Description: "timeout seconds 1..60 (default 20)"},
			{Name: "run_in_background", Type: tools.ParamBool, Description: "background run (not supported in the synchronous kernel — returns a note)"},
		},
		Returns: "{stdout, stderr, exit_code, duration_ms}",
	}
}

func (powerShellTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	cmd := argStr(args, "command")
	if cmd == "" {
		return tools.Result{}, fmt.Errorf("command required")
	}
	if bg, _ := args["run_in_background"].(bool); bg {
		return tools.Result{Note: "background run not supported (synchronous kernel) — run foreground or use TaskCreate to record a long job", Output: map[string]any{"ran": false}}, nil
	}
	// Same denylist guard as the shell tool.
	lower := strings.ToLower(cmd)
	for _, p := range shellDenyPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return tools.Result{}, fmt.Errorf("[PETUNJUK, bukan salahmu] perintah memuat pola berbahaya %q (destruktif/ga bisa di-undo) jadi diblok — ini aturan tetap. Cari cara aman (hapus file spesifik, bukan rekursif). Ragu? tanya owner (doktrin ERR_SHELL_DANGEROUS)", p)
		}
	}
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		if p2, e2 := exec.LookPath("powershell"); e2 == nil {
			pwsh = p2
		} else {
			return tools.Result{}, fmt.Errorf("[PETUNJUK, bukan salahmu] pwsh tidak ada di host ini (%s). Pakai tool Bash untuk perintah shell di sini", runtime.GOOS)
		}
	}
	shared := tools.FromSharedDir(ctx)
	if shared == "" {
		return tools.Result{}, fmt.Errorf("PowerShell: shared dir not in context")
	}
	timeoutSec := 20
	if n, ok := argInt(args, "timeout"); ok && n > 0 {
		timeoutSec = int(n)
	}
	if timeoutSec < 1 {
		timeoutSec = 1
	}
	if timeoutSec > 60 {
		timeoutSec = 60
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	c := exec.CommandContext(runCtx, pwsh, "-NoProfile", "-NonInteractive", "-Command", cmd)
	c.Dir = shared
	c.Env = scrubEnv()
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
				"exit_code": 124, "duration_ms": elapsed,
			}}, nil
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return tools.Result{}, fmt.Errorf("pwsh exec: %w", runErr)
		}
	}
	return tools.Result{Output: map[string]any{
		"stdout": outBuf.String(), "stderr": errBuf.String(),
		"exit_code": exitCode, "duration_ms": elapsed,
	}}, nil
}

// =============================================================================
// Task family — durable task ledger over agent_runs (this agent's own store).
// =============================================================================

const taskRunsDDL = `CREATE TABLE IF NOT EXISTS agent_runs (
	id TEXT PRIMARY KEY, label TEXT, state TEXT NOT NULL DEFAULT 'pending',
	checkpoint TEXT, updated TEXT)`

type taskCreateTool struct{}

func (taskCreateTool) Name() string       { return "TaskCreate" }
func (taskCreateTool) Capability() string { return "state:write" }
func (taskCreateTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Record a task in this agent's durable task ledger and get a task_id. Note: the kernel runs synchronously, so this REGISTERS the task (subject/prompt/status) for tracking — it does not spawn a parallel background agent. Drive execution yourself, then TaskUpdate/TaskStop and read TaskOutput.",
		Params: []tools.Param{
			{Name: "subject", Type: tools.ParamString, Description: "short task subject/title", Required: true},
			{Name: "description", Type: tools.ParamString, Description: "what the task is"},
			{Name: "activeForm", Type: tools.ParamString, Description: "present-tense label shown while running"},
			{Name: "prompt", Type: tools.ParamString, Description: "the instruction/prompt for the task"},
		},
		Returns: "{task_id, status, subject}",
	}
}

func (taskCreateTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("TaskCreate: store not in context")
	}
	subject := argStr(args, "subject")
	if subject == "" {
		return tools.Result{}, fmt.Errorf("subject required")
	}
	db := store.DB()
	if _, err := db.Exec(taskRunsDDL); err != nil {
		return tools.Result{}, fmt.Errorf("TaskCreate schema: %w", err)
	}
	// NOTE: separator is "_" not "-" on purpose. "task-<digits>" contains the
	// substring "sk-<digits>" which matches the OpenAI-key shape in the result
	// secret-redactor (loket_wire.go secretRe) and gets nuked to [REDACTED]
	// before the model can chain the id into TaskOutput/Update/Stop. Found via
	// a real mr-flow test, not a direct tool call.
	id := fmt.Sprintf("task_%d", time.Now().UnixNano())
	meta := map[string]any{
		"description": argStr(args, "description"),
		"activeForm":  argStr(args, "activeForm", "active_form"),
		"prompt":      argStr(args, "prompt"),
		"output":      "",
	}
	cp, _ := json.Marshal(meta)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(
		"INSERT OR REPLACE INTO agent_runs (id,label,state,checkpoint,updated) VALUES (?,?, 'pending', ?, ?)",
		id, subject, string(cp), now); err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: map[string]any{"task_id": id, "status": "pending", "subject": subject}}, nil
}

type taskUpdateTool struct{}

func (taskUpdateTool) Name() string       { return "TaskUpdate" }
func (taskUpdateTool) Capability() string { return "state:write" }
func (taskUpdateTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Update a task's status in the ledger (e.g. pending|in_progress|completed). Optionally append output text.",
		Params: []tools.Param{
			{Name: "taskId", Type: tools.ParamString, Description: "task id from TaskCreate", Required: true},
			{Name: "status", Type: tools.ParamString, Description: "new status (pending|in_progress|completed|...)", Required: true},
			{Name: "output", Type: tools.ParamString, Description: "optional output text to store"},
		},
		Returns: "{task_id, status}",
	}
}

func (taskUpdateTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("TaskUpdate: store not in context")
	}
	id := argStr(args, "taskId", "task_id")
	status := argStr(args, "status")
	if id == "" || status == "" {
		return tools.Result{}, fmt.Errorf("taskId + status required")
	}
	db := store.DB()
	if _, err := db.Exec(taskRunsDDL); err != nil {
		return tools.Result{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	// Optionally merge output into checkpoint json.
	if out := argStr(args, "output"); out != "" {
		var cur string
		_ = db.QueryRow("SELECT COALESCE(checkpoint,'') FROM agent_runs WHERE id=?", id).Scan(&cur)
		meta := map[string]any{}
		if cur != "" {
			_ = json.Unmarshal([]byte(cur), &meta)
		}
		meta["output"] = out
		cp, _ := json.Marshal(meta)
		_, _ = db.Exec("UPDATE agent_runs SET checkpoint=? WHERE id=?", string(cp), id)
	}
	r, err := db.Exec("UPDATE agent_runs SET state=?, updated=? WHERE id=?", status, now, id)
	if err != nil {
		return tools.Result{}, err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return tools.Result{}, fmt.Errorf("[PETUNJUK, bukan salahmu] ga ada task %q di ledger. task_id valid datang dari TaskCreate — bikin dulu via TaskCreate, jangan nebak id (doktrin ERR_FILE_NOT_FOUND)", id)
	}
	return tools.Result{Output: map[string]any{"task_id": id, "status": status}}, nil
}

type taskStopTool struct{}

func (taskStopTool) Name() string       { return "TaskStop" }
func (taskStopTool) Capability() string { return "state:write" }
func (taskStopTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Mark a task terminal (stopped) in the ledger so it is not resumed.",
		Params: []tools.Param{
			{Name: "task_id", Type: tools.ParamString, Description: "task id to stop", Required: true},
		},
		Returns: "{task_id, status}",
	}
}

func (taskStopTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("TaskStop: store not in context")
	}
	id := argStr(args, "task_id", "taskId")
	if id == "" {
		return tools.Result{}, fmt.Errorf("task_id required")
	}
	db := store.DB()
	if _, err := db.Exec(taskRunsDDL); err != nil {
		return tools.Result{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r, err := db.Exec("UPDATE agent_runs SET state='stopped', updated=? WHERE id=?", now, id)
	if err != nil {
		return tools.Result{}, err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return tools.Result{}, fmt.Errorf("[PETUNJUK, bukan salahmu] ga ada task %q di ledger. task_id valid datang dari TaskCreate — cek dulu sebelum stop (doktrin ERR_FILE_NOT_FOUND)", id)
	}
	return tools.Result{Output: map[string]any{"task_id": id, "status": "stopped"}}, nil
}

type taskOutputTool struct{}

func (taskOutputTool) Name() string       { return "TaskOutput" }
func (taskOutputTool) Capability() string { return "state:read" }
func (taskOutputTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Read a task's current state + stored output from the ledger. (block/timeout are accepted for compatibility but the kernel is synchronous, so this returns the current snapshot immediately.)",
		Params: []tools.Param{
			{Name: "task_id", Type: tools.ParamString, Description: "task id to read", Required: true},
			{Name: "block", Type: tools.ParamBool, Description: "compat only — ignored (no async wait)"},
			{Name: "timeout", Type: tools.ParamInt, Description: "compat only — ignored"},
		},
		Returns: "{task_id, state, output, subject}",
	}
}

func (taskOutputTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("TaskOutput: store not in context")
	}
	id := argStr(args, "task_id", "taskId")
	if id == "" {
		return tools.Result{}, fmt.Errorf("task_id required")
	}
	db := store.DB()
	if _, err := db.Exec(taskRunsDDL); err != nil {
		return tools.Result{}, err
	}
	var label, state, cp string
	err := db.QueryRow("SELECT COALESCE(label,''), state, COALESCE(checkpoint,'') FROM agent_runs WHERE id=?", id).Scan(&label, &state, &cp)
	if err != nil {
		return tools.Result{}, fmt.Errorf("[PETUNJUK, bukan salahmu] ga ada task %q di ledger. task_id valid datang dari TaskCreate (doktrin ERR_FILE_NOT_FOUND)", id)
	}
	output := ""
	if cp != "" {
		meta := map[string]any{}
		if json.Unmarshal([]byte(cp), &meta) == nil {
			if o, ok := meta["output"].(string); ok {
				output = o
			}
		}
	}
	return tools.Result{Output: map[string]any{"task_id": id, "state": state, "output": output, "subject": label}}, nil
}

// =============================================================================
// ScheduleWakeup — durable one-shot wakeup record (wakeups table)
// =============================================================================

type scheduleWakeupTool struct{}

func (scheduleWakeupTool) Name() string       { return "ScheduleWakeup" }
func (scheduleWakeupTool) Capability() string { return "state:write" }
func (scheduleWakeupTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Record a one-shot self-wakeup: after delaySeconds, the agent loop should re-fire `prompt`. Writes a durable wakeups row (due time, prompt, reason). The engine/agent loop fires due wakeups (like cron schedules).",
		Params: []tools.Param{
			{Name: "delaySeconds", Type: tools.ParamInt, Description: "seconds from now to wake up", Required: true},
			{Name: "reason", Type: tools.ParamString, Description: "short reason (telemetry/visible)", Required: true},
			{Name: "prompt", Type: tools.ParamString, Description: "the input to fire on wake-up", Required: true},
		},
		Returns: "{wakeup_id, due, delay_seconds}",
	}
}

func (scheduleWakeupTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("ScheduleWakeup: store not in context")
	}
	delay, ok := argInt(args, "delaySeconds")
	if !ok || delay <= 0 {
		return tools.Result{}, fmt.Errorf("delaySeconds required (positive int)")
	}
	prompt := argStr(args, "prompt")
	reason := argStr(args, "reason")
	if prompt == "" || reason == "" {
		return tools.Result{}, fmt.Errorf("prompt + reason required")
	}
	db := store.DB()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS wakeups (
		id TEXT PRIMARY KEY, due_unix INTEGER NOT NULL, prompt TEXT, reason TEXT,
		fired INTEGER NOT NULL DEFAULT 0, created TEXT)`); err != nil {
		return tools.Result{}, fmt.Errorf("ScheduleWakeup schema: %w", err)
	}
	now := time.Now().UTC()
	due := now.Add(time.Duration(delay) * time.Second)
	id := fmt.Sprintf("wake-%d", now.UnixNano())
	if _, err := db.Exec(
		"INSERT INTO wakeups (id,due_unix,prompt,reason,fired,created) VALUES (?,?,?,?,0,?)",
		id, due.Unix(), prompt, reason, now.Format(time.RFC3339)); err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: map[string]any{
		"wakeup_id": id, "due": due.Format(time.RFC3339), "delay_seconds": delay,
	}}, nil
}

// =============================================================================
// Monitor — poll a command until its output matches `until`, bounded by timeout
// =============================================================================

type monitorTool struct{}

func (monitorTool) Name() string       { return "Monitor" }
func (monitorTool) Capability() string { return "exec:shell" }
func (monitorTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Repeatedly run `command` (every ~2s) until its combined output contains the `until` substring, or timeout_ms elapses. Synchronous + bounded (timeout cap 60s). Use for waiting on a condition (a file to appear, a process to report ready).",
		Params: []tools.Param{
			{Name: "until", Type: tools.ParamString, Description: "substring that signals the condition is met", Required: true},
			{Name: "command", Type: tools.ParamString, Description: "shell command to poll", Required: true},
			{Name: "description", Type: tools.ParamString, Description: "what you're waiting for"},
			{Name: "timeout_ms", Type: tools.ParamInt, Description: "max wait ms (default 30000, cap 60000)"},
			{Name: "persistent", Type: tools.ParamBool, Description: "compat only — synchronous kernel cannot keep watching after return"},
		},
		Returns: "{matched, iterations, last_stdout, last_stderr, last_exit, elapsed_ms}",
	}
}

func (monitorTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	until := argStr(args, "until")
	cmd := argStr(args, "command")
	if until == "" || cmd == "" {
		return tools.Result{}, fmt.Errorf("until + command required")
	}
	lower := strings.ToLower(cmd)
	for _, p := range shellDenyPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return tools.Result{}, fmt.Errorf("[PETUNJUK, bukan salahmu] perintah Monitor memuat pola berbahaya %q jadi diblok — ini aturan tetap. Pakai perintah observasi yang aman (read-only). Ragu? tanya owner (doktrin ERR_SHELL_DANGEROUS)", p)
		}
	}
	shared := tools.FromSharedDir(ctx)
	if shared == "" {
		return tools.Result{}, fmt.Errorf("Monitor: shared dir not in context")
	}
	timeoutMs := int64(30000)
	if n, ok := argInt(args, "timeout_ms"); ok && n > 0 {
		timeoutMs = n
	}
	if timeoutMs > 60000 {
		timeoutMs = 60000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	t0 := time.Now()
	iterations := 0
	var lastOut, lastErr string
	lastExit := 0
	for {
		iterations++
		runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		var c *exec.Cmd
		if runtime.GOOS == "windows" {
			c = exec.CommandContext(runCtx, "cmd", "/C", cmd)
		} else {
			c = exec.CommandContext(runCtx, "/bin/sh", "-c", cmd)
		}
		c.Dir = shared
		c.Env = scrubEnv()
		var outBuf, errBuf bytes.Buffer
		const outCap = 64 * 1024
		c.Stdout = &capWriter{buf: &outBuf, cap: outCap}
		c.Stderr = &capWriter{buf: &errBuf, cap: outCap}
		runErr := c.Run()
		cancel()
		lastOut, lastErr = outBuf.String(), errBuf.String()
		lastExit = 0
		if runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				lastExit = exitErr.ExitCode()
			}
		}
		if strings.Contains(lastOut, until) || strings.Contains(lastErr, until) {
			return tools.Result{Output: map[string]any{
				"matched": true, "iterations": iterations,
				"last_stdout": lastOut, "last_stderr": lastErr,
				"last_exit": lastExit, "elapsed_ms": time.Since(t0).Milliseconds(),
			}}, nil
		}
		if time.Now().After(deadline) {
			return tools.Result{Output: map[string]any{
				"matched": false, "iterations": iterations,
				"last_stdout": lastOut, "last_stderr": lastErr,
				"last_exit": lastExit, "elapsed_ms": time.Since(t0).Milliseconds(),
			}, Note: "timeout reached before condition met"}, nil
		}
		select {
		case <-ctx.Done():
			return tools.Result{}, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// =============================================================================
// SendUserFile — send file(s) to the user over Telegram (sendDocument)
// =============================================================================

type sendUserFileTool struct{}

func (sendUserFileTool) Name() string       { return "SendUserFile" }
func (sendUserFileTool) Capability() string { return "net:fetch:telegram" }
func (sendUserFileTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Send file(s) to the user via Telegram. Paths are relative to the agent shared workspace. Sends to the first chat in TELEGRAM_ALLOWED_CHATS. Bot token from agent secrets.",
		Params: []tools.Param{
			{Name: "files", Type: tools.ParamArray, Description: "list of file paths (relative to shared workspace)", Required: true},
			{Name: "caption", Type: tools.ParamString, Description: "optional caption for the file(s)"},
			{Name: "status", Type: tools.ParamString, Description: "optional status note (compat)"},
		},
		Returns: "{sent:[{file, message_id}], count}",
	}
}

func (sendUserFileTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("SendUserFile: store not in context")
	}
	shared := tools.FromSharedDir(ctx)
	if shared == "" {
		return tools.Result{}, fmt.Errorf("SendUserFile: shared dir not in context")
	}
	// Collect files (array of strings, or a single string).
	var files []string
	switch v := args["files"].(type) {
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				files = append(files, strings.TrimSpace(s))
			}
		}
	case string:
		if strings.TrimSpace(v) != "" {
			files = append(files, strings.TrimSpace(v))
		}
	}
	if len(files) == 0 {
		return tools.Result{}, fmt.Errorf("files required (non-empty list of paths)")
	}
	secrets, serr := store.Secrets()
	if serr != nil {
		return tools.Result{}, fmt.Errorf("read secrets: %w", serr)
	}
	token := strings.TrimSpace(secrets["TELEGRAM_BOT_TOKEN"])
	if token == "" {
		return tools.Result{}, fmt.Errorf("[PETUNJUK, bukan salahmu] TELEGRAM_BOT_TOKEN belum ada di secrets agent. Minta owner isi di Settings → Token/API Keys. Jangan hardcode atau nebak token (doktrin ERR_SECRET_MISSING)")
	}
	chatID, cerr := firstAllowedChat(secrets)
	if cerr != nil {
		return tools.Result{}, cerr
	}
	caption := argStr(args, "caption")

	sent := []map[string]any{}
	for _, rel := range files {
		if filepath.IsAbs(rel) {
			return tools.Result{}, fmt.Errorf("SendUserFile: path must be relative: %q", rel)
		}
		abs := filepath.Join(shared, rel)
		if relCheck, err := filepath.Rel(shared, abs); err != nil || strings.HasPrefix(relCheck, "..") {
			return tools.Result{}, fmt.Errorf("[PETUNJUK, bukan salahmu] path %q keluar dari workspace — diblok demi keamanan. Pakai path relatif DI DALAM shared workspace (doktrin ERR_WORKSPACE_ESCAPE)", rel)
		}
		msgID, err := telegramSendDocument(ctx, token, chatID, abs, caption)
		if err != nil {
			return tools.Result{}, fmt.Errorf("send %q: %w", rel, err)
		}
		sent = append(sent, map[string]any{"file": rel, "message_id": msgID})
		caption = "" // caption only on first file
	}
	return tools.Result{Output: map[string]any{"sent": sent, "count": len(sent)}}, nil
}

// telegramSendDocument — multipart POST /bot<token>/sendDocument.
func telegramSendDocument(ctx context.Context, token string, chatID int64, absPath, caption string) (int64, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return 0, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if caption != "" {
		_ = mw.WriteField("caption", caption)
	}
	fw, err := mw.CreateFormFile("document", filepath.Base(absPath))
	if err != nil {
		return 0, err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return 0, err
	}
	mw.Close()

	url := fmt.Sprintf("%s/bot%s/sendDocument", telegramAPIBase, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("telegram api: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var tg struct {
		OK     bool   `json:"ok"`
		Desc   string `json:"description"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &tg) != nil || !tg.OK {
		return 0, fmt.Errorf("telegram sendDocument failed: %s", tg.Desc)
	}
	return tg.Result.MessageID, nil
}

// =============================================================================
// StructuredOutput — capture a structured JSON payload as the agent's result
// =============================================================================

type structuredOutputTool struct{}

func (structuredOutputTool) Name() string       { return "StructuredOutput" }
func (structuredOutputTool) Capability() string { return "" }
func (structuredOutputTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Emit a structured result. Pass your findings as a JSON object/array; it is validated as present and returned verbatim as the structured output. Use this when a caller asked for machine-readable output.",
		Params: []tools.Param{
			{Name: "findings", Type: tools.ParamObject, Description: "the structured payload (object or array)", Required: true},
		},
		Returns: "{findings, ok}",
	}
}

func (structuredOutputTool) Run(_ context.Context, args map[string]any) (tools.Result, error) {
	v, present := args["findings"]
	if !present || v == nil {
		return tools.Result{}, fmt.Errorf("findings required")
	}
	return tools.Result{Output: map[string]any{"findings": v, "ok": true}}, nil
}

// =============================================================================
// Workflow — register a multi-step orchestration script (durable run record)
// =============================================================================

type workflowTool struct{}

func (workflowTool) Name() string       { return "Workflow" }
func (workflowTool) Capability() string { return "rpc:agent-invoke" }
func (workflowTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Register a multi-step orchestration script in the run ledger and get a run_id. Note: the kernel is synchronous and does NOT yet execute the script as a parallel multi-agent fan-out — this REGISTERS it (durable record) so a coordinator can run/inspect it. Does not claim to have run it.",
		Params: []tools.Param{
			{Name: "script", Type: tools.ParamString, Description: "the workflow script body"},
			{Name: "scriptPath", Type: tools.ParamString, Description: "path to a workflow script (relative to shared)"},
		},
		Returns: "{run_id, status, note}",
	}
}

func (workflowTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	store, ok := tools.FromStore(ctx)
	if !ok {
		return tools.Result{}, fmt.Errorf("Workflow: store not in context")
	}
	script := argStr(args, "script")
	scriptPath := argStr(args, "scriptPath", "script_path")
	if script == "" && scriptPath == "" {
		return tools.Result{}, fmt.Errorf("script or scriptPath required")
	}
	db := store.DB()
	if _, err := db.Exec(taskRunsDDL); err != nil {
		return tools.Result{}, err
	}
	id := fmt.Sprintf("wf-%d", time.Now().UnixNano())
	meta := map[string]any{"script": script, "scriptPath": scriptPath, "output": ""}
	cp, _ := json.Marshal(meta)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(
		"INSERT OR REPLACE INTO agent_runs (id,label,state,checkpoint,updated) VALUES (?,?, 'registered', ?, ?)",
		id, "workflow", string(cp), now); err != nil {
		return tools.Result{}, err
	}
	return tools.Result{
		Output: map[string]any{"run_id": id, "status": "registered"},
		Note:   "workflow registered (durable record). Synchronous kernel: no parallel fan-out executed yet — a coordinator runs it.",
	}, nil
}
