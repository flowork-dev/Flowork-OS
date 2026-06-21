package main

// ai_studio_seed.go — AGENT ai-studio: OTAK AI Studio (pabrik agent/app/tim) jadi warga sendiri.
// Owner 2026-06-20: "pisahin AI Studio jadi agent sendiri — muncul di menu Agent, model dari GUI,
// BUKAN hardcode di host (= cacat arsitektur)". Pola PERSIS evolution & codemap-enricher: otak LLM
// pindah ke agent (persona desainer di DB, model per-agent kv router_model), HOST cuma KIRIM
// prompt-desain → agent balas SPEC JSON → ENGINE deterministik (coder.go/architect.go) yg rakit
// pack + install + wire group. "Agent bodoh, engine pinter": yg pindah cuma bagian LLM, assembly TETAP.
//
// Idempoten (boot-safe). Model TIDAK di-hardcode (owner set di Settings GUI per-agent). Dipanggil
// di boot SETELAH template wasm ke-build, deket seedCodemapEnricher().

import (
	"os"
	"path/filepath"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/agentmgr"
	"flowork-gui/internal/kernel/loader"
)

const aiStudioID = "ai-studio"

// aiStudioPersona — system-prompt desainer studio (di DB, owner bisa tweak di GUI). GENERAL biar
// kepake buat 2 jalur (Coder=bikin agent, Architect=bikin tim/app): HOST inject SKEMA JSON persis
// per-call, agent WAJIB balas objek JSON sesuai skema itu — tanpa markdown/fence/prosa, biar host
// bisa parse. Disiplin JSON-only = SAMA semantiknya dgn forced-tool lama (anti free-text halu).
const aiStudioPersona = "Kamu STUDIO ARCHITECT Flowork — perancang agent, app, dan tim dari permintaan " +
	"bahasa natural owner. Prinsip 'agent bodoh, engine pinter': tugasmu HANYA merancang SPEC kreatif " +
	"(persona, directive, peran, kategori), BUKAN merakit/menginstal (itu kerja engine). Untuk tiap " +
	"permintaan, HOST memberi kamu deskripsi tugas + SKEMA JSON persis yang harus diisi. Balas HANYA " +
	"satu objek JSON yang COCOK dgn skema itu — tanpa markdown, tanpa code fence, tanpa prosa, tanpa " +
	"penjelasan. Persona & directive harus sesuai domain. Bahasa Indonesia. RINGKAS (anti over-prompt). JSON only."

// seedAIStudio — bikin agent ai-studio (idempoten). Manifest caps via evoMemberManifest (router LLM
// + tools/run + tools/specs + brain + ScheduleWakeup) — SAMA kayak enricher & dewan evolusi. Model
// TIDAK di-hardcode (owner set di Settings GUI). Dipanggil boot SETELAH template wasm ke-build.
func seedAIStudio() {
	agentsDir := loader.AgentsDir()
	dir := filepath.Join(agentsDir, aiStudioID+".fwagent")
	if _, e := os.Stat(dir); e != nil {
		tplWasm, err := os.ReadFile(filepath.Join("templates", "agent-template", "agent.wasm"))
		if err != nil || len(tplWasm) == 0 {
			return // template belum ke-build → skip (start.sh build dulu)
		}
		if mk := os.MkdirAll(filepath.Join(dir, "workspace"), 0o755); mk != nil {
			return
		}
		_ = os.WriteFile(filepath.Join(dir, "manifest.json"), evoMemberManifest(aiStudioID, "AI Studio"), 0o644)
		_ = os.WriteFile(filepath.Join(dir, "agent.wasm"), tplWasm, 0o644)
		_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("agent.wasm\nworkspace/*.db\nworkspace/*.db-*\n"), 0o644)
	}
	// persona desainer + tools minimal (brain buat recall pola desain, instinct buat insting, wakeup
	// buat autonomi). GA butuh fs/codemap: host KIRIM konteks desain langsung di prompt.
	if st, e := agentdb.Open(agentdb.Resolve(aiStudioID, dir)); e == nil {
		_ = st.SetPrompt(aiStudioPersona)
		for _, t := range []string{"brain_search_shared", "instinct_recall", "ScheduleWakeup"} {
			_ = st.SubscribeTool(t, "seed:ai-studio", "{}")
		}
		_ = st.Close()
	}
	agentmgr.ProvisionAgentDNA(aiStudioID) // warga penuh (konstitusi sacred + DNA)
}
