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
// 2026-06-15 (owner-approved, Aola): boot auto-start support — (1) ResolveLlamaBin()
//   cross-OS binary resolver (FLOWORK_LLAMA_BIN → <exe-dir>/bin → PATH) so main.go
//   can auto-start the local model on boot (Linux/Mac/Win, all editions); (2) ctx
//   window default 8192→16384 (env FLOWORK_CTX) — doctrine-enriched prompts run
//   ~9-12k tokens; 8192 rejected them ("exceed context") → silent failover to cloud
//   ("agent stuck on Claude"); (3) --reasoning off default (env FLOWORK_REASONING) —
//   stock models burn the completion budget thinking → empty content. --jinja kept.
//
// 2026-06-16 (owner-approved, Aola): MoE/VRAM tuning knobs (env-gated, portable defaults
//   = zero regression) — (1) FLOWORK_CPU_MOE=1 → --cpu-moe (MoE experts on CPU/RAM,
//   attention on GPU); (2) FLOWORK_KV_TYPE=q8_0 → -fa on + -ctk/-ctv quant (KV-cache
//   quantization → thinking fits VRAM, no OOM). MEASURED gemma4 26B-A4B on RTX 4060 8GB:
//   FLOWORK_NGL=99 + FLOWORK_CPU_MOE=1 + FLOWORK_KV_TYPE=q8_0 → ~30 tok/s @ ~3GB VRAM
//   (was CPU-only crawl + thinking-OOM). Per-machine values live in a LOCAL env (not
//   committed) so the repo stays portable. RE-LOCKED — jangan diubah AI lain tanpa owner.
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
		if binPath = ResolveLlamaBin(); binPath == "" {
			binPath = "llama-server" // PATH fallback; Start() errors clearly if absent
		}
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

// ResolveLlamaBin — cross-OS llama-server resolution for boot auto-start.
// Order: $FLOWORK_LLAMA_BIN → <exe-dir>/bin/llama-server[.exe] → <exe-dir>/llama-server[.exe]
//   → PATH. Returns "" if nothing resolves anywhere (callers treat "" as "no local
//   runtime present" → skip auto-start; the manual GUI path falls back to PATH name).
func ResolveLlamaBin() string {
	if p := strings.TrimSpace(os.Getenv("FLOWORK_LLAMA_BIN")); p != "" && fileExists(p) {
		return p
	}
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		for _, c := range []string{
			filepath.Join(d, "bin", "llama-server"), filepath.Join(d, "bin", "llama-server.exe"),
			filepath.Join(d, "llama-server"), filepath.Join(d, "llama-server.exe"),
		} {
			if fileExists(c) {
				return c
			}
		}
	}
	if lp, err := exec.LookPath("llama-server"); err == nil {
		return lp
	}
	return ""
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
	// Context window. Doctrine-enriched prompts (constitution + brain + history +
	// tools) run ~9-12k tokens; the old 8192 default REJECTED them ("exceed context
	// size") → the router silently failed over to cloud ("agent stuck on Claude").
	// 16384 gives headroom on modest RAM; bump via FLOWORK_CTX (e.g. 32768) on a
	// roomy machine.
	cw := strings.TrimSpace(os.Getenv("FLOWORK_CTX"))
	if cw == "" {
		cw = "16384"
	}
	args := []string{
		"-m", ggufPath,
		"--port", strconv.Itoa(r.port),
		"--host", "127.0.0.1",
		"--jinja", // ← REQUIRED for tool-calling. See the warning box above.
		"-c", cw,
	}
	// Reasoning/thinking: a STOCK (un-fine-tuned) model burns the whole completion
	// budget "thinking" and returns empty content. Default OFF; a fine-tuned model
	// with short <think> can set FLOWORK_REASONING=on (or auto).
	reasoning := strings.TrimSpace(os.Getenv("FLOWORK_REASONING"))
	if reasoning == "" {
		reasoning = "off"
	}
	args = append(args, "--reasoning", reasoning)
	// Clean separation: any model "thinking" (channel/<think> tags) goes to
	// message.reasoning_content, NEVER leaks into message.content. Safe no-op when
	// reasoning=off or on a non-thinking model. Future-proof for when the evolution
	// debate-council runs with thinking on. Does not affect --jinja tool-call parsing.
	args = append(args, "--reasoning-format", "deepseek")
	// Optional GPU offload. Portable default is CPU-safe (a flashdisk runs on
	// unknown hardware — a hardcoded -ngl could OOM a small GPU). On a known
	// machine set $FLOWORK_NGL (e.g. 35, or 99 for "all") to offload layers to the GPU.
	if ngl := strings.TrimSpace(os.Getenv("FLOWORK_NGL")); ngl != "" {
		args = append(args, "-ngl", ngl)
	}
	// MoE offload (FLOWORK_CPU_MOE=1): keep Mixture-of-Experts weights on CPU/RAM while
	// attention runs on the GPU. Lets a big MoE (e.g. gemma4 26B-A4B, only 4B active)
	// FLY on a small-VRAM card — measured RTX 4060 8GB: ~30 tok/s using just ~3GB VRAM
	// (vs CPU-only crawl). No-op on a dense model. Pair with FLOWORK_NGL=99 (all attention
	// layers → GPU). This is what makes a 14GB model usable on 8GB VRAM without OOM.
	if os.Getenv("FLOWORK_CPU_MOE") == "1" {
		args = append(args, "--cpu-moe")
	}
	// KV-cache quantization (FLOWORK_KV_TYPE=q8_0|q4_0): halves/quarters KV-cache memory,
	// so long context / "thinking" fits the VRAM headroom WITHOUT OOM (the old thinking-OOM
	// fix). Requires flash-attention → we force -fa on when a quant type is set.
	if kt := strings.TrimSpace(os.Getenv("FLOWORK_KV_TYPE")); kt != "" {
		args = append(args, "-fa", "on", "-ctk", kt, "-ctv", kt)
	}
	cmd := exec.Command(r.binPath, args...)
	// PORTABLE (audit #10 2026-06-15): prefer shared libs (libggml/libllama .so) yang
	// se-folder sama binary (router/bin/), biar self-contained — gak gantung ke build dir
	// luar (mis. /home/mrflow/llama.cpp). make-distributable bundle binary+libs per-OS.
	binDir := filepath.Dir(r.binPath)
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+binDir+string(os.PathListSeparator)+os.Getenv("LD_LIBRARY_PATH"))
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
