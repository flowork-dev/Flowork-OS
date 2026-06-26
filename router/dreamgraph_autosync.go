// dreamgraph_autosync.go — DreamGraph (router cognitive graph) auto-populate + auto-update.
// NON-frozen extension seam (Rule 7). File baru, gak nyentuh file frozen.
//
// AKAR (Rule 5): tabel cognitive_nodes/cognitive_edges KOSONG + gak ada trigger terjadwal →
// tab Brain "FLowork Knowledge Graph (DreamGraph)" tampil kosong. RunDreamCycle disabled (mock+
// data-loss). SyncGraphToRAG cuma kepanggil pas CRUD node/edge manual.
//
// FIX (cabut-akar): boot-populate + ticker mirror sumber (constitution/persona/skill/agent) ke
// graph via brain.SyncGraphToRAG — idempotent, MIRROR-only (gak hapus memory). Auto-update saat
// ada pengetahuan baru lewat ticker (default 5 menit).
//
// KEBENARAN DI GUI (owner 2026-06-26): switch via fwswitch registry (FLOWORK_DREAMGRAPH_AUTOSYNC,
// FLOWORK_DREAMGRAPH_SYNC_MIN) → diatur dari tab "🎛️ Switch Fitur", BUKAN hardcode. Loop re-baca
// switch tiap siklus → ganti di GUI langsung kepakai tanpa restart.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flowork-os/flowork_Router/internal/brain"
	"github.com/flowork-os/flowork_Router/internal/store"
)

var dreamGraphSyncMu sync.Mutex // serialize sync (anti SQLite lock bentrok ticker vs manual)

// dreamGraphAutoSyncEnabled — switch GUI FLOWORK_DREAMGRAPH_AUTOSYNC (default ON).
func dreamGraphAutoSyncEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FLOWORK_DREAMGRAPH_AUTOSYNC"))) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}

// dreamGraphSyncInterval — switch GUI FLOWORK_DREAMGRAPH_SYNC_MIN (menit, default 5).
func dreamGraphSyncInterval() time.Duration {
	m := 5
	if s := strings.TrimSpace(os.Getenv("FLOWORK_DREAMGRAPH_SYNC_MIN")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			m = n
		}
	}
	return time.Duration(m) * time.Minute
}

// dreamGraphSyncOnce — apply brain path (lintas-proses settings) lalu mirror sumber → graph.
func dreamGraphSyncOnce(ctx context.Context) error {
	dreamGraphSyncMu.Lock()
	defer dreamGraphSyncMu.Unlock()
	d, _ := store.Open()
	s, _ := store.LoadSettings(d)
	applyBrainPath(s)
	if !brain.Available() {
		return nil
	}
	return brain.SyncGraphExtended(ctx) // B+C: core+instincts+knowledge (switch GUI)
}

// startDreamGraphAutoSync — boot-populate sekali + loop poll (re-baca switch GUI tiap siklus =
// live-adjust). Goroutine best-effort, gak pernah blok serve.
func startDreamGraphAutoSync(ctx context.Context) {
	go func() {
		if dreamGraphAutoSyncEnabled() {
			if err := dreamGraphSyncOnce(ctx); err != nil {
				log.Printf("WARN: dreamgraph boot sync: %v", err)
			} else {
				log.Printf("dreamgraph: boot sync OK (graph mirror sumber)")
			}
		}
		const poll = 30 * time.Second
		t := time.NewTicker(poll)
		defer t.Stop()
		last := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if !dreamGraphAutoSyncEnabled() {
					last = time.Now() // reset: pas dinyalain lagi, gak langsung nembak
					continue
				}
				if time.Since(last) < dreamGraphSyncInterval() {
					continue
				}
				last = time.Now()
				if err := dreamGraphSyncOnce(ctx); err != nil {
					log.Printf("WARN: dreamgraph sync: %v", err)
				}
			}
		}
	}()
}

// dreamGraphSyncHandler — POST /api/brain/graph/sync → trigger manual (tombol GUI "Sync Now").
func dreamGraphSyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := dreamGraphSyncOnce(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var nodes, edges int
	if db, err := brain.OpenRW(); err == nil {
		_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM cognitive_nodes").Scan(&nodes)
		_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM cognitive_edges").Scan(&edges)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nodes": nodes, "edges": edges})
}
