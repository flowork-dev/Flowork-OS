// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// 2026-06-12 (owner-approved): first-run config seed. `-export-seed <path>`
//   writes the current config with secrets blanked (store.ExportSeed); on normal
//   boot the embedded seed/router-config.seed.json is imported when the DB has no
//   providers yet, so a fresh download reproduces the owner's exact setup (the
//   user only pastes tokens). Additive — never touches an already-configured DB.
// 2026-06-13 (owner-approved, autonomous mesh task): mDNS discovery now announces the
//   ACTUAL listen port (parsed from -addr) instead of a hardcoded 2402, so nodes on
//   non-default ports / multiple nodes per host mesh correctly. Bug fix only — enables the
//   mesh, removes nothing.
// 2026-06-22 (owner-approved, F5/D32-INC4): + boot goroutine ticker brain.RebuildFreshIndex
//   (fresh-recall index buat drawer federation baru). Additive — gak ngubah jalur lain.
// Reason: Audit pass — audit pass surface review.

// flow_router Entry Point.

package main

import (
	"context"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/flowork-os/flowork_Router/internal/brain"
	"github.com/flowork-os/flowork_Router/internal/localai"
	"github.com/flowork-os/flowork_Router/internal/mesh"
	"github.com/flowork-os/flowork_Router/internal/policy"
	"github.com/flowork-os/flowork_Router/internal/store"

	// Side-effect import: each filter's init() registers itself with rtk.
	// New filters plug in by dropping a file under internal/rtk/filters/.
	_ "github.com/flowork-os/flowork_Router/internal/rtk/filters"

	// Side-effect imports: each translator pair self-registers via init().
	_ "github.com/flowork-os/flowork_Router/internal/translator/request"
	_ "github.com/flowork-os/flowork_Router/internal/translator/response"

	// Side-effect imports: each provider catalog file self-registers via init().
	_ "github.com/flowork-os/flowork_Router/internal/providers/embedding"
	_ "github.com/flowork-os/flowork_Router/internal/providers/image"
	_ "github.com/flowork-os/flowork_Router/internal/providers/tts"

	// Side-effect import: web-search vendor catalog (tavily/brave/serpapi/duckduckgo).
	_ "github.com/flowork-os/flowork_Router/internal/search"
)

//go:embed web/static
var webFS embed.FS

//go:embed seed/router-config.seed.json
var seedConfigJSON []byte

const version = "1.0.0-phase1.5-all-features-functional"

func main() {
	addr := flag.String("addr", "127.0.0.1:2402", "listen address")
	exportSeed := flag.String("export-seed", "", "write the current config (secrets blanked) to this path as a first-run seed, then exit")
	flag.Parse()

	// Section 13 phase 2: long-lived ctx untuk goroutine (mesh discovery,
	// future scheduler). Cancel saat shutdown.
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	_ = ctx

	log.Printf("flow_router %s starting on %s", version, *addr)
	log.Printf("Data: %s", store.DBPath())

	// Init storage + seed defaults
	d, err := store.Open()
	if err != nil {
		log.Fatalf("storage init: %v", err)
	}
	defer store.Close()

	// -export-seed: snapshot the current config (secrets blanked) and exit. Used
	// to regenerate seed/router-config.seed.json whenever the owner's setup changes.
	if *exportSeed != "" {
		raw, merr := json.MarshalIndent(store.ExportSeed(d), "", "  ")
		if merr != nil {
			log.Fatalf("export-seed marshal: %v", merr)
		}
		if werr := os.WriteFile(*exportSeed, append(raw, '\n'), 0o644); werr != nil {
			log.Fatalf("export-seed write: %v", werr)
		}
		log.Printf("config seed written to %s (secrets blanked)", *exportSeed)
		return
	}

	// First-run seed: reproduce the owner's exact config on a fresh install (no
	// providers yet). No-op on an already-configured DB. Runs before SeedDefaults
	// so the shipped setup wins over the generic catalog fallback.
	if n := store.SeedFromBundleJSON(d, seedConfigJSON); n != nil {
		log.Printf("first-run config seed applied: %v", n)
	}
	if err := store.SeedDefaults(d); err != nil {
		log.Printf("WARN: seed defaults: %v", err)
	}
	if err := store.AugmentTierTags(d); err != nil {
		log.Printf("WARN: augment tier tags: %v", err)
	}
	if err := store.SeedDefaultPricing(d); err != nil {
		log.Printf("WARN: seed pricing: %v", err)
	}
	// Seed the Flowork doctrine (Aola's soul) into a fresh brain on first boot — only when the brain
	// is enabled. Idempotent + embedded → ships to OS/USB/Android with no external files. Sacred
	// secrets are NOT here (they belong in code/secret-store, never in an LLM-injected brain).
	if s, _ := store.LoadSettings(d); s != nil && s.Brain.Enabled {
		if s.Brain.DBPath != "" {
			brain.SetDBPath(s.Brain.DBPath)
		}
		if nd, nc, err := brain.SeedDoctrine(); err != nil {
			log.Printf("WARN: seed doctrine: %v", err)
		} else if nd > 0 || nc > 0 {
			log.Printf("brain: Flowork doctrine seeded — %d drawers + %d constitution", nd, nc)
		}
	}
	if err := store.PurgeExpiredSessions(d); err != nil {
		log.Printf("WARN: purge sessions: %v", err)
	}
	// Section 13 mesh foundation: generate ed25519 identity kalau belum
	// ada, simpan di mesh_identity. Idempotent — call tiap boot OK.
	if id, err := mesh.EnsureIdentity(d, version); err != nil {
		log.Printf("WARN: mesh identity: %v", err)
	} else {
		shortKey := id.PubKeyHex
		if len(shortKey) > 16 {
			shortKey = shortKey[:16]
		}
		log.Printf("Mesh identity: %s... (host=%s)", shortKey, id.Hostname)
		// Section 13 phase 2: mDNS discovery goroutine. Pure Go UDP
		// multicast. Best-effort — kalau port busy → announce-only mode.
		pubkeyBytes, _ := hex.DecodeString(id.PubKeyHex)
		// Announce the ACTUAL listen port (from -addr), not a hardcoded 2402 — otherwise a
		// node on a non-default port (or multiple nodes per host) advertises the wrong port
		// and peers can never reach it. Falls back to 2402 if -addr has no parseable port.
		meshPort := 2402
		if _, ps, perr := net.SplitHostPort(*addr); perr == nil {
			if p, aerr := strconv.Atoi(ps); aerr == nil && p > 0 {
				meshPort = p
			}
		}
		discovery := mesh.NewDiscovery(pubkeyBytes, meshPort, version, d)
		// Section 19: karma gate — ignore mDNS announces from peers whose trust
		// score fell below the floor (auto-quarantine misbehaving hosts).
		discovery.WhitelistCheck = mesh.KarmaGate(d)
		if derr := discovery.Start(ctx); derr != nil {
			log.Printf("WARN: mesh discovery: %v", derr)
		}
	}

	// Section 15 phase 2: gossip engine. Push pending packets to random
	// peers every 10s.
	gossipEngine := mesh.NewGossipEngine(d)
	gossipEngine.Start(ctx)
	defer gossipEngine.Stop()

	// Section 27 phase 2: policy budget cron evaluator.
	policyEngine := policy.New(d)
	policyEngine.Start(ctx)
	defer policyEngine.Stop()
	policyEngineRef = policyEngine

	loadMITMCaptureState()
	loadLearnCaptureState()     // 3E/D13: auto-capture toggle (kv, GUI) — owner 2026-06-21
	loadLocalAIAutostartState() // local-AI autostart toggle (kv, GUI; migrasi sekali dari env)
	startTunnelWatchdog()
	providers, _ := store.ListProviders(d)
	log.Printf("Providers loaded: %d", len(providers))
	for _, p := range providers {
		status := "off"
		if p.IsActive {
			status = "on"
		}
		log.Printf("  - [%s] %s (%s, priority=%d)", status, p.Name, p.AuthType, p.Priority)
	}

	// Boot auto-start of the local model (cross-OS, all editions): if the user
	// enabled a local provider AND the GGUF + llama-server binary are present,
	// bring llama-server up on boot so "Start Flowork" = sovereign model ready —
	// no GUI click. Goroutine (never blocks serve); skips silently for cloud-only
	// setups. Disable with FLOWORK_LOCALAI_AUTOSTART=0.
	go maybeAutostartLocalAI(providers)

	// F5 (D32-INC4 enabler): fresh-recall index. Rebuild on boot + tiap 2 menit (change-
	// detect → murah) supaya drawer federation BARU (recovery-instinct INC-4) ke-recall
	// SEBELUM vindex utama di-rebuild manual. Best-effort, goroutine — gak pernah blok serve.
	go func() {
		if n, err := brain.RebuildFreshIndex(ctx); err != nil {
			log.Printf("WARN: fresh-index boot rebuild: %v", err)
		} else if n > 0 {
			log.Printf("brain: fresh-recall index built (%d federation drawers)", n)
		}
		t := time.NewTicker(2 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := brain.RebuildFreshIndex(ctx); err != nil {
					log.Printf("WARN: fresh-index rebuild: %v", err)
				} else if n > 0 {
					log.Printf("brain: fresh-recall index refreshed (%d federation drawers)", n)
				}
			}
		}
	}()

	mux := http.NewServeMux()

	// All HTTP routes live in routes.go, grouped per domain.
	registerRoutes(mux)

	srv := &http.Server{
		Addr: *addr,
		// Middleware chain (outermost first):
		//   apiKeyMiddleware    — gates /v1 + /v1beta with flow_router API keys
		//                         (opt-in via settings.RequireApiKey), enforces
		//                         per-key caps, injects the key into context.
		//   authEnforceMiddleware — OPT-IN GUI session gate (settings.RequireLogin);
		//                         exempts /v1, /api/auth, health, static, root.
		Handler: apiKeyMiddleware(authEnforceMiddleware(mux)),
		// Slowloris guard: caps on every phase of the HTTP transaction so a
		// stalled client cannot pin a server goroutine forever. ReadHeader is
		// the most important — request-line + headers must arrive in 15s.
		// Write/Idle are generous because /v1 streams completions for minutes.
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
		WriteTimeout:      10 * time.Minute,
	}

	// Graceful shutdown: /api/shutdown fires shutdownTriggerCh; SIGINT/SIGTERM
	// also closes the server cleanly.
	shutdownTriggerCh = make(chan struct{}, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-shutdownTriggerCh:
			log.Printf("flow_router: shutdown requested via API")
		case s := <-sigCh:
			log.Printf("flow_router: signal %s received", s)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stopMITMOnShutdown() // drain MITM interceptor + clear DNS hijack before exit
		_ = srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("flow_router serve error: %v", err)
		os.Exit(1)
	}
	log.Printf("flow_router stopped cleanly")
}

// STABLE — verified 2026-06-15 (owner-approved). Boot auto-start PROVEN: router
// boot → ResolveLlamaBin() finds <exe-dir>/bin/llama-server → llama-server UP on
// :8088 with `--jinja -c 16384 --reasoning off -ngl $FLOWORK_NGL`, cross-OS. Don't
// change the gating (active :8088 provider) or resolution without re-testing a cold
// boot — the whole "Start Flowork = sovereign model ready" UX depends on it.
//
// maybeAutostartLocalAI — OPT-IN (audit #10 2026-06-15): LLM lokal BERAT + kebanyakan user
// pakai API/cloud, lagian target Android gak pake LLM lokal. Jadi DEFAULT = TIDAK auto-start.
// User aktifin lewat TOMBOL GUI router (POST /api/localai/runtime {action:start}) saat mau
// pakai lokal, atau set FLOWORK_LOCALAI_AUTOSTART=1 buat opt-in auto-start tiap boot.
// "Provider local active" (failover chain) != auto-spawn proses (beda concern, lihat audit #6).
// GUI status/stop kontrol instance yg sama (shared runtime ref). Cross-OS / semua edisi.
func maybeAutostartLocalAI(providers []store.ProviderConnection) {
	// Sumber kebenaran = setting GUI (kv 'localai:autostart'), BUKAN env (owner 2026-06-21).
	// loadLocalAIAutostartState() udah migrasi dari env sekali kalau kv belum ada.
	if !localAIAutostartEnabled() {
		log.Printf("localai autostart: OFF — nyalain lewat toggle GUI router (Settings → Local AI autostart)")
		return
	}
	want := false
	for _, p := range providers {
		if !p.IsActive {
			continue
		}
		if base, _ := p.Data[store.CfgBaseURL].(string); strings.Contains(base, ":8088") {
			want = true
			break
		}
	}
	if !want {
		return // no active local provider → user is cloud-only; leave it.
	}
	gguf := localai.ResolveFloworkBrain()
	bin := localai.ResolveLlamaBin()
	if gguf == "" || bin == "" {
		log.Printf("localai autostart: skip — gguf=%q bin=%q (set FLOWORK_BRAIN_GGUF / FLOWORK_LLAMA_BIN or install the model)", gguf, bin)
		return
	}
	localAIRuntimeMu.Lock()
	if localAIRuntimeRef == nil {
		localAIRuntimeRef = localai.NewRuntime(bin, 0)
	}
	rt := localAIRuntimeRef
	localAIRuntimeMu.Unlock()
	log.Printf("localai autostart: starting flowork-brain (bin=%s)", bin)
	if err := rt.Start(localai.FloworkBrainModel, gguf); err != nil {
		log.Printf("localai autostart: failed (%v) — agents fall back to cloud/priority", err)
		return
	}
	log.Printf("localai autostart: flowork-brain UP on :8088")
}
