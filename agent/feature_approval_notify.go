// feature_approval_notify.go — sisa F-B: notif Telegram ke owner pas ada antrian
// approval PENDING baru (biar owner ga harus buka GUI buat tau ada yang nunggu).
// 📄 Dok: FLowork_os/lock/approval-gate.md
//
// NON-FROZEN sibling (deletable, seam feature-registry — pola feature_deadair.go):
// host-side poller MURAH (baca approval_queue tiap menit, NOL token/LLM), skip
// agent yang ga punya tabel (anti polusi DB, pola wakeup_engine). Dedup in-memory:
// tiap row cuma dinotif sekali per proses; restart → pending yang masih nunggu
// dinotif ulang sekali (pengingat, bukan spam). Switch: FLOWORK_APPROVAL_NOTIFY.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"flowork-gui/internal/kernelhost"
)

func approvalNotifyEnabled() bool {
	v := strings.TrimSpace(os.Getenv("FLOWORK_APPROVAL_NOTIFY"))
	return v == "" || v == "1" || strings.EqualFold(v, "true") // default ON
}

func init() {
	RegisterFeature(Feature{Name: "approval-notify", Phase: PhaseSeed, Apply: func(d *Deps) {
		if d.Host == nil {
			return
		}
		go approvalNotifyLoop(d.Ctx, d.Host)
	}})
}

func approvalNotifyLoop(ctx context.Context, host *kernelhost.Host) {
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	seen := map[string]bool{} // "<agent>#<id>" → udah dinotif (dedup per proses)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if !approvalNotifyEnabled() {
				continue
			}
			var lines []string
			for _, id := range host.AgentIDs() {
				store, err := host.OpenAgentStore(id)
				if err != nil {
					continue
				}
				// Anti polusi: skip agent yang belum pernah punya antrian (pola
				// wakeup_engine) — jangan bikin tabel di DB agent yang ga butuh.
				var tbl string
				if store.DB().QueryRow(
					"SELECT name FROM sqlite_master WHERE type='table' AND name='approval_queue'").
					Scan(&tbl) != nil {
					store.Close()
					continue
				}
				rows, lerr := store.ListApprovalQueue("pending", 20)
				store.Close()
				if lerr != nil {
					continue
				}
				for _, r := range rows {
					key := fmt.Sprintf("%s#%d", id, r.ID)
					if seen[key] {
						continue
					}
					seen[key] = true
					reason := strings.TrimSpace(r.Reason)
					if reason == "" {
						reason = r.ToolName
					}
					lines = append(lines, fmt.Sprintf(
						"• %s — queue_id=%d · tool %s · %s", id, r.ID, r.ToolName, reason))
				}
			}
			if len(lines) == 0 {
				continue
			}
			_ = notifyOwnerTelegram(ctx, fmt.Sprintf(
				"🔐 %d aksi nunggu APPROVAL lo:\n%s\n\nPutusin di GUI tab Protector "+
					"(approve/reject). Approved berlaku 1 jam per tool+args.",
				len(lines), strings.Join(lines, "\n")))
		}
	}
}
