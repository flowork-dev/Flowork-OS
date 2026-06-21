// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Mr.Dev · floworkos.com
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-22
// Reason: B4 (roadmap_brain.md #4) — auto-sync sumber→Cognitive Graph. Ganti
//
//	re-run manual `_scratch_cgm/graphsync` jadi ticker host idempotent: projeksi
//	skills/constitution/edu-errors/brain_drawers mr-flow ke graph (UpsertNode +
//	embedding bge-m3) biar graph SELALU cermin sumber (unified-recall ga basi).
//	CHANGE-DETECTION: skip embed kalau label node == sumber (hemat router :2402);
//	cuma row BARU/BERUBAH yang re-embed. Throttle 30min. Additive + data-only
//	(sumber TIDAK diubah; graph cuma mirror). Reuse pola graphsync. Privasi D8:
//	konten owner (constitution/drawer) LOKAL only — graph per-agent ga ke mesh.
//	Host-side non-frozen (logic-brain inti = FROZEN di internal/agentdb).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/kernelhost"
	"flowork-gui/internal/routerclient"
)

const (
	graphSyncInterval = 30 * time.Minute
	graphSyncAgent    = "mr-flow" // agent kanonik (brain kaya); scope sama kaya graphsync B1
)

var (
	lastGraphSync    time.Time
	graphSyncBrandRe = regexp.MustCompile(`(?i)\b(claude|anthropic|fable|gemini|opus|sonnet|haiku|chatgpt|openai)\b`)
)

func gsTrim(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// SyncSourcesToGraph (B4) — projeksi sumber lokal (skills/constitution/edu/drawer) mr-flow
// ke cognitive graph, IDEMPOTENT + change-detection (skip embed kalau label SAMA → hemat
// router). Throttle graphSyncInterval. Balikin jumlah node BARU/BERUBAH yg di-project.
// Dipanggil dari ticker host main.go. roadmap_brain.md #4.
func SyncSourcesToGraph(ctx context.Context, host *kernelhost.Host) int {
	if time.Since(lastGraphSync) < graphSyncInterval {
		return 0
	}
	lastGraphSync = time.Now()

	store, err := host.OpenAgentStore(graphSyncAgent)
	if err != nil {
		return 0
	}
	defer store.Close()
	db := store.DB()

	// change-detection: id→label node existing (skip re-embed kalau ga berubah).
	existing := map[string]string{}
	if nodes, e := store.ListCogNodes(5000); e == nil {
		for _, n := range nodes {
			existing[n.ID] = n.Label
		}
	}

	rc := routerclient.New("")
	scope := "agent:" + graphSyncAgent
	changed := 0

	put := func(id, label, typ, domain, props string) {
		if graphSyncBrandRe.MatchString(label) {
			return // white-label gate (D8)
		}
		if existing[id] == label {
			return // unchanged → skip embed (hemat router)
		}
		ectx, cc := context.WithTimeout(ctx, 30*time.Second)
		vec, e := rc.EmbedText(ectx, "", label)
		cc()
		if e != nil {
			return
		}
		if _, ue := store.UpsertNode(agentdb.CogNode{
			ID: id, Label: label, Type: typ, WhereDomain: domain, Properties: props,
			SourceKind: "verified", SourceRef: id, Confidence: 0.9, Status: "active",
			Embedding: agentdb.Quantize(vec),
		}); ue == nil {
			changed++
		}
	}

	// Skills → type=skill
	if rows, e := db.Query(`SELECT id, COALESCE(trigger,''), COALESCE(instructions,'') FROM skills WHERE COALESCE(archived,0)=0`); e == nil {
		for rows.Next() {
			var id, trig, instr string
			if rows.Scan(&id, &trig, &instr) != nil {
				continue
			}
			name := strings.TrimPrefix(id, "skill:")
			label := "SKILL " + name + ": " + gsTrim(trig, 200)
			if trig == "" {
				label = "SKILL " + name + ": " + gsTrim(instr, 200)
			}
			put(scope+"/skill/"+name, label, "skill", "agent-skill", `{"skill_id":"`+id+`"}`)
		}
		rows.Close()
	}

	// Constitution → type=doctrine (sacred)
	if rows, e := db.Query(`SELECT id, rule, COALESCE(sacred,0) FROM constitution`); e == nil {
		for rows.Next() {
			var id, rule string
			var sacred int
			if rows.Scan(&id, &rule, &sacred) != nil {
				continue
			}
			props := fmt.Sprintf(`{"sacred":%v,"rule_id":"%s"}`, sacred == 1, id)
			put(scope+"/constitution/"+id, gsTrim(rule, 380), "doctrine", "constitution", props)
		}
		rows.Close()
	}

	// Edu errors → type=edu_error
	if rows, e := db.Query(`SELECT code, COALESCE(category,''), COALESCE(title,''), COALESCE(remediation,'') FROM educational_errors_cache`); e == nil {
		for rows.Next() {
			var code, cat, title, rem string
			if rows.Scan(&code, &cat, &title, &rem) != nil {
				continue
			}
			label := code + " (" + cat + "): " + gsTrim(title, 80) + " -> " + gsTrim(rem, 240)
			props, _ := json.Marshal(map[string]any{"code": code, "category": cat})
			put(scope+"/edu/"+code, gsTrim(label, 400), "edu_error", "education", string(props))
		}
		rows.Close()
	}

	// Knowledge drawers (brain_drawers lokal) → type=knowledge. Konten owner LOKAL only (D8).
	if rows, e := db.Query(`SELECT id, content, COALESCE(wing,''), COALESCE(room,'') FROM brain_drawers WHERE deleted_at IS NULL`); e == nil {
		for rows.Next() {
			var id, content, wing, room string
			if rows.Scan(&id, &content, &wing, &room) != nil {
				continue
			}
			dom := wing
			if dom == "" {
				dom = "general"
			}
			props, _ := json.Marshal(map[string]any{"drawer_id": id, "wing": wing, "room": room})
			put(scope+"/knowledge/drawer-"+id, gsTrim(content, 400), "knowledge", dom, string(props))
		}
		rows.Close()
	}

	return changed
}
