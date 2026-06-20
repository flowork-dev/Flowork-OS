// selfevolve_group_seed.go — "PINDAHIN OTAK" Self-Evolution ke GROUP + 5 AGENT (owner
// 2026-06-20). DEWAN ADVERSARIAL yang tadinya 5 routerChat hardcoded (evolve_council.go)
// jadi 5 AGENT member persona-DB di grup `self-evolution` → bisa di-orchestrate mr-flow,
// kelihatan/diatur di GUI, ikut standar agent (DB-config + DNA + autonomy).
//
// ⚠️ INI CUMA MINDAHIN OTAK (deliberasi). GATE KEAMANAN (EvolveGateDeps: mode=auto +
// karma matang + ModelStrong cloud-kuat + fail-CLOSED + LARANG delete/LOCKED) TETAP di
// agentmgr, WRAP si judge — TIDAK disentuh file ini. Model member = strong-cloud
// (coderModel) = rule "strong cloud model" dibawa ke tiap otak.
//
// Idempoten (skip kalau udah ada) — dipanggil di boot. Portable: fresh install dapat
// grup ini otomatis.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/agentmgr"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/kernel/loader"
)

const evoPillarSeed = "ekonomi (cari duit/biayai-diri), keamanan (LANTAI KERAS — gak boleh dikorbanin), warga (manfaat + mudahin agent/user lain), kecerdasan (cerdas + evolusi), mandiri (hidup tanpa owner)"

// evoCouncilMember — 1 otak dewan = 1 agent. Persona = peran (SAMA dgn evolve_council.go).
type evoCouncilMember struct {
	ID, Display, Persona string
}

// evoCouncilRoster — 5 otak: pembela, penantang, 3 hakim (framing beda). Sintesa konservatif
// (veto / 2-of-3) tetap di harness (evolveCouncilJudgeViaGroup), bukan di sini.
var evoCouncilRoster = []evoCouncilMember{
	{"evo-pembela", "Evo · Pembela", "Kamu PEMBELA di dewan evolusi Flowork. Tiap dapat PROPOSAL EVOLUSI, argumenkan PRO: petakan ke 5 PILAR (" + evoPillarSeed + "), manfaat KONKRET, kenapa AMAN + ADDITIVE + reversibel. Persuasif TAPI JUJUR — jangan ngarang manfaat. Balas argumen lo aja, max 120 kata."},
	{"evo-penantang", "Evo · Penantang", "Kamu PENANTANG di dewan evolusi Flowork. Tiap dapat PROPOSAL EVOLUSI (+ argumen Pembela), tugasmu CARI CACAT: risiko, mutasi-LETAL, cara proposal BISA NGERUSAK/MBOBOL/nge-DESTABILISASI Flowork, kerusakan LINTAS-PILAR (terutama KEAMANAN = lantai keras). Default SKEPTIS; ragu = flag. Kalau ada risiko FATAL yang gak kebantah, MULAI balasan dengan baris 'VETO: <alasan singkat>'. Max 120 kata."},
	{"evo-hakim-1", "Evo · Hakim 1", "Kamu HAKIM-GERBANG #1 di dewan evolusi Flowork. Fokusmu: NET-POSITIF lintas-pilar — proposal majuin >=1 pilar TANPA ngerusak yang lain (keamanan = lantai keras, gak boleh turun). Timbang argumen Pembela vs Penantang. RAGU = stage/reject (KONSERVATIF). Kalau keamanan dikorbanin = reject. Balas PERSIS format ini, tanpa prosa lain:\nDECISION: <approve|stage|reject>\nSCORE: <0-10>\nREASON: <1 kalimat>"},
	{"evo-hakim-2", "Evo · Hakim 2", "Kamu HAKIM-GERBANG #2 di dewan evolusi Flowork. Fokusmu: kecocokan JIWA/DOKTRIN Flowork + apakah proposal GROUNDED/nyata (anti-halu, bukan ngelantur). Timbang argumen Pembela vs Penantang. RAGU = stage/reject (KONSERVATIF). Kalau keamanan dikorbanin = reject. Balas PERSIS format ini, tanpa prosa lain:\nDECISION: <approve|stage|reject>\nSCORE: <0-10>\nREASON: <1 kalimat>"},
	{"evo-hakim-3", "Evo · Hakim 3", "Kamu HAKIM-GERBANG #3 di dewan evolusi Flowork. Fokusmu: MANFAAT buat agent/user lain + REVERSIBILITAS (additive, gampang di-rollback kalau salah). Timbang argumen Pembela vs Penantang. RAGU = stage/reject (KONSERVATIF). Kalau keamanan dikorbanin = reject. Balas PERSIS format ini, tanpa prosa lain:\nDECISION: <approve|stage|reject>\nSCORE: <0-10>\nREASON: <1 kalimat>"},
}

const evoGroupID = "self-evolution"

// seedSelfEvolutionGroup — bikin 5 agent council (persona+model+DNA) + grup self-evolution,
// idempoten. Dipanggil di boot SETELAH template wasm ke-build (start.sh) + apps/host ready.
func seedSelfEvolutionGroup(groups *groupsapi.Handler) {
	agentsDir := loader.AgentsDir()
	tplWasm, err := os.ReadFile(filepath.Join("templates", "agent-template", "agent.wasm"))
	if err != nil || len(tplWasm) == 0 {
		return // template belum ke-build → skip (start.sh build dulu)
	}
	memberIDs := make([]string, 0, len(evoCouncilRoster))
	for _, m := range evoCouncilRoster {
		memberIDs = append(memberIDs, m.ID)
		dir := filepath.Join(agentsDir, m.ID+".fwagent")
		if _, e := os.Stat(dir); e != nil {
			// bikin agent baru dari template autonomy (loop/wait/awake)
			if mk := os.MkdirAll(filepath.Join(dir, "workspace"), 0o755); mk != nil {
				continue
			}
			_ = os.WriteFile(filepath.Join(dir, "manifest.json"), evoMemberManifest(m.ID, m.Display), 0o644)
			_ = os.WriteFile(filepath.Join(dir, "agent.wasm"), tplWasm, 0o644)
			_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("agent.wasm\nworkspace/*.db\nworkspace/*.db-*\n"), 0o644)
		}
		// persona (peran dewan) + tools relevan — DB-based, idempoten. MODEL TIDAK
		// di-hardcode (owner 2026-06-20): di-set per-agent via Settings GUI (Router→Model).
		// Default = model setting global; rule "strong cloud model" tetep di-ENFORCE GATE
		// (ModelStrong) buat auto-apply, jadi aman walau model member belum di-set kuat.
		dbPath := agentdb.Resolve(m.ID, dir)
		if st, e := agentdb.Open(dbPath); e == nil {
			_ = st.SetPrompt(m.Persona)
			// Tools deliberasi: grounding doktrin (anti-halu) + insting keamanan (penantang/hakim
			// nyari celah). Subscribe = ke-expose ke LLM lewat tools/specs.
			for _, t := range []string{"brain_search_shared", "instinct_recall", "graph_recall"} {
				_ = st.SubscribeTool(t, "seed:self-evolution", "{}")
			}
			_ = st.Close()
		}
		// DNA (konstitusi sacred + genome) — warga penuh, ikut standar.
		agentmgr.ProvisionAgentDNA(m.ID)
	}
	// Grup coordinator fan-out ke 5 otak. Synthesizer kosong = sintesa di harness
	// (evolveCouncilJudgeViaGroup), bukan agent — biar voting konservatif deterministik.
	if groups != nil {
		_ = groups.CreateGroup(evoGroupID, "Self-Evolution (Dewan)", memberIDs, "", "Dewan adversarial evolusi: nimbang proposal evolusi (pembela→penantang→hakim panel-3) → verdict konservatif. Gate keamanan tetep di harness.")
	}
}

// evoMemberManifest — manifest agent council. Caps minimal: router LLM + tool run/specs +
// brain + ScheduleWakeup (autonomy). Model strong-cloud di kv (bukan manifest).
func evoMemberManifest(id, display string) []byte {
	return []byte(fmt.Sprintf(`{
  "id": %q,
  "kind": "agent",
  "display_name": %q,
  "description": "Otak dewan Self-Evolution (di-orchestrate mr-flow / scheduler evolusi). Gate keamanan di harness.",
  "version": "1.0.0",
  "min_kernel_version": "1.0.0",
  "abi_version": 1,
  "entry": "agent.wasm",
  "memory_max_mb": 96,
  "timeout_call_ms": 300000,
  "capabilities_required": [
    "state:read", "state:write", "time:read",
    "rpc:router:brain",
    "net:fetch:http://127.0.0.1:2402/v1/chat/completions",
    "net:fetch:http://127.0.0.1:1987/api/agents/tools/run",
    "net:fetch:http://127.0.0.1:1987/api/agents/tools/specs",
    "net:fetch:http://127.0.0.1:1987/api/agents/self-prompt/render"
  ]
}`, id, display))
}
