// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/flow_router
// Locked at: 2026-05-30
// Reason: Section 25 phase 2 LocalAI runtime — llama-server subprocess
//   lifecycle (start/stop/health). Single-instance model loaded at a time.
//   Phase 3 (multi-model swap, GPU layer config, llama.cpp build self-
//   compile) → tambah file baru.
//
// 2026-06-14 (owner-approved, Aola): added portable "flowork-brain" model
//   auto-resolution (ship router/models/flowork-brain.gguf next to binary —
//   no system Ollama needed) + the --jinja flag so the model's embedded chat
//   template parses <tool_call> into native tool_calls. Without --jinja the
//   tool call leaks into content = the train/serve skew "narration halu"
//   (root-caused + proven via live test, see the project root-cause writeup).
//
// runtime.go — Section 25 phase 2: llama-server subprocess.

package localai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Runtime — manages llama-server subprocess.
type Runtime struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	modelName string
	port      int
	binPath   string
	cli       *http.Client
}

// NewRuntime — caller supply path ke llama-server binary (or assume
// PATH-resolved "llama-server"). Default port 8088 (anti collide dengan
// kernel 1987 / router 2402 / mDNS 5353).
func NewRuntime(binPath string, port int) *Runtime {
	if binPath == "" {
		binPath = "llama-server"
	}
	if port <= 0 {
		port = 8088
	}
	return &Runtime{
		binPath: binPath,
		port:    port,
		cli:     &http.Client{Timeout: 10 * time.Second},
	}
}

// FloworkBrainModel — the canonical local model name. An agent opts in to the
// local brain by requesting this model; the runtime auto-resolves its GGUF from
// the portable models/ dir shipped next to the binary (no system Ollama needed).
const FloworkBrainModel = "flowork-brain"

// ResolveFloworkBrain — portable GGUF lookup for flowork-brain, mirroring the
// resolution order of internal/brain.brain.go. Order:
//   1. $FLOWORK_BRAIN_GGUF      (explicit override)
//   2. <exe-dir>/models/flowork-brain.gguf            (portable: ship models/ next to binary)
//   3. <exe-dir>/../router/models/flowork-brain.gguf  (dev/build layout)
//   4. ./router/models/flowork-brain.gguf             (run from repo root)
// Returns "" if not found anywhere.
func ResolveFloworkBrain() string {
	if p := os.Getenv("FLOWORK_BRAIN_GGUF"); p != "" && fileExists(p) {
		return p
	}
	var cands []string
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		cands = append(cands,
			filepath.Join(d, "models", "flowork-brain.gguf"),
			filepath.Join(d, "..", "router", "models", "flowork-brain.gguf"),
		)
	}
	cands = append(cands, filepath.Join("router", "models", "flowork-brain.gguf"))
	for _, c := range cands {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// Start — spawn llama-server with model file. Stop existing first kalau ada.
// Caller pass GGUF path. Best-effort: kalau binary tidak ada, return error.
func (r *Runtime) Start(modelName, ggufPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		_, _ = r.cmd.Process.Wait()
		r.cmd = nil
	}
	// Default to the canonical local brain + auto-resolve its portable GGUF so a
	// caller can just ask for "flowork-brain" with no path (ship-with-repo).
	if modelName == "" {
		modelName = FloworkBrainModel
	}
	if ggufPath == "" && modelName == FloworkBrainModel {
		ggufPath = ResolveFloworkBrain()
	}
	if modelName == "" || ggufPath == "" {
		return fmt.Errorf("model_name + gguf_path required (flowork-brain.gguf not found under models/ — run from repo root or set $FLOWORK_BRAIN_GGUF)")
	}
	// ┌───────────────────────────────────────────────────────────────────────┐
	// │  ⚠️  DO NOT REMOVE --jinja.  READ THIS BEFORE EDITING THIS ARG LIST.    │
	// ├───────────────────────────────────────────────────────────────────────┤
	// │ PAST MISTAKE — cost ~1 YEAR + 13 refactors + 2 months of training:     │
	// │                                                                         │
	// │ The local model emits tool calls as TEXT:  <tool_call>{...}</tool_call> │
	// │ llama-server only converts that text into a structured `tool_calls`     │
	// │ response when launched WITH --jinja (it then uses the GGUF's embedded   │
	// │ chat template, which for Qwen parses <tool_call>). WITHOUT --jinja the  │
	// │ tool call stays as plain `content` → the caller never sees a tool_call  │
	// │ → the tool never runs → the model then HALLUCINATES the tool result     │
	// │ ("narration halu").                                                     │
	// │                                                                         │
	// │ We hunted this bug for a YEAR in the WRONG layers — code refactors,     │
	// │ more training, a feedback loop — because the real fault was this single │
	// │ missing flag at the train↔serve seam. The harness was always fine; a    │
	// │ heavily fine-tuned 7B was also damaged (catastrophic forgetting), which │
	// │ masked it further. Full root cause + live proof:                        │
	// │     the project root-cause writeup                                        │
	// │                                                                         │
	// │ RULES for future maintainers (human or AI):                             │
	// │  • Keep --jinja. If tool-calling "halu" returns, check THIS flag first. │
	// │  • Add knowledge/persona via RAG — NOT by retraining the model.         │
	// │  • Never heavy-fine-tune a small base; it breaks tool-format + language.│
	// │  • If you must swap models, re-verify with a multi-tool probe.          │
	// └───────────────────────────────────────────────────────────────────────┘
	args := []string{
		"-m", ggufPath,
		"--port", strconv.Itoa(r.port),
		"--host", "127.0.0.1",
		"--jinja", // ← REQUIRED for tool-calling. See the warning box above.
		"-c", "8192",
	}
	// Optional GPU offload. Portable default is CPU-safe (a flashdisk runs on
	// unknown hardware — a hardcoded -ngl could OOM a small GPU). On a known
	// machine set $FLOWORK_NGL (e.g. 35) to offload layers to the GPU.
	if ngl := strings.TrimSpace(os.Getenv("FLOWORK_NGL")); ngl != "" {
		args = append(args, "-ngl", ngl)
	}
	cmd := exec.Command(r.binPath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start llama-server: %w", err)
	}
	r.cmd = cmd
	r.modelName = modelName
	// Wait for health up to 30s.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if r.healthy() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("llama-server start timeout (port %d)", r.port)
}

// Stop — terminate process. Best-effort.
func (r *Runtime) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	_ = r.cmd.Process.Kill()
	_, _ = r.cmd.Process.Wait()
	r.cmd = nil
	r.modelName = ""
	return nil
}

// Status — return current state.
type Status struct {
	Running   bool   `json:"running"`
	ModelName string `json:"model_name"`
	Port      int    `json:"port"`
	Healthy   bool   `json:"healthy"`
}

func (r *Runtime) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := Status{
		Port:      r.port,
		ModelName: r.modelName,
	}
	if r.cmd != nil && r.cmd.Process != nil {
		st.Running = true
		// Health check without holding lock-after-IO is fine — race accepted.
	}
	if st.Running {
		st.Healthy = r.healthyUnlocked()
	}
	return st
}

func (r *Runtime) healthy() bool {
	return r.healthyUnlocked()
}

func (r *Runtime) healthyUnlocked() bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/health", r.port)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := r.cli.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return strings.Contains(string(body), "ok") || resp.StatusCode == 200
}
