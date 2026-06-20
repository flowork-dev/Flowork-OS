// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-20 (auto-compact orchestrator).
// LOCKED ≠ FREEZE. Urutan digest→VERIFY→trim + fail-safe + skip-busy = anti-fatal; jangan ubah tanpa izin.
package agentmgr

// autocompact.go — AUTO-COMPACT konteks per-agent biar AI ga halu pas konteks panjang (owner
// 2026-06-20: "kalau konteks udah panjang, semua agent otomatis compact + masukin pengalaman ke
// brain kayak dream; pastikan work, FATAL jika salah"). Trigger by UKURAN konteks (bukan cuma cron
// 12 jam). Urutan AMAN: digest→VERIFY→trim. Fail-safe: digest gagal = ga trim (no loss).

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/httpx"
)

const (
	compactDefaultMaxInteractions = 400              // live interaksi non-deleted → trigger compact
	compactDefaultKeepRecent      = 60               // sisain N interaksi terbaru (konteks recent tetap utuh)
	compactBusyWindow             = 90 * time.Second // skip agent yg baru aktif (mungkin mid-task)
)

// compactConfig — ambang + toggle dari GUI/KV (owner kontrol). Default aman kalau belum di-set.
func compactConfig() (maxLive, keepRecent int, enabled bool) {
	maxLive, keepRecent, enabled = compactDefaultMaxInteractions, compactDefaultKeepRecent, true
	db, err := floworkdb.Shared()
	if err != nil {
		return
	}
	if v, _ := db.GetKV("compact_max_interactions"); strings.TrimSpace(v) != "" {
		if n, e := strconv.Atoi(strings.TrimSpace(v)); e == nil && n > 0 {
			maxLive = n
		}
	}
	if v, _ := db.GetKV("compact_keep_recent"); strings.TrimSpace(v) != "" {
		if n, e := strconv.Atoi(strings.TrimSpace(v)); e == nil && n >= 0 {
			keepRecent = n
		}
	}
	if v, _ := db.GetKV("compact_enabled"); strings.TrimSpace(v) == "0" {
		enabled = false
	}
	return
}

// AutoCompactAgent — compact 1 agent kalau konteks lewat ambang. digest pending → VERIFY → trim.
// FATAL-SAFE: (1) digest gagal → ga trim. (2) trim cuma yg UDAH di-digest. (3) skip agent mid-task.
func AutoCompactAgent(agentID string, maxLive, keepRecent int) (trimmed int64, digested int, note string) {
	store, err := openAgentStore(agentID)
	if err != nil {
		return 0, 0, "open: " + err.Error()
	}
	live, undigested, _, last, serr := store.CompactStats()
	store.Close()
	if serr != nil {
		return 0, 0, "stats: " + serr.Error()
	}
	if live < maxLive {
		return 0, 0, "under-threshold"
	}
	// SKIP mid-task: agent baru aktif → jangan ganggu konteks kerjanya (anti-fatal).
	if last != "" {
		if t, e := time.Parse(time.RFC3339, last); e == nil && time.Since(t) < compactBusyWindow {
			return 0, 0, "busy"
		}
	}
	// 1. DIGEST pending → brain (loop bounded, 100/batch). Gagal = STOP, JANGAN trim.
	if undigested > 0 {
		for i := 0; i < 20; i++ {
			_, n, derr := DigestAgent(agentID, 2)
			if derr != nil {
				return 0, digested, "digest GAGAL (ga trim, no loss): " + derr.Error()
			}
			digested += n
			if n == 0 {
				break
			}
		}
	}
	// 2. VERIFY: pastikan ga ada sisa undigested SEBELUM trim (kalau masih ada = jangan trim).
	store2, err := openAgentStore(agentID)
	if err != nil {
		return 0, digested, "reopen: " + err.Error()
	}
	defer store2.Close()
	if _, undig2, _, _, _ := store2.CompactStats(); undig2 > 0 {
		return 0, digested, "masih undigested abis digest → ga trim (fail-safe)"
	}
	// 3. TRIM: soft-delete interaksi yg UDAH di-brain, sisain N terbaru. Recoverable.
	trimmed, terr := store2.TrimDigestedInteractions(keepRecent)
	if terr != nil {
		return 0, digested, "trim: " + terr.Error()
	}
	return trimmed, digested, "ok"
}

// AutoCompactAllAgents — cek semua agent, compact yg lewat ambang. Resilient per-agent (1 rusak
// ga nyeret yg lain). Dipanggil cron berkala (lihat main). Hemat: cuma agent over-threshold yg
// kena LLM digest; sisanya cuma 1 query COUNT.
func AutoCompactAllAgents(agentIDs []string) {
	maxLive, keepRecent, enabled := compactConfig()
	if !enabled {
		return
	}
	for _, id := range agentIDs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[auto-compact] %s PANIC (di-skip): %v", id, r)
				}
			}()
			tr, dg, note := AutoCompactAgent(id, maxLive, keepRecent)
			if tr > 0 || dg > 0 {
				log.Printf("[auto-compact] %s: digested=%d trimmed=%d (%s)", id, dg, tr, note)
			}
		}()
	}
}

// CompactConfigHandler — GET status / POST set ambang auto-compact (owner GUI). KV global.
//
//	GET  /api/compact/config → {enabled, max_interactions, keep_recent}
//	POST {enabled?:bool, max_interactions?:int, keep_recent?:int}
func CompactConfigHandler(w http.ResponseWriter, r *http.Request) {
	db, err := floworkdb.Shared()
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "db: " + err.Error()})
		return
	}
	if r.Method == http.MethodPost {
		var b struct {
			Enabled         *bool `json:"enabled"`
			MaxInteractions *int  `json:"max_interactions"`
			KeepRecent      *int  `json:"keep_recent"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Enabled != nil {
			v := "1"
			if !*b.Enabled {
				v = "0"
			}
			_ = db.SetKV("compact_enabled", v)
		}
		if b.MaxInteractions != nil && *b.MaxInteractions > 0 {
			_ = db.SetKV("compact_max_interactions", strconv.Itoa(*b.MaxInteractions))
		}
		if b.KeepRecent != nil && *b.KeepRecent >= 0 {
			_ = db.SetKV("compact_keep_recent", strconv.Itoa(*b.KeepRecent))
		}
		httpx.WriteJSON(w, map[string]any{"ok": true})
		return
	}
	maxLive, keepRecent, enabled := compactConfig()
	httpx.WriteJSON(w, map[string]any{
		"enabled": enabled, "max_interactions": maxLive, "keep_recent": keepRecent,
		"note": "Tiap " + "15 menit agent yg interaksi non-deleted-nya > max bakal di-digest ke brain + trim (sisain keep_recent terbaru). Pengalaman ga ilang (pindah ke brain).",
	})
}

// CompactAgentHandler — POST /api/agents/compact?id=<agent>[&force=1]. Manual trigger (owner
// kontrol / GUI / QC). force=1 = abaikan ambang (compact walau di bawah). Default hormati ambang.
func CompactAgentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (POST)"})
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("id"))
	if agentID == "" {
		httpx.WriteJSON(w, map[string]any{"error": "agent id required"})
		return
	}
	maxLive, keepRecent, _ := compactConfig()
	if r.URL.Query().Get("force") == "1" {
		maxLive = 0 // paksa: ambang 0 = selalu lewat
	}
	tr, dg, note := AutoCompactAgent(agentID, maxLive, keepRecent)
	httpx.WriteJSON(w, map[string]any{
		"ok": true, "agent": agentID, "digested": dg, "trimmed": tr, "note": note,
		"max_interactions": maxLive, "keep_recent": keepRecent,
	})
}
