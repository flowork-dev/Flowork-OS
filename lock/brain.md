# BRAIN — Arsitektur Memori Flowork: Subsistem & Cara Terhubung
> Dokumen referensi (white-label). Menjelaskan SEMUA subsistem memori, file penghubung,
> keputusan teknologi, dan cara mereka tersambung. Owner: Mr.Dev. Update terakhir: 2026-06-22.
> ⚠️ File ini KE-TRACK repo → NOL data personal owner (mekanisme generic doang).

---

## 0. FILOSOFI INTI
Memori Flowork = **dua lapis + satu substrat pemersatu**:
- **Lapis sumber** (authoritative): tiap subsistem punya tabel/store sendiri (skills, constitution, drawers, mistakes, dst). Plug-and-play, terisolasi per-agent.
- **Substrat pemersatu**: **Cognitive Graph** (`cognitive_nodes` + `cognitive_edges`) = mirror semua subsistem dalam 1 format, supaya bisa di-recall **by-makna** (semantic) lintas-subsistem + di-viz di GUI.
- **Lem-nya**: **embedding** (vektor makna, bge-m3). Tiap node punya embedding → recall = cari node paling mirip query secara cosine, BUKAN cuma keyword.

Prinsip: sumber tetap raja; graph = lapis-akses terpadu. Recall 3-lapis (verbatim FTS + semantic graph + instinct).

---

## 1. LAPIS PENYIMPANAN (storage)

### 1.1 LOCAL per-agent — SQLite `state.db`
- Lokasi kanonik mr-flow: `agent/agents/mr-flow/workspace/state.db` (di REPO, bukan `~/.flowork`).
- Tiap agent punya `state.db` SENDIRI (isolasi: agent A rusak ga sentuh B).
- Teknologi: **SQLite** (driver `modernc.org/sqlite`, pure-Go, WAL mode, `WITHOUT ROWID` di kv).
- Tabel kunci: `cognitive_nodes`, `cognitive_edges`, `cognitive_identity_alias`, `brain_drawers` (+`brain_fts*` FTS5), `skills`, `constitution`, `educational_errors_cache`, `mistakes_local`, `kv`, `tool_memory`, `learning_record_log`, `agent_runs`, `wakeups`, `interactions`, `decisions`, `codemap_*`.

### 1.2 SHARED — Router brain `flowork-brain.sqlite`
- Lokasi: `router/brain/flowork-brain.sqlite` (~jutaan drawers: security/training/knowledge umum).
- Mesin **embedding** (bge-m3, dim 1024) + **vecindex** ada di ROUTER (`:2402`). Agent "pinjem hitungan" via HTTP.
- Akses dari agent: tool `brain_search_shared` (capability `rpc:router:brain`).

**2-tier brain:** brain PRIBADI lokal (`brain_search`) vs korpus LUAS shared (`brain_search_shared`). Insting/pengetahuan umum di shared; pengalaman/data personal di lokal.

---

## 2. SUBSISTEM MEMORI (sumber → file → peran)

| # | Subsistem | Sumber (tabel/store) | File pengelola | Isi |
|---|---|---|---|---|
| 1 | **Knowledge base** | Router `flowork-brain.sqlite` | `tools/builtins/brain.go` | korpus luas shared (jutaan drawer) |
| 2 | **Knowledge drawer** | `brain_drawers` (+`brain_fts` FTS5) | `agentdb/brain_drawers.go`, `tools/builtins/brain_local.go` | memori verbatim per-agent (wing/room) |
| 3 | **Constitution** | `constitution` | `agentdb/constitution.go` | 8 aturan sacred (always_inject, amplitude, lens) |
| 4 | **Typed Memory** | `kv`, `tool_memory` | `tools/builtins` (memory_get/set) | key-value + config toggle |
| 5 | **Personas** | node `type=agent`/`persona` + `kv.prompt` | `agentdb/cognitive_graph.go` | identitas/peran agent colony |
| 6 | **Instincts** | `cognitive_nodes type=instinct` | `agentdb/cognitive_recall.go`, `tools/builtins/instinct_recall.go` | pola "WHEN→THEN" coding/security |
| 7 | **Skills** | `skills` | `agentdb` skills accessor | prosedur reusable (trigger+instructions) |
| 8 | **Error edukasi** | `educational_errors_cache` (statis) + `mistakes_local` (dinamis) + recovery-instinct | `agentdb/edu_errors_seed.go`, `agentdb/mistakes.go`, `mistake_promote_job.go` | doktrin anti-stuck + lesson dari pengalaman |

---

## 3. SUBSTRAT PEMERSATU — Cognitive Graph

### 3.1 Skema node (`cognitive_nodes`) — file `agentdb/cognitive_graph.go`
`CogNode` = skema **W5H1** (who/what/why/where/when + how):
```
ID (URN: <scope>/<type>/<local_id>) · Label (WHAT) · Type (person|concept|skill|
doctrine|persona|memory|knowledge|instinct|edu_error|fact|preference|trait|event|
project|agent|code|tool|...) · Why · Who(JSON) · WhereDomain · WhenValid ·
Properties(JSON) · SourceKind (user_said|agent_inferred|verified|strong_model_unverified) ·
SourceRef · Confidence(0..1) · Status (active|quarantined|obsolete|shadow) ·
Embedding([]byte, 8-bit quantized) · HitCount · Version
```
- API tulis: `func (s *Store) UpsertNode(n CogNode) (added bool, err error)` — idempotent by ID. Edge: `UpsertEdge(CogEdge)`. Baca tetangga: `Neighbors(id)` (out+in, **status='active' only**).
- API baca (GUI): `ListCogNodes(limit)` — ORDER BY `hit_count DESC, last_seen_at DESC` (default 500, max 5000). `ListCogEdges(limit)` (no status-filter).
- **W5H1 KE-ISI (BRAIN.md B2, `graphwire`):** dulu pengetahuan numpuk di Label doang. Sekarang label insting `"WHEN <X> -> <Y>"` di-PECAH ke field terstruktur → **`when_valid`=X (trigger/WHEN)**, **`properties.how`=Y (aksi/HOW)**. **HOW = dimensi prioritas** (insting penemu owner: "bagaimana caranya agar..." bukan "mungkin nggak ya"). ~1167 node `when_valid` keisi, ~897 `properties.how`.

### 3.2 Skema edge (`cognitive_edges`) — relasi berarah
`CogEdge`: from_id · to_id · relation_type (kosakata tetap: `member_of`/`taught`/`uses`/`part_of`/`depends_on`/`governed_by`/`belongs_to`/dst) · strength · confidence · source_kind · status.
- **2 jenis edge:** (a) **SEMANTIK** (twin: person↔event↔trait, `taught`/`uses`/keluarga `member_of` — status=`active`) → IKUT recall. (b) **STRUKTURAL** (konektivitas GUI: node→hub→root, status=`shadow`) → TIDAK ikut recall.
- ⚠️ **TRIK KUNCI shadow-edge (BRAIN.md B2, tool `graphwire`):** node `instinct`/`edu_error`/`skill`/`knowledge` di-recall by-**embedding** (bukan traversal). Biar GUI NYAMBUNG (bukan titik melayang), tiap node dikasih edge `member_of` → **hub-node** per-domain (`concept/hub-coding-instinct`/`hub-security-instinct`/`hub-recovery`/`hub-mindset`/`hub-skills`/`hub-constitution`/`hub-edu`/`hub-knowledge`) → `concept/brain-root`. **Hub-node + edge-hub = `status='shadow'`.** Akibatnya:
  - `Neighbors()` (recall, query `WHERE status='active'`) **SKIP** edge-shadow → fact-sheet BERSIH (no hub-junk).
  - `SearchNodesByEmbedding` (filter `status='active'`) **SKIP** hub-node → ga ke-seed recall.
  - `ListCogEdges`/`ListCogNodes` (GUI, **NO** status-filter) → hub + edge-shadow **TAMPIL** → GUI ngumpul rapi.
  - → **GUI nyambung + recall bersih, TANPA edit kode/rebuild** (data-only, conf+strength edge rendah 0.2 jaga-jaga).

### 3.3 Tabel pendukung
- `cognitive_identity_alias` (co-reference: alias→canonical) — file `cognitive_coref.go`.
- `cognitive_tension` (konflik fakta) · `cognitive_digest_log` (jejak digestion).

---

## 4. LEM SEMANTIK — Embedding (bge-m3) + Quantize

**Alur (PENTING — ini yang bikin recall by-makna):**
1. Teks (label node / query) → **`routerclient.EmbedText(ctx, model, text)`** (`routerclient/embed.go`) → HTTP `POST :2402/v1/embeddings` (OpenAI-compatible) → vektor float32 dim **1024** (bge-m3, mesin di router).
2. **`agentdb.Quantize(vec []float32) []byte`** (`cognitive_resolve.go`) → 8-bit (1 byte/dim, ~99% recall vs float; pola vecindex router) → simpan ke kolom `embedding` BLOB node.
3. Recall: query di-embed → quantize → **`SearchNodesByEmbedding(typ, queryEmb, k)`** (`cognitive_recall.go`) → cosine top-k node `active`.

**Kenapa di router, bukan di agent:** mesin embed berat (model) → 1 instance di router, semua agent pinjem. Agent cuma simpan hasil quantize (ringan).

---

## 5. MEKANISME RECALL (3-lapis)

| Lapis | Tool | File | Cara | Sumber |
|---|---|---|---|---|
| **Verbatim (lokal)** | `brain_search` | `tools/builtins/brain_local.go` | FTS5/BM25 keyword | `brain_drawers`+`brain_fts` |
| **Verbatim (shared)** | `brain_search_shared` | `tools/builtins/brain.go` | BM25/FTS remote (rpc:router:brain) | router `flowork-brain.sqlite` |
| **Semantic graph** | `graph_recall` | `tools/builtins/cognitive_tools.go` → `agentdb/cognitive_recall.go` | embed query → `SearchNodesByEmbedding` (semua type) → `RecallFactSheet` (seed+rank, budget-capped) | `cognitive_nodes/edges` |
| **Instinct** | `instinct_recall` | `tools/builtins/instinct_recall.go` | embed query → `SearchNodesByEmbedding(type='instinct')` budget 1400ch | `cognitive_nodes type=instinct` |
| **Mistakes** | `mistakes_recall` | `tools/builtins/mistakes_recall.go` → `agentdb/mistakes_recall.go` | `LIKE` keyword (BUKAN semantic) | `mistakes_local` |
| **Edu (statis)** | `edu_error_lookup` | `agentdb/edu_errors.go` | by-Code exact | `educational_errors_cache` |
| **Codemap** | `codemap_search` | `tools/builtins/codemap_tools.go` | substring node kode | `codemap_nodes` |
| **Tool registry** | `tool_search` | `tools/builtins/v9_extras.go` | substring nama/cap/desc | registry tools |

**`RecallFactSheet`** (`cognitive_recall.go`): seed (embedding + label) → rangkai fact-sheet ringkas budget-capped. Ranking saat ini `confidence×strength` (bukan pure query-relevance — keterbatasan known).

---

## 6. CARA TIAP SUBSISTEM MASUK KE GRAPH

Ada **3 jalur** node bisa lahir di `cognitive_nodes`:

### 6.1 EKSTRAKSI (otomatis, dari interaksi) — digestion
- `cognitive_extract.go` (ekstrak node/edge dari teks chat) + `cognitive_dream.go` (digest batch via agent `dream-digester`).
- Gerbang: `cognitive_gate.go` (validation gate, anti-halu) → `cognitive_resolve.go` (`ResolveByEmbedding` dedup) → UpsertNode.
- Hook: `agentmgr/cognitive_digest_cron.go` (ticker) + auto-compact.
- Reasoning di AGENT `dream-digester` (model GUI): `dream_digester_seed.go` → `host.InvokeAgentMessage`.

### 6.2 PROJEKSI (manual/batch, dari tabel sumber) — scratch tools (`_scratch_cgm/`)
- `instproj/main.go` — instinct corpus (router brain room) → `type=instinct` (+embedding).
- `graphsync/main.go` — **skills/constitution/edu_errors/drawers → graph** (+embedding). [BRAIN.md FASE B1]
- `secinstinct/main.go` + `redistil/main.go` — distil korpus mentah → instinct (white-label+leak-gate) → ingest router brain.
- `addinstinct/main.go` — seed meta-instinct manual (mis. 5 meta security/coding + safety "reframing=refuse").
- `graphwire/main.go` — **[BRAIN.md FASE B2]** (A) W5H1-fill (`when_valid`/`properties.how` dari label insting), (B) konek edge `member_of` **status=shadow** node→hub→root (GUI nyambung, recall bersih), (HOW) seed **HOW-instinct** mindset penemu (`where_domain='mindset'`, conf 0.95).
- Pola umum: baca sumber → `EmbedText` → `Quantize` → `UpsertNode(type, embedding)`. Idempotent (id stabil).

### 6.3 PEMBELAJARAN (dari pengalaman) — loop
- **3E loop-belajar** (`agentmgr/learning_feed.go` + `agentdb/learning_log.go`): router capture model-kuat → `recordings` → distil (dream-digester) → SHADOW node (`source_kind=strong_model_unverified`) → promote-on-repetisi.
- **D32 recovery-instinct** (`mistake_promote_job.go`, non-beku): `mistakes_local` `hit_count≥3` → `type=instinct where_domain='recovery'` (+embedding) → recall semantic. Gate repetisi = anti-degenerasi.

---

## 7. AUTO-RECALL (inti "kenal owner") — file `agent/agents/mr-flow/main.go` (fungsi `fetchAutoRecall`)
- `fetchAutoRecall(userText)` di-panggil TIAP TURN → jalanin `graph_recall`(query=userText, budget 2800) + `brain_search`(query=userText, k=5) → inject fakta relevan ke **Tier-3** prompt + **directive TEGAS**.
- **2 directive (string di `b.WriteString`):**
  - graph: `[FAKTA TERVERIFIKASI tentang Mr.Dev... JAWAB pakai fakta ini & HUBUNGKAN fakta yang berkaitan. JANGAN bilang "gak punya data/inget" kalau bisa disimpulkan...]`. ("HUBUNGKAN" = biar model nyambungin fakta tersebar, mis. "X taught owner" + "owner uses Y" → "X guru Y owner".)
  - brain: `[FAKTA VERBATIM dari memori lo (drawer tersimpan) — JAWAB PAKAI INI. JANGAN bilang "gak tau / ga ada catatan" kalau jawabannya ADA di bawah]` (diperkuat 2026-06-22 biar model 26B ga ngabaikan drawer).
- Akar: brain/graph dulu cuma tool-driven → model lemah ga manggil → "gak punya data" walau fakta ada. Sekarang auto-nongol.
- Model = GUI per-agent (`cfg.Router.Model`), bukan hardcode (mandat AI-in-agent).
- ⚠️ **K11 KNOWN-MISS (recall ~93.3%):** query RELASI **terbalik** (mis. "siapa <peran-X> gw?" — nyari subjek dari relasi) kadang miss → `graph_recall` ga nge-SEED node yg bener buat frasa itu (embedding query ga match label node person yg sering generik spt "User"). Fakta ADA + model PAKAI pas query **sebut nama entitas-nya langsung**. **K11/K12: JANGAN graph-hack ranking** — jalur bener = verbatim coverage (brain_search). Stronger model (Opus) dapet 2 arah.

---

## 8. GUI — Cognitive Graph tab
- Front-end: `agent/web/tabs/cognitive.js` (D3 **force-directed graph**, "balls connected"). `TYPE_COLOR` map warna per-type + legend + truncate label (anti-berantakan) + klik node → detail.
- Fetch: `GET /api/agents/cognitive/graph?id=<agent>&limit=2000`.
- Back-end handler: `agentmgr/cognitive_handlers.go` `CognitiveGraphHandler` → `ListCogNodes` + edges.
- web di-EMBED ke binary (`//go:embed web` di `main.go`) → ubah GUI = rebuild host.

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
- `edu_errors_seed.go` / `edu_errors.go` — katalog doktrin edukasi (statis, 28).
- `constitution.go` — 8 aturan sacred.

**tools/builtins (jembatan LLM ↔ memori):**
- `cognitive_tools.go` (graph_recall) · `instinct_recall.go` · `brain.go` (shared) · `brain_local.go` (lokal) · `brain_immune.go` (antibody) · `mistakes_recall.go` · `codemap_tools.go` · `v9_extras.go` (tool_search) · `claude_tools.go` (Task/Schedule/etc).
- `tool_specs.go` (agentmgr) — gerbang tool MANA yang di-expose ke LLM (core + primaryExtra + subscription, cap 51).

**host non-beku (orkestrasi loop):**
- `agent/main.go` — wiring + ticker (1 menit: RunDueWakeups, RunQueuedTasks, PromoteRecurringMistakes).
- `wakeup_engine.go` (ScheduleWakeup) · `task_worker.go` (background task) · `mistake_promote_job.go` (D32) · `dream_digester_seed.go` (digest agent) · `learning_feed.go`/`learning_log.go` (3E).

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

## 11. ALUR END-TO-END (contoh: 1 fakta dari chat → recall)
1. Owner ngomong fakta di chat → `interactions` tersimpan.
2. Ticker digest (`cognitive_digest_cron`) → agent `dream-digester` ekstrak → `cognitive_extract` → gerbang `cognitive_gate` (anti-halu) → dedup `ResolveByEmbedding` → `UpsertNode` (label di-`EmbedText`→`Quantize`→embedding).
3. Lain kali owner tanya (kata beda) → `fetchAutoRecall` (mr-flow main.go) → `graph_recall` embed query → `SearchNodesByEmbedding` cosine → fact-sheet → inject Tier-3 → LLM jawab pakai fakta.
4. GUI: node muncul di tab Cognitive Graph (D3), warna per-type.

**Untuk subsistem (skills/constitution/edu/drawer):** langkah-2 diganti **projeksi** (`graphsync`: baca tabel sumber → EmbedText → Quantize → UpsertNode type sesuai). Recall + GUI sama.

---

## 12. RINGKAS — "siapa nyambung ke siapa"
```
            ┌─────────────── COGNITIVE GRAPH (cognitive_nodes/edges) ───────────────┐
            │  substrat pemersatu — tiap node punya EMBEDDING (lem semantic)         │
            └───────▲────────▲────────▲────────▲────────▲────────▲──────────────────┘
   projeksi (graphsync) │        │        │        │        │   ekstraksi/digest (dream)
   ┌────────┬───────────┴──┬─────┴───┬────┴────┬───┴─────┬──┴──────┐         ▲
 skills  constitution  edu_errors  drawers  instinct  recovery   personas    │
(skills) (constitution)(edu_cache)(brain_  (corpus/  (mistakes  (agent     interactions
                                   drawers) instproj) _local)    nodes)
            │                                                        │
   recall: graph_recall / instinct_recall / brain_search(_shared) / mistakes_recall
            │                                                        │
        fetchAutoRecall (tiap turn) ──→ LLM (model GUI)        GUI cognitive.js (D3)
            ▲                                                        ▲
        EmbedText(router bge-m3) + Quantize(8-bit) ←── lem semantic ─┘
```
Router brain (`flowork-brain.sqlite`, shared 5jt) = sumber knowledge-base luas, diakses `brain_search_shared` (rpc:router:brain), + mesin embedding (bge-m3).

---

## 13. BRAIN-CORE — file inti buat di-FREEZE (kandidat BRAIN_FREEZE)
> Owner 2026-06-22: **freeze SEMUA jalur brain** — lindungi dari AI yg ngubah TANPA SADAR (internal-evolusi DAN eksternal spt asisten-AI pas autonom). Pola = extend `KERNEL_FREEZE` (SHA256 manifest + `TestBrainFreeze` + Guardian baseline + appliance dm-verity). **Komentar file2 ini bakal DIHAPUS → diganti rujukan `// arsitektur: lihat lock/brain.md`** (clean code; semua "kenapa" pindah ke doc ini).

**A. Inti recall/graph — `agent/internal/agentdb/`:**
`cognitive_graph.go` · `cognitive_recall.go` · `cognitive_resolve.go` · `cognitive_extract.go` · `cognitive_dream.go` · `cognitive_gate.go` · `cognitive_coref.go` · `cognitive_temporal.go` · `cognitive_heal.go` · `cognitive_embed_backfill.go` · `cognitive_codemap.go` · `brain_drawers.go` · `mistakes.go`/`mistakes_promote.go`/`mistakes_recall.go` · `edu_errors.go`/`edu_errors_seed.go` · `constitution.go`.

**B. Tool jembatan — `agent/internal/tools/builtins/`:**
`cognitive_tools.go` (graph_recall) · `instinct_recall.go` · `brain.go` · `brain_local.go` · `brain_immune.go` · `mistakes_recall.go`.

**C. Embedding:** `agent/internal/routerclient/embed.go` (+ `Quantize` ada di `cognitive_resolve.go`).

**D. Auto-recall:** fungsi `fetchAutoRecall` di `agent/agents/mr-flow/main.go`. ⚠️ main.go CAMPUR brain + non-brain (tool-loop/persona/ghost-guard) → freeze **granular** (pisah fetchAutoRecall ke file sendiri dulu) ATAU freeze main.go penuh (lebih kaku).

**D2. Auto-capture recovery (D32 INC-2) — `agent/agents/mr-flow/recovery_capture.go`:** logic-brain `captureRecovery`/`toolErrClass`/`recoveryCaptureSkip` DI-EKSTRAK dari main.go = realisasi PERTAMA pola granular §13.D (main.go = list/wiring EDITABLE, logic-brain = file terpisah FROZEN). Tool ERROR→tool SAMA SUKSES dalam loop → `mistake_log` → pipeline INC-1. Dipanggil 1 baris dari tool-loop main.go. FROZEN.

**E. Loop non-beku yg NYENTUH brain (boleh evolve tapi hati2):** `dream_digester_seed.go` · `mistake_promote_job.go` · `learning_feed.go`/`agentdb/learning_log.go` · `agentmgr/cognitive_digest_cron.go` · `graph_autosync.go` (B4 auto-sync sumber→graph, ticker host + change-detection; soft-lock, kandidat freeze).

**F. GUI — ⛔ TIDAK di-freeze (owner 2026-06-22):** `web/tabs/cognitive.js` + `agentmgr/cognitive_handlers.go` = jalur GUI/viz (warna/legend/filter masih EVOLVE). **Jangan dikunci** — biar bebas berkembang.

**⛔ JANGAN di-freeze:** **GUI** (cognitive.js + cognitive_handlers.go — viz berkembang) · **main.go** (fetchAutoRecall di sini; main.go bakal jadi LIST/wiring doang — nano-modular, nanti) · **scratch** (`_scratch_cgm/*` — gitignored, sekali-pakai) · **DATA** (db/`cognitive_nodes`/embedding/drawer — TUMBUH terus; freeze cuma buat CODE).

**STATUS 2026-06-22:** **30 file brain-LOGIC** (A+B+C+E) di-STRIP komentar + header minimal + SHA256 di `KERNEL_FREEZE.md` + chattr +i. Verified: gofmt clean · go build · go vet · brain-tests PASS · TestKernelFreeze OK (kode+perilaku IDENTIK, strip=comment-only). **+1 file (D2) 2026-06-22:** `recovery_capture.go` (D32 INC-2, di-ekstrak dari main.go) di-freeze juga (chattr+hash) = **31 file brain-core**. **SISA (nanti):** OS-sealer otomatis (N3).
