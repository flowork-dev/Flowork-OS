// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30 (re-locked 2026-06-11; +env hook 2026-06-12)
// 2026-06-12 OWNER-APPROVED (frozen unfreeze→refreeze, hash regenerated): added the
//   `EnvForwardKeys` PLUG-AND-PLAY hook so buildAgentEnv can forward extra global
//   env keys (connector tokens, future secrets) registered from NON-FROZEN code —
//   so adding a new token NEVER requires unlocking this file again. This is meant to
//   be the LAST edit to the env-forward path; new keys go through the hook.
// 2026-06-03 TWEAK (param-only): InvokeAgentMessage deadline 180s→300s (selaras
//   manifest timeout_call_ms=300000) — fix synth crew 6-agent kena deadline.
//   Cap doang, ga ngubah orkestrasi/isolasi.
// 2026-06-15 OWNER-APPROVED (FASE 2 autonomi): InvokeAgentMessage skarang catat
//   SELF-AWARENESS di choke-point (recordInvokeSelfKnowledge) — interaction+decision+karma
//   per invoke buat SEMUA agent (dulu cuma mr-flow legacy opt-in). Best-effort, additive,
//   nol ubah orkestrasi/isolasi. Fuel buat gate auto-commit R7 + ngidupin menu Diagnostics.
// 2026-06-11 OWNER-APPROVED (frozen file unfreeze→refreeze, KERNEL_FREEZE.md hash
//   regenerated): buildAgentEnv forward-allowlist + DEVTO_API_KEY/X_AUTH_TOKEN/
//   X_CT0/LINKEDIN_COOKIE so a per-platform promo group reads its own publishing
//   key from Settings → API Keys via os.Getenv. Curated allowlist only — does NOT
//   widen the env surface beyond these named keys. No orchestration/isolation change.
//   (2nd pass same day, owner-approved, hash re-regenerated): +FWOS_CHAT_ID/
//   FWOS_BOT_TOKEN so the promo group shares each published article to the
//   FLOWORK_OS Telegram group. Same curated-allowlist pattern.
// Reason: Kernel orchestrator (CRITICAL). Audit pass:
//   - Boot: per-agent rejection isolation (one bad agent ngga kill boot)
//   - Workspace mkdir 0o755, state.db touch
//   - Broker.Approve dari manifest.CapabilitiesRequired only
//   - mu.Lock per public lifecycle method (Reload, handleAgentChange)
//   - No direct SQL — delegate to agentdb.Store (proper layer separation)
//   - HTTP handlers (RPC/Status/Agents/UISchema) input-validated
//   - Host fn callbacks (logInteraction/Decision/karmaUpdate/dispatchSlash)
//     resolve pluginID dari ctx unexported key — anti spoof
//   - Cron: retention sweep + promote + workspace rebuild per-agent
//   - Watcher: fsnotify hot-reload dengan debounce (delegate ke loader)
//   - AgentIDs(), CapsCheckerForAgent() — read-only accessors
//
// Package kernelhost — embedded kernel runtime. Single-binary embedded kernel — wazero runtime + capability broker
// + scanner running in-process. Sebelum: kernel terpisah di :1988.
// Sekarang: satu binary, satu port (:1987).
//
//
// Host = singleton state untuk wazero runtime + broker + loader scan.
// HTTP handlers di-attach ke mux GUI lama via methods di Host.

package kernelhost

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/httpx"
	"flowork-gui/internal/kernel/broker"
	"flowork-gui/internal/kernel/loader"
	"flowork-gui/internal/kernel/runtime"
	"flowork-gui/internal/routerclient"
	"flowork-gui/internal/slashcmd"
)

// LiveEntry — per-agent record yang Host pegang setelah scan + load.
type LiveEntry struct {
	Discovery loader.Discovery
	Instance  *runtime.Instance
	Enabled   bool // false = user toggle off (skip boot, no daemon)
}

// Host — single embedded kernel instance untuk seluruh proses.
type Host struct {
	Runtime   *runtime.Runtime
	Broker    *broker.Broker
	AgentsDir string
	SharedDir string // ~/.flowork/shared/ — cross-agent workspace

	mu    sync.Mutex
	lives []LiveEntry

	// storeCache — *agentdb.Store per pluginID, di-reuse (fix bug.md #3: anti
	// open/close state.db tiap pesan). WAL cross-connection visibility sudah
	// diverifikasi: reader fresh tetap liat tulisan store cached. Di-close
	// semua saat Host.Close().
	storeCache sync.Map
}

// cachedStore — buka state.db sekali per agent lalu reuse. agentdb.Store
// mutex-protected + WAL → aman concurrent + reader lain tetap liat tulisannya.
// JANGAN Close hasilnya (lifetime = Host); di-close di Host.Close().
func (h *Host) cachedStore(pluginID, agentPath string) (*agentdb.Store, error) {
	if v, ok := h.storeCache.Load(pluginID); ok {
		return v.(*agentdb.Store), nil
	}
	dbPath := agentdb.Resolve(pluginID, agentPath)
	store, err := agentdb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open state.db: %w", err)
	}
	if actual, loaded := h.storeCache.LoadOrStore(pluginID, store); loaded {
		_ = store.Close() // race: handle lain udah ke-store duluan
		return actual.(*agentdb.Store), nil
	}
	return store, nil
}

// SharedSubfolders — struktur subfolder standar di shared workspace
// per agent (HARDCODED). Tiap agent dapet folder `<shared>/<id>/` dengan
// 6 subfolder ini auto-created saat boot. Plus `_global/` di root shared
// untuk bahan bareng lintas agent.
var SharedSubfolders = []string{
	"tools",    // script/tool yang agent bikin (.py, .sh, .go) — bisa diakses agent lain
	"job",      // output kerjaan (hasil scrape/process)
	"document", // markdown, notes, report
	"media",    // audio, video, image
	"cache",    // cache temporary (agent boleh hapus sendiri)
	"log",      // log per-agent
}

// sharedWorkspaceDir — HARDCODED ke root project.
//
// Konvensi standar: `<project-root>/workspace/`. Project root = cwd
// kalau ada folder `agents/` di sana, else fallback ke `~/.flowork/workspace/`
// (untuk binary yang di-run headless tanpa source tree).
func sharedWorkspaceDir() string {
	// Root via ProjectRoot() (env FLOWORK_PROJECT_ROOT > cwd) — fix bug.md #2.
	root := agentdb.ProjectRoot()
	if stat, err := os.Stat(filepath.Join(root, "agents")); err == nil && stat.IsDir() {
		return filepath.Join(root, "workspace")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".flowork", "workspace")
	}
	return "/tmp/flowork-workspace"
}

// ensureAgentWorkspace bikin folder workspace per-agent + return path-nya.
// HARDCODED konvensi: `<source>/workspace/` (source) atau `<staged>/workspace/`.
// Plus touch state.db kosong supaya agent runtime bisa langsung open.
func ensureAgentWorkspace(agentID, stagedPath string) (string, error) {
	ws := agentdb.SourceWorkspace(agentID, stagedPath)
	if err := os.MkdirAll(ws, 0o755); err != nil {
		return "", err
	}
	// state.db — HARDCODED di workspace/state.db. Kernel jamin file ada,
	// schema/migrasi dikerjakan agentdb.Open() lewat ensureSchema().
	dbPath := filepath.Join(ws, "state.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		f, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			return "", fmt.Errorf("touch state.db: %w", err)
		}
		_ = f.Close()
	}
	return ws, nil
}

// ensureAgentSharedSpace bikin folder agent di shared workspace +
// 6 subfolder standar (tools/job/document/media/cache/log). Idempotent.
//
// Layout: `<shared>/<agentID>/{tools,job,document,media,cache,log}/`.
// Kalau salah satu subfolder udah ada → skip. Folder `_global/` di root
// shared juga di-ensure (1x saat boot pertama).
func ensureAgentSharedSpace(sharedRoot, agentID string) error {
	// Root shared + _global (lintas-agent).
	if err := os.MkdirAll(filepath.Join(sharedRoot, "_global"), 0o755); err != nil {
		return fmt.Errorf("mkdir _global: %w", err)
	}
	agentShared := filepath.Join(sharedRoot, agentID)
	for _, sub := range SharedSubfolders {
		if err := os.MkdirAll(filepath.Join(agentShared, sub), 0o755); err != nil {
			return fmt.Errorf("mkdir shared/%s/%s: %w", agentID, sub, err)
		}
	}
	return nil
}

// hasSharedCap cek manifest agent declare "fs:shared".
// Tanpa cap ini, /shared ngga di-mount → agent ngga bisa intip workspace tetangga.
func hasSharedCap(m *loader.Manifest) bool {
	if m == nil {
		return false
	}
	for _, c := range m.CapabilitiesRequired {
		if c == "fs:shared" {
			return true
		}
	}
	return false
}

// dangerousCapPrefixes — capabilities that grant host-level power. A side-loaded
// agent must not self-grant these merely by listing them in its own manifest.
var dangerousCapPrefixes = []string{"exec:", "rpc:agent-invoke", "fs:shared", "secret:"}

func isDangerousCap(c string) bool {
	for _, p := range dangerousCapPrefixes {
		if strings.HasPrefix(c, p) {
			return true
		}
	}
	return false
}

func capsContains(caps []string, want string) bool {
	for _, c := range caps {
		if c == want {
			return true
		}
	}
	return false
}

var privilegedWarnOnce sync.Once

// filterPrivilegedCaps gates dangerous capabilities behind an owner allowlist.
// FLOWORK_PRIVILEGED_AGENTS (comma-separated agent IDs) lists which agents may
// hold exec/power/agent-invoke/shared-fs caps. If SET, dangerous caps are
// stripped from any agent not on the list. If UNSET, behaviour is unchanged
// (dev-trust) but a one-time warning is logged — set the allowlist before
// loading untrusted agents.
func filterPrivilegedCaps(agentID string, caps []string) []string {
	allow := strings.TrimSpace(os.Getenv("FLOWORK_PRIVILEGED_AGENTS"))
	if allow == "" {
		for _, c := range caps {
			if isDangerousCap(c) {
				privilegedWarnOnce.Do(func() {
					log.Printf("SECURITY: agent %q auto-granted dangerous capability %q from its own manifest (dev-trust). Set FLOWORK_PRIVILEGED_AGENTS=<ids> to restrict before loading untrusted agents.", agentID, c)
				})
				break
			}
		}
		return caps
	}
	listed := false
	for _, id := range strings.Split(allow, ",") {
		if strings.TrimSpace(id) == agentID {
			listed = true
			break
		}
	}
	if listed {
		return caps
	}
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		if isDangerousCap(c) {
			log.Printf("SECURITY: stripped dangerous capability %q from non-privileged agent %q (not in FLOWORK_PRIVILEGED_AGENTS)", c, agentID)
			continue
		}
		out = append(out, c)
	}
	return out
}

// Boot init wazero + scan + load semua agent. Caller nyalain hot-reload
// + auto-boot daemon di goroutine sendiri (lihat Host.StartWatcher /
// Host.AutoBootDaemons).
func Boot(ctx context.Context) (*Host, error) {
	agentsDir := loader.AgentsDir()
	sharedDir := sharedWorkspaceDir()

	rt := runtime.New(ctx)
	br := broker.New()
	// Host di-allocate dulu supaya callback ke host_log_interaction bisa
	// closure-capture h (callback baru jalan saat agent invoke RPC — sampai
	// situ h sudah fully populated).
	h := &Host{
		Runtime:   rt,
		Broker:    br,
		AgentsDir: agentsDir,
		SharedDir: sharedDir,
	}
	if err := rt.Bootstrap(ctx, br.IsApproved, rt.Get, h.logInteraction, h.logDecision, h.karmaUpdate, h.dispatchSlash); err != nil {
		return nil, fmt.Errorf("runtime bootstrap: %w", err)
	}

	log.Printf("kernel: agents dir  %s", agentsDir)
	log.Printf("kernel: shared dir  %s", sharedDir)
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir shared: %w", err)
	}
	discoveries, err := loader.Scan(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("agent scan: %w", err)
	}

	var accepted, rejected int
	for _, d := range discoveries {
		if d.State == loader.StateFailed {
			log.Printf("kernel: rejected %s: %s", d.Path, d.RejectReason)
			rejected++
			h.lives = append(h.lives, LiveEntry{Discovery: d})
			continue
		}
		wasm, rerr := os.ReadFile(d.Path + "/" + d.Manifest.Entry)
		if rerr != nil {
			log.Printf("kernel: rejected %s: read entry: %v", d.Path, rerr)
			rejected++
			d.State = loader.StateFailed
			d.RejectReason = "read entry: " + rerr.Error()
			h.lives = append(h.lives, LiveEntry{Discovery: d})
			continue
		}

		effCaps := filterPrivilegedCaps(d.Manifest.ID, d.Manifest.CapabilitiesRequired)
		br.Approve(d.Manifest.ID, effCaps)

		inst, ierr := rt.LoadInstance(ctx, d.Manifest.ID, wasm, d.Manifest.MemoryMaxMB)
		if ierr != nil {
			log.Printf("kernel: rejected %s: instantiate: %v", d.Path, ierr)
			rejected++
			d.State = loader.StateFailed
			d.RejectReason = "instantiate: " + ierr.Error()
			h.lives = append(h.lives, LiveEntry{Discovery: d})
			continue
		}

		// Workspace per-agent (mandatory isolation, HARDCODED) + shared.
		ws, werr := ensureAgentWorkspace(d.Manifest.ID, d.Path)
		if werr != nil {
			log.Printf("kernel: rejected %s: workspace: %v", d.Path, werr)
			rejected++
			d.State = loader.StateFailed
			d.RejectReason = "workspace: " + werr.Error()
			h.lives = append(h.lives, LiveEntry{Discovery: d})
			continue
		}
		// Shared workspace — mount /shared into the guest ONLY when the agent
		// EFFECTIVELY holds fs:shared (after the privileged-cap filter). Otherwise
		// a non-privileged / non-allowlisted agent would get cross-agent filesystem
		// access via WASI even though the fs:shared cap token was stripped.
		shared := ""
		if capsContains(effCaps, "fs:shared") {
			if err := ensureAgentSharedSpace(h.SharedDir, d.Manifest.ID); err != nil {
				log.Printf("kernel: warn shared subdirs %s: %v", d.Manifest.ID, err)
			}
			shared = h.SharedDir
		}
		inst.SetWorkspaces(ws, shared)

		// SQLite per-agent — HARDCODED di `<workspace>/state.db`.
		dbPath := agentdb.Resolve(d.Manifest.ID, d.Path)
		store, sErr := agentdb.Open(dbPath)
		if sErr != nil {
			log.Printf("kernel: rejected %s: open db: %v", d.Manifest.ID, sErr)
			rejected++
			d.State = loader.StateFailed
			d.RejectReason = "open db: " + sErr.Error()
			h.lives = append(h.lives, LiveEntry{Discovery: d})
			continue
		}
		// One-time migrate config.json → DB (idempotent).
		if err := store.MigrateFromJSON(d.Path); err != nil {
			log.Printf("kernel: warn migrate config.json %s: %v", d.Manifest.ID, err)
		}

		// Cek user toggle off — kalau disabled, unload instance + skip
		// auto-boot. Agent tetap muncul di list dengan enabled=false;
		// re-enable lewat /api/agents/toggle trigger ReloadAgent.
		disabled := store.Disabled()
		if disabled {
			_ = store.Close()
			_ = h.Runtime.Unload(ctx, d.Manifest.ID)
			log.Printf("kernel: %s disabled by user — skip daemon", d.Manifest.ID)
			d.State = loader.StateReady
			h.lives = append(h.lives, LiveEntry{Discovery: d, Enabled: false})
			accepted++
			continue
		}

		// Forward FLOWORK_* + DB config + workspace paths ke agent env.
		env := buildAgentEnv(d, store, ws, shared)
		_ = store.Close()
		if len(env) > 0 {
			inst.SetEnv(env)
		}

		log.Printf("kernel: loaded %s v%s (%s) caps=%d ws=%s db=%s",
			d.Manifest.ID, d.Manifest.Version, d.Manifest.Kind,
			len(d.Manifest.CapabilitiesRequired), ws, dbPath)
		accepted++
		d.State = loader.StateReady
		h.lives = append(h.lives, LiveEntry{Discovery: d, Instance: inst, Enabled: true})
	}
	log.Printf("kernel: agent scan complete: %d accepted, %d rejected", accepted, rejected)
	return h, nil
}

// AutoBootDaemons — spawn satu goroutine per agent yang declare `boot`
// di exposes_rpc. Goroutine call inst.Call(ctx, "boot", "{}") yang
// blocking sampai agent main() return atau ctx cancel.
func (h *Host) AutoBootDaemons(ctx context.Context) {
	for _, l := range h.lives {
		if l.Instance == nil || l.Discovery.Manifest == nil || !l.Enabled {
			continue
		}
		id := l.Discovery.Manifest.ID
		hasBoot := false
		for _, m := range l.Discovery.Manifest.ExposesRPC {
			if m.Name == "boot" {
				hasBoot = true
				break
			}
		}
		if !hasBoot {
			continue
		}
		inst := l.Instance
		go func() {
			log.Printf("kernel: daemon-boot %s", id)
			_, err := inst.Call(ctx, "boot", []byte("{}"))
			if err != nil {
				log.Printf("kernel: daemon-boot %s exited: %v", id, err)
			} else {
				log.Printf("kernel: daemon-boot %s exited cleanly", id)
			}
		}()
	}
}

// callOnLoad invokes a module's optional on_load entry (§8.A: init / register /
// migrate) right after it loads. Best-effort + bounded; a module with no on_load
// just returns "unknown function", which is ignored.
func (h *Host) callOnLoad(ctx context.Context, id string) {
	inst := h.Runtime.Get(id)
	if inst == nil {
		return
	}
	c2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, _ = inst.Call(c2, "on_load", []byte("{}"))
}

// callOnStop invokes a module's optional on_stop entry (§8.A: death-letter /
// cleanup) BEFORE it is unloaded. It runs OUTSIDE the runtime lock — Runtime.Get
// returns before we Call — so a module that touches the bus in on_stop cannot
// deadlock the unload. Best-effort + bounded.
func (h *Host) callOnStop(ctx context.Context, id string) {
	inst := h.Runtime.Get(id)
	if inst == nil {
		return
	}
	c2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, _ = inst.Call(c2, "on_stop", []byte("{}"))
}

// AutoOnLoad calls on_load (§8.A) on every loaded+enabled module. Run AFTER the
// HTTP loket endpoint is listening, so a module's on_load can use loket caps
// (store/brain/etc) — at AutoBootDaemons time the loket isn't up yet.
func (h *Host) AutoOnLoad(ctx context.Context) {
	h.mu.Lock()
	ids := make([]string, 0, len(h.lives))
	for _, l := range h.lives {
		if l.Instance != nil && l.Discovery.Manifest != nil && l.Enabled {
			ids = append(ids, l.Discovery.Manifest.ID)
		}
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.callOnLoad(ctx, id)
	}
}

// AutoOnStop calls on_stop (§8.A) on every loaded+enabled module. Run on graceful
// shutdown BEFORE the HTTP server closes, so a module's on_stop (death-letter) can
// still reach loket caps.
func (h *Host) AutoOnStop(ctx context.Context) {
	h.mu.Lock()
	ids := make([]string, 0, len(h.lives))
	for _, l := range h.lives {
		if l.Instance != nil && l.Discovery.Manifest != nil && l.Enabled {
			ids = append(ids, l.Discovery.Manifest.ID)
		}
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.callOnStop(ctx, id)
	}
}

// StartRetentionCron — spawn goroutine yang jalan retention sweep tiap
// interval. Default 24h interval (per roadmap section 8). Iterate semua
// agent di h.lives, call agentdb.Store.RunRetentionSweep per agent.
//
// Aman terhadap shutdown — listen ke ctx.Done(). Sweep per agent serial
// (open-on-demand pattern, same as logInteraction/logDecision).
//
// Roadmap section 8.
func (h *Host) StartRetentionCron(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		// Initial delay supaya ngga jalan persis boot — kasih 1 menit warm-up.
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			h.runRetentionSweepAllAgents(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	log.Printf("kernel: retention cron armed (interval=%s)", interval)
}

// runRetentionSweepAllAgents — iterate snapshot lives, call sweep per agent.
// Hold h.mu sebentar buat snapshot — release sebelum heavy sweep (long-running
// per agent ngga lock semua agent lain).
func (h *Host) runRetentionSweepAllAgents(ctx context.Context) {
	h.mu.Lock()
	type sweepTarget struct {
		ID   string
		Path string
	}
	targets := make([]sweepTarget, 0, len(h.lives))
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Enabled {
			targets = append(targets, sweepTarget{
				ID:   l.Discovery.Manifest.ID,
				Path: l.Discovery.Path,
			})
		}
	}
	h.mu.Unlock()

	for _, t := range targets {
		if ctx.Err() != nil {
			return
		}
		report, err := h.RunRetentionForAgent(t.ID)
		if err != nil {
			log.Printf("kernel: retention sweep %s failed: %v", t.ID, err)
			continue
		}
		log.Printf("kernel: retention sweep %s done: %+v", t.ID, report)
	}
}

// RunRetentionForAgent — sweep satu agent dengan default retention windows.
// Caller: cron, atau admin endpoint POST /api/agents/retention/sweep?id=.
// Return agentdb.RetentionReport.
func (h *Host) RunRetentionForAgent(agentID string) (agentdb.RetentionReport, error) {
	h.mu.Lock()
	var agentPath string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == agentID {
			agentPath = l.Discovery.Path
			break
		}
	}
	h.mu.Unlock()

	if agentPath == "" {
		return agentdb.RetentionReport{}, fmt.Errorf("agent not loaded: %s", agentID)
	}
	dbPath := agentdb.Resolve(agentID, agentPath)
	store, err := agentdb.Open(dbPath)
	if err != nil {
		return agentdb.RetentionReport{}, fmt.Errorf("open state.db: %w", err)
	}
	defer store.Close()

	return store.RunRetentionSweep(agentdb.DefaultRetention()), nil
}

// StartWatcher — fsnotify watcher untuk hot-reload. Spawn goroutine yang
// receive change events + reload agent.
func (h *Host) StartWatcher(ctx context.Context) error {
	watcher, err := loader.NewWatcher(h.AgentsDir)
	if err != nil {
		return err
	}
	go watcher.Run(ctx)
	go func() {
		for ch := range watcher.Listener() {
			log.Printf("kernel: agent folder change: %s %s", ch.Kind, ch.AgentID)
			h.handleAgentChange(ctx, ch)
		}
	}()
	log.Printf("kernel: hot-reload watcher armed on %s", h.AgentsDir)
	return nil
}

// ReloadAgent — programmatic trigger reload satu agent. Dipanggil dari
// ConfigHandler setelah config save (fsnotify ngga reliable buat inner
// subfolder write).
func (h *Host) ReloadAgent(agentID string) error {
	h.mu.Lock()
	var path string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == agentID {
			path = l.Discovery.Path
			break
		}
	}
	h.mu.Unlock()
	if path == "" {
		return fmt.Errorf("agent not loaded: %s", agentID)
	}
	h.handleAgentChange(context.Background(), loader.Change{
		Kind:    loader.ChangeUpdated,
		Path:    path,
		AgentID: agentID,
	})
	return nil
}

// handleAgentChange react ke watcher event. Race-aware retry untuk
// extract yang belum selesai nulis wasm.
func (h *Host) handleAgentChange(ctx context.Context, ch loader.Change) {
	switch ch.Kind {
	case loader.ChangeRemoved:
		h.callOnStop(ctx, ch.AgentID) // §8.A: let the module write its death-letter first
		_ = h.Runtime.Unload(ctx, ch.AgentID)
		h.Broker.Revoke(ch.AgentID)
		h.mu.Lock()
		filtered := h.lives[:0]
		for _, l := range h.lives {
			if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == ch.AgentID {
				continue
			}
			filtered = append(filtered, l)
		}
		h.lives = filtered
		h.mu.Unlock()
		log.Printf("kernel: unloaded %s", ch.AgentID)
		return
	case loader.ChangeAdded, loader.ChangeUpdated:
		var m *loader.Manifest
		var wasm []byte
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(500 * time.Millisecond)
			}
			raw, err := os.ReadFile(ch.Path + "/manifest.json")
			if err != nil {
				continue
			}
			parsed, err := loader.Parse(raw)
			if err != nil {
				log.Printf("kernel: hot-reload skip %s: parse: %v", ch.AgentID, err)
				return
			}
			body, err := os.ReadFile(ch.Path + "/" + parsed.Entry)
			if err != nil || len(body) == 0 {
				continue
			}
			m = parsed
			wasm = body
			break
		}
		if m == nil || len(wasm) == 0 {
			log.Printf("kernel: hot-reload skip %s: source still incomplete", ch.AgentID)
			return
		}
		h.callOnStop(ctx, m.ID) // §8.A: death-letter before the old instance is replaced
		_ = h.Runtime.Unload(ctx, m.ID)
		effCaps := filterPrivilegedCaps(m.ID, m.CapabilitiesRequired)
		h.Broker.Approve(m.ID, effCaps)
		inst, err := h.Runtime.LoadInstance(ctx, m.ID, wasm, m.MemoryMaxMB)
		if err != nil {
			log.Printf("kernel: hot-reload load %s failed: %v", m.ID, err)
			return
		}
		// Mount workspaces (idempotent — mkdir already-exists is fine).
		d := loader.Discovery{Path: ch.Path, Manifest: m, State: loader.StateReady}
		ws, werr := ensureAgentWorkspace(m.ID, ch.Path)
		if werr != nil {
			log.Printf("kernel: hot-reload workspace %s: %v", m.ID, werr)
			return
		}
		shared := ""
		if capsContains(effCaps, "fs:shared") {
			if err := ensureAgentSharedSpace(h.SharedDir, m.ID); err != nil {
				log.Printf("kernel: warn shared subdirs %s (hot-reload): %v", m.ID, err)
			}
			shared = h.SharedDir
		}
		inst.SetWorkspaces(ws, shared)

		// SQLite per-agent (HARDCODED di workspace/state.db).
		dbPath := agentdb.Resolve(m.ID, ch.Path)
		var store *agentdb.Store
		disabled := false
		if s, sErr := agentdb.Open(dbPath); sErr == nil {
			store = s
			_ = store.MigrateFromJSON(ch.Path)
			disabled = store.Disabled()
		} else {
			log.Printf("kernel: warn open db %s (hot-reload): %v", m.ID, sErr)
		}

		// Kalau user toggle off — unload instance + skip daemon, tetap
		// list-in dengan enabled=false.
		if disabled {
			if store != nil {
				_ = store.Close()
			}
			_ = h.Runtime.Unload(ctx, m.ID)
			h.mu.Lock()
			filtered := h.lives[:0]
			for _, l := range h.lives {
				if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == m.ID {
					continue
				}
				filtered = append(filtered, l)
			}
			h.lives = append(filtered, LiveEntry{Discovery: d, Enabled: false})
			h.mu.Unlock()
			log.Printf("kernel: hot-reload %s → disabled (daemon stopped)", m.ID)
			return
		}

		// Inject env (FLOWORK_* + agent config + workspace mounts).
		env := buildAgentEnv(d, store, ws, shared)
		if len(env) > 0 {
			inst.SetEnv(env)
		}
		if store != nil {
			_ = store.Close()
		}
		h.mu.Lock()
		filtered := h.lives[:0]
		for _, l := range h.lives {
			if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == m.ID {
				continue
			}
			filtered = append(filtered, l)
		}
		h.lives = append(filtered, LiveEntry{Discovery: d, Instance: inst, Enabled: true})
		h.mu.Unlock()

		h.callOnLoad(ctx, m.ID) // §8.A: init after the (re)loaded module is live

		// Auto-boot kalau agent declare `boot`.
		hasBoot := false
		for _, exp := range m.ExposesRPC {
			if exp.Name == "boot" {
				hasBoot = true
				break
			}
		}
		if hasBoot {
			id := m.ID
			go func() {
				log.Printf("kernel: daemon-boot %s (hot-reload)", id)
				_, err := inst.Call(ctx, "boot", []byte("{}"))
				if err != nil {
					log.Printf("kernel: daemon-boot %s exited: %v", id, err)
				}
			}()
		}
		log.Printf("kernel: loaded %s v%s (%s)", m.ID, m.Version, m.Kind)
	}
}

// buildAgentEnv collect env vars yang agent perlu:
//   - Forward FLOWORK_* dari proses caller (mis. legacy fallback)
//   - Inject FLOWORK_AGENT_CONFIG (JSON dari SQLite store) + secrets sebagai env
//   - Inject FLOWORK_AGENT_ID, FLOWORK_AGENT_WORKSPACE, FLOWORK_SHARED_WORKSPACE,
//     FLOWORK_AGENT_DB (WASI mount points; "" kalau ngga di-mount)
//
// `store` opsional — kalau nil, env config + secrets di-skip (boot tanpa store
// hanya kepake oleh code path lama saat migrasi).
// EnvForwardKeys — PLUG-AND-PLAY extension point (set ONCE from non-frozen init,
// see main.go wiring connections.GlobalSecretEnvKeys). For the agent being built,
// buildAgentEnv forwards the EXTRA global env keys (Settings → API Keys) this hook
// returns FOR THAT agent, on top of the hardcoded base list below. Per-agent so a
// connector's token reaches ONLY that connector (not every agent). Register new
// tokens/secrets HERE from non-frozen code — never unlock this frozen file again to
// add a key. nil → base list only.
var EnvForwardKeys func(agentID string) []string

func buildAgentEnv(d loader.Discovery, store *agentdb.Store, workspaceMount, sharedMount string) map[string]string {
	out := map[string]string{}
	fwdKeys := []string{
		"FLOWORK_TG_BOT_TOKEN",
		"FLOWORK_TG_ALLOWED_CHATS",
		"FLOWORK_ROUTER_URL",
		"FLOWORK_LLM_MODEL",
		// Publishing credentials set in Settings → API Keys (os.Setenv'd before agents boot),
		// forwarded so a per-platform promo group reads its own key via os.Getenv. Curated
		// allowlist — only these Settings keys reach agents, not every global key.
		"DEVTO_API_KEY",
		"X_AUTH_TOKEN",
		"X_CT0",
		"LINKEDIN_COOKIE",
		// Share-to-community: the FLOWORK_OS Telegram group id + the bot token used
		// to post there (the promo group shares each published article). Settings →
		// API Keys, forwarded here so the group reads them via os.Getenv.
		"FWOS_CHAT_ID",
		"FWOS_BOT_TOKEN",
	}
	// Plug-and-play: non-frozen code may register MORE global keys to forward for
	// THIS agent (connector tokens, future secrets) without ever editing this frozen
	// file. Per-agent so a connector's token reaches only that connector.
	if EnvForwardKeys != nil && d.Manifest != nil {
		fwdKeys = append(fwdKeys, EnvForwardKeys(d.Manifest.ID)...)
	}
	for _, key := range fwdKeys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			out[key] = v
		}
	}
	if d.Manifest != nil {
		out["FLOWORK_AGENT_ID"] = d.Manifest.ID
	}
	if workspaceMount != "" {
		// Guest-side mount point — agent baca `os.ReadFile("/workspace/x")`.
		out["FLOWORK_AGENT_WORKSPACE"] = "/workspace"
		// SQLite per-agent terisolasi di dalam workspace.
		// Agent buka via guest path; kernel sudah touch file-nya saat boot.
		out["FLOWORK_AGENT_DB"] = "/workspace/state.db"
	}
	if sharedMount != "" {
		out["FLOWORK_SHARED_WORKSPACE"] = "/shared"
	}
	if store != nil {
		if raw, err := store.LoadJSON(); err == nil && len(raw) > 2 {
			out["FLOWORK_AGENT_CONFIG"] = string(raw)
		}
		// Per-agent isolated secrets — Telegram token, Google API key,
		// dll. Expand jadi env var supaya agent baca via os.Getenv(KEY)
		// tanpa parse JSON.
		if secrets, err := store.Secrets(); err == nil {
			for k, v := range secrets {
				if k = strings.TrimSpace(k); k != "" {
					out[k] = v
				}
			}
		}
	}
	return out
}

// ── HTTP handlers ──────────────────────────────────────────────────────────

// StatusHandler — GET /api/kernel/status.
func (h *Host) StatusHandler(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, map[string]any{
		"service":    "flowork-kernel-embedded",
		"agents_dir": h.AgentsDir,
		"loaded":     h.Runtime.Loaded(),
		"accepted":   h.countByState(loader.StateReady),
		"rejected":   h.countByState(loader.StateFailed),
	})
}

func (h *Host) countByState(s loader.State) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, l := range h.lives {
		if l.Discovery.State == s {
			n++
		}
	}
	return n
}

// AgentsHandler — GET /api/kernel/agents.
func (h *Host) AgentsHandler(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	out := make([]map[string]any, 0, len(h.lives))
	for _, l := range h.lives {
		d := l.Discovery
		entry := map[string]any{
			"path":    d.Path,
			"state":   string(d.State),
			"enabled": l.Enabled,
		}
		if d.Manifest != nil {
			entry["id"] = d.Manifest.ID
			entry["version"] = d.Manifest.Version
			entry["kind"] = string(d.Manifest.Kind)
			entry["display_name"] = d.Manifest.DisplayName
			entry["description"] = d.Manifest.Description
			entry["author"] = d.Manifest.Author
			entry["capabilities_required"] = d.Manifest.CapabilitiesRequired
			wsHost := filepath.Join(d.Path, "workspace")
			entry["workspace_host"] = wsHost
			entry["workspace_guest"] = "/workspace"
			// SQLite state.db per-agent (di dalam workspace folder).
			dbHost := filepath.Join(wsHost, "state.db")
			entry["db_host"] = dbHost
			entry["db_guest"] = "/workspace/state.db"
			if fi, err := os.Stat(dbHost); err == nil {
				entry["db_size"] = fi.Size()
			} else {
				entry["db_size"] = 0
			}
			// Pakai EFFECTIVE cap (Broker, post privileged-filter) biar report jujur —
			// kalau filter nyabut fs:shared, jangan ngaku punya /shared (audit #3 2026-06-15).
			if (h.Broker != nil && h.Broker.IsApproved(d.Manifest.ID, "fs:shared")) ||
				(h.Broker == nil && hasSharedCap(d.Manifest)) {
				entry["shared_host"] = h.SharedDir
				entry["shared_guest"] = "/shared"
			}
			if exposes := d.Manifest.ExposesRPC; len(exposes) > 0 {
				names := make([]string, 0, len(exposes))
				for _, m := range exposes {
					names = append(names, m.Name)
				}
				entry["exposes_rpc"] = names
			}
		}
		if d.RejectReason != "" {
			entry["reject_reason"] = d.RejectReason
		}
		out = append(out, entry)
	}
	h.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		ai, _ := out[i]["id"].(string)
		bj, _ := out[j]["id"].(string)
		return ai < bj
	})
	httpx.WriteJSON(w, map[string]any{"plugins": out, "count": len(out)})
}

// UISchemaHandler — GET /api/agents/ui-schema?id=<id>. Return
// manifest.UISchema (kalau ada) supaya frontend bisa render section
// extra di popup setting. Kalau agent ngga declare schema, return
// `{sections: []}` kosong.
func (h *Host) UISchemaHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		httpx.WriteJSON(w, map[string]any{"error": "id required"})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, l := range h.lives {
		if l.Discovery.Manifest == nil || l.Discovery.Manifest.ID != id {
			continue
		}
		schema := l.Discovery.Manifest.UISchema
		if schema == nil {
			httpx.WriteJSON(w, map[string]any{"sections": []any{}})
			return
		}
		httpx.WriteJSON(w, schema)
		return
	}
	httpx.WriteJSON(w, map[string]any{"error": "agent not found: " + id})
}

// RPCHandler — POST /api/kernel/rpc. Body: {plugin, function, args}.
func (h *Host) RPCHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	var body struct {
		Plugin   string          `json:"plugin"`
		Function string          `json:"function"`
		Args     json.RawMessage `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "parse: " + err.Error()})
		return
	}
	if body.Plugin == "" || body.Function == "" {
		httpx.WriteJSON(w, map[string]any{"error": "plugin + function required"})
		return
	}
	inst := h.Runtime.Get(body.Plugin)
	if inst == nil {
		httpx.WriteJSON(w, map[string]any{"error": "plugin not loaded: " + body.Plugin})
		return
	}
	if len(body.Args) == 0 {
		body.Args = []byte("{}")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	respBytes, err := inst.Call(ctx, body.Function, body.Args)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if len(respBytes) == 0 {
		respBytes = []byte("{}")
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(respBytes)
}

// logInteraction — implementasi `runtime.InteractionLogger`. Resolve
// pluginID → agent folder via h.lives, open state.db, append row di tabel
// `interactions`. Hold h.mu sepanjang Open+Insert supaya agent ngga bisa
// di-Unload paralel sambil log row.
//
// Plugin cuma log ke state.db nya sendiri — pluginID di-set kernel dari
// ctx (WithGuestPluginID), bukan dari payload. Anti spoof.
//
// Open-per-call: cheap untuk single-writer SQLite + insert kecil. Cache
// Store per pluginID belum di-implement (premature opt — Mr.Flow chat
// throughput sekarang low). Lihat audit deferred items di Changelog.
func (h *Host) logInteraction(pluginID, channel, direction, actor, content string, metadata map[string]any) error {
	if pluginID == "" {
		return fmt.Errorf("pluginID required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	var agentPath string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == pluginID {
			agentPath = l.Discovery.Path
			break
		}
	}
	if agentPath == "" {
		return fmt.Errorf("agent not loaded: %s", pluginID)
	}

	store, err := h.cachedStore(pluginID, agentPath)
	if err != nil {
		return err
	}
	_, err = store.LogInteraction(channel, direction, actor, content, metadata)
	return err
}

// logDecision — implementasi `runtime.DecisionLogger`. Pola sama dengan
// logInteraction: resolve pluginID → agent folder, open state.db, insert
// row decision. Hold h.mu sepanjang Open+Log (anti race dengan Unload).
// Return decision ID supaya caller bisa reuse buat cross-ref future.
//
// Plugin cuma log ke state.db nya sendiri — pluginID dari ctx (anti spoof).
// Anti over-prompt: data ini ngga boleh auto-inject ke system prompt;
// dashboard / tool call only.
//
// TODO Section 8 (perf): open-on-demand pattern serial Mr.Flow chat
// triggers 2 logInteraction + 1 logDecision = 3 SQLite Open per pesan.
// Cache *Store per pluginID di sync.Map kalau warga > 1 atau chat freq
// scale up.
func (h *Host) logDecision(pluginID, decisionType, rationale, outcome string, inputs map[string]any, refInteractionID int64) (int64, error) {
	if pluginID == "" {
		return 0, fmt.Errorf("pluginID required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	var agentPath string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == pluginID {
			agentPath = l.Discovery.Path
			break
		}
	}
	if agentPath == "" {
		return 0, fmt.Errorf("agent not loaded: %s", pluginID)
	}

	store, err := h.cachedStore(pluginID, agentPath)
	if err != nil {
		return 0, err
	}
	return store.LogDecision(decisionType, rationale, outcome, inputs, refInteractionID)
}

// karmaUpdate — implementasi `runtime.KarmaUpdater`. Resolve pluginID →
// agent folder, open state.db, call IncrementKarma atau AverageUpdateKarma
// tergantung op. Return current value (post-update) ke caller.
//
// Hold h.mu sepanjang Open+Update (race-safe pattern Section 1).
// Section 5 roadmap.
func (h *Host) karmaUpdate(pluginID, op, key string, value float64) (float64, error) {
	if pluginID == "" {
		return 0, fmt.Errorf("pluginID required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	var agentPath string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == pluginID {
			agentPath = l.Discovery.Path
			break
		}
	}
	if agentPath == "" {
		return 0, fmt.Errorf("agent not loaded: %s", pluginID)
	}

	store, err := h.cachedStore(pluginID, agentPath)
	if err != nil {
		return 0, err
	}
	switch op {
	case "increment":
		return store.IncrementKarma(key, value)
	case "average":
		return store.AverageUpdateKarma(key, value)
	default:
		return 0, fmt.Errorf("unknown op %q (use 'increment' or 'average')", op)
	}
}

// workspaceCataloged — set agent yang workspace-meta-nya UDAH di-catalog (1×/boot).
// Choke-point trigger RebuildWorkspaceMetaForAgent lazy (async) → menu Workspace keisi
// buat SEMUA agent tanpa nunggu admin klik "rebuild".
var workspaceCataloged sync.Map

// recordMistake — best-effort: catat 1 baris di Mistakes journal store agent. Pola sama
// logInteraction (hold h.mu sepanjang resolve+write, anti race Unload).
func (h *Host) recordMistake(agentID, category, title, content string) {
	if h == nil || agentID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var agentPath string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == agentID {
			agentPath = l.Discovery.Path
			break
		}
	}
	if agentPath == "" {
		return
	}
	store, err := h.cachedStore(agentID, agentPath)
	if err != nil {
		return
	}
	_, _, _ = store.AddMistake(category, title, content, "invoke-choke-point")
}

// recordInvokeSelfKnowledge — SELF-AWARENESS (fuel R7), owner-approved 2026-06-15.
// Di CHOKE-POINT InvokeAgentMessage: catat interaction(in/out)+decision+karma+mistake(on real
// error)+catalog workspace di store agent yang di-invoke → SEMUA menu Diagnostics ke-cover
// OTOMATIS buat SEMUA agent (dulu cuma mr-flow legacy opt-in). Best-effort: error di-swallow
// biar nggak ganggu invoke. karma invoke_success/invoke_fail = track-record gate auto-commit R7.
func (h *Host) recordInvokeSelfKnowledge(agentID, caller, text, reply string, callErr error) {
	if h == nil || agentID == "" {
		return
	}
	// Catalog workspace-meta 1×/boot/agent (async, non-blocking) → menu Workspace keisi.
	if _, done := workspaceCataloged.LoadOrStore(agentID, true); !done {
		go func() { _, _ = h.RebuildWorkspaceMetaForAgent(agentID) }()
	}
	clip := func(s string, n int) string {
		if len(s) > n {
			return s[:n]
		}
		return s
	}
	_ = h.logInteraction(agentID, "invoke", "in", caller, clip(text, 4000), nil)
	if callErr != nil {
		emsg := callErr.Error()
		_ = h.logInteraction(agentID, "invoke", "out", agentID, "error: "+clip(emsg, 2000), nil)
		_, _ = h.logDecision(agentID, "agent_invoke", "invoke gagal: "+clip(emsg, 240), "fail", map[string]any{"caller": caller}, 0)
		_, _ = h.karmaUpdate(agentID, "increment", "invoke_fail", 1)
		// Mistakes journal: cuma error NYATA (skip timeout/cancel = infra, bukan kesalahan agent).
		if !strings.Contains(emsg, "context canceled") && !strings.Contains(emsg, "context deadline") {
			h.recordMistake(agentID, "invoke_error", "invoke gagal", clip(emsg, 1000))
		}
		return
	}
	_ = h.logInteraction(agentID, "invoke", "out", agentID, clip(reply, 8000), nil)
	_, _ = h.logDecision(agentID, "agent_invoke", "invoke ok", "ok", map[string]any{"caller": caller, "reply_len": len(reply)}, 0)
	_, _ = h.karmaUpdate(agentID, "increment", "invoke_success", 1)
}

// SharedDirForAgent — return absolute path ke shared workspace per agent
// (`<SharedDir>/<agentID>/`). Buat dispatcher tool ops yang butuh fs access.
//
// Self-healing: the per-agent workspace is created on demand here. The boot-time
// ensureAgentSharedSpace only fires for agents holding fs:shared (which mounts the
// shared root into the GUEST sandbox), but a loket-native agent reaches file/bash
// tools through the in-process bridge (host-side cwd = this dir) WITHOUT needing
// the cross-agent mount. Creating it lazily lets any agent — including future
// copied "ant" agents — run fs/exec tools the first time without a manual mkdir.
func (h *Host) SharedDirForAgent(agentID string) (string, error) {
	if agentID == "" {
		return "", fmt.Errorf("agentID required")
	}
	dir := filepath.Join(h.SharedDir, agentID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("ensure workspace %q: %w", agentID, err)
	}
	return dir, nil
}

// OpenAgentStore — Section 18: convenience opener buat scheduler. Resolves
// agent folder dari h.lives + open agentdb.Store. Caller WAJIB Close().
func (h *Host) OpenAgentStore(agentID string) (*agentdb.Store, error) {
	if h == nil || agentID == "" {
		return nil, fmt.Errorf("nil host or empty agentID")
	}
	h.mu.Lock()
	var path string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == agentID {
			path = l.Discovery.Path
			break
		}
	}
	h.mu.Unlock()
	if path == "" {
		return nil, fmt.Errorf("agent %q not loaded", agentID)
	}
	dbPath := agentdb.Resolve(agentID, path)
	return agentdb.Open(dbPath)
}

// InvokeAgentMessage — Section 18 scheduler executor: call WASM agent
// handle_message RPC dengan task text. Return reply text or error.
func (h *Host) InvokeAgentMessage(ctx context.Context, agentID, text, caller string) (string, error) {
	// 300s: selaras manifest timeout_call_ms=300000. Worker taskflow (riset
	// multi-tool serialized) + synthesizer (file_read 5 file + generate panjang)
	// butuh >180s — synth crew 6-agent kena deadline di 180s (run 11 promo).
	// Cap, BUKAN wait tetap: agent cepet tetep balik cepet. Scheduler aman.
	return h.InvokeAgentMessageTimeout(ctx, agentID, text, caller, 300*time.Second)
}

// InvokeAgentMessageTimeout — sama spt InvokeAgentMessage tapi cap-nya CALLER yang
// tentuin. Dipake OPS-1 async-group: GROUP module orkestrasi SELURUH crew (debate/
// parallel) + synthesizer DI DALAM satu handle_message — jadi butuh budget jauh >300s
// (mis. 25 menit). Cap = ctx host (wazero hormati cancel), BUKAN hard-cap WASM.
func (h *Host) InvokeAgentMessageTimeout(ctx context.Context, agentID, text, caller string, timeout time.Duration) (string, error) {
	if h == nil || h.Runtime == nil {
		return "", fmt.Errorf("nil host/runtime")
	}
	inst := h.Runtime.Get(agentID)
	if inst == nil {
		return "", fmt.Errorf("agent %q not loaded", agentID)
	}
	args := map[string]any{
		"text": text,
		"user": caller,
	}
	bodyJSON, _ := json.Marshal(args)
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	respBytes, err := inst.Call(callCtx, "handle_message", bodyJSON)
	if err != nil {
		h.recordInvokeSelfKnowledge(agentID, caller, text, "", err) // self-awareness choke-point
		return "", err
	}
	var out struct {
		Reply string `json:"reply"`
		Error string `json:"error"`
	}
	if jerr := json.Unmarshal(respBytes, &out); jerr != nil {
		reply := string(respBytes)
		h.recordInvokeSelfKnowledge(agentID, caller, text, reply, nil)
		return reply, nil
	}
	if out.Error != "" {
		aerr := fmt.Errorf("%s", out.Error)
		h.recordInvokeSelfKnowledge(agentID, caller, text, "", aerr)
		return "", aerr
	}
	h.recordInvokeSelfKnowledge(agentID, caller, text, out.Reply, nil)
	return out.Reply, nil
}

// AgentIDs — Section 16 phase 2: enumerate loaded agent IDs (snapshot,
// thread-safe). Buat caller (main.go) iterate untuk multi-warga commands
// loading + watcher setup.
//
// Dedupe: kernel kadang scan multiple roots (Documents/agents +
// .flowork/agents) yang punya same id — rejected sebagai "plugin already
// loaded" tapi LiveEntry tetap di-append. Caller cuma butuh distinct id.
func (h *Host) AgentIDs() []string {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	seen := map[string]bool{}
	out := make([]string, 0, len(h.lives))
	for _, l := range h.lives {
		if l.Discovery.Manifest == nil {
			continue
		}
		id := l.Discovery.Manifest.ID
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// CapsCheckerForAgent — Section 12: return closure bound ke broker
// IsApproved untuk agent tertentu. Sandbox (tools.SandboxRun) pakai
// untuk capability gate. Return nil kalau Broker ngga di-set (default-allow).
func (h *Host) CapsCheckerForAgent(agentID string) func(capability string) bool {
	if h == nil || h.Broker == nil || agentID == "" {
		return nil
	}
	return func(capability string) bool {
		return h.Broker.IsApproved(agentID, capability)
	}
}

// PromoteReport — outcome RunPromoteForAgent. Agregat hasil submit ke
// router. Section 7 phase 1.
type PromoteReport struct {
	StartedAt       string   `json:"started_at"`
	FinishedAt      string   `json:"finished_at"`
	Eligible        int      `json:"eligible"`        // mistakes lokal yang qualify
	Submitted       int      `json:"submitted"`       // sukses POST + sukses mark promoted lokal
	UpsertExisting  int      `json:"upsert_existing"` // router return added=false (audit fix N1 typo)
	Failed          int      `json:"failed"`
	LocalMarkFailed int      `json:"local_mark_failed"` // router OK tapi SetMistakePromoted gagal — next sweep akan re-submit (audit fix C2)
	Errors          []string `json:"errors,omitempty"`
}

// RunPromoteForAgent — Section 7: list mistakes lokal warga yang eligible
// (tier='raw' + hit_count ≥ 3 + belum promoted), submit ke Router brain
// global, lalu mark `tier='promoted'` di mistakes_local. Caller: admin
// endpoint POST /api/agents/promote/run?id= atau cron loop.
//
// Min hit_count default 3 (sama dengan Router validation di SubmitMistake).
// Open store on-demand (consistent pattern dengan RetentionSweep).
func (h *Host) RunPromoteForAgent(agentID string) (PromoteReport, error) {
	rep := PromoteReport{StartedAt: time.Now().UTC().Format(time.RFC3339)}

	h.mu.Lock()
	var agentPath string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == agentID {
			agentPath = l.Discovery.Path
			break
		}
	}
	h.mu.Unlock()
	if agentPath == "" {
		return rep, fmt.Errorf("agent not loaded: %s", agentID)
	}

	dbPath := agentdb.Resolve(agentID, agentPath)
	store, err := agentdb.Open(dbPath)
	if err != nil {
		return rep, fmt.Errorf("open state.db: %w", err)
	}
	defer store.Close()

	eligible, err := store.ListMistakesEligibleForPromote(3, 50)
	if err != nil {
		rep.Errors = append(rep.Errors, "list eligible: "+err.Error())
		rep.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return rep, nil
	}
	rep.Eligible = len(eligible)
	if len(eligible) == 0 {
		rep.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return rep, nil
	}

	// Router URL: ambil dari agent kv config kalau ada, else default.
	routerURL := routerclient.DefaultRouterURL
	if cfg, lerr := store.Load(); lerr == nil {
		if v, ok := cfg["router_url"].(string); ok && v != "" {
			routerURL = v
		}
	}
	client := routerclient.New(routerURL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// maxErrorMsgs — cap rep.Errors supaya respons ngga bloat kalau
	// 50 mistake semua failed (50 × 200-char msg = 10KB). Audit fix #11.
	const maxErrorMsgs = 10
	appendErr := func(msg string) {
		if len(rep.Errors) < maxErrorMsgs {
			rep.Errors = append(rep.Errors, msg)
		}
	}

	for _, m := range eligible {
		resp, serr := client.SubmitMistake(ctx, routerclient.SubmitMistakeReq{
			AgentID:  agentID,
			Category: m.Category,
			Title:    m.Title,
			Content:  m.Content,
			HitCount: m.HitCount,
		})
		if serr != nil {
			rep.Failed++
			appendErr(fmt.Sprintf("submit id=%d: %v", m.ID, serr))
			continue
		}
		// Audit fix C3: router return resp.ID = 0 = signal "ngga tersimpan"
		// padahal HTTP OK. JANGAN mark promoted lokal (else stale state).
		if resp.ID <= 0 {
			rep.Failed++
			appendErr(fmt.Sprintf("submit id=%d: router returned invalid id", m.ID))
			continue
		}
		// Mark promoted di lokal. Audit fix C2: kalau SetMistakePromoted
		// gagal, classify as LocalMarkFailed (BUKAN Submitted) supaya
		// caller tau lokal stale → next sweep akan re-submit (router
		// atomic UPSERT handle dup) tapi minor hit_count inflation.
		if perr := store.SetMistakePromoted(m.ID, resp.ID); perr != nil {
			appendErr(fmt.Sprintf("set promoted id=%d: %v", m.ID, perr))
			rep.LocalMarkFailed++
			continue
		}
		rep.Submitted++
		if !resp.Added {
			rep.UpsertExisting++
		}
	}
	rep.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	return rep, nil
}

// RebuildWorkspaceMetaForAgent — Section 6: scan agent shared workspace
// folder (`<SharedDir>/<agentID>/`) + register file ke tabel workspace_meta.
// Caller: admin endpoint POST /api/agents/workspace-meta?id=&action=rebuild.
//
// Hold h.mu sebentar buat resolve dbPath, lalu release sebelum heavy
// scan (RebuildIndexFromDir bisa take seconds untuk folder besar — ngga
// monopoli h.mu yang share dengan logInteraction/logDecision/karma).
func (h *Host) RebuildWorkspaceMetaForAgent(agentID string) (agentdb.RebuildIndexReport, error) {
	h.mu.Lock()
	var agentPath string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == agentID {
			agentPath = l.Discovery.Path
			break
		}
	}
	h.mu.Unlock()

	if agentPath == "" {
		return agentdb.RebuildIndexReport{}, fmt.Errorf("agent not loaded: %s", agentID)
	}
	dbPath := agentdb.Resolve(agentID, agentPath)
	store, err := agentdb.Open(dbPath)
	if err != nil {
		return agentdb.RebuildIndexReport{}, fmt.Errorf("open state.db: %w", err)
	}
	defer store.Close()

	// Shared workspace path: <SharedDir>/<agentID>/
	workspaceRoot := filepath.Join(h.SharedDir, agentID)
	return store.RebuildIndexFromDir(workspaceRoot), nil
}

// SlashDispatcherFunc — set di main.go untuk inject actual slashcmd
// dispatcher. Signature: (ctx, pluginID, text, caller) → (resultText,
// cmdName, error). Ctx pre-populated dengan store/agent/caller — caller
// di main.go invoke slashcmd.Dispatch(ctx, text).
//
// Section 15: pass ctx supaya slash commands bisa akses Store via
// slashcmd.FromStore — productive commands (/stats /tools etc) butuh DB.
//
// Nil-safe: kalau ngga di-set, dispatchSlash return error "slash not
// wired". Callback in-process — Section 15 ctx passing supersedes
// previous anti-circular note (slashcmd no-longer depends on kernel).
var SlashDispatcherFunc func(ctx context.Context, pluginID, text, caller string) (string, string, error)

// dispatchSlash — implementasi runtime.SlashDispatcher. Resolve via
// callback `SlashDispatcherFunc`. Plus per-agent log invocation via
// store.LogSlashInvocation. Roadmap section 17.
func (h *Host) dispatchSlash(pluginID, text, caller string) (string, string, error) {
	if pluginID == "" {
		return "", "", fmt.Errorf("pluginID required")
	}
	if SlashDispatcherFunc == nil {
		return "", "", fmt.Errorf("slash dispatcher not wired")
	}
	// Resolve agent path supaya bisa log invocation per-warga.
	h.mu.Lock()
	var agentPath string
	for _, l := range h.lives {
		if l.Discovery.Manifest != nil && l.Discovery.Manifest.ID == pluginID {
			agentPath = l.Discovery.Path
			break
		}
	}
	h.mu.Unlock()
	if agentPath == "" {
		return "", "", fmt.Errorf("agent not loaded: %s", pluginID)
	}

	// Section 15: open store upfront supaya bisa pass via ctx ke slash
	// commands (slashcmd.FromStore extract). Reuse for log invocation
	// post-dispatch. Best-effort — kalau open gagal, dispatch tanpa store
	// (commands butuh store akan return error gracefully).
	dbPath := agentdb.Resolve(pluginID, agentPath)
	store, oerr := agentdb.Open(dbPath)

	ctx := context.Background()
	if oerr == nil && store != nil {
		ctx = slashcmd.WithStore(ctx, store)
	}
	ctx = slashcmd.WithCaller(ctx, caller)
	ctx = slashcmd.WithAgent(ctx, pluginID)

	// Dispatch + capture timing.
	t0 := time.Now()
	resultText, cmdName, dErr := SlashDispatcherFunc(ctx, pluginID, text, caller)
	durationMs := time.Since(t0).Milliseconds()

	// Best-effort log to per-agent state.db.
	if oerr == nil && store != nil {
		args := ""
		if idx := strings.IndexAny(text, " \t"); idx >= 0 {
			args = strings.TrimSpace(text[idx+1:])
		}
		errText := ""
		if dErr != nil {
			errText = dErr.Error()
		}
		_, _ = store.LogSlashInvocation(cmdName, args, caller, resultText, errText, durationMs)
		_ = store.Close()
	}

	return resultText, cmdName, dErr
}

// Close release semua resource.
func (h *Host) Close(ctx context.Context) error {
	// Tutup semua cached *Store (fix bug.md #3) sebelum runtime shutdown.
	h.storeCache.Range(func(_, v any) bool {
		if s, ok := v.(*agentdb.Store); ok {
			_ = s.Close()
		}
		return true
	})
	return h.Runtime.Close(ctx)
}
