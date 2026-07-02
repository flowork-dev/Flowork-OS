// provision_dna.go — SATU PINTU DNA agent (2026-06-20).
//
// Pabrik AI Studio: tiap agent baru harus LAHIR sebagai warga penuh — bukan
// nunggu restart. Fungsi ini ngumpulin semua seeding DNA INTRINSIK per-agent
// (konstitusi sacred, edu-errors, antibody immune, schema cognitive graph) jadi
// satu panggilan IDEMPOTENT. Dipanggil dari:
//   - boot loop (main.go) — semua agent existing.
//   - install path (plugin_handler.go) — agent baru hasil AI Studio LANGSUNG.
//
// DNA BERSAMA (instinct/graph/otak-kolektif) TIDAK di-copy ke sini — itu
// di-REFERENSI lewat pipa tool (coreExposedTools: graph_recall / instinct_recall
// / brain_search_shared) ke SHARED brain. Update shared sekali → semua agent
// lihat, ga ada salinan basi.
package agentmgr

import "log"

// DNAResult — ringkasan apa yang baru ke-seed (0 = udah ada, idempotent).
type DNAResult struct {
	EduErrors     int
	Constitution  int
	Antibodies    int
	SelfKnowledge int
	GraphNodes    int
	GraphEdges    int
	Synced        bool
}

// ProvisionAgentDNA seed DNA intrinsik ke satu agent (idempotent, aman dipanggil
// berkali-kali). Error per-langkah di-log, ga nge-fatal — best-effort provisioning
// (1 langkah gagal ga boleh nahan yang lain / nge-brick agent).
func ProvisionAgentDNA(agentID string) DNAResult {
	var res DNAResult
	store, err := openAgentStore(agentID)
	if err != nil {
		log.Printf("provision-dna: buka store %s gagal: %v", agentID, err)
		return res
	}
	defer store.Close()

	if n, e := store.SeedEduErrors(); e == nil {
		res.EduErrors = n
	}
	// CABANG edu-errors (non-frozen, override DO-UPDATE): refresh ERR_TOOL_NOT_FOUND ke jalur
	// self-evolving (tool_create) + ERR_TOOL_GC_REMOVED (deletion-aware). edu_errors_seed.go frozen
	// pakai DO-NOTHING → ga bisa refresh dari sana; ext ini yg nyebar override ke semua agent.
	if n, e := store.SeedEduErrorsExt(); e == nil {
		res.EduErrors += n
	}
	// Konstitusi sacred (5W1H/identity/anti-halu + sync-honest/recall-first).
	// Idempotent-upsert: aturan sacred baru auto-nyebar.
	if n, e := store.SeedSacredConstitution(); e == nil {
		res.Constitution = n
	}
	// Agent EXTENSION (non-primary) ga punya brain_search_shared 5jt → rapihin
	// rule anti-halu biar ga nyuruh pake tool yang ga dia punya. SEBELUM sync.
	if !IsPrimaryAgent(agentID) {
		_, _ = store.TuneConstitutionForExtension()
	}
	if updated, e := store.SyncConstitutionSlot(); e == nil {
		res.Synced = updated
	}
	if n, e := store.SeedAntibodies(); e == nil {
		res.Antibodies = n
	}
	// Self-knowledge: kartu FITUR/SUBSISTEM Flowork (distilasi lock/*.md) → brain lokal
	// tiap agent, biar TAHU kemampuan dirinya (recall via brain_search, bukan ngarang).
	// DNA statik (sekelas konstitusi/antibody), idempotent: no-op kalau room udah keisi.
	if n, e := store.SeedSelfKnowledge(); e == nil {
		res.SelfKnowledge = n
	}
	// owner 2026-06-23 "GUI = KEBENARAN UTAMA, hardcode HARAM": tool default primary (browser/task/
	// sleep-wait/brain-extra) DULU hardcode-force di ToolSpecsHandler — sekarang DI-SEED jadi subscription
	// (SEKALI, sentinel-flag) → owner curate/uncheck per-agent di GUI Tools-catalog. brain_add ditambah
	// (brain INPUT). Idempoten: sentinel cegah re-seed (hormati uncheck owner). Ant (non-primary) ga di-seed.
	if IsPrimaryAgent(agentID) {
		if done, _ := store.IsSubscribed("__primary_seeded_v1__"); !done {
			for _, t := range append([]string{"brain_add"}, primaryExtraTools...) {
				_ = store.SubscribeTool(t, "manual", "{}")
			}
			_ = store.SubscribeTool("__primary_seeded_v1__", "seed", "{}")
			log.Printf("provision-dna %s: seeded %d default primary tools (GUI-curate)", agentID, len(primaryExtraTools)+1)
		}
	}

	// Pastiin schema cognitive graph ke-create (CountCognitiveGraph manggil
	// ensureCognitiveGraphSchema) → graph_recall + digestion siap dari turn-1,
	// ga nunggu operasi cognitive pertama.
	res.GraphNodes, res.GraphEdges = store.CountCognitiveGraph()

	if res.EduErrors+res.Constitution+res.Antibodies+res.SelfKnowledge > 0 || res.Synced {
		log.Printf("provision-dna %s: edu=%d constitution=%d antibody=%d selfknowledge=%d synced=%v graph=%d/%d",
			agentID, res.EduErrors, res.Constitution, res.Antibodies, res.SelfKnowledge, res.Synced, res.GraphNodes, res.GraphEdges)
	}
	return res
}
