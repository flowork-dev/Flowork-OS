// === LOCKED FILE (soft) === Status: STABLE — DO NOT MODIFY without owner approval (LOCKED ≠ FREEZE).
// Owner: Aola Sahidin (Mr.Dev) · Locked 2026-06-16. Reason: R7 Milestone D trigger terjadwal. VERIFIED
// E2E: schedule config get/set; run=1 mode=off→reflect 5, no auto-apply (gate closed); run=1 mode=auto→
// reflect + AUTO-APPLY 3 behavior (2 skill+1 agent), 2 core di-skip (review). Loop otonom penuh nyala.
//
// selfevolve_schedule.go — R7 fase-2b Milestone D: TRIGGER TERJADWAL (cron refleksi berkala).
// Owner-approved 2026-06-16. Sebelumnya refleksi cuma manual (tombol). Sekarang organisme
// refleksi-diri SENDIRI tiap interval (owner pilih "terjadwal" sbg pemicu) → generate proposal;
// kalau mode=AUTO → auto-apply proposal BEHAVIOR (additive ~/.flowork, gated). Core proposals
// tetep di-stage/review owner (ga auto dari cron — terlalu deliberate). Interval di KV (lokal).
//
// Loop otonom penuh: cron → refleksi → (mode=auto) auto-apply behavior. Semua di-gate berlapis;
// mode=off / interval=0 → diam total. Anti-spam: tick kasar 10mnt + cek "udah lewat interval?".

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"flowork-gui/internal/agentmgr"
	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/kernelhost"
)

const evolveScheduleTick = 10 * time.Minute // tick kasar; interval sebenernya dari KV (jam)

// evolveCycleMu — guard ANTI-CONCURRENT: cegah dua siklus jalan barengan (tick goroutine +
// tombol "run now" manual) → dobel proposal/apply. TryLock: kalau lagi jalan, skip.
var evolveCycleMu sync.Mutex

func evolveScheduleKV() (*floworkdb.Store, error) { return floworkdb.Shared() }

// runEvolveScheduledCycle — satu siklus terjadwal: cek due → refleksi → (auto) auto-apply behavior.
// force=true (tombol "run now") lewatin cek interval. Balikin ringkasan buat respons/log.
func runEvolveScheduledCycle(host *kernelhost.Host, fdb *floworkdb.Store, groups *groupsapi.Handler, force bool) map[string]any {
	if !evolveCycleMu.TryLock() {
		return map[string]any{"skipped": "siklus lain lagi jalan (anti-concurrent)"}
	}
	defer evolveCycleMu.Unlock()
	db, err := evolveScheduleKV()
	if err != nil {
		return map[string]any{"error": "db: " + err.Error()}
	}
	hours := 0.0
	if v, _ := db.GetKV("evolve_schedule_hours"); strings.TrimSpace(v) != "" {
		hours, _ = strconv.ParseFloat(strings.TrimSpace(v), 64)
	}
	if !force {
		if hours <= 0 {
			return map[string]any{"skipped": "jadwal OFF (interval 0)"}
		}
		if lastStr, _ := db.GetKV("evolve_schedule_last"); strings.TrimSpace(lastStr) != "" {
			if last, perr := time.Parse(time.RFC3339, strings.TrimSpace(lastStr)); perr == nil {
				if time.Since(last) < time.Duration(hours*float64(time.Hour)) {
					return map[string]any{"skipped": "belum waktunya"}
				}
			}
		}
	}
	_ = db.SetKV("evolve_schedule_last", time.Now().UTC().Format(time.RFC3339))

	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	saved, rerr := agentmgr.EvolveReflectOnce(ctx, evolveProposer(), "refleksi terjadwal otonom")
	if rerr != nil {
		return map[string]any{"error": "refleksi: " + rerr.Error()}
	}
	out := map[string]any{"ok": true, "reflected": len(saved), "proposals": saved}
	// mode=auto → auto-apply behavior (gated internal; return kosong kalau belum auto).
	applied := agentmgr.EvolveScheduleAutoApply(evolveGateDeps(), evolveApplier(host, fdb, groups), evolveCouncilJudge(), saved)
	if len(applied) > 0 {
		out["auto_applied"] = applied
	}
	return out
}

// startEvolveScheduler — goroutine ticker. ctx cancel → stop. Idempotent via interval-check.
func startEvolveScheduler(ctx context.Context, host *kernelhost.Host, fdb *floworkdb.Store, groups *groupsapi.Handler) {
	go func() {
		t := time.NewTicker(evolveScheduleTick)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = runEvolveScheduledCycle(host, fdb, groups, false)
			}
		}
	}()
}

// evolveScheduleHandler — GET status / POST {hours} set interval / POST ?run=1 fire sekarang.
func evolveScheduleHandler(host *kernelhost.Host, fdb *floworkdb.Store, groups *groupsapi.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db, err := evolveScheduleKV()
		if err != nil {
			tfWriteJSON(w, 0, map[string]any{"error": "db: " + err.Error()})
			return
		}
		if r.Method == http.MethodPost {
			if r.URL.Query().Get("run") == "1" {
				tfWriteJSON(w, 0, runEvolveScheduledCycle(host, fdb, groups, true))
				return
			}
			var b struct {
				Hours *float64 `json:"hours"`
			}
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b.Hours != nil {
				h := *b.Hours
				if h < 0 {
					h = 0
				}
				_ = db.SetKV("evolve_schedule_hours", strconv.FormatFloat(h, 'f', -1, 64))
			}
			tfWriteJSON(w, 0, map[string]any{"ok": true})
			return
		}
		hours := 0.0
		if v, _ := db.GetKV("evolve_schedule_hours"); strings.TrimSpace(v) != "" {
			hours, _ = strconv.ParseFloat(strings.TrimSpace(v), 64)
		}
		last, _ := db.GetKV("evolve_schedule_last")
		tfWriteJSON(w, 0, map[string]any{
			"hours": hours, "enabled": hours > 0, "last_run": last,
			"note": "Refleksi-diri otomatis tiap N jam. mode=auto → auto-apply proposal behavior (additive). Core tetep di-stage buat review. 0 = OFF.",
		})
	}
}
