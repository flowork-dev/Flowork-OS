// task_worker.go — drive background (queued) tasks async (Phase 6 / E, opsi-1 minimal).
//
// AKAR (roadmap E): kernel WASM SINKRON → ga ada background-agent paralel. Pola yang
// TERBUKTI = durable ledger + poller non-beku (lihat wakeup_engine.go). Worker ini
// MIRROR wakeup_engine: tiap tick scan agent_runs state='queued' (di-set TaskCreate
// background:true), drive via host.InvokeAgentMessage di goroutine BOUNDED + panic-
// recover, mark 'done'+output, notify owner. Kernel TIDAK disentuh; async hidup DI
// ATAS kernel (lapis non-beku, package main = wiring). Additive + DORMANT: tanpa task
// queued = no-op (perilaku identik sebelum fitur ini).
//
// Bounded (maxConcurrentTasks): model lokal serialize slot-nya → cap kecil = anti-
// overwhelm + fault-containment (1 task panic → host SELAMAT, task lain jalan terus).
package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"flowork-gui/internal/kernelhost"
)

// maxConcurrentTasks — batas task background yang jalan barengan. Kecil = aman (model
// serialize, hindari resource contention). Try-acquire non-blocking: slot penuh →
// task tetap 'queued', di-pick tick berikut.
const maxConcurrentTasks = 2

var taskSem = make(chan struct{}, maxConcurrentTasks)

// RunQueuedTasks — drive setiap task ber-state 'queued' lintas semua agent, async.
// Return jumlah task yang MULAI di-drive tick ini. Mirror RunDueWakeups; bedanya:
// eksekusi di goroutine (non-blocking poller) + bounded semaphore (anti-overwhelm).
func RunQueuedTasks(ctx context.Context, host *kernelhost.Host) int {
	started := 0
	for _, id := range host.AgentIDs() {
		store, err := host.OpenAgentStore(id)
		if err != nil {
			continue
		}
		db := store.DB()
		// Mayoritas agent ga punya agent_runs → skip murah (jangan polusi tiap DB).
		var tbl string
		if db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='agent_runs'").Scan(&tbl) != nil {
			store.Close()
			continue
		}
		type qt struct{ id, label, prompt string }
		var queued []qt
		rows, qerr := db.Query("SELECT id, label, COALESCE(checkpoint,'') FROM agent_runs WHERE state='queued' ORDER BY updated LIMIT 10")
		if qerr == nil {
			for rows.Next() {
				var r qt
				var cp string
				if rows.Scan(&r.id, &r.label, &cp) == nil {
					var meta map[string]any
					if json.Unmarshal([]byte(cp), &meta) == nil {
						if p, ok := meta["prompt"].(string); ok {
							r.prompt = p
						}
					}
					queued = append(queued, r)
				}
			}
			rows.Close()
		}
		for _, q := range queued {
			if strings.TrimSpace(q.prompt) == "" {
				// Ga ada prompt buat di-drive → tandai error biar ga muter (anti-stuck).
				db.Exec("UPDATE agent_runs SET state='error', updated=? WHERE id=? AND state='queued'",
					time.Now().UTC().Format(time.RFC3339), q.id)
				continue
			}
			// Try-acquire slot non-blocking: penuh → biarin 'queued' (tick berikut).
			select {
			case taskSem <- struct{}{}:
			default:
				continue
			}
			// Mark 'running' DULU (anti re-pick tick berikut, sama pola wakeup mark-fired).
			now := time.Now().UTC().Format(time.RFC3339)
			res, e := db.Exec("UPDATE agent_runs SET state='running', updated=? WHERE id=? AND state='queued'", now, q.id)
			if e != nil {
				<-taskSem
				continue
			}
			if n, _ := res.RowsAffected(); n == 0 {
				// Race: task udah ke-pick proses lain → lepas slot, skip.
				<-taskSem
				continue
			}
			started++
			go driveQueuedTask(ctx, host, id, q.id, q.label, q.prompt)
		}
		store.Close()
	}
	return started
}

// driveQueuedTask — eksekusi 1 task background di goroutine sendiri. Panic-recover =
// fault-containment (host SELAMAT). Buka store FRESH (poller udah Close store-nya).
func driveQueuedTask(ctx context.Context, host *kernelhost.Host, agentID, taskID, label, prompt string) {
	defer func() {
		<-taskSem // lepas slot APAPUN yang terjadi
		if r := recover(); r != nil {
			log.Printf("[queued-task] PANIC drive %s (host selamat): %v", taskID, r)
			if st, e := host.OpenAgentStore(agentID); e == nil {
				st.DB().Exec("UPDATE agent_runs SET state='error', updated=? WHERE id=?",
					time.Now().UTC().Format(time.RFC3339), taskID)
				st.Close()
			}
		}
	}()

	// 300s = SAMA dengan turn normal/wakeup (biar loop panjang sempat checkpoint +
	// jadwalin lanjutan via ScheduleWakeup; jangan turunin — lihat wakeup_engine.go).
	ictx, cancel := context.WithTimeout(ctx, 300*time.Second)
	reply, ierr := host.InvokeAgentMessage(ictx, agentID, prompt, "task")
	cancel()

	text := strings.TrimSpace(reply)
	var emitted map[string]any
	if json.Unmarshal([]byte(reply), &emitted) == nil {
		if rv, ok := emitted["reply"].(string); ok {
			text = strings.TrimSpace(rv)
		}
	}
	finalState := "done"
	if ierr != nil {
		finalState = "error"
		if text == "" {
			text = "(task error: " + ierr.Error() + ")"
		}
	}

	// Tulis hasil balik ke ledger (state + checkpoint.output) — buka store fresh.
	if st, e := host.OpenAgentStore(agentID); e == nil {
		db := st.DB()
		var cp string
		db.QueryRow("SELECT COALESCE(checkpoint,'{}') FROM agent_runs WHERE id=?", taskID).Scan(&cp)
		meta := map[string]any{}
		_ = json.Unmarshal([]byte(cp), &meta)
		meta["output"] = text
		nb, _ := json.Marshal(meta)
		db.Exec("UPDATE agent_runs SET state=?, checkpoint=?, updated=? WHERE id=?",
			finalState, string(nb), time.Now().UTC().Format(time.RFC3339), taskID)
		st.Close()
	}

	// Notify owner (best-effort; Telegram bisa ke-block network → di-log, ga fatal).
	icon := "✅"
	if finalState == "error" {
		icon = "⚠️"
	}
	msg := icon + " Task background selesai: " + label
	if text != "" {
		msg += "\n\n" + text
	}
	if nerr := notifyOwnerTelegram(ctx, msg); nerr != nil {
		log.Printf("[queued-task] notify owner gagal (%s): %v", taskID, nerr)
	}
	log.Printf("[queued-task] %s %s (agent %s)", finalState, taskID, agentID)
}
