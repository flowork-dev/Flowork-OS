# STANDAR AGENT FLOWORK — Kontrak Tiap Warga

> Detail dicontek dari `/home/mrflow/Documents/roadmap_agent.md` (§1-6) + hasil sesi
> 2026-06-20. Ini standar WAJIB tiap agent (mr-flow = template referensi). Tiap entitas
> mikir (agent crew, AI Studio, Evolution) ikut kontrak SAMA.

Agent = **OTAK LOKAL**: punya `workspace/state.db` sendiri (terisolasi total). Config &
skill **di DATABASE, BUKAN file .md** (prinsip visi owner: ".md = file sampah"). Loader
scan dir `*.fwagent` di `~/.flowork/agents/` (data-home).

---

## 1. DUA-LAPIS OTAK (gimana nyambung)
```
ROUTER (:2402) — OTAK BERSAMA (shared)          ← Knowledge 859K · Instinct · KG-collective ·
   ▲                                              Constitution-umum · Persona-library · Skill Registry
   │ capability rpc:router:brain (HTTP)
   │ PIPA tool: brain_search_shared, instinct_recall, graph_recall
AGENT (:1987) — OTAK LOKAL (state.db)           ← persona DB · brain lokal (drawers/recall) ·
                                                  cognitive graph lokal · interactions/karma/dll
```
**Pembagian:** UMUM (instinct/knowledge/KG-collective) → router. PERSONAL (twin owner:
sejarah/keluarga) → agent LOKAL (HARAM ke router/mesh).

---

## 2. 8 SYARAT WAJIB TIAP AGENT (§5 roadmap)
Tiap agent baru WAJIB lahir dengan ini. Sebagian **otomatis dari HOST** (ga perlu di-kode
di wasm) — ditandai 🤖; sebagian di **manifest/persona** — ditandai 📝.

1. 📝 **Config DB-based** — persona di `config.prompt` + `self_prompt`(versioned) +
   `constitution`(sacred). HARAM baca `.md`. (Pola mr-flow `agents/mr-flow/main.go`.)
2. 🤖 **Konstitusi sacred** — auto-seed via `ProvisionAgentDNA` (idempotent, boot+install).
   8 rule: 5w1h-gate, identity-guard, anti-halu, 00-mission-sacred, 01-core-behaviors,
   sync-honest, recall-first, **autonomy-mode**. Auto-nyebar ke SEMUA agent.
3. 🤖 **Pipa OTAK BERSAMA (genome)** — `coreExposedTools` auto-kasih `brain_search_shared`
   + `instinct_recall` + `graph_recall` → tiap agent nyolok Knowledge/Instinct/KG router.
4. 🤖 **Cognitive Graph LOKAL** — `cognitive_nodes/edges` di state.db sendiri (schema
   ke-ensure ProvisionAgentDNA), diisi via dream/digest, dibaca via `graph_recall`.
   Memori relasional per-agent (owner-facing→twin; lain→konsep domain).
5. 📝 **Identitas penting masuk BRAIN/recall**, bukan cuma persona — 26B percaya
   recall(brain) > persona (§4.4). Fakta owner WAJIB ke drawer/recall.
6. 📝 **Capability `rpc:router:brain`** di manifest.
7. 🤖 **Schedule per-agent** — cron→task (agentdb `schedules` + engine auto-tick/menit).
   Fungsional, KEEP (beda dari global schedule).
8. 🤖 **Autonomus PENUH (loop/wait/awake/auto-continue)** — embedded di wasm template
   (§8): tool-loop (chain), ghost-guard, ScheduleWakeup (tidur→bangun via RunDueWakeups),
   loop TIME-BOUND (bukan cap-angka), + AUTO-CONTINUE deterministik (budget abis →
   harness jadwalin lanjutan sendiri → nyambung lintas-turn sampe SELESAI). Worker baru
   dari template otomatis dapat semua ini. Detail = §8.

---

## 3. KONEK KE APP (izin-app, GUI=truth)
App = platform dipakai-bersama user+agent (`~/.flowork/apps/<id>/`). Agent pakai app
lewat **tool** `app_<id>_<op>` (hyphen→`_`), di-gate **capability `app:<id>`**.
- **Izin per-agent** disimpen di tabel `app_grants` (state.db) — di-toggle di GUI agent
  (Setting → Tools → "Aplikasi yang boleh dipakai"). Centang → cap di-grant LIVE.
- App WAJIB punya ≥1 operasi `"tool": true` (konektor agent) biar bisa dipakai agent;
  kalau 0 = user-only. (Detail: `agent/apps/README.MD`.)
- Backend: `app_grants_handler.go` + `internal/agentdb/app_grants.go`.

## 4. KONEK KE BRAIN (shared + lokal)
- **Shared (router):** `brain_search_shared` (Knowledge 859K by-makna), `instinct_recall`
  (insting coding/security), `graph_recall` (KG). Butuh cap `rpc:router:brain`. Auto via genome.
- **Lokal (agent):** `brain_search` (drawer lokal), `memory_get/set` (USER.md/MEMORY.md),
  recall twin. Privat, HARAM ke router.

## 5. KONEK KE KONSTITUSI/DNA
`ProvisionAgentDNA(agentID)` (host, boot+install) seed: konstitusi sacred + genome pipe +
cognitive-graph schema + edu-errors + antibody immune. Agent baru = warga penuh tanpa nunggu
restart. Constitution decision (owner, A): agent-sacred canonical di-inject; router-65
TIDAK dipaksa-inject (lapis shared, di-query).

## 6. ROUTING (3-jalur, sejak 2026-06-20)
Dulu: `mr-flow→group→agent` (selalu via group). SEKARANG mr-flow (& orchestrator) bisa:
1. **Jawab sendiri** — pake tool (web_search/brain/app).
2. **DIRECT ke 1 agent** — `agent_command(agent_id, text)` (cap `rpc:agent-invoke`).
3. **GROUP/crew** — `task_run(category, subject)` (cek `task_list` live dulu).
Anti-halu: jangan ngarang agent_id/run_id; kalau ga ada target → jawab sendiri.

---

## 7. MANIFEST MINIMAL (skeleton)
```json
{
  "id": "<agent-id>", "kind": "agent", "display_name": "<Nama>",
  "version": "1.0.0", "abi_version": 1, "entry": "agent.wasm",
  "memory_max_mb": 64, "timeout_call_ms": 300000,
  "capabilities_required": [
    "state:read", "state:write", "time:read",
    "rpc:router:brain",                       // otak bersama (brain/instinct/KG)
    "net:fetch:http://127.0.0.1:2402/v1/chat/completions",  // LLM router
    "net:fetch:http://127.0.0.1:1987/api/agents/tools/run", // jalanin tool
    "net:fetch:http://127.0.0.1:1987/api/agents/tools/specs"
    // + "rpc:agent-invoke" kalau orchestrator (direct-agent)
    // + "app:<id>" lewat app_grants (GUI), JANGAN hardcode app worker
  ],
  "exposes_rpc": [{ "name": "handle_message",
    "input_schema": {"type":"object","properties":{"text":{"type":"string"}}} }]
}
```
Caps app (`app:*`) JANGAN ditaruh manifest — itu di-grant lewat **app_grants/GUI** (GUI=truth).
Caps tool subscribed (net:fetch:*/exec:*) buat agent privileged auto-grant dari subscription
(`grantSubscribedToolCaps`, main.go).

## 8. KONTRAK WASM + AUTONOMY (agent code)
Agent = "bodoh, engine pinter" — behavior dari persona DB + DNA, bukan wasm. Tapi
**autonomy-loop ada DI wasm** (harness), dan **template `templates/agent-template/` udah
embed semua ini** (2026-06-20, sama kayak mr-flow) → tiap agent baru langsung bisa
looping/wait/awake:
- `selfID()` baca `FLOWORK_AGENT_ID` (= manifest.id) → wasm SAMA jalan buat agent apa pun.
- RPC `handle_message`: system = persona(config.prompt) + **DNA** (`/api/agents/self-prompt/
  render` field `rendered` = konstitusi sacred + directive) ; tools = `/api/agents/tools/specs`
  (subscription). → **callLLM TOOL-LOOP**.
- **LOOPING**: LLM → eksekusi tool (`/api/agents/tools/run`) → feed hasil → ulang (chain),
  1 tool/iter (`parallel_tool_calls:false`, anti router-400). Bukan 1 call doang.
- **GHOST-GUARD**: model narasi "mau ngapain / lanjut ke X" TANPA manggil tool → PAKSA 1
  putaran (panggil tool / ScheduleWakeup), bounded `maxGhostNudges`. Anti janji-kosong.
- **WAIT/AWAKE (sleep→wake)**: tool `ScheduleWakeup(delaySeconds, reason, prompt)` → tulis
  row `wakeups` durable → host `RunDueWakeups` (per-menit) re-invoke agent dgn `prompt`.
  Wakeup invoke budget = **300s** (SAMA turn normal, owner 2026-06-20) — JANGAN lebih kecil dari
  `loopBudgetMs`, kalau ngga tugas panjang yg di-RESUME via wakeup ke-cut sebelum sempat
  checkpoint+jadwalin lanjutan → rantai PUTUS = mati di tengah. Agent kebangun sendiri & lanjut.
- **TIME-BOUND, BUKAN cap-angka** (`loopBudgetMs≈200s`): loop jalan TERUS dalam budget
  waktu turn (turn-timeout 290s = backstop). Kerja autonomus panjang ga dikerangkeng angka.
- **AUTO-CONTINUE deterministik (unbounded over time)**: budget abis & BELUM kelar →
  HARNESS sendiri jadwalin ScheduleWakeup (`[LANJUTAN OTOMATIS #N] ...===TUGAS===<task>`)
  → nyambung lintas-turn sampe model bilang `SELESAI` (kelar dalam budget) ATAU `maxAuto
  Continue=50` (anti-runaway). Counter #N ride di prompt (stateless). GA ngandelin model
  milih (deterministik). = "kerja seharian" dipotong chunk yang nyambung otomatis.
- Build: **standard wasip1** (`GOWORK=off GOOS=wasip1 GOARCH=wasm go build`), BUKAN tinygo.

> Manifest WAJIB subscribe `ScheduleWakeup` (+ tools/specs, tools/run, router) biar
> wait/awake + loop jalan. Worker baru dari template otomatis dapat semua ini.

## 9. BUILD + DEPLOY
Repo ga simpan `agent.wasm` (gitignore). Build → deploy ke `~/.flowork/agents/<id>.fwagent/
agent.wasm` (yang ke-load loader) → restart. Lihat `agents/mr-flow/readme.md` (anti-ketuker
source vs deployed).

---

## 10. AUTO-COMPACT KONTEKS (anti-halu) — owner 2026-06-20
Konteks panjang → AI halu. Tiap agent WAJIB bisa di-compact: **digest pengalaman ke brain
(cognitive graph) → trim interaksi lama**. Pengalaman TIDAK hilang (pindah ke brain, recallable),
cuma raw interaksi lama di-soft-delete. **FATAL-safe** (urutan ga boleh kebalik):

1. Ukur konteks (`CompactStats`: interaksi non-deleted + undigested).
2. **DIGEST** pending → brain (`DigestAgent`, idempoten via `cognitive_digest_log`).
3. **VERIFY** undigested==0.
4. **TRIM** (`TrimDigestedInteractions`): soft-delete HANYA yg UDAH di brain, sisain N terbaru.
   Yg belum di-digest TIDAK pernah disentuh (teruji `TestAutoCompactSafety`).

**Trigger 3 cara:** (a) OTOMATIS cron 15-mnt (agent yg lewat ambang). (b) MANUAL di **menu Agent**:
`🧠 Compact Now` (per-agent) + `🧠 Compact All` (semua, async). (c) **TRIGGER terjadwal**:
target `system:compact-all` (menu Triggers). Ambang di **Settings → 🧠 Auto-Compact**
(`compact_max_interactions`/`compact_keep_recent`/`compact_enabled` di KV global).

Aturan: JANGAN compact agent mid-task (skip kalau ada aktivitas <90s); digest gagal → JANGAN trim.

---

## 11. EVOLUSI: LOOP REVIEWER↔FIXER + ANTI-TIMER (khusus tim evolusi)
Khusus core-apply evolusi (bukan tiap agent, tapi standar tim evo): evo-coder GENERATE → evo-reviewer
AUDIT (keamanan korpus-hacking via instinct_recall + kualitas) → TEMUAN → coder FIX → ulang
sampai `LOLOS`. **DOKTRIN ANTI-TIMER**: JANGAN kasih wall-clock ke loop evolusi ("dibatasin =
mati di tengah"). Stop = konvergensi (LOLOS) / no-progress detector (temuan sama 2×). Per-call
timeout (300s) OK (anti-hang). Core-apply ASYNC (status `coding`, ga blok HTTP).

---
**Referensi:** template baru = `templates/agent-template/`. Standar penuh + alasan tiap
poin = `roadmap_agent.md`. Otak/memori (CGM, instinct, dream) = `roadmap_opus8.md`.
