// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// feature_worklog_trigger.go — F-D (roadmap): WIRE seam triggers.WorklogPendingReader ke papan
// kerja nyata → trigger tipe `worklog-pending` bisa fire pas ada task NYANGKUT (bukan cuma PC-idle).
// Bikin MANDOR bisa kebangun by-KERJAAN. Pakai helper worklog yang udah ada (openAgentDB/Collect/…
// di feature_worklog.go, package sama). NOL sentuh frozen. Gate FLOWORK_WORKLOG. Dok: lock/worklog.md
package main

import (
	"time"

	"flowork-gui/internal/triggers"
	"flowork-gui/internal/worklog"
)

func init() {
	RegisterFeature(Feature{Name: "worklog-trigger", Phase: PhaseRoute, Apply: func(d *Deps) {
		triggers.WorklogPendingReader = func() int {
			if !worklogEnabled() || d.Host == nil {
				return 0 // worklog off → fail-safe (trigger ga fire)
			}
			items := worklog.Collect(d.Host.AgentIDs(), openAgentDB(d.Host),
				worklogOrchestrator(), worklogStaleMin(), true, time.Now().UTC())
			n := 0
			for _, it := range items {
				if it.Stale { // NYANGKUT = butuh rekonsiliasi mandor
					n++
				}
			}
			return n
		}
	}})
}
