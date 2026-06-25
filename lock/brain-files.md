# BRAIN — PETA FILE & KEPUTUSAN TEKNOLOGI (pecahan `lock/brain.md`)

> Bagian arsitektur memori Flowork. Overview + index: **`lock/brain.md`**. Topik file ini: peta file lengkap (file→peran) · keputusan teknologi (kenapa).
> ⚠️ KE-TRACK repo → NOL data personal owner.

---

## 9. PETA FILE LENGKAP (file → peran)

**agentdb (data + logika memori):**
- `cognitive_graph.go` — CogNode/CogEdge struct + UpsertNode/ListCogNodes (substrat).
- `cognitive_recall.go` — SearchNodesByEmbedding + RecallFactSheet (recall semantic).
- `cognitive_resolve.go` — Quantize (8-bit) + ResolveByEmbedding (dedup/entity-resolution).
- `cognitive_extract.go` / `cognitive_dream.go` — ekstraksi + digestion node dari interaksi.
- `cognitive_gate.go` — validation gate (anti-halu sebelum masuk graph).
- `cognitive_coref.go` — identity alias (co-reference, anti-fragmentasi identitas).
- `cognitive_temporal.go` — fakta berubah seiring waktu (versioning).
- `cognitive_heal.go` — self-heal graph (integrity).
- `cognitive_embed_backfill.go` — isi embedding node lama.
- `cognitive_codemap.go` — codemap (struktur kode dirinya) ke graph.
- `brain_drawers.go` — drawer verbatim + FTS5.
- `mistakes.go` / `mistakes_promote.go` / `mistakes_recall.go` — jurnal mistake + gerbang promote + recall.
- `recovery_generalize.go` — **D32 INC-3** generalisasi recovery-instinct (Lapis A strip privasi + Lapis B coarsen LLM + `GeneralizeRecovery` shadow + `PromoteRecoveryShadows` gerbang; dedup by kelas-error deterministik).
- `edu_errors_seed.go` / `edu_errors.go` — katalog doktrin edukasi (statis, 28).
- `constitution.go` — 8 aturan sacred.

**tools/builtins (jembatan LLM ↔ memori):**
- `cognitive_tools.go` (graph_recall) · `instinct_recall.go` · `brain.go` (shared) · `brain_local.go` (lokal) · `brain_immune.go` (antibody) · `mistakes_recall.go` · `codemap_tools.go` · `v9_extras.go` (tool_search) · `claude_tools.go` (Task/Schedule/etc).
- `tool_specs.go` (agentmgr) — gerbang tool MANA yang di-expose ke LLM (core + primaryExtra + subscription, cap 51).

**host non-beku (orkestrasi loop):**
- `agent/main.go` — wiring + ticker (1 menit: RunDueWakeups, RunQueuedTasks, PromoteRecurringMistakes).
- `wakeup_engine.go` (ScheduleWakeup) · `task_worker.go` (background task) · `mistake_promote_job.go` (D32 INC-1 promote) · `graph_autosync.go` (**B4** auto-sync sumber→graph, ticker+change-detection, FROZEN) · `dream_digester_seed.go` (digest agent) · `learning_feed.go`/`learning_log.go` (3E).

**agent-side mr-flow brain (FROZEN, di-panggil dari main.go):**
- `agents/mr-flow/recovery_capture.go` (**D32 INC-2** capture error→recovery; nano-modular: logic-brain terpisah dari orkestrator main.go).
- `agents/mr-flow/recall_gate.go` (**N1-C** gate auto-recall `isTrivialChat`+`trivialChatTokens`; nano-modular: di-ekstrak dari main.go, FROZEN).
- `agents/mr-flow/working_set.go` (**D18-P1** `activeTaskFor`: TUGAS AKTIF persist lintas-sesi via kv; nano-modular: di-ekstrak dari main.go, FROZEN).

**routerclient (jembatan ke router):**
- `embed.go` (EmbedText → bge-m3) · routerclient (ChatComplete → LLM).

**GUI:** `web/tabs/cognitive.js` · `agentmgr/cognitive_handlers.go`.

**scratch projector (`_scratch_cgm/`, gitignored — tool sekali-pakai, BUKAN bagian runtime):** instproj · graphsync · graphwire · secinstinct · redistil · addinstinct.

---

## 10. KEPUTUSAN TEKNOLOGI (kenapa)

| Pilihan | Kenapa |
|---|---|
| **SQLite (pure-Go modernc, WAL)** | Portable/plug-and-play/multi-OS, no server, embedded 1-file. Per-agent isolasi. WAL = concurrent read + 1 writer. |
| **bge-m3 embedding (dim 1024)** | Multilingual, kualitas semantic bagus, bisa lokal (di router). Recall by-makna lintas bahasa. |
| **8-bit quantize embedding** | 1 byte/dim (vs 4) → hemat 4× storage, ~99% recall kejaga. Pola vecindex router. |
| **Embedding di ROUTER (bukan tiap agent)** | Mesin berat → 1 instance shared, agent pinjem hitungan. |
| **FTS5/BM25 (brain_fts)** | Recall verbatim/keyword cepat (komplemen semantic). |
| **Cognitive Graph (node+edge, W5H1)** | Memori terstruktur + relasi + 1 substrat pemersatu buat recall lintas-subsistem + viz. |
| **Recall by-embedding (node melayang) buat instinct** | Insting = "kalau situasi X" → cocok by-MAKNA, ga butuh edge eksplisit. Skala besar (ribuan) tanpa ledakan edge. |
| **2-tier brain (lokal + router shared)** | Privasi (D8): data personal di lokal, pengetahuan umum di shared. |
| **Reasoning di AGENT (model GUI), host orkestrasi** | Mandat AI-in-agent: model swappable per-agent dari GUI, bukan hardcode. |
| **Worker non-beku di atas kernel sinkron** | Kernel WASM beku (isolasi/keamanan abadi); async (wakeup/task/promote) hidup di lapis non-beku via durable ledger + poller. |
| **D3 force-graph (GUI)** | Viz relasi natural, vendored (no build-step front-end). |
| **Gerbang repetisi (hit_count) sebelum promote** | Anti-degenerasi self-loop (SGS): cuma pola berulang yang jadi insting/recovery. |

---
