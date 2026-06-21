// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// 2026-06-22 OWNER-APPROVED buka-lock (B4 auto-sync sumber→graph, re-locked): +1 baris
//   ticker → SyncSourcesToGraph(ctx,host): projeksi skills/constitution/edu/drawer mr-flow
//   ke cognitive graph (job non-beku graph_autosync.go), IDEMPOTENT + change-detection
//   (skip embed kalau label SAMA → hemat router) + throttle 30min. Ganti re-run manual
//   _scratch_cgm/graphsync. Additive + data-only (sumber ga diubah; graph cuma mirror). Re-locked.
// 2026-06-21 OWNER-APPROVED buka-lock (D32 inc-1 error-edukasi-adaptif, re-locked): +1
//   baris ticker → PromoteRecurringMistakes(ctx,host): mistake recurring (hit_count>=3)
//   → project jadi recovery-instinct embedded (semantic-recallable) via job non-beku
//   mistake_promote_job.go. Tutup gap: gate promote ga ke-wire + recall mistakes LIKE
//   (bukan semantik). Kernel ga disentuh; reuse ListMistakesEligibleForPromote+UpsertNode.
//   Additive + dormant (tanpa mistake hit>=3 = no-op). Re-locked.
// 2026-06-21 OWNER-APPROVED buka-lock (Phase 6/E async opsi-1, re-locked): +1 baris di
//   ticker poller (deket RunDueWakeups) → RunQueuedTasks(ctx,host) drive task background
//   (state 'queued', di-set TaskCreate background:true) ASYNC via worker non-beku
//   task_worker.go. Kernel TIDAK disentuh; mirror pola wakeup. Additive + DORMANT (tanpa
//   task queued = no-op). Re-locked.
// 2026-06-21 OWNER-APPROVED (Phase 3E/D13 loop-belajar): +route POST /api/agents/learning/digest
//   (agentmgr.LearningDigestHandler) + AUTO-TRIGGER ticker 30 menit (goroutine, per-tick recover)
//   → distil recording model kuat (capture real-time di ROUTER, toggle GUI) → shadow → promote-on-
//   repetisi. Saklar = toggle capture GUI router (env DIHAPUS, owner "kebenaran di GUI"). Logic di
//   agentmgr/learning_feed.go + agentdb/learning_log.go (LOCKED). Additive (route + ticker).
// 2026-06-21 OWNER-APPROVED (AI-IN-AGENT mandate G1): +seedDreamDigester() & wire
//   agentmgr.DigestLLMOverride = dreamDigestLLM(host) (deket seedCodemapEnricher/AIStudio)
//   → reasoning CGM digest/extraction pindah dari host+model-global ke AGENT dream-digester
//   (model GUI). Additive, 2 baris. Re-locked.
// 2026-06-20 OWNER-APPROVED: DB-DRIVEN self-heal groups ("pake db biar flexibel").
//   +reconcileDeadCrews(fdb, agentsDir) di boot (abis seedSocialDefaults): hapus
//   task_category yg SEMUA member agent-nya udah ga ada (dir-existence, aman dari race
//   load) → DeleteCategory (cascade task_agents + trigger_rules). Realisasi "hapus
//   agent → crew auto-ilang". Anti-halu-crew-mati lapis arsitektur. Re-locked.
// 2026-06-20 OWNER-APPROVED (autonomy grant, not hash-frozen): wire CGM digestion —
//   (1) route POST /api/agents/cognitive/digest (manual trigger, owner-controlled),
//   (2) Tier-2 deep digest hook in the existing 12h dream cron (gated by env
//   FLOWORK_CGM_AUTODIGEST=1, default OFF). Deploy of P1 proven loop. Additive; logic
//   in non-locked agentmgr/cognitive_digest_cron.go. No new exec surface, manifest unchanged.
// 2026-06-20 OWNER-APPROVED (autonomy grant): boot agent-provisioning loop →
//   satu pintu agentmgr.ProvisionAgentDNA(id) (gantiin seeding inline; SAMA
//   dipanggil di plugin_handler.go install) → agent baru hasil AI Studio lahir
//   warga penuh (konstitusi+graph schema) tanpa nunggu restart. Additive.
// Locked at: 2026-05-30 (re-locked 2026-06-11; +group seed & secret-central 2026-06-12)
// 2026-06-12 OWNER-APPROVED: boot migrates connector secrets to Settings → API Keys
//   (connections.MigrateSchemaSecretsToGlobal) and wires the frozen env-forward hook
//   (kernelhost.EnvForwardKeys = connections.GlobalSecretEnvKeys) so connector tokens
//   live ONLY in Settings yet still reach their connector via env.
// 2026-06-12 OWNER-APPROVED: boot calls groupsAPI.SeedFromJSON() to reconcile each
//   group's roster with its committed group.json mirror (export configured groups,
//   restore them on a fresh install). Additive — no-op for plain agents.
// 2026-06-13 OWNER-APPROVED (unfreeze→review→refreeze, autonomous): apps now load from the data
//   home (~/.flowork/apps) instead of the hardcoded "apps" CWD path — Pilar 6 portability fix so
//   the 2 apps (flowalpha, notepad) appear whether run from a project, the portable cache, or the
//   appliance. Path-resolution change ONLY: no new exec surface (apps still need install/exec
//   consent), same data-home trust level as agents; not a hash-frozen file (manifest unchanged).
// 2026-06-11 OWNER-APPROVED: inject saved Settings → API Keys + Default Router/Model
//   (FLOWORK_LLM_MODEL, ROUTER_DEFAULT_URL from KV) into the process env BEFORE
//   kernelhost.Boot, so agents see them at load time (fixes the boot-order defect
//   where keys set after Boot never reached agents). Mounts /api/settings/router-default.
//   UPPER_SNAKE + IsSensitiveEnvKey filter preserved — no new env surface.
// Reason: Single-binary bootstrap. Audit pass — flag.Parse addr, embed.FS
//   static, SIGINT/SIGTERM signal handling, srv timeout (ReadHeader 15s,
//   Idle 120s), graceful shutdown 5s, scheduler/walletalert/watchdog
//   engines wired dengan ctx propagation, mock API fallback handler.
//   Owner-mode auth stub OK untuk single-user.
//
// flowork-gui — Flowork microkernel + control panel, single binary.
//
// Sebelumnya GUI lama (:1987) proxy ke kernel terpisah (:1988). Sekarang
// kernel embedded via internal/kernelhost — satu port saja: :1987.
//
// Konsep: tiap agent punya tombol Setting yang buka popup (router / prompt
// / tools / schedule) — state warga terisolasi di state.db (agentdb). DI LUAR
// itu ada halaman Settings GLOBAL (owner-level) + DB global flowork.db
// (floworkdb): auth single-owner, API key, wallet personal. Warga TIDAK
// menyimpan apa pun di flowork.db.
//
// Packages aktif:
//
//	internal/httpx       JSON writer + no-cache middleware
//	internal/kernel      wazero runtime + capability broker + loader (WASI plugin)
//	internal/kernelhost  embed kernel + HTTP handlers (/api/kernel/*)
//	internal/agentmgr    /api/agents/* (upload/download/remove/config)

package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/agentmgr"
	fwapps "flowork-gui/internal/apps"
	"flowork-gui/internal/codescan"
	"flowork-gui/internal/connections"
	"flowork-gui/internal/floworkauth"
	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/guardian"
	"flowork-gui/internal/httpx"
	"flowork-gui/internal/kernel/loader"
	"flowork-gui/internal/kernelhost"
	"flowork-gui/internal/loket"
	"flowork-gui/internal/marketdata"
	"flowork-gui/internal/mcphub"
	"flowork-gui/internal/scanapi"
	"flowork-gui/internal/scheduler"
	"flowork-gui/internal/settingsapi"
	"flowork-gui/internal/sidecar"
	"flowork-gui/internal/slashcmd"
	slashbuiltins "flowork-gui/internal/slashcmd/builtins"
	slashcustom "flowork-gui/internal/slashcmd/custom"
	"flowork-gui/internal/tools"
	"flowork-gui/internal/tools/builtins"
	"flowork-gui/internal/triggers"
	"flowork-gui/internal/watchdog"
)

//go:embed web
var webFS embed.FS

const version = "2.6.0"

func main() {
	addr := flag.String("addr", "127.0.0.1:1987", "listen address")
	armFlag := flag.Bool("arm", false, "Guardian: rekam baseline integritas + aktifkan, lalu keluar")
	disarmFlag := flag.Bool("disarm", false, "Guardian: matikan (buat update kernel/binary yang disengaja), lalu keluar")
	flag.Parse()

	// Guardian CLI (FASE 1): arm/disarm dari shell lokal (sudah trusted). Jalan lalu exit.
	if *armFlag {
		now := time.Now().UTC().Format(time.RFC3339)
		v, err := guardian.Arm(guardian.CoreFilesFromManifest(), now, true)
		if err != nil {
			log.Fatalf("guardian arm: %v", err)
		}
		if v.Sealed {
			log.Printf("guardian ARMED + OS-SEALED (%s) — %d artefak immutable (binary+manifest+vault) + deteksi kernel. Disarm dulu sebelum update.", v.SealMethod, len(v.Baseline))
		} else {
			log.Printf("guardian ARMED (detection-only) — %d artefak dijaga via hash. Seal OS gagal/no-root; jalankan `sudo flowork --arm` buat immutability penuh.", len(v.Baseline))
		}
		return
	}
	if *disarmFlag {
		if err := guardian.Disarm(); err != nil {
			log.Fatalf("guardian disarm: %v", err)
		}
		log.Printf("guardian DISARMED — verifikasi integritas mati. Re-arm (`--arm`) setelah update.")
		return
	}

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("embed sub: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Section 11 phase 1a-1f: register 11 builtin tools (echo, now,
	// memory_x3, file_x3, brain_search, telegram_send, webfetch).
	// Section 14 phase 1: register 3 builtin slash commands (help, echo, ping).
	// Both Init panic on duplicate name — early bug catch.
	builtins.Init()
	slashbuiltins.Init()
	// A2 isolation: per-process loopback secret. The kernel injects this + the
	// calling agent's VERIFIED id into self-API requests, so the tools/run
	// handler can bind execution to the real caller — one agent can no longer run
	// tools under another agent's id via ?id=. Set before any agent boots.
	if sb := make([]byte, 24); true {
		if _, rerr := rand.Read(sb); rerr == nil {
			_ = os.Setenv("FLOWORK_LOOPBACK_SECRET", hex.EncodeToString(sb))
		}
	}
	// Section 12 phase 2: register 3 built-in interceptor (workspace-path,
	// sensitive-file, persona-inject). SandboxRunV2 di agentmgr panggil chain
	// sebelum 3-gate sandbox.
	tools.InitDefaultInterceptors()

	// Section 17 phase 3: register pre-/post-hook framework + decisions log
	// + rate limit. DispatchWithHooks wraps locked Dispatch.
	slashcmd.InitHooks()

	// Section 17/15: wire slash dispatcher callback. Kernelhost pre-populate
	// ctx (Store/Caller/Agent) sebelum invoke ini supaya productive commands
	// (/stats /tools dst.) bisa akses store via slashcmd.FromStore.
	kernelhost.SlashDispatcherFunc = func(ctx context.Context, pluginID, text, caller string) (string, string, error) {
		result, cmdName, err := slashcmd.DispatchWithHooks(ctx, text)
		return result.Text, cmdName, err
	}

	// Centralize connector secrets: relocate any token still living in a per-agent
	// store into Settings → API Keys (global) BEFORE the env injection below reads
	// it, then register the frozen env-forward hook so each connector's token (and
	// any future connector's) reaches it via env — with no further frozen edit.
	if n := connections.MigrateSchemaSecretsToGlobal(); n > 0 {
		log.Printf("connections: centralized %d connector secret(s) into Settings → API Keys", n)
	}
	kernelhost.EnvForwardKeys = connections.GlobalSecretEnvKeys

	// Backfill each agent's GUI-editable prompt (kv.prompt) from its prompt.md when
	// unset, so the persona shows up + is editable in the GUI and reaches the wasm
	// via FLOWORK_AGENT_CONFIG.prompt. Non-destructive (never overwrites a GUI edit).
	if n := agentmgr.SeedPromptsFromMd(loader.AgentsDir()); n > 0 {
		log.Printf("agents: seeded %d GUI prompt(s) from prompt.md", n)
	}

	// Inject saved Settings → API Keys into the process env BEFORE agents boot. The kernel
	// builds each agent's env (buildAgentEnv) at load time from os.Getenv, so the keys MUST be
	// set first — otherwise agents boot before the keys exist and never see them (the old order
	// injected them after kernelhost.Boot, an architectural defect). floworkdb.Shared() is an
	// idempotent singleton, safe to open here. UPPER_SNAKE only; reserved/sensitive names skipped.
	if fdbEarly, ferr := floworkdb.Shared(); ferr == nil {
		// Migrasi sekali: enkripsi secret plaintext lama (privkey/API key) at-rest (owner
		// 2026-06-20 "pindahin saja"). Idempotent — yg udah enc + password-hash di-skip.
		if n, merr := fdbEarly.MigrateSecretsEncrypt(); merr == nil && n > 0 {
			log.Printf("secrets: %d kredensial plaintext di-enkripsi at-rest", n)
		}
		if secrets, serr := fdbEarly.AllSecrets(); serr == nil {
			for k, v := range secrets {
				if k == strings.ToUpper(k) && strings.TrimSpace(v) != "" && !settingsapi.IsSensitiveEnvKey(k) {
					_ = os.Setenv(k, v)
				}
			}
		}
		// Settings → Default Router (generic KV not secrets). Model TIDAK lagi di-mirror
		// ke env (owner 2026-06-20: "kebenaran di GUI bukan env") — coderModel/loket/cgm baca
		// floworkdb.DefaultModelShared() langsung. Router URL masih perlu env (routerclient pakai).
		if u, _ := fdbEarly.GetKV("router_default_url"); strings.TrimSpace(u) != "" {
			_ = os.Setenv("ROUTER_DEFAULT_URL", strings.TrimSpace(u))
		}
	}

	host, err := kernelhost.Boot(ctx)
	if err != nil {
		log.Fatalf("kernel boot: %v", err)
	}
	defer host.Close(context.Background())

	// Section 16 phase 2: multi-warga + hot-reload fsnotify.
	// Enumerate semua agent → resolve <sharedDir>/<agentID>/commands/ → load
	// all + start fsnotify watcher debounce 500ms.
	commandsDirs := []string{}
	for _, agentID := range host.AgentIDs() {
		if shared, derr := host.SharedDirForAgent(agentID); derr == nil && shared != "" {
			commandsDirs = append(commandsDirs, filepath.Join(shared, "commands"))
		}
	}
	if loaded, skipped, lerr := slashcustom.LoadFromDirs(commandsDirs); lerr != nil {
		log.Printf("custom slash load: %v", lerr)
	} else if loaded > 0 || skipped > 0 {
		log.Printf("custom slash: loaded=%d skipped=%d across %d dirs", loaded, skipped, len(commandsDirs))
	}
	if werr := slashcustom.StartWatcher(ctx, commandsDirs); werr != nil {
		log.Printf("custom slash watcher: %v (hot-reload disabled)", werr)
	}

	// Wire ConfigHandler → kernel reload callback. Tanpa ini, save config
	// dari popup ngga restart daemon → env baru ngga kebawa.
	agentmgr.AgentIDsFunc = host.AgentIDs
	agentmgr.Reload = host.ReloadAgent
	// agent_command delegation tool: let router agent (Mr.Flow) invoke a
	// specialist agent and relay the reply (power control is direct now, not delegated).
	builtins.InvokeAgentFunc = host.InvokeAgentMessage
	agentmgr.RetentionSweep = func(agentID string) (any, error) {
		return host.RunRetentionForAgent(agentID)
	}
	agentmgr.WorkspaceRebuildIndex = func(agentID string) (any, error) {
		return host.RebuildWorkspaceMetaForAgent(agentID)
	}
	agentmgr.PromoteRun = func(agentID string) (any, error) {
		return host.RunPromoteForAgent(agentID)
	}
	agentmgr.SharedDirForAgent = func(agentID string) (string, error) {
		return host.SharedDirForAgent(agentID)
	}
	agentmgr.CapsCheckerForAgent = func(agentID string) func(capability string) bool {
		return host.CapsCheckerForAgent(agentID)
	}

	// Section 18 phase 1: scheduler engine. Tick 60s align ke top-of-minute.
	// Executor: kalau task mulai `/` → slash dispatch; selain itu → RPC
	// handle_message ke agent WASM (sama path Telegram).
	schedEngine := scheduler.New(
		host.AgentIDs,
		func(agentID string) (*agentdb.Store, error) {
			return host.OpenAgentStore(agentID)
		},
		func(ctx context.Context, agentID, scheduleID, task string) (string, error) {
			// Scheduler executor: forward task as user message ke agent
			// WASM. Agent's doHandle akan deteksi leading `/` dan route ke
			// slash dispatcher (Section 17 phase 2 parity), atau LLM
			// kalau plain text.
			task = strings.TrimSpace(task)
			reply, runErr := host.InvokeAgentMessage(ctx, agentID, task, "scheduler")
			// Section 18 phase 2: decisions log + karma update sinkron.
			if store, oerr := host.OpenAgentStore(agentID); oerr == nil {
				defer store.Close()
				outcome := "success"
				rationale := "schedule fire: " + task
				karmaKey := "schedule_success_count"
				if runErr != nil {
					outcome = "fail"
					rationale = "schedule fire fail: " + runErr.Error()
					if len(rationale) > 512 {
						rationale = rationale[:512] + "…"
					}
					karmaKey = "schedule_fail_count"
				}
				inputs := map[string]any{
					"schedule_id": scheduleID,
					"task":        task,
					"agent_id":    agentID,
					"reply_len":   len(reply),
				}
				_, _ = store.LogDecision("schedule_fire", rationale, outcome, inputs, 0)
				_, _ = store.IncrementKarma(karmaKey, 1)
			}
			return reply, runErr
		},
	)
	schedEngine.Start(ctx)
	defer schedEngine.Stop()
	agentmgr.SchedulerFireFunc = func(agentID, scheduleID string) (int64, error) {
		return schedEngine.FireNow(ctx, agentID, scheduleID)
	}

	// Section 26 phase 2: watchdog cron evaluator. Tick 60s, default rules
	// (protector_burst ≥10/60s CRITICAL, scanner_critical_burst ≥5/1h HIGH,
	// tool_call_storm ≥100/60s WARNING). 1h cooldown anti-spam per rule.
	watchdogEngine := watchdog.New(
		host.AgentIDs,
		host.OpenAgentStore,
		func(ctx context.Context, agentID, channel, message string) error {
			log.Printf("[watchdog→%s] agent=%s %s", channel, agentID, message)
			return nil
		},
	)
	watchdogEngine.Start(ctx)
	defer watchdogEngine.Stop()
	agentmgr.WatchdogFireFunc = func() (int, int) {
		return watchdogEngine.FireNow(ctx)
	}

	host.AutoBootDaemons(ctx)
	if err := host.StartWatcher(ctx); err != nil {
		log.Printf("kernel watcher start failed: %v (hot-reload disabled)", err)
	}
	// Section 8: retention cron — sweep tiap 24h, hard-delete grace 90 hari.
	host.StartRetentionCron(ctx, 24*time.Hour)
	// FASE 8: Curator skill harian — consolidate dup + arsip stale semua agent.
	go func() {
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("skill-curator PANIC (ticker selamat): %v", r)
						}
					}()
					if rep := agentmgr.CurateAllAgentsSkills(host.AgentIDs()); len(rep) > 0 {
						log.Printf("skill-curator: %d agent dirapihin", len(rep))
					}
				}()
			}
		}
	}()

	// Roadmap 2 B3: Dream cron (shared-worker) — tiap 12 jam, konsolidasi pola
	// berulang tiap agent jadi eureka brain drawer. Compute 1× per tick, tulis
	// ke state.db lokal masing-masing (anti-boros). Per-tick recover (ticker selamat).
	go func() {
		t := time.NewTicker(12 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("dream PANIC (ticker selamat): %v", r)
						}
					}()
					formed := 0
					for _, agentID := range host.AgentIDs() {
						if store, derr := host.OpenAgentStore(agentID); derr == nil {
							if res, derr := store.RunDream(now); derr == nil && res.EurekasFormed > 0 {
								log.Printf("dream: %d eureka baru → %s", res.EurekasFormed, agentID)
								formed += res.EurekasFormed
							}
							// Roadmap 2 B5: immune sweep — quarantine drawer injection/halu.
							if q, derr := store.ScanAndQuarantine(); derr == nil && q > 0 {
								log.Printf("immune: %d drawer dikarantina → %s", q, agentID)
							}
							store.Close()
						}
					}
					if formed > 0 {
						log.Printf("dream: total %d eureka baru lintas agent", formed)
					}
					// CGM Tier-2 deep digestion (roadmap §4.6/D16) — percakapan → cognitive
					// graph. Gated by env FLOWORK_CGM_AUTODIGEST=1 (default OFF, owner opt-in).
					// Resilient: per-agent error di-skip (lihat agentmgr.DigestAllAgents).
					if dg := agentmgr.DigestAllAgents(host.AgentIDs()); dg > 0 {
						log.Printf("cgm: %d interactions dicerna ke cognitive graph (Tier-2)", dg)
					}
				}()
			}
		}
	}()

	// 3E/D13 LOOP-BELAJAR auto-trigger (owner 2026-06-21): capture REAL-TIME di router (toggle
	// GUI), distil-nya BATCH tiap 30 menit → recordings model kuat jadi shadow → promote on
	// repetisi. No-op kalau ga ada recording baru (anti-replay learning_record_log). Per-tick
	// recover (1 error ga nyeret yg lain). Saklar = toggle capture GUI (capture OFF → ga ada
	// recording baru → trigger no-op).
	go func() {
		t := time.NewTicker(30 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("3E loop-belajar PANIC (ticker selamat): %v", r)
						}
					}()
					if ls, err := agentmgr.DigestRecordings(100); err == nil && ls.Added > 0 {
						log.Printf("3E loop-belajar: %d node shadow baru (%d processed, %d promoted)", ls.Added, ls.Processed, ls.Promoted)
					}
				}()
			}
		}
	}()

	// AUTO-COMPACT konteks (owner 2026-06-20: "kalau konteks panjang, semua agent compact +
	// masukin pengalaman ke brain kayak dream; FATAL jika salah"). Beda dari dream 12h: trigger
	// by UKURAN konteks → tiap 15 menit cek; cuma agent yg LEWAT AMBANG di-digest→verify→trim
	// (anti-halu konteks panjang). Hemat (mayoritas cuma 1 query COUNT). Per-tick recover.
	go func() {
		t := time.NewTicker(15 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("auto-compact PANIC (ticker selamat): %v", r)
						}
					}()
					agentmgr.AutoCompactAllAgents(host.AgentIDs(), false)
				}()
			}
		}
	}()

	// DNA INTRINSIK tiap agent (idempotent, satu pintu — lihat agentmgr/
	// provision_dna.go): edu-errors + konstitusi sacred (incl anti-janji-palsu/
	// recall-first) + tune-extension + sync slot + antibody immune + ensure schema
	// cognitive graph. SAMA persis dipanggil di install path (plugin_handler.go) →
	// agent baru hasil AI Studio lahir warga penuh tanpa nunggu restart. Pipa ke
	// instinct/graph/otak-kolektif via coreExposedTools (referensi shared brain).
	for _, agentID := range host.AgentIDs() {
		agentmgr.ProvisionAgentDNA(agentID)
	}

	// Background code scanner — watch source repo + kode buatan AI; auto-scan
	// file yang berubah, deteksi bug/celah dari tiap update. Critical/high →
	// audit log + push Telegram ke owner. (Tampil di tab Scanner sbg "auto:*".)
	codescanRoot := strings.TrimSpace(os.Getenv("FLOWORK_CODESCAN_ROOT"))
	if codescanRoot == "" {
		codescanRoot = agentdb.ProjectRoot()
	}
	codescanEngine := codescan.New(
		host.AgentIDs, host.OpenAgentStore, host.SharedDirForAgent,
		func(nctx context.Context, title, body string) error {
			return notifyOwnerTelegram(nctx, title+"\n\n"+body)
		},
		codescanRoot,
	)
	codescanEngine.Start(ctx)

	// DB global Flowork (owner-level: auth, settings, API key, wallet personal).
	// TERPISAH dari state.db per-warga (agentdb) — warga tetap terisolasi.
	fdb, err := floworkdb.Shared()
	if err != nil {
		log.Fatalf("flowork.db open: %v", err)
	}
	// Plug-and-Play Phase 3: drop-folder auto-install (.fwpack di ~/.flowork/dropbox/).
	startPluginDropWatcher(host, fdb)
	// TOOL-PACK plug-and-play: re-register tool plugin dari marker tool.json (persist).
	if nt := reregisterToolPacksOnBoot(host); nt > 0 {
		log.Printf("kernel: re-registered %d tool-pack(s) from disk", nt)
	}
	if ns := reregisterSlashPacksOnBoot(host); ns > 0 {
		log.Printf("kernel: re-registered %d slash-pack(s) from disk", ns)
	}
	// Inject API key tersimpan → env, supaya engine (wallet, dll) langsung
	// pakai konfigurasi dari Settings tanpa hardcode/restart. Hanya key
	// UPPER_SNAKE (env-var) yang di-set; password hash (lowercase) di-skip.
	if secrets, serr := fdb.AllSecrets(); serr == nil {
		for k, v := range secrets {
			// Never re-inject a reserved env name (PATH/LD_*/FLOWORK_*/…) even if one
			// somehow got persisted — a loader/PATH hijack or a forged loopback secret
			// must not survive a restart. Same gate the POST handler enforces.
			if k == strings.ToUpper(k) && strings.TrimSpace(v) != "" && !settingsapi.IsSensitiveEnvKey(k) {
				_ = os.Setenv(k, v)
			}
		}
	}
	// Ensure the task tables exist. Categories are NOT hardcoded/seeded — they are
	// auto-registered when an agent/task pack is installed (plug-and-play). No agent
	// installed → no category, so AI Studio shows real state, not a phantom crew.
	if serr := fdb.EnsureTaskSchema(); serr != nil {
		log.Printf("taskflow schema: %v", serr)
	}
	// boot hygiene: run 'running' zombie dari proses lama (mati/restart) ditandai
	// 'interrupted' + KABARIN owner ke Telegram (dulu ilang diem-diem → bug "task
	// selesai tapi ga ada laporan"). notify_chat di-persist per-run.
	if orphans, oerr := fdb.MarkRunningInterrupted(); oerr != nil {
		log.Printf("taskflow boot sweep: %v", oerr)
	} else {
		for _, o := range orphans {
			if strings.TrimSpace(o.NotifyChat) != "" {
				notifyTelegram(host, o.NotifyChat, fmt.Sprintf(
					"⚠️ Task %s — \"%s\" (run #%d) ke-interrupt pas Flowork restart, hasilnya ga kelar. Kirim ulang ya bro.",
					o.CategoryID, o.InputText, o.ID))
			}
		}
	}
	// TRIGGER engine (ROADMAP 3): papan kosong event-driven (Schedule = tipe `time`).
	// Reuse: InvokeAgentMessage (aksi) + notifyOwnerTelegram (deliver). Hook ke tick di bawah.
	_ = fdb.EnsureTriggerSchema()
	go func() { _ = fdb.SweepTriggerKeys(30) }() // retensi ledger dedup
	trigEngine := &triggers.Engine{Store: fdb, Invoke: host.InvokeAgentMessage, Notify: notifyOwnerTelegram,
		// Aksi SISTEM buat trigger (owner 2026-06-20 "all compact ke triger"): target_kind=system.
		SystemAction: func(ctx context.Context, action string) (string, error) {
			switch action {
			case "compact-all":
				go agentmgr.AutoCompactAllAgents(host.AgentIDs(), true) // async (bisa lama, LLM)
				return "compact-all dijadwalkan (jalan di background)", nil
			default:
				return "", fmt.Errorf("system action ga dikenal: %s", action)
			}
		},
	}

	// APPS platform (ROADMAP 4): program dipakai MANUSIA (GUI) & AGENT (tool) di state yang SAMA,
	// core LINTAS BAHASA (runtime:process). Load apps/<id>/ → daftarkan operasi sbg tool agent.
	// Apps: laci DATA (installed user) + overlay CONTENT (bawaan shippable). Resolver
	// tunggal lewat paket sidecar (roadmap_sidecar Fase 0/1). Legacy-default:
	// sidecar.AppsDataDir() == ~/.flowork/apps pas FLOWORK_SIDECAR kosong → zero perubahan.
	appsMgr := fwapps.NewManager(sidecar.AppsDataDir()).WithContent(sidecar.AppsContentDir())
	fwapps.SetDefault(appsMgr) // target install/uninstall package-level (gerbang seragam + HTTP)
	if err := appsMgr.Load(); err != nil {
		log.Printf("apps load: %v", err)
	}
	defer appsMgr.Shutdown()

	// "GUI = KEBENARAN" (owner 2026-06-20): SETELAH SEMUA tool keregister (builtin
	// init() + tool-pack reregister + app appsMgr.Load di atas) — baru auto-grant
	// capability dari tool yg dicentang di GUI. Telat dikit ga apa: broker.Approve
	// nambah cap ke set yg dipakai tiap tool-call (bukan snapshot saat boot). Kalau
	// jalan kepagian (sebelum app/tool-pack register), tools.Lookup balik false →
	// app:flowalpha/mcp:* ga ke-grant (bug yg bikin app_flowalpha tetep ditolak).
	for _, agentID := range host.AgentIDs() {
		grantSubscribedToolCaps(host, agentID)
		seedAndApplyAppCaps(host, agentID) // izin-app per-agent (app_grants) → cap app:<id>
	}

	// SCHEDULER LOOPING: tiap menit cek jadwal task → fire Category Task otomatis +
	// notify Telegram (mis. tiap jam 9 pagi: analisa saham A → keputusan ke chat).
	go func() {
		t := time.NewTicker(1 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("task-scheduler PANIC (ticker selamat): %v", r)
						}
					}()
					if n := RunDueSchedules(host, fdb); n > 0 {
						log.Printf("task-scheduler: %d jadwal di-fire", n)
					}
					trigEngine.Tick(ctx)                      // ROADMAP 3: proses aturan trigger poll (time/file-watch/…)
					if n := RunDueWakeups(ctx, host); n > 0 { // ScheduleWakeup: fire one-shot wakeup yang due
						log.Printf("wakeup: %d di-fire", n)
					}
					if n := RunQueuedTasks(ctx, host); n > 0 { // E (Phase 6): drive task background (queued) async
						log.Printf("queued-task: %d di-drive", n)
					}
					if n := PromoteRecurringMistakes(ctx, host); n > 0 { // D32: mistake recurring → recovery-instinct (embedded)
						log.Printf("mistake-promote: %d → recovery-instinct", n)
					}
					if n := SyncSourcesToGraph(ctx, host); n > 0 { // B4: auto-sync sumber→graph (throttle 30min, change-detect)
						log.Printf("graph-autosync: %d node sumber→graph diperbarui", n)
					}
				}()
			}
		}
	}()
	authMgr := floworkauth.NewManager(fdb)
	settingsAPI := settingsapi.New(fdb)

	// Groups (§F2) — list/edit GROUP modules (roster + synthesizer + task) in their
	// own loket store, the same per-module path the kernel uses (isolation kept).
	groupsAPI := groupsapi.New(groupsapi.Deps{
		AgentIDs: host.AgentIDs,
		LoketStorePath: func(module string) (string, error) {
			staged := filepath.Join(loader.AgentsDir(), module+".fwagent")
			return filepath.Join(filepath.Dir(agentdb.Resolve(module, staged)), "loket.db"), nil
		},
		AgentsDir:     loader.AgentsDir(),
		GroupWasmPath: "templates/group-template/agent.wasm",
		Toggle:        agentmgr.ToggleAgent, // group on/off cascades to coordinator + members
		TelegramToken: func() string { // push slash menu Telegram = group LIVE (fix nyangkut group lama)
			if fdbT, err := floworkdb.Shared(); err == nil {
				v, _ := fdbT.GetSecret("TELEGRAM_BOT_TOKEN")
				return v
			}
			return ""
		},
	})
	// Keep each group's roster in sync with its committed group.json mirror: export
	// configured groups (so they can be committed) and restore them on a fresh
	// install. No-op for plain agents and already-synced groups.
	if restored, exported := groupsAPI.SeedFromJSON(); restored > 0 || exported > 0 {
		log.Printf("groups: %d roster(s) restored, %d mirrored to group.json", restored, exported)
	}
	// Plug-and-play: auto-discover every group and sync the list to the orchestrator
	// so the Telegram slash menu + Mr.Flow's ask_group cover ALL groups, no config.
	if n := groupsAPI.SyncToOrchestrator(); n > 0 {
		log.Printf("groups: synced %d group(s) to orchestrator (%s) for slash + ask_group", n, groupsapi.OrchestratorID)
	}
	// Portable defaults: a FRESH clone gets the SAME social schedule + public Telegram
	// invite as the owner (tokens stay in Settings, never pushed). No-op on a machine
	// that already has a schedule / config.
	seedSocialDefaults(fdb, loader.AgentsDir())
	// SELF-EVOLUTION → GROUP + 5 AGENT (owner 2026-06-20 "pindahin otaknya"): bikin dewan
	// adversarial (pembela/penantang/3 hakim) jadi 5 agent persona-DB + grup self-evolution
	// → mr-flow bisa jalanin. Idempoten. GATE KEAMANAN (mode/karma/ModelStrong) TETAP di
	// harness (selfevolve.go), GA disentuh. Cuma mindahin otak.
	seedSelfEvolutionGroup(groupsAPI)
	seedCodemapEnricher() // agent enrich codemap (model GUI, ga hardcode — owner 2026-06-20)
	seedAIStudio()        // agent ai-studio: otak AI Studio (bikin agent/app/tim), model GUI — owner 2026-06-20
	seedDreamDigester()   // agent dream-digester: otak CGM digest/extraction, model GUI — owner 2026-06-21 (AI-IN-AGENT)
	agentmgr.DigestLLMOverride = dreamDigestLLM(host) // digest reasoning lewat agent (model GUI), bukan global
	seedAppJudge()        // G5: agent app-judge (verifier adversarial), model GUI haiku — owner 2026-06-21 (AI-IN-AGENT)
	seedScanDistiller()   // G6: agent scan-distiller (generator check privat), model GUI haiku — owner 2026-06-21
	scanapi.DistillModelResolver = func() string { return utilityAgentModel(scanDistillerID) } // distiller model dari agent GUI
	// DB-DRIVEN SELF-HEAL (owner 2026-06-20 "pake db biar flexibel"): crew/category yg
	// SEMUA member agent-nya udah dihapus (dir ga ada) = mati → auto-clean (cascade
	// task_agents + trigger_rules). Realisasi "hapus agent → crew auto-ilang", lepas
	// dari jalur delete mana pun. Pakai dir-existence (BUKAN AgentIDs) → aman dari
	// race agent-belum-load (dir ada walau lagi loading = di-keep).
	reconcileDeadCrews(fdb, loader.AgentsDir())

	mux := http.NewServeMux()

	// Auth — single-owner password (floworkauth). Session cookie in-memory.
	mux.HandleFunc("/api/auth/me", authMgr.MeHandler)
	mux.HandleFunc("/api/auth/login", authMgr.LoginHandler)
	mux.HandleFunc("/api/auth/register", authMgr.RegisterHandler)
	mux.HandleFunc("/api/auth/logout", authMgr.LogoutHandler)
	mux.HandleFunc("/api/auth/change-password", authMgr.ChangePasswordHandler)
	mux.HandleFunc("/api/owner/auto-verify", ownerAutoVerify)
	mux.HandleFunc("/api/system/health", systemHealth)

	// Groups (§F2) — GUI tab "Group" reads/edits group rosters.
	mux.HandleFunc("/api/groups", groupsAPI.ListHandler)
	mux.HandleFunc("/api/groups/config", groupsAPI.ConfigHandler)
	mux.HandleFunc("/api/groups/create", groupsAPI.CreateHandler)
	mux.HandleFunc("/api/groups/delete", groupsAPI.DeleteHandler)
	mux.HandleFunc("/api/groups/toggle", groupsAPI.ToggleHandler) // group on/off (cascade to members)
	mux.HandleFunc("/api/groups/reset", groupsAPI.ResetHandler)   // restore bundled groups if deleted

	// Page routes — FileServer cuma map exact filename (/login.html), jadi
	// /login & /register butuh handler eksplisit yang serve embedded HTML.
	servePage := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			data, rerr := fs.ReadFile(staticFS, name)
			if rerr != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(data)
		}
	}
	mux.HandleFunc("/login", servePage("login.html"))
	mux.HandleFunc("/register", servePage("register.html"))

	// Settings (owner-level, flowork.db global).
	mux.HandleFunc("/api/settings/keys", settingsAPI.KeysHandler)
	mux.HandleFunc("/api/settings/router-default", settingsAPI.RouterDefaultHandler)
	mux.HandleFunc("/api/settings/notify", settingsAPI.NotifyHandler)
	settingsapi.TestNotifyFunc = notifyOwnerTelegram
	// Owner-notify HUB (§ FlowAlpha): a loopback-only endpoint so local sandboxed apps/scanner
	// can push an owner Telegram alert WITHOUT holding the token. Additive; reuses
	// notifyOwnerTelegram. Reachable without a session ONLY for a local non-browser caller
	// (isPublicPath gates it + cross-site browser drive-by is cut there).
	mux.HandleFunc("/api/notify", notifyHubHandler)
	// Settings → YouTube (owner-level OAuth via GUI, no .scratch)
	mux.HandleFunc("/api/settings/youtube", settingsAPI.YouTubeStatusHandler)
	mux.HandleFunc("/api/settings/youtube/credentials", settingsAPI.YouTubeCredentialsHandler)
	mux.HandleFunc("/api/settings/youtube/connect", settingsAPI.YouTubeConnectHandler)
	mux.HandleFunc("/api/settings/youtube/disconnect", settingsAPI.YouTubeDisconnectHandler)
	mux.HandleFunc("/api/settings/youtube/config", settingsAPI.YouTubeConfigHandler)

	// Kernel introspection (list agent, RPC call).
	mux.HandleFunc("/api/kernel/status", host.StatusHandler)
	mux.HandleFunc("/api/kernel/agents", host.AgentsHandler)
	mux.HandleFunc("/api/kernel/rpc", host.RPCHandler)
	mux.HandleFunc("/api/agents/ui-schema", host.UISchemaHandler)

	// "Papan kosong" microkernel — the single loket: ONE endpoint where a module
	// makes call(cap, args). ADDITIVE + non-breaking; runs beside the legacy
	// kernel. Loopback-only (caller id is kernel-stamped via the loopback secret).
	loketSvc := wireLoket(host)
	mux.HandleFunc("/api/kernel/call", loketSvc.CallHandler)
	mux.HandleFunc("/api/kernel/gui", loketSvc.GUIHandler)
	mux.HandleFunc("/api/kernel/webhook/", loketSvc.WebhookHandler)

	// Connections — universal connector registry (telegram/discord/email/cli/...).
	// Install reuses the .fwpack gerbang (kind:channel); these cover list/toggle/
	// config/uninstall. Each connector is self-contained in its own folder.
	mux.HandleFunc("/api/connections", connections.ListHandler)
	mux.HandleFunc("/api/connections/toggle", connections.ToggleHandler)
	mux.HandleFunc("/api/connections/config", connections.ConfigHandler)
	mux.HandleFunc("/api/connections/uninstall", connections.UninstallHandler)

	// MCP connectors (Jenis 2: external MCP servers as agent tool-sources).
	// Owner-gated. Installed connectors auto-start below.
	mux.HandleFunc("/api/mcp", mcphub.ListHandler)
	mux.HandleFunc("/api/mcp/install", mcphub.InstallHandler)
	mux.HandleFunc("/api/mcp/enable", mcphub.EnableHandler)
	mux.HandleFunc("/api/mcp/disable", mcphub.DisableHandler)
	mux.HandleFunc("/api/mcp/uninstall", mcphub.UninstallHandler)
	// Auto-start installed MCP connectors (best-effort) so their tools are registered
	// for agents right away. In a goroutine — a slow MCP server must not delay boot.
	go func() {
		ec, ecancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer ecancel()
		mcphub.Default.EnableAll(ec)
	}()

	// Agent manager (upload .fwagent.zip, config per agent).
	mux.HandleFunc("/api/agents/upload", agentmgr.UploadHandler)
	mux.HandleFunc("/api/agents/download", agentmgr.DownloadHandler)
	mux.HandleFunc("/api/agents/remove", agentmgr.RemoveHandler)
	mux.HandleFunc("/api/agents/config", agentmgr.ConfigHandler)
	mux.HandleFunc("/api/agents/mcp", agentmgr.AgentMCPHandler)    // per-agent MCP opt-out checklist
	mux.HandleFunc("/api/agents/duplicate", agentDuplicateHandler) // copy an agent (the "copas" button)
	mux.HandleFunc("/api/agents/toggle", agentmgr.ToggleHandler)
	mux.HandleFunc("/api/agents/db/reset", agentmgr.DBResetHandler)
	mux.HandleFunc("/api/agents/interactions", agentmgr.InteractionsHandler)
	mux.HandleFunc("/api/agents/decisions", agentmgr.DecisionsHandler)
	mux.HandleFunc("/api/agents/mistakes", agentmgr.MistakesHandler)
	mux.HandleFunc("/api/agents/retention/sweep", agentmgr.RetentionSweepHandler)
	mux.HandleFunc("/api/agents/death-letter", agentmgr.DeathLetterHandler)
	mux.HandleFunc("/api/agents/karma", agentmgr.KarmaHandler)
	mux.HandleFunc("/api/agents/workspace-meta", agentmgr.WorkspaceMetaHandler)
	mux.HandleFunc("/api/agents/promote/run", agentmgr.PromoteRunHandler)
	mux.HandleFunc("/api/agents/edu-errors", agentmgr.EduErrorsHandler)
	mux.HandleFunc("/api/agents/tools/registry", agentmgr.ToolRegistryHandler)
	mux.HandleFunc("/api/agents/tool-invocations", agentmgr.ToolInvocationsHandler)
	mux.HandleFunc("/api/agents/tools/run", agentmgr.ToolRunHandler)
	mux.HandleFunc("/api/agents/tools/specs", agentmgr.ToolSpecsHandler)
	mux.HandleFunc("/api/agents/slash/run", agentmgr.SlashRunHandler)
	mux.HandleFunc("/api/agents/slash/registry", agentmgr.SlashRegistryHandler)
	mux.HandleFunc("/api/agents/slash-invocations", agentmgr.SlashInvocationsHandler)
	mux.HandleFunc("/api/agents/router-skills/list", agentmgr.RouterSkillsListHandler)
	mux.HandleFunc("/api/agents/router-skills/get", agentmgr.RouterSkillsGetHandler)
	// Section 13 phase 2 — tool subscriptions + suggest.
	mux.HandleFunc("/api/agents/tools/catalog", agentmgr.ToolCatalogHandler)
	mux.HandleFunc("/api/agents/tools/my", agentmgr.ToolMyHandler)
	mux.HandleFunc("/api/agents/tools/subscribe", agentmgr.ToolSubscribeHandler)
	mux.HandleFunc("/api/agents/tools/unsubscribe", agentmgr.ToolUnsubscribeHandler)
	mux.HandleFunc("/api/agents/tools/suggest", agentmgr.ToolSuggestHandler)
	// FASE 4/5: Category Task — trigger + CRUD kategori/crew + run timeline.
	mux.HandleFunc("/api/taskflow/run", taskflowRunHandler(host, fdb))
	mux.HandleFunc("/api/taskflow/categories", taskflowCategoriesHandler(fdb))
	mux.HandleFunc("/api/taskflow/category", taskflowCategoryHandler(fdb))
	mux.HandleFunc("/api/taskflow/category/delete", taskflowCategoryDeleteHandler(fdb))
	mux.HandleFunc("/api/taskflow/runs", taskflowRunsHandler(fdb))
	mux.HandleFunc("/api/taskflow/run-detail", taskflowRunDetailHandler(fdb))
	// Plug-and-Play: install task pack (.fwpack) → extract agent + daftarin kategori+crew.
	mux.HandleFunc("/api/plugins/install", pluginInstallHandler(host, fdb))
	mux.HandleFunc("/api/plugins/uninstall", pluginUninstallHandler(fdb))
	mux.HandleFunc("/api/plugins/export", pluginExportHandler(fdb))
	mux.HandleFunc("/api/plugins/verify", pluginVerifyHandler()) // VERIFIER dry-run gate (loopback)
	// CODER (AI Utama 2.2): generate agent baru → verify → Approval Queue (owner-gated).
	mux.HandleFunc("/api/coder/generate", coderGenerateHandler(host))
	mux.HandleFunc("/api/coder/pending", coderPendingHandler())
	mux.HandleFunc("/api/coder/approve", coderApproveHandler(host, fdb, groupsAPI)) // P3: auto-group (agent WAJIB group)
	mux.HandleFunc("/api/coder/reject", coderRejectHandler())
	// ARCHITECT (group/team creator): bikin TIM utuh dari 1 prompt → design (Opus) →
	// generate tiap specialist+lead (mesin coder yg sama) → groupsapi.CreateGroup.
	// Hasil: group muncul di tab Group, chat via /api/chat agent=<group_id>, slash auto.
	mux.HandleFunc("/api/architect/build", architectBuildHandler(host, fdb, groupsAPI))
	// CHAT SESSIONS (ChatGPT-style tab in Group): persistent sessions + full-context
	// conversation with a group (talk to a team) or the Architect (brainstorm + build).
	if cerr := fdb.EnsureChatSchema(); cerr != nil {
		log.Printf("chat: EnsureChatSchema: %v", cerr)
	}
	mux.HandleFunc("/api/chat/sessions", chatSessionsHandler(fdb))
	mux.HandleFunc("/api/chat/sessions/rename", chatSessionRenameHandler(fdb))
	mux.HandleFunc("/api/chat/sessions/delete", chatSessionDeleteHandler(fdb))
	mux.HandleFunc("/api/chat/sessions/meta", chatSessionMetaHandler(fdb))
	mux.HandleFunc("/api/chat/sessions/messages", chatMessagesHandler(fdb))
	mux.HandleFunc("/api/chat/send", chatSendHandler(host, fdb, groupsAPI))
	// REAPER (AI Utama 2.4): apoptosis — surface app broken/failing → owner reap.
	mux.HandleFunc("/api/reaper/candidates", reaperCandidatesHandler(host, fdb))
	mux.HandleFunc("/api/reaper/reap", reaperReapHandler(fdb))
	// CHANNEL HTTP/CLI (roadmap Channels, langkah aman): transport ke-2 ke mr-flow
	// channel-agnostic core. Test-harness doktrin (jalur sama Telegram). Ga sentuh daemon live.
	mux.HandleFunc("/api/chat", chatHandler(host))
	// TOOL-PACK plug-and-play (multi-KIND): install/uninstall/list tool plugin.
	mux.HandleFunc("/api/tools/install", toolInstallHandler(host))
	mux.HandleFunc("/api/tools/uninstall", toolUninstallHandler())
	mux.HandleFunc("/api/tools/installed", toolInstalledHandler())
	// SLASH-PACK plug-and-play (multi-KIND): install/uninstall/list slash command.
	mux.HandleFunc("/api/slash/install", slashInstallHandler(host))
	mux.HandleFunc("/api/slash/uninstall", slashUninstallHandler())
	mux.HandleFunc("/api/slash/installed", slashInstalledHandler())
	// SCANNER allowlist (owner-editable gerbang, agent-locked) — defensif P1.
	_ = fdb.EnsureScanSchema()
	mux.HandleFunc("/api/scanner/allowlist", scanapi.ScannerAllowlistHandler(fdb))
	mux.HandleFunc("/api/scanner/allowlist/delete", scanapi.ScannerAllowlistDeleteHandler(fdb))
	mux.HandleFunc("/api/scanner/allowlist/check", scanapi.ScannerAllowlistCheckHandler(fdb))
	mux.HandleFunc("/api/scanner/run", scanapi.ScannerRunHandler(fdb, host.OpenAgentStore))   // GATED-EXEC + mirror ke Threat Radar
	mux.HandleFunc("/api/scanner/runs", scanapi.ScannerRunsHandler(fdb))                      // audit history
	mux.HandleFunc("/api/scanner/findings", scanapi.ScannerFindingsHandler(fdb))              // finding terstruktur (P2.2b)
	mux.HandleFunc("/api/scanner/findings/verify", scanapi.ScannerFindingVerifyHandler(fdb))  // owner konfirmasi (P1.4)
	mux.HandleFunc("/api/scanner/findings/triage", scanapi.ScannerTriageHandler(fdb))         // RAG triage 5jt (P1.3)
	mux.HandleFunc("/api/scanner/findings/push", scanapi.ScannerPushHandler(fdb))             // sync finding → tracker brain
	mux.HandleFunc("/api/scanner/trackers", scanapi.ScannerTrackersHandler())                 // dashboard laporan (immune+pentest)
	mux.HandleFunc("/api/scanner/registry", scanapi.ScannerRegistryHandler(fdb))              // KATALOG arsenal (auditor+trivy+nuclei pack)
	mux.HandleFunc("/api/scanner/registry/toggle", scanapi.ScannerRegistryToggleHandler(fdb)) // install/uninstall pack nuclei
	mux.HandleFunc("/api/scanner/checks/add", scanapi.ScannerCheckAddHandler())               // ingest check privat (distill 5jt/komunitas) + gerbang nuclei -validate
	mux.HandleFunc("/api/scanner/checks/delete", scanapi.ScannerCheckDeleteHandler())
	mux.HandleFunc("/api/scanner/distill", scanapi.ScannerDistillHandler())                      // GENERATOR: LLM baca 5jt → template → validate → ingest
	mux.HandleFunc("/api/scanner/distill/corpus", scanapi.ScannerDistillCorpusHandler())         // nyisir corpus (exploitdb) → ribuan check, dedup+resumable
	mux.HandleFunc("/api/scanner/bodyscan", scanapi.ScannerBodyScanHandler(host.OpenAgentStore)) // SCAN TUBUH FLOWORK: scan kode semua repo → tulis ke Threat Radar
	mux.HandleFunc("/api/scanner/efficacy", scanapi.ScannerEfficacyHandler())                    // LAPIS EFIKASI: saringan false-positive (run lawan target bersih → karantina)
	mux.HandleFunc("/api/scanner/packs/install", scanapi.ScannerPackInstallHandler())            // kind:scanner .fwpack plug-and-play
	mux.HandleFunc("/api/scanner/packs/uninstall", scanapi.ScannerPackUninstallHandler())
	mux.HandleFunc("/api/scanner/packs/installed", scanapi.ScannerPacksInstalledHandler())
	// MARKET DATA (investment team "eyes"): read-through Yahoo proxy (cookie+crumb
	// server-side) so WASM analyst agents just GET this on localhost. Non-frozen.
	mux.HandleFunc("/api/market/quote", marketdata.QuoteHandler())
	// TRIGGER (ROADMAP 3): otomasi event→aksi (plug-and-play, agnostic). Schedule = tipe `time`.
	mux.HandleFunc("/api/triggers", triggersHandler(trigEngine))
	mux.HandleFunc("/api/triggers/delete", triggersDeleteHandler(trigEngine))
	mux.HandleFunc("/api/triggers/duplicate", triggersDuplicateHandler(trigEngine)) // clone a schedule/trigger (owner request)
	mux.HandleFunc("/api/triggers/toggle", triggersToggleHandler(trigEngine))
	mux.HandleFunc("/api/triggers/run", triggersRunHandler(trigEngine))
	mux.HandleFunc("/api/triggers/runs", triggersRunsHandler(trigEngine))
	mux.HandleFunc("/api/triggers/types", triggersTypesHandler())
	mux.HandleFunc("/api/triggers/hook/", triggersHookHandler(trigEngine)) // webhook intake (public, secret-gated)
	// APPS (ROADMAP 4): launcher + invoke operasi (1 pintu utk human GUI & agent tool) + state + aset GUI.
	mux.HandleFunc("/api/apps", appsListHandler(appsMgr))
	mux.HandleFunc("/api/apps/op", appsOpHandler(appsMgr))
	mux.HandleFunc("/api/apps/install", appsInstallHandler())           // upload .fwpack → hot-reload
	mux.HandleFunc("/api/apps/uninstall", appsUninstallHandler())       // stop + unregister + rm
	mux.HandleFunc("/api/apps/stop", appsStopHandler(appsMgr))          // stop core when its GUI tab closes
	mux.HandleFunc("/api/agents/apps", appGrantsHandler(appsMgr, host)) // izin-app per-agent (GUI=truth): list app terinstall + toggle cap app:<id>
	mux.HandleFunc("/api/apps/state", appsStateHandler(appsMgr))
	mux.HandleFunc("/api/apps/", appsUIHandler(appsMgr)) // /api/apps/<id>/ui/* (iframe sandbox)
	// GUARDIAN (FASE 1): status + arm/disarm. Owner-session gated (lewat authMgr.Middleware).
	mux.HandleFunc("/api/guardian/status", guardianStatusHandler())
	mux.HandleFunc("/api/guardian/arm", guardianArmHandler())
	mux.HandleFunc("/api/guardian/disarm", guardianDisarmHandler(authMgr))
	// Scheduler looping: CRUD jadwal recurring task.
	mux.HandleFunc("/api/taskflow/schedules", taskflowSchedulesHandler(fdb))
	mux.HandleFunc("/api/taskflow/schedule", taskflowScheduleAddHandler(fdb))
	mux.HandleFunc("/api/taskflow/schedule/delete", taskflowScheduleDeleteHandler(fdb))
	// FASE 7: config MCP buat GUI copy-paste ke AI eksternal.
	mux.HandleFunc("/api/mcp/config", mcpConfigHandler)
	// FASE 8: Curator skill (per-agent) — list+grade + jalanin curator.
	mux.HandleFunc("/api/agents/skills", agentmgr.SkillsListHandler)
	mux.HandleFunc("/api/agents/skills/curate", agentmgr.SkillsCurateHandler)
	mux.HandleFunc("/api/skills/export", skillsExportHandler) // A2 fase-1: bundle SKILL.md → .fwskill pack
	mux.HandleFunc("/api/skills/import", skillsImportHandler) // A2 fase-1: import .fwskill (owner-gated + verifier)
	mux.HandleFunc("/api/agents/scheduler/runs", agentmgr.SchedulerRunsHandler)
	mux.HandleFunc("/api/agents/scheduler/trigger", agentmgr.SchedulerTriggerHandler)
	mux.HandleFunc("/api/agents/sneakernet/export", agentmgr.SneakernetExportHandler)
	mux.HandleFunc("/api/agents/sneakernet/import", agentmgr.SneakernetImportHandler)
	mux.HandleFunc("/api/agents/mesh/identity", agentmgr.MeshIdentityHandler)
	mux.HandleFunc("/api/agents/mesh/peers", agentmgr.MeshPeersHandler)
	mux.HandleFunc("/api/agents/watchdog/tick", agentmgr.WatchdogTickHandler)
	mux.HandleFunc("/api/agents/finance/ledger", agentmgr.FinanceLedgerHandler)
	mux.HandleFunc("/api/agents/finance/summary", agentmgr.FinanceSummaryHandler)
	mux.HandleFunc("/api/agents/finance/budget", agentmgr.FinanceBudgetHandler)
	mux.HandleFunc("/api/agents/finance/check_budget", agentmgr.FinanceCheckBudgetHandler)
	mux.HandleFunc("/api/agents/protector/rules", agentmgr.ProtectorRulesHandler)
	mux.HandleFunc("/api/agents/protector/test", agentmgr.ProtectorTestHandler)
	mux.HandleFunc("/api/agents/protector/audit", agentmgr.ProtectorAuditHandler)
	mux.HandleFunc("/api/agents/scanner/scan", agentmgr.ScannerScanHandler)
	mux.HandleFunc("/api/agents/scanner/runs", agentmgr.ScannerRunsHandler)
	mux.HandleFunc("/api/agents/scanner/findings", agentmgr.ScannerFindingsHandler)
	mux.HandleFunc("/api/agents/scanner/auditors", agentmgr.ScannerAuditorsHandler)
	mux.HandleFunc("/api/agents/audit/log", agentmgr.AuditLogHandler)
	mux.HandleFunc("/api/agents/watchdog/alerts", agentmgr.WatchdogAlertsHandler)
	mux.HandleFunc("/api/agents/codemap/index", agentmgr.CodemapIndexHandler)
	mux.HandleFunc("/api/agents/codemap/nodes", agentmgr.CodemapNodesHandler)
	// CGM (Cognitive Graph Memory) read API — additive, plug-and-play (roadmap_opus8.md
	// Phase 3 §4.9). Owner-approved edit to this LOCKED file (autonomy grant 2026-06-19):
	// reason = wire 2 read-only handlers for the new Cognitive Graph GUI tab.
	mux.HandleFunc("/api/agents/cognitive/graph", agentmgr.CognitiveGraphHandler)
	mux.HandleFunc("/api/agents/cognitive/tensions", agentmgr.CognitiveTensionsHandler)
	// CGM digestion manual trigger (POST) — deploy proven loop (P1, 2026-06-20).
	// Owner-controlled; always runs. Auto-digest in dream cron gated by env.
	mux.HandleFunc("/api/agents/cognitive/digest", agentmgr.CognitiveDigestHandler)
	mux.HandleFunc("/api/agents/learning/digest", agentmgr.LearningDigestHandler) // Phase 3E/D13 loop-belajar (gated FLOWORK_LEARN_LOOP=1)
	mux.HandleFunc("/api/agents/compact", agentmgr.CompactAgentHandler) // auto-compact manual trigger (owner/QC)
	mux.HandleFunc("/api/compact/config", agentmgr.CompactConfigHandler) // ambang auto-compact (GUI Settings)
	mux.HandleFunc("/api/agents/compact-all", func(w http.ResponseWriter, r *http.Request) { // Compact All (menu Agent)
		if r.Method != http.MethodPost {
			httpx.WriteJSON(w, map[string]any{"error": "method not allowed (POST)"})
			return
		}
		// ASYNC: compact-all (force) digest LLM tiap agent → bisa puluhan menit, JANGAN blok HTTP.
		force := r.URL.Query().Get("force") == "1"
		ids := host.AgentIDs()
		go agentmgr.AutoCompactAllAgents(ids, force)
		httpx.WriteJSON(w, map[string]any{
			"ok": true, "async": true, "agents": len(ids),
			"note": "Compact All jalan di background buat " + strconv.Itoa(len(ids)) + " agent (digest→trim). Cek log / refresh nanti.",
		})
	})
	mux.HandleFunc("/api/agents/zombie/findings", agentmgr.ZombieFindingsHandler)
	mux.HandleFunc("/api/agents/zombie/ack", agentmgr.ZombieAckHandler)
	mux.HandleFunc("/api/agents/zombie/scan", agentmgr.ZombieScanHandler)
	mux.HandleFunc("/api/agents/self-prompt", agentmgr.SelfPromptHandler)
	mux.HandleFunc("/api/agents/self-prompt/render", agentmgr.SelfPromptRenderHandler)

	// Legacy reference GUI compat shim — map paths dari reference tabs ke
	// agent-scoped endpoint (default agent mr-flow).
	mux.HandleFunc("/api/finance/snapshot", agentmgr.FinanceSnapshotCompatHandler)
	// R8 self-finance fase-1: menu setting kredensial wallet EVM (privkey lokal terenkripsi, ga ke-commit).
	mux.HandleFunc("/api/finance/wallet", financeWalletHandler())
	mux.HandleFunc("/api/brain/prompt-templates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			agentmgr.PromptTemplatesUpsertCompatHandler(w, r)
			return
		}
		agentmgr.PromptTemplatesListCompatHandler(w, r)
	})
	mux.HandleFunc("/api/brain/prompt-templates/detail", agentmgr.PromptTemplatesDetailCompatHandler)
	mux.HandleFunc("/api/brain/prompt-templates/update", agentmgr.PromptTemplatesUpsertCompatHandler)
	mux.HandleFunc("/api/brain/prompt-templates/delete", agentmgr.PromptTemplatesDeleteCompatHandler)
	mux.HandleFunc("/api/protector", agentmgr.ProtectorListCompatHandler)
	mux.HandleFunc("/api/protector/add", agentmgr.ProtectorAddCompatHandler)
	mux.HandleFunc("/api/protector/remove", agentmgr.ProtectorRemoveCompatHandler)
	mux.HandleFunc("/api/protector/toggle", agentmgr.ProtectorToggleCompatHandler)
	mux.HandleFunc("/api/protector/test", agentmgr.ProtectorTestCompatHandler)
	mux.HandleFunc("/api/codemap/graph", agentmgr.CodemapGraphCompatHandler)
	mux.HandleFunc("/api/codemap/status", agentmgr.CodemapStatusCompatHandler)
	mux.HandleFunc("/api/codemap/zombies", agentmgr.CodemapZombiesCompatHandler)
	mux.HandleFunc("/api/codemap/reindex", agentmgr.CodemapReindexCompatHandler)
	mux.HandleFunc("/api/codemap/roots", agentmgr.CodemapRootsCompatHandler)
	mux.HandleFunc("/api/codemap/docs", agentmgr.CodemapDocsCompatHandler)
	// R6 self-map semantik: lapisan MAKNA di atas self-map deterministik (LLM di-inject).
	mux.HandleFunc("/api/codemap/enrich", agentmgr.CodemapEnrichHandler(codemapSemanticSummarizer(host)))
	mux.HandleFunc("/api/codemap/semantic", agentmgr.CodemapSemanticHandler)
	// R7 self-evolution fase-1: refleksi-diri (baca self-map → usul perbaikan → backlog + karma).
	mux.HandleFunc("/api/evolve/reflect", agentmgr.EvolveReflectHandler(evolveProposer()))
	mux.HandleFunc("/api/evolve/proposals", agentmgr.EvolveProposalsHandler)
	// R7 fase-2 control-plane: saklar self-modify (off|stage|auto) + gate berlapis (KRUSIAL).
	mux.HandleFunc("/api/evolve/config", agentmgr.EvolveConfigHandler(evolveGateDeps()))
	// Capability eval: buktiin model layak self-modify (compile+run, kalibrasi Opus 4.7).
	mux.HandleFunc("/api/evolve/eval", evolveEvalHandler())
	// R7 fase-2b ENGINE EKSEKUSI (behavior-layer): apply proposal add-agent/skill/app →
	// reuse architect → ~/.flowork (additive, di luar git). Gate: saklar+model (anti-lokal).
	mux.HandleFunc("/api/evolve/apply", agentmgr.EvolveApplyHandler(evolveGateDeps(), evolveApplier(host, fdb, groupsAPI)))
	// R7 fase-2b core-apply (🔴 DEV-only): proposal core 'NEW:' → git-worktree sandbox →
	// codegen → test-gate → STAGE diff buat review. Edit existing/LOCKED/delete = error edukasi.
	mux.HandleFunc("/api/evolve/core-apply", agentmgr.EvolveCoreApplyHandler(evolveGateDeps(), evolveCoreApplier(host)))
	mux.HandleFunc("/api/evolve/council", agentmgr.EvolveCouncilHandler(evolveCouncilJudgeViaGroup(host))) // A1: dewan adversarial via grup self-evolution (5 agent: pembela/penantang/hakim panel-3)
	mux.HandleFunc("/api/evolve/proposal/delete", agentmgr.EvolveProposalDeleteHandler)                    // hapus usulan (1 / bulk by-status)
	mux.HandleFunc("/api/evolve/stages", agentmgr.EvolveStagesHandler)
	// Milestone C STAGE review: owner Approve (commit isi yg direview + maybe push) / Reject staged diff.
	mux.HandleFunc("/api/evolve/stage-action", agentmgr.EvolveStageActionHandler(evolveGateDeps(), evolveStageCommitter()))
	// B3 auto-push config: token GitHub organisme (lokal di ~/.flowork, ga ke-commit) + saklar.
	mux.HandleFunc("/api/evolve/push-config", evolvePushConfigHandler())
	// B3 boot-rollback: penyebab organisme hampir mati (commit letal di-revert watchdog) — biar dia tau.
	mux.HandleFunc("/api/evolve/rollback-log", evolveRollbackLogHandler())
	// Milestone D trigger terjadwal: refleksi-diri otomatis tiap N jam (+ auto-apply behavior pas auto).
	mux.HandleFunc("/api/evolve/schedule", evolveScheduleHandler(host, fdb, groupsAPI))
	startEvolveScheduler(ctx, host, fdb, groupsAPI) // goroutine: cron refleksi (interval dari KV; 0=off)
	mux.HandleFunc("/api/agents/protector/approval/queue", agentmgr.ApprovalQueueHandler)
	mux.HandleFunc("/api/agents/protector/approve_pending", agentmgr.ApproveHandler)
	mux.HandleFunc("/api/agents/protector/reject_pending", agentmgr.RejectHandler)
	mux.HandleFunc("/api/agents/tool-audit", agentmgr.ToolAuditHandler)
	// Section 9 reference GUI tab (doktrin_edukasi.js) — compat shim
	// → agents/edu-errors agent-scoped (lihat legacy_compat_v2.go).
	mux.HandleFunc("/api/settings/educational-errors", agentmgr.EduErrorsCompatHandler)
	// Tool Registry reference GUI tab (warga_caps.js) — single-warga shim.
	mux.HandleFunc("/api/warga-caps/warga", agentmgr.WargaListCompatHandler)
	mux.HandleFunc("/api/warga-caps/catalog", agentmgr.WargaCapsCatalogCompatHandler)
	mux.HandleFunc("/api/warga-caps/effective", agentmgr.WargaCapsEffectiveCompatHandler)
	mux.HandleFunc("/api/warga-caps/override", agentmgr.WargaCapsOverrideCompatHandler)
	mux.HandleFunc("/api/warga-caps/seed", agentmgr.WargaCapsSeedCompatHandler)
	// Audit Log reference GUI tab (commits.js) — adapt audit → git-style.
	mux.HandleFunc("/api/commits", agentmgr.CommitsCompatHandler)

	// Catch-all stub utk path /api/* yang gak diregister.
	mux.HandleFunc("/api/", mockAPI)

	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// GUARDIAN boot gate (FASE 1): kalau armed & integritas binary/kernel gagal → SAFE-MODE
	// + alert owner. SafeModeMiddleware = lapis paling LUAR (blok exec/install saat safe-mode),
	// di atas auth — nol perubahan ke kernel beku.
	guardianAutoArm() // ONE-CLICK: auto-jaga (detection) pas start, kecuali sudah OS-lock eksplisit
	guardianBootCheck()
	// GUARDIAN sentinel (FASE 3): pengawas runtime — tiap 5 mnt cek integritas + seal-drift +
	// cap-drift (agent dapet cap berbahaya baru → eskalasi). Pasif kalau belum di-arm.
	sentinelEvery := 5 * time.Minute
	if s := os.Getenv("FLOWORK_GUARDIAN_INTERVAL_SEC"); s != "" {
		if n, e := strconv.Atoi(s); e == nil && n > 0 {
			sentinelEvery = time.Duration(n) * time.Second
		}
	}
	go guardian.RunSentinel(ctx, sentinelEvery, guardianDangerCaps, func(msg string) {
		nctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = notifyOwnerTelegram(nctx, msg)
	})
	srv := &http.Server{
		Addr:              *addr,
		Handler:           httpx.NoCache(guardian.SafeModeMiddleware(authMgr.Middleware(mux))),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		<-ctx.Done()
		// §8.A: let every module write its death-letter (on_stop) while the loket
		// endpoint is still up, BEFORE we close the HTTP server.
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 8*time.Second)
		host.AutoOnStop(stopCtx)
		stopCancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		_ = srv.Shutdown(shutCtx)
	}()
	// Lifecycle on_load (§8.A): fire once the loket endpoint is listening, so a
	// module's on_load can reach loket caps. Brief delay lets the server bind first.
	go func() {
		time.Sleep(1500 * time.Millisecond)
		host.AutoOnLoad(ctx)
	}()
	log.Printf("flowork-gui %s listening on http://%s (agents dir: %s)", version, *addr, host.AgentsDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}

// ── Auth / system stubs ────────────────────────────────────────────────────

func ownerAutoVerify(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, map[string]any{"verified": true})
}

// grantSubscribedToolCaps — "GUI = KEBENARAN" (owner 2026-06-20): tool yg di-CENTANG
// di GUI settings (config.tools = subscribed) → capability-nya AUTO-GRANT ke agent,
// biar "centang tool = beneran bisa pakai", bukan pajangan. Subscribe ≠ capability by
// default (gate isolasi) — pass ini yg nyambungin.
// AMAN: PRIVILEGED-ONLY (FLOWORK_PRIVILEGED_AGENTS). Agent untrusted tetep manifest-
// only → checkbox nyasar ga ngasih cap bahaya (exec:power/shell/net:fetch:*). Agent
// owner (mr-flow) = privileged → semua tool yg dicentang dapet cap-nya.
func grantSubscribedToolCaps(host *kernelhost.Host, agentID string) {
	if host == nil || host.Broker == nil {
		return
	}
	priv := false
	for _, id := range strings.Split(os.Getenv("FLOWORK_PRIVILEGED_AGENTS"), ",") {
		if strings.TrimSpace(id) == agentID {
			priv = true
			break
		}
	}
	if !priv {
		return // untrusted: manifest-only (anti security-hole)
	}
	store, err := host.OpenAgentStore(agentID)
	if err != nil {
		return
	}
	// GUI checkbox nulis ke tabel tool_subscriptions (BUKAN config blob) — itu
	// sumber kebenaran "tool yg dicentang".
	subs, lerr := store.ListSubscriptions()
	store.Close()
	if lerr != nil || len(subs) == 0 {
		return
	}
	set := map[string]bool{}
	for _, c := range host.Broker.Approved(agentID) {
		set[c] = true
	}
	added := 0
	for _, ts := range subs {
		name := ts.ToolName
		if name == "" {
			continue
		}
		if t, ok := tools.Lookup(name); ok {
			// app:<id> caps di-SKIP di sini — itu digate fitur izin-app per-agent
			// (app_grants / applyAppCaps), biar "Allowed Apps" di GUI = satu-satunya
			// gerbang app (centang app = bisa, uncentang = ga bisa). Subscribe tool
			// app ga otomatis kasih akses app — owner pilih eksplisit di list app.
			if c := t.Capability(); c != "" && !strings.HasPrefix(c, "app:") && !set[c] {
				set[c] = true
				added++
			}
		}
	}
	if added == 0 {
		return
	}
	caps := make([]string, 0, len(set))
	for c := range set {
		caps = append(caps, c)
	}
	host.Broker.Approve(agentID, caps)
	log.Printf("[caps] %s: +%d capability auto-grant dari %d tool ke-subscribe (GUI=truth, privileged)", agentID, added, len(subs))
}

// seedAndApplyAppCaps — sinkron cap app:<id> agent dari tabel app_grants (fitur
// "Allowed Apps" di GUI). MIGRASI one-time: app_grants kosong + belum pernah
// di-seed → isi dari app-tool yg udah di-subscribe (biar setup lama ga regres +
// langsung ke-centang di GUI). Abis itu app_grants authoritative — owner
// centang/uncentang di GUI = kebenaran. Dipanggil pas boot (setelah app & tool
// keregister) supaya tools.Lookup nemu app-tool buat derivasi seed.
func seedAndApplyAppCaps(host *kernelhost.Host, agentID string) {
	if host == nil {
		return
	}
	store, err := host.OpenAgentStore(agentID)
	if err != nil {
		return
	}
	defer store.Close()
	if !store.AppGrantsSeeded() {
		seedAppGrantsFromSubscriptions(store)
		_ = store.MarkAppGrantsSeeded()
	}
	applyAppCaps(host, agentID, store)
}

// seedAppGrantsFromSubscriptions — derivasi app yg "udah dipakai": tiap tool yg
// di-subscribe dengan capability app:<id> → GrantApp(<id>). One-time (dipagar
// AppGrantsSeeded di pemanggil) biar uncentang-semua ga ke-reset balik.
func seedAppGrantsFromSubscriptions(store *agentdb.Store) {
	subs, err := store.ListSubscriptions()
	if err != nil {
		return
	}
	for _, ts := range subs {
		if t, ok := tools.Lookup(ts.ToolName); ok {
			if c := t.Capability(); strings.HasPrefix(c, "app:") {
				_ = store.GrantApp(strings.TrimPrefix(c, "app:"))
			}
		}
	}
}

// reconcileDeadCrews — DB-driven self-heal: hapus task_category yg SEMUA member
// agent-nya udah ga ada (dir ga eksis) → cascade task_agents + trigger_rules
// (lewat DeleteCategory). "Hapus agent → crew auto-ilang", lepas dari jalur delete.
// Pakai DIR-EXISTENCE (bukan host.AgentIDs) → AMAN dari race agent-belum-load:
// dir ada walau lagi loading = di-keep. Skip kategori crew-kosong (lagi disetup).
func reconcileDeadCrews(fdb *floworkdb.Store, agentsDir string) {
	ids, err := fdb.CategoryIDs()
	if err != nil {
		return
	}
	for _, id := range ids {
		cat, err := fdb.GetCategory(id)
		if err != nil || cat == nil || len(cat.Crew) == 0 {
			continue
		}
		alive := false
		for _, m := range cat.Crew {
			if m.AgentID == "" {
				continue
			}
			if st, e := os.Stat(filepath.Join(agentsDir, m.AgentID+".fwagent")); e == nil && st.IsDir() {
				alive = true
				break
			}
		}
		if !alive {
			if derr := fdb.DeleteCategory(id); derr == nil {
				log.Printf("[reconcile] crew mati '%s' dihapus (semua %d member ga ada lagi)", id, len(cat.Crew))
			}
		}
	}
}

// seedSocialDefaults makes a FRESH clone match the owner's setup. It loads the
// committed seed/social.seed.json and:
//
//	(a) installs the 2-trending-1-promo social schedule — ONLY when trigger_rules is
//	    empty, so it never clobbers a machine that already has its own schedule;
//	(b) sets each social group's non-secret flowork_tele_link kv when it's absent.
//
// Secrets are NOT here: tokens (X cookies, Dev.to key, bot tokens) stay in Settings →
// API Keys (flowork.db, never pushed). So `git clone` → same setup, user only fills tokens.
func seedSocialDefaults(fdb *floworkdb.Store, agentsDir string) {
	raw, err := os.ReadFile("seed/social.seed.json")
	if err != nil {
		return // no seed file shipped → nothing to do
	}
	var seed struct {
		Schedule []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Cron   string `json:"cron"`
			Target string `json:"target"`
			Prompt string `json:"prompt"`
			Kind   string `json:"kind"` // "group" (default) or "agent"
		} `json:"schedule"`
		GroupConfig  map[string]string `json:"group_config"`
		ConfigGroups []string          `json:"config_groups"`
		KV           map[string]string `json:"kv"`
	}
	if json.Unmarshal(raw, &seed) != nil {
		return
	}
	// (kv) Setting global default — set HANYA kalau kosong, jadi: fresh clone dapat
	// pengalaman owner (model/yt default), nilai PERSONAL = "change me" sampai user isi,
	// dan mesin owner (kv sudah terisi) TIDAK ketimpa.
	for k, v := range seed.KV {
		if cur, _ := fdb.GetKV(k); strings.TrimSpace(cur) == "" {
			_ = fdb.SetKV(k, v)
		}
	}
	// (a) schedule — AUTO-UPDATE friendly (P1.5). Install each release-shipped rule whose
	// ID has NEVER been seeded before; never overwrite a rule the user already has, and
	// never re-add one the user deleted. A kv ledger ("seeded_schedule_ids") records every
	// id we have ever seeded, so updates are additive: a NEW rule shipped in a later release
	// appears on an already-populated machine, while existing/edited/deleted rules are left
	// exactly as the user left them. (Fresh install: table + ledger empty → all installed.)
	if len(seed.Schedule) > 0 {
		existing := map[string]bool{}
		if cur, lerr := fdb.ListTriggers(); lerr == nil {
			for _, t := range cur {
				existing[t.ID] = true
			}
		}
		seeded := map[string]bool{}
		if raw, _ := fdb.GetKV("seeded_schedule_ids"); raw != "" {
			var ids []string
			if json.Unmarshal([]byte(raw), &ids) == nil {
				for _, id := range ids {
					seeded[id] = true
				}
			}
		}
		added := 0
		for _, s := range seed.Schedule {
			if !existing[s.ID] && !seeded[s.ID] { // genuinely new → install once
				kind := s.Kind
				if kind == "" {
					kind = "group"
				}
				_ = fdb.UpsertTrigger(floworkdb.Trigger{
					ID: s.ID, Name: s.Name, TypeID: "time",
					Config: `{"cron":"` + s.Cron + `"}`,
					Target: s.Target, TargetKind: kind,
					Prompt: s.Prompt, Deliver: "telegram", Enabled: true,
				})
				added++
			}
			seeded[s.ID] = true // remember forever (covers pre-existing + just-added)
		}
		ids := make([]string, 0, len(seeded))
		for id := range seeded {
			ids = append(ids, id)
		}
		if b, err := json.Marshal(ids); err == nil {
			_ = fdb.SetKV("seeded_schedule_ids", string(b))
		}
		if added > 0 {
			log.Printf("seed: installed %d NEW social schedule rule(s) from seed/social.seed.json (additive update, user data preserved)", added)
		}
	}
	// (b) non-secret per-group config (e.g. the PUBLIC Telegram invite) — set if absent.
	for _, gid := range seed.ConfigGroups {
		staged := filepath.Join(agentsDir, gid+".fwagent")
		path := filepath.Join(filepath.Dir(agentdb.Resolve(gid, staged)), "loket.db")
		st, oerr := loket.OpenStore(path)
		if oerr != nil {
			continue
		}
		for k, v := range seed.GroupConfig {
			if cur, _, _ := st.KVGet(k); strings.TrimSpace(cur) == "" {
				_ = st.KVSet(k, v)
			}
		}
		_ = st.Close()
	}
}

// notifyOwnerTelegram — push pesan ke owner via Telegram. Baca config dari
// SETTINGS GLOBAL (floworkdb) — BUKAN dari agent. Owner-level notif terpisah
// dari AI agent (yang sengaja terisolasi / plug-and-play). Diam (log only)
// kalau Telegram belum di-set di Settings.
func notifyOwnerTelegram(ctx context.Context, text string) error {
	// Owner notifications have their OWN home: Settings → Notifications, stored in the
	// global floworkdb (NOTIFY_TG_TOKEN + notify_tg_chat). This is deliberately a
	// SEPARATE concern from any connector's chat token — the kernel pushing alerts to
	// the owner vs. a connector receiving messages. Read here, never from an agent.
	fdb, err := floworkdb.Shared()
	if err != nil {
		return err
	}
	token, _ := fdb.GetSecret("NOTIFY_TG_TOKEN")
	chatID, _ := fdb.GetKV("notify_tg_chat")
	token = strings.TrimSpace(token)
	chatID = strings.TrimSpace(chatID)
	if token == "" || chatID == "" {
		log.Printf("[codescan→notify] (telegram notif belum di-set di Settings) %s", truncStr(text, 120))
		return nil
	}
	form := url.Values{"chat_id": {chatID}, "text": {text}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.telegram.org/bot"+token+"/sendMessage", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cl := &http.Client{Timeout: 10 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram sendMessage status %d", resp.StatusCode)
	}
	return nil
}

// notifyHubHandler — POST /api/notify {"text":"..."} → owner Telegram via notifyOwnerTelegram.
// Loopback-only (isPublicPath admits it only for a local non-browser caller; cross-site browser
// drive-by is cut there). Lets sandboxed local apps push an owner alert without the token.
func notifyHubHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "invalid json"})
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		httpx.WriteJSON(w, map[string]any{"error": "text required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := notifyOwnerTelegram(ctx, text); err != nil {
		httpx.WriteJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"ok": true})
}

// truncStr — potong string buat log.
func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func systemHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, map[string]any{
		"status":  "ok",
		"version": version,
		"ts":      time.Now().UTC().Format(time.RFC3339),
	})
}

// mockAPI — shape-friendly stub untuk unregistered /api/* paths.
func mockAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	switch {
	case strings.HasSuffix(path, "/list"),
		strings.HasSuffix(path, "/all"),
		strings.HasSuffix(path, "/inbox"),
		strings.HasSuffix(path, "/recent"):
		httpx.WriteJSON(w, map[string]any{"data": []any{}, "count": 0})
	case strings.HasSuffix(path, "/status"),
		strings.HasSuffix(path, "/state"),
		strings.HasSuffix(path, "/config"):
		httpx.WriteJSON(w, map[string]any{"ok": true})
	default:
		httpx.WriteJSON(w, map[string]any{"path": path, "method": r.Method})
	}
}
