// Owner: Mr.Dev · github.com/flowork-os/Flowork-OS · floworkos.com
// ⚠️ FROZEN brain-core — jangan edit tanpa unfreeze owner. Arsitektur & alasan: lihat lock/brain.md
//
// cognitive_antibody_job.go — F4 host orchestrator: ANTIBODY KOLEKTIF lintas-agent.
//
// Recovery-instinct (INC-3) yg ditemukan INDEPENDEN oleh >=N agent (kelas-error sama) =
// pola terbukti umum → push ke SEMUA agent yg belum punya + mark collective (conf 0.95).
// Imunitas kolektif: sekali cukup agent kebal suatu error, semua agent ikut kebal.
//
// DORMANT pas cuma 1 agent yg punya recovery suatu kelas (ga ada konvergensi) → 0 dampak.
// Logic per-agent di agentdb/cognitive_antibody.go. Privacy-safe by-construction (INC-3).

package main

import (
	"context"
	"log"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/kernelhost"
	"flowork-gui/internal/routerclient"
)

// antibodyMinAgents — kelas recovery yg dipunyai >= sekian agent (independen) → naik jadi
// antibody kolektif. 2 = minimal "lebih dari satu agent setuju" (anti-noise 1-agent).
const antibodyMinAgents = 2

// antibodyPlan — rencana per kelas: label terbaik + agent yg UDAH punya.
type antibodyPlan struct {
	label string
	conf  float64
	have  map[string]bool
}

// computeCollectiveAntibodies — PURE: dari peta agent→[kelas recovery], cari kelas yg
// dipunyai >=minAgents agent DISTINCT. Return kelas→plan (label terbaik + set agent yg punya).
// Testable tanpa I/O.
func computeCollectiveAntibodies(perAgent map[string][]agentdb.ClassLabel, minAgents int) map[string]*antibodyPlan {
	plans := map[string]*antibodyPlan{}
	for agent, classes := range perAgent {
		for _, cl := range classes {
			p := plans[cl.Class]
			if p == nil {
				p = &antibodyPlan{have: map[string]bool{}}
				plans[cl.Class] = p
			}
			p.have[agent] = true
			if cl.Confidence > p.conf {
				p.conf, p.label = cl.Confidence, cl.Label
			}
		}
	}
	out := map[string]*antibodyPlan{}
	for class, p := range plans {
		if len(p.have) >= minAgents {
			out[class] = p
		}
	}
	return out
}

// PromoteCollectiveAntibodies — gather kelas recovery lintas agent → kelas konvergen
// (>=N agent) → push ke agent yg belum punya. Return jumlah push. Resilient.
func PromoteCollectiveAntibodies(ctx context.Context, host *kernelhost.Host) int {
	agentIDs := host.AgentIDs()
	perAgent := map[string][]agentdb.ClassLabel{}
	for _, id := range agentIDs {
		store, err := host.OpenAgentStore(id)
		if err != nil {
			continue
		}
		perAgent[id] = store.ActiveRecoveryClasses()
		store.Close()
	}
	plans := computeCollectiveAntibodies(perAgent, antibodyMinAgents)
	if len(plans) == 0 {
		return 0
	}
	rc := routerclient.New("")
	pushed := 0
	for class, p := range plans {
		// embed label sekali (buat recall by-makna di agent tujuan).
		ectx, cancel := context.WithTimeout(ctx, 30*time.Second)
		vec, eerr := rc.EmbedText(ectx, "", p.label)
		cancel()
		if eerr != nil {
			continue
		}
		emb := agentdb.Quantize(vec)
		for _, id := range agentIDs {
			if p.have[id] {
				continue // udah punya
			}
			store, err := host.OpenAgentStore(id)
			if err != nil {
				continue
			}
			scope := "agent:" + id
			if !store.HasActiveRecoveryClass(scope, class) {
				if _, uerr := store.UpsertCollectiveAntibody(scope, class, p.label, emb); uerr == nil {
					pushed++
					log.Printf("[antibody] kelas '%s' → push ke %s (collective, %d agent setuju)", class, id, len(p.have))
				}
			}
			store.Close()
		}
	}
	return pushed
}
