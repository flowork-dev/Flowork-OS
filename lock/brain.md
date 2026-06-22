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
- Lokasi: `router/brain/flowork-brain.sqlite` (~860k drawers, 859.808 per 2026-06-22: security/training/knowledge umum; dulu ~5jt, sampah dibersihin).
- Mesin **embedding** (bge-m3, dim 1024) + **vecindex** ada di ROUTER (`:2402`). Agent "pinjem hitungan" via HTTP.
- Akses dari agent: tool `brain_search_shared` (capability `rpc:router:brain`).

**2-tier brain:** brain PRIBADI lokal (`brain_search`) vs korpus LUAS shared (`brain_search_shared`). Insting/pengetahuan umum di shared; pengalaman/data personal di lokal.

---

## 2. SUBSISTEM MEMORI (sumber → file → peran)

| # | Subsistem | Sumber (tabel/store) | File pengelola | Isi |
|---|---|---|---|---|
| 1 | **Knowledge base** | Router `flowork-brain.sqlite` | `tools/builtins/brain.go` | korpus luas shared (~860k drawer) |
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

⚠️ **fact-sheet `graph_recall` = EDGES doang** (relasi `X —rel→ Y`), BUKAN label-node standalone (temuan N2 2026-06-22). Akibat: node `knowledge`/drawer-projeksi yang GA punya edge → **invisible di graph_recall** walau ke-seed by-embedding. Jadi **verbatim-drawer cuma bantu `brain_search`, BUKAN graph_recall.** Buat jawab query relasi-kebalik (mis. "siapa guru gitar gw") → fakta WAJIB ada sebagai **EDGE** (mis. `Irin —taught→ User`, outgoing-dari-seed) ATAU **verbatim drawer** (jalur brain_search, model 26B pakai). K11/K12: JANGAN graph-hack ranking; tutup gap via verbatim drawer + data-fix edge salah-atribusi (lihat N2: cabut halu `User —is_a→ Best Guitarist` → re-point ke Irin).

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
- **⚡ B4 AUTO-SYNC (produksi, 2026-06-22) — `graph_autosync.go` (host non-beku, FROZEN):** versi OTOMATIS dari `graphsync` scratch. Ticker tiap 30min projeksi skills/constitution/edu/drawers → graph + **CHANGE-DETECTION** (`SyncSourcesToGraph`: skip `EmbedText` kalau label node == sumber → cuma row BARU/BERUBAH yang re-embed → hemat router). Ganti re-run manual. Graph SELALU cermin sumber tanpa re-run tangan.

### 6.3 PEMBELAJARAN (dari pengalaman) — loop
- **3E loop-belajar** (`agentmgr/learning_feed.go` + `agentdb/learning_log.go`): router capture model-kuat → `recordings` → distil (dream-digester) → SHADOW node (`source_kind=strong_model_unverified`) → promote-on-repetisi.
- **D32 recovery-instinct (loop 3-tahap, FROZEN):**
  - **(INC-2 CAPTURE)** `recovery_capture.go` (di-panggil 1 baris dari mr-flow tool-loop): tool ERROR lalu tool yg SAMA SUKSES dalam loop → `mistake_log` "WHEN <tool> <kelas> -> recovered" (kelas error BEBAS path/data owner — privasi). Reuse pipeline mistake.
  - **(INC-1 PROMOTE)** `mistake_promote_job.go` (non-beku, ticker 1-menit): `mistakes_local` `hit_count≥3` (eligible) → kirim ke INC-3 generalize → SHADOW instinct. Lalu **GERBANG** `PromoteRecoveryShadows(2)` (di ticker yg SAMA, BUKAN nyandar autodigest yg default-OFF): recovery-instinct SHADOW yg `hit_count≥2` → ACTIVE → baru ke-recall.
  - **(INC-3 GENERALIZE, `recovery_generalize.go`)** raw recovery → instinct UMUM privacy-safe: **Lapis A** strip deterministik (path/url/email/token/hex + nama-personal allowlist-runtime → JAMIN 0 data owner walau LLM meleset) → **Lapis B** coarsen via dream-digester (model Haiku) jadi pola "WHEN <umum> -> <aksi>" (re-strip + brand-check atas output) → `type=instinct where_domain='recovery'` SHADOW (+embedding buat recall by-makna). ⚠️ IDENTITAS/DEDUP pakai **kunci KELAS-error deterministik** (mis. `recov-not-found`), BUKAN embedding output LLM — sebab LLM coarsen non-deterministik (teks goyang tiap call) → embedding-dedup ga reliable → instinct nyangkut shadow. Kelas stabil → recovery kelas-sama lintas-tool nyatu ke 1 node → hit naik → gerbang firable. → agent ga ngulang stuck yg udah ke-recover (hemat token).
- **D32-INC4 SHARE → SHARED-BRAIN (`recovery_share_job.go`, host, FROZEN):** recovery-instinct generik+verified → `SelectPromotableRecoveryInstincts` (`federation_recovery.go`, FROZEN) → double-check privasi deterministik (StripDeterministic==self && !ContainsBrand) → `PromoteDrawer` mem_type=`recovery_instinct` → imunitas kolektif (agent lain recall via `brain_search_shared`). Anti-double `federation_cognitive_log`. ⚠️ "consensus 9-lapis" cuma 6/9 NYATA (audit) → INC-4 reuse lapis 1-6 + gate privasi; consensus N-of-M (L7-9) + antibody kolektif = BLOCKED multi-peer mesh (roadmap F).
- **C COLLECTIVE GRAPH (`cognitive_share_job.go`, host, FROZEN):** fakta UMUM (concept/skill/knowledge + relasi) → `SelectPromotableCognitiveNodes/Edges` (default-DENY: type-allowlist + verified + BUKAN person-linked) → `cleanForShare` strict → `PromoteDrawer` mem_type=`collective_knowledge`. Privasi D8 3-lapis.
- **F5 FRESH-RECALL (router `internal/brain/fresh_index.go`, soft-lock):** index VECTOR kedua kecil in-memory isinya drawer federation (`recovery_instinct`/`collective_knowledge`), rebuild periodik (change-detect) → di-merge ADDITIF di `SemanticRetrieve` (fresh kosong → 0 regresi). Akar: vindex utama di-build manual+cached → drawer baru ga ke-recall sampe reindex. AMAN: index 859k GAK disentuh. (F5 enabler recall INC-4/C.)
- **D COLD-ARCHIVE (`cognitive_archive.go` + `cognitive_archive_job.go`, FROZEN):** node tua+low-hit+tipe-BULK → `status='archived'` (recall auto-skip, reversible). GATED >50k node aktif (anti-premature, 0 dampak di ~2k). Tipe identitas/instinct/skill ga pernah di-archive.
- **E RACE-GUARD (`task_worker.go`, FROZEN):** worker async (ledger `agent_runs`) + `agentBusySet` → MAKS 1 bg-task per agent (anti korup `__d18_active_task` kv); lintas-agent paralel. Fix di worker, BUKAN lock choke-point (anti-deadlock group-call).
- **F1-F3 CONSENSUS 9-LAPIS MESH (`router/internal/mesh/`, soft-lock):** jalur knowledge dari PEER mesh (`ProcessKnowledgePacket`) lengkap 9-lapis: L1-6 (signature/freshness/karma/quarantine/injection) + **L7** near-dup (trigram offline / embedding-injectable) + **L8 consensus N-of-M** (`consensus_phase3.go`: ≥N peer DISTINCT endorse near-same, ATAU 1 peer trusted-karma; sybil-resist distinct-pubkey) + **L9** promote-decision (agregat di ProcessKnowledgePacket). Federation OWNER (INC-4/C) TIDAK lewat sini. DORMANT single-node (0 peer).
- **F4 ANTIBODY KOLEKTIF (`cognitive_antibody.go` + `cognitive_antibody_job.go`, FROZEN):** recovery-instinct yg ditemukan INDEPENDEN ≥N agent (kelas sama) → push ke SEMUA agent + mark collective (conf 0.95). Imunitas kolektif. Dedup by kelas-error. Dormant pas 1 agent.
- **ANN/IVF (`router/internal/brain/vecindex/ann.go`, soft-lock):** index approximate (k-means cluster + probe nprobe → SearchSubset exact) buat skala >jutaan. ADDITIVE — Index flat TIDAK disentuh (tetap jalur live, recall@10=0.985); ANN = kapabilitas siap (recall@10=0.918 @ ~3× lebih cepet), flip pas jutaan node + flat fallback. BUKAN rip-replace.

### 6.4 AUTO-COMPACT KONTEKS (anti-halu konteks panjang) — `agentmgr/autocompact.go` + `digest_model.go` (FROZEN)
- **Masalah→solusi:** interaksi numpuk → konteks kepanjangan → AI halu. Tiap 15 menit (cron) ATAU tombol GUI, agent yg interaksi non-deleted > ambang (default 400) → **digest pengalaman ke brain (jalur 6.1)** → **trim** raw interaksi lama (sisain `keep_recent` terbaru, default 60). Pengalaman GA ilang — pindah ke brain, bisa di-recall.
- **Urutan FATAL-SAFE (`AutoCompactAgent`):** (1) DIGEST pending → brain; gagal → STOP, JANGAN trim. (2) VERIFY 0 sisa undigested SEBELUM trim. (3) TRIM cuma yg UDAH ke-brain (`TrimDigestedInteractions`, soft-delete reversible). + skip agent mid-task (busy <90s). Jadi digest gagal = no trim = **NO LOSS**.
- **CHUNKING (owner 2026-06-22, `cognitive_dream.go` `DigestPendingInteractions`):** extract-call dipecah per **6000 char**. Batch gede (puluhan ribu char) bikin model nyerah → balikin kosong/prosa → ParseExtraction gagal → digest gagal → ga pernah trim (terbukti QC live). Per-chunk digest+mark SENDIRI; chunk gagal → interaksinya stay undigested (no loss, retry tick berikut). 1 interaksi solo boleh > budget (tetep 1 chunk). `firstErr` ke-return → AutoCompact tau belum tuntas (ga trim sampe semua chunk sukses).
- **MODEL-PICKER (owner 2026-06-22, `digest_model.go` + KV `compact_model`):** model reasoning buat digest compact BISA dipilih owner (Settings → Auto-Compact, **free-text**). Di-set → **SEMUA** jalur compact (cron / Compact All / per-agent) pake model itu. **KOSONG = model LOKAL `flowork-brain`** (bukan cloud) — biar compact tetep jalan **TANPA langganan** (tujuan freeze/standalone: kalau token cloud habis, digest ke-brain tetep hidup). `DigestAgentModel` reuse pipeline digest yg SAMA, cuma swap model di `DigestDeps` (bypass `DigestLLMOverride`). Jalur digest non-compact (dream cron) TIDAK disentuh (no regression).
- **Bukti empiris (2026-06-22):** model lokal flowork-brain di-test isolasi (temp DB, 32 interaksi=6688 char → 2 chunk via router :2402) → digest OK **13 node/10 edge**, trim **32→5**, 0 leak, **offline**. `internal/agentdb/live_local_digest_test.go` (gated `FLOWORK_LIVE_DIGEST=1`, ga ikut suite biasa). Compact terbukti jalan tanpa cloud. ✓
- GUI `web/tabs/settings.js` `renderCompact` (NON-frozen, §13.F). Route: `POST /api/agents/compact?id=&force=1` (per-agent) · `POST /api/agents/compact-all?force=1` (Compact All) · `GET/POST /api/compact/config` (ambang+toggle+model).

---

## 7. AUTO-RECALL (inti "kenal owner") — file `agent/agents/mr-flow/main.go` (fungsi `fetchAutoRecall`)
- `fetchAutoRecall(userText)` di-panggil TIAP TURN → jalanin `graph_recall`(query=userText, budget 2800) + `brain_search`(query=userText, k=5) → inject fakta relevan ke **Tier-3** prompt + **directive TEGAS**.
- **N1-C GATE (2026-06-22): skip recall pas pesan TRIVIAL.** Helper `isTrivialChat(q)` + set `trivialChatTokens` (sapaan/ack/filler) → `fetchAutoRecall` panggil di awal → return "" kalau SEMUA token pesan trivial ("halo"/"makasih bro") → `graph_recall` + `brain_search` GA jalan (hemat ~200-250 token + 2 tool-call/turn). KONSERVATIF: 1 kata substantif matahin gate → query identitas/relasi ("siapa gw") TETAP ke-recall (0 regresi; unit 30/30 + e2e dbgchat). **DI-EKSTRAK ke `agents/mr-flow/recall_gate.go` (FROZEN, pola nano-modular spt recovery_capture.go); main.go cuma manggil (wiring, tetap editable).**
- **D18-P1 WORKING-SET (2026-06-22): TUGAS AKTIF persist lintas-sesi.** `activeTaskFor(userText)` (di `agents/mr-flow/working_set.go`, FROZEN): request SUBSTANTIF (reuse `isTrivialChat`) → simpan kv `__d18_active_task` (`memory_set`/tool_memory); trivial chat ga ngubah. main.go inject hasilnya BOTTOM-salient tiap turn → goal ga ke-scroll keluar window 16-turn / ga ilang walau restart. Verified e2e (model lanjut tugas di turn lain). + **D18-P0** observability: log `D18-ctx: sys/recall/history/tools` per turn (instrumentasi, di main.go). Desain capstone D18 (fase P0→P4) = doc lokal owner (di luar repo).
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
Router brain (`flowork-brain.sqlite`, shared ~860k) = sumber knowledge-base luas, diakses `brain_search_shared` (rpc:router:brain), + mesin embedding (bge-m3).

---

## 13. BRAIN-CORE — file inti buat di-FREEZE (kandidat BRAIN_FREEZE)
> Owner 2026-06-22: **freeze SEMUA jalur brain** — lindungi dari AI yg ngubah TANPA SADAR (internal-evolusi DAN eksternal spt asisten-AI pas autonom). Pola = extend `KERNEL_FREEZE` (SHA256 manifest + `TestBrainFreeze` + Guardian baseline + appliance dm-verity). **Komentar file2 ini bakal DIHAPUS → diganti rujukan `// arsitektur: lihat lock/brain.md`** (clean code; semua "kenapa" pindah ke doc ini).

**A. Inti recall/graph — `agent/internal/agentdb/`:**
`cognitive_graph.go` · `cognitive_recall.go` · `cognitive_resolve.go` · `cognitive_extract.go` · `cognitive_dream.go` · `cognitive_gate.go` · `cognitive_coref.go` · `cognitive_temporal.go` · `cognitive_heal.go` · `cognitive_embed_backfill.go` · `cognitive_codemap.go` · `brain_drawers.go` · `mistakes.go`/`mistakes_promote.go`/`mistakes_recall.go` · `recovery_generalize.go` (**D32 INC-3** generalisasi recovery) · `federation_recovery.go` (**INC-4** select recovery share) · `cognitive_archive.go` (**D** cold-archive) · `cognitive_antibody.go` (**F4** antibody kolektif) · `edu_errors.go`/`edu_errors_seed.go` · `constitution.go`.
> **Host orchestrator FROZEN (loop non-beku brain-pathway):** `recovery_share_job.go` (INC-4 share) · `cognitive_share_job.go` (C collective) · `cognitive_archive_job.go` (D archive sweep) · `cognitive_antibody_job.go` (F4 antibody) · `task_worker.go` (E worker + race-guard). Pola = `mistake_promote_job.go`/`graph_autosync.go`.
> **Router brain (soft-lock, konvensi router, NON-chattr):** F1-F3 consensus mesh (`internal/mesh/{karma_toolshare_filter,consensus_phase3,pipeline}.go`) · F5 fresh-recall (`internal/brain/fresh_index.go`) · ANN (`internal/brain/vecindex/ann.go`) · `routerclient.go` (SSRF-fix).

**B. Tool jembatan — `agent/internal/tools/builtins/`:**
`cognitive_tools.go` (graph_recall) · `instinct_recall.go` · `brain.go` · `brain_local.go` · `brain_immune.go` · `mistakes_recall.go`.

**C. Embedding:** `agent/internal/routerclient/embed.go` (+ `Quantize` ada di `cognitive_resolve.go`).

**D. Auto-recall:** fungsi `fetchAutoRecall` di `agent/agents/mr-flow/main.go`. ⚠️ main.go CAMPUR brain + non-brain (tool-loop/persona/ghost-guard) → freeze **granular** (pisah fetchAutoRecall ke file sendiri dulu) ATAU freeze main.go penuh (lebih kaku).

**D2. Auto-capture recovery (D32 INC-2) — `agent/agents/mr-flow/recovery_capture.go`:** logic-brain `captureRecovery`/`toolErrClass`/`recoveryCaptureSkip` DI-EKSTRAK dari main.go = realisasi PERTAMA pola granular §13.D (main.go = list/wiring EDITABLE, logic-brain = file terpisah FROZEN). Tool ERROR→tool SAMA SUKSES dalam loop → `mistake_log` → pipeline INC-1. Dipanggil 1 baris dari tool-loop main.go. FROZEN.

**D3. Gate auto-recall (N1-C) — `agent/agents/mr-flow/recall_gate.go`:** `isTrivialChat(q)` + `trivialChatTokens` DI-EKSTRAK dari main.go (pola granular §13.D). `fetchAutoRecall` skip recall (graph+brain) kalau pesan cuma sapaan/ack/filler → hemat ~200-250 token + 2 tool-call/turn trivial. KONSERVATIF (1 kata substantif matahin gate → recall sah tetap jalan). Dipanggil 1 baris dari main.go. FROZEN.

**D4. Working-set (D18-P1) — `agent/agents/mr-flow/working_set.go`:** `activeTaskFor(userText)` DI-EKSTRAK dari main.go (pola granular §13.D). TUGAS AKTIF (request substantif, reuse `isTrivialChat`) di-persist ke kv `__d18_active_task` → di-inject bottom-salient tiap turn → goal ga ilang lintas-sesi/restart. Trivial chat ga ngubah. Dipanggil 1 baris dari main.go. FROZEN. (P0 observability `D18-ctx` log = di main.go, non-frozen.)

**E. Loop non-beku yg NYENTUH brain (boleh evolve tapi hati2):** `dream_digester_seed.go` · `mistake_promote_job.go` · `learning_feed.go`/`agentdb/learning_log.go` · `agentmgr/cognitive_digest_cron.go` · `graph_autosync.go` (B4 auto-sync sumber→graph, ticker host + change-detection; **FROZEN** chattr+hash 2026-06-22 = 32 file brain-core).

**E2. AUTO-COMPACT (FROZEN 2026-06-22) — `agentmgr/autocompact.go` + `agentmgr/digest_model.go`:** orkestrator compact (digest→VERIFY→trim, FATAL-SAFE, skip-busy) + **model-picker** (KV `compact_model`, **default LOKAL flowork-brain**) → SEMUA jalur compact (cron/Compact All/per-agent) hormati model pilihan owner. Owner minta freeze jalur manual+auto compact. Chunking ada di `cognitive_dream.go` (§A, udah FROZEN). `DigestAgentModel` REUSE `cognitive_digest_cron.go` (§E, FROZEN) — manggil fungsinya, ga ngedit. GUI `renderCompact` (`settings.js`) = NON-frozen (§F, viz berkembang). Arsitektur lengkap: §6.4. Bukti lokal-digest LULUS (`live_local_digest_test.go`, gated).

**G. SKILL SUBSYSTEM (FROZEN 2026-06-22, chattr+manifest) — 9 file skill-only router:** `handlers_skills_invoke.go` · `handlers_skillpack.go` · `handlers_skillregistry.go` · `handlers_brain_skills.go` · `internal/store/skills.go` · `internal/skillpack/skillpack.go`+`karma.go` · `internal/skillregistry/registry.go` · `internal/brain/skills.go`. Owner minta "freeze SEMUA file skill". **DEVIASI sadar dari konvensi "router NON-chattr":** file skill ini di-`chattr +i` + masuk manifest (path `../router/...`, TestKernelFreeze cek dari cwd `agent/`). Yang terlindungi: engine invoke/render-template/store + registry (3-gerbang: karma-gate+sign+verify)/pack/karma + dynamic-skills loader. ⛔ Shared NON-frozen (ga bisa chattr, ada kode lain): `handlers_resources.go` (skill CRUD = thin delegate ke `store/skills.go` FROZEN) · `internal/brain/init.go` (schema `skills` + tabel lain). GUI skill tab = NON-frozen (§F). Arsitektur: §14.

**H. CONSTITUTION + PERSONA + ENRICHMENT (FROZEN 2026-06-22, owner: "freeze SEMUA file yg berhubungan, BIAR self-documenting — kalau AI nyasar langsung tau ini BY-DESIGN & baca brain.md"):** +11 file desain-inti di-`chattr +i` + manifest, tiap file di-prepend pointer `// FROZEN brain-core … baca lock/brain.md`. **Agent:** `agentdb/federation.go` (promote drawer→shared) · `agentdb/constitution_tier.go` (TuneConstitutionForExtension) · `agentdb/constitution_upsert.go`. **Router (DEVIASI sadar dari "router NON-chattr" — owner minta FREEZE bukan soft-lock):** `internal/router/brainenrich.go` ⭐ (maybeEnrichBrain — jalur SUNTIK knowledge+skill+doktrin ke prompt; INI yg bikin "paham") · `internal/router/brain_constitution.go` (maybeInjectConstitution — render 5W1H-gate+anti-halu) · `internal/brain/semantic.go` (SemanticRetrieve — recall by-makna) · `internal/brain/write.go` (Add/Update/SoftDeleteConstitution + AddDrawer) · `internal/brain/explore.go` (ListConstitution) · `internal/brain/crud.go` (AddPersona/Update/Delete + drawer crud) · `internal/brain/views.go` (ListPersonas/ListByType) · `internal/brain/seed_doctrine.go` (seed doktrin). ALASAN: ini jalur yg, kalau diedit diam-diam (mis. "hemat token" → matiin enrichment/potong constitution), MERUSAK desain "paham + insting + 5W+H + anti-asal-ceplos". chattr = edit GAGAL → AI sadar BY-DESIGN → baca §6.4/§14/§7. Build agent+router OK.

**F. GUI — ⛔ TIDAK di-freeze (owner 2026-06-22):** `web/tabs/cognitive.js` + `agentmgr/cognitive_handlers.go` = jalur GUI/viz (warna/legend/filter masih EVOLVE). **Jangan dikunci** — biar bebas berkembang.

**⛔ JANGAN di-freeze:** **GUI** (cognitive.js + cognitive_handlers.go — viz berkembang) · **main.go** (fetchAutoRecall di sini; main.go bakal jadi LIST/wiring doang — nano-modular, nanti) · **scratch** (`_scratch_cgm/*` — gitignored, sekali-pakai) · **DATA** (db/`cognitive_nodes`/embedding/drawer — TUMBUH terus; freeze cuma buat CODE).

**STATUS 2026-06-22:** **52 file brain-core FROZEN** (+E2 AUTO-COMPACT `agentmgr/autocompact.go`+`agentmgr/digest_model.go` — orkestrator compact + model-picker default-LOKAL, owner minta freeze jalur compact; chunking `cognitive_dream.go` udah frozen; bukti lokal-digest→trim LULUS offline 32→5/13 node; +F4 `cognitive_antibody.go`+`cognitive_antibody_job.go`; F1-F3 consensus mesh + F5 fresh-recall + ANN = router soft-lock) (chattr +i + SHA256 di `KERNEL_FREEZE.md`, TestKernelFreeze 128 hash PASS): 30 brain-LOGIC + **D2** `recovery_capture.go` (D32 INC-2) + **B4** `graph_autosync.go` + **D3** `recall_gate.go` (N1-C) + **D4** `working_set.go` (D18-P1) + **D32-INC3** `recovery_generalize.go` (generalisasi recovery, e2e infra-real PASS 0 leak) + **6 BARU 2026-06-22** (INC-4/C/D/E): `federation_recovery.go` + `recovery_share_job.go` (INC-4 share→shared-brain) · `cognitive_share_job.go` (C collective graph) · `cognitive_archive.go` + `cognitive_archive_job.go` (D cold-archive, gated) · `task_worker.go` (E worker race-guard). Semua additive, unit/`-race` PASS, 0-regresi, di-push 2 repo. (Recall payoff INC-4/C nunggu deploy + F5 router fresh-recall.) **+ 7 brain-dep di-AUDIT-bersih + freeze 2026-06-22** (owner: "cek bug+keamanan, kalau ga ada freeze"): `federation_cognitive.go` (gate privasi C; +fix: edge anti-double pakai label, exclude personal diperluas person/persona/trait/preference) · `brain_federation.go` · `routerclient/federation.go`+`brain_search.go` · `brain_dream.go` · `codemap_tools.go`+`codemap_files_tool.go`. Pre-freeze fix SSRF di `routerclient.go` (userinfo-bypass `user@host` → exfil; net/url+tolak-userinfo; soft-lock NON-chattr krn infra HTTP). Router `dream_cycle.go`+`seed_doctrine.go` = soft-lock (konvensi router). Pola **nano-modular**: file brain-pathway terpisah → FREEZE; orkestrator (`main.go`) tetap EDITABLE. **+ DOC INI (`lock/brain.md`) di-FREEZE 2026-06-22 (chattr +i)** — lindungi arsitektur kanonik dari edit-tak-sadar AI; unfreeze sadar (`sudo chattr -i`) buat update. **+ §13.G SKILL SUBSYSTEM 2026-06-22 (owner: "freeze SEMUA file skill"):** +9 file skill-only router di-chattr+i + manifest (`../router/...`) → **79→88 hash**; registry komunitas (publish/pull) terbukti e2e round-trip (repo `flowork-os/flowork-skills` public, seed `ringkas-terstruktur`). Arsitektur skill: §14. **+ §13.H CONSTITUTION+PERSONA+ENRICHMENT 2026-06-22 (owner: "freeze SEMUA file desain, biar self-documenting"):** +11 file (enrichment/constitution/persona/semantic/drawer-write/seed-doctrine) di-chattr+manifest + pointer brain.md → **88→99 hash**. Tujuan: kalau AI nyasar (mis. mau "hemat token" dgn matiin enrichment/potong constitution), edit GAGAL → sadar BY-DESIGN → baca doc ini. **+ FASE-A JANTUNG 2026-06-22 (owner: "freeze SEMUA jalur brain+Skills+...+Mesh, jantung Flowork; mr-flow juga"):** +18 file → **99→117 hash**: **MESH** (17 file `../router/internal/mesh/*` — consensus/CRDT/gossip/karma/sign/identity/peers/packet/dst, federation jalur) di-chattr+manifest (dulu soft-lock); **mr-flow** `agents/mr-flow/main.go` (jantung agent owner, stabil pasca anchor-fix) chattr+manifest + `manifest.json` chattr (non-.go, manual-verify). Tiap file di-prepend pointer brain.md. Bagian dari rencana **main.go HOST agnostic** (FASE B nanti: registry pattern → main.go bootstrap buta → freeze). **+ FASE-B AGNOSTIC HOST 2026-06-22 (owner: "main.go gw freeze, bikin agnostic"):** `agent/main.go` di-refactor jadi BOOTSTRAP BUTA — feature-registry pattern (`feature_registry.go`: Deps+Feature+RegisterFeature+applyPhase). SEMUA ~220 route HTTP DIPINDAH ke `feature_*.go` (auth/compat/platform/agents/chat/ops/dev/apps) yg self-register lewat `init()`. main.go cuma: build Deps → applyPhase(Wire/Route/Seed) → ListenAndServe. **Nambah fitur = file feature_*.go BARU, NOL sentuh main.go.** `main.go` + `feature_registry.go` di-chattr+manifest (**117→119 hash**) = RANGKA beku. `feature_*.go` = TITIK-TUMBUH (editable). Wiring-hook + seed TETAP di main.go (boot-ordering sensitif; stabil/fixed → main.go tetap freezable). Runtime-verified e2e (login owner + kernel/agents/groups/settings 200 via registry, 0 route ilang). **+ FASE-C CABUT-CABANG + GAP-FREEZE 2026-06-22 (owner: "pastikan SEMUA jalur Mesh/Skills/Knowledge-base/drawer/Constitution/Memory/Personas/Instincts/enrich/Compact/DreamGraph s.d. mr-flow & agent lain BEKU; QC banyak simulasi; kalau jalur bercabang—ada logic numpang—bikin jalur baru sebab jalur ini diabadikan"):** audit nemu 2 CABANG (jalur brain numpang di file EDITABLE shared → bisa ke-drop diam-diam tanpa unfreeze) → dicabut jadi jalur sendiri + sisa gap di-freeze → **119→128 hash**. **CABANG dicabut:** (1) `agent/internal/tools/builtins/builtins.go` (EDITABLE, daftar tool campur) dulu `Register` 11 tool BRAIN bareng tool non-brain → **dicabut ke `builtins_brain.go` (BARU, FROZEN, `init()` self-register)**: brain_search_shared/brain_add/brain_search/brain_get · graph_recall · mistake_recall · brain_immune_scan/brain_verify · brain_promote_shared · codemap_search/codemap_stats (pola sama `brain_dream.go`/`instinct_recall.go` yg udah self-register). (2) `router/handlers_resources.go` (SHARED providers/combos/apiKeys/proxyPools/media) dulu nampung skill CRUD → **dicabut ke `handlers_skills_crud.go` (BARU, FROZEN)** (`/api/skills` + `/api/skills/<id>`, delegate murni ke `store/skills.go` FROZEN). **GAP di-freeze:** `agentdb/immune.go` (immune ScanAndQuarantine) · `agentdb/compact.go` (CompactStats/TrimDigestedInteractions) · 5 `router/handlers_mesh*.go` (mesh/_transport/_stack/_advanced/_ratelimit — HTTP mesh) · `router/internal/brain/doctrine_seed.json` (chattr-only, non-.go → manual-verify, ga di TestKernelFreeze). **QC BANYAK-SIMULASI LULUS:** unit agent+router 0-FAIL · hermetik 13 skenario (DreamGraph 6 / Drawer-Memory 3 / Instincts 4) PASS · `builtins_brain_test.go` (13 tool brain TETAP ter-register pasca-cabut) PASS · redeploy via watchdog (agent+router rebuilt, deployed=source mtime cocok) · live-e2e 6 subsistem (Skills-CRUD count:5 / Constitution alwaysOn+topK20 / Knowledge-shared hits / Mesh can_publish / local-AI "Siap" / agent-CGM 200) UP · **capstone mr-flow recall integrated BENAR** (Owner/Mr.Dev + lumpuh + anak Adrian/Arkana/Shanon + misi → bukti DreamGraph+recall+constitution+local-AI+mr-flow end-to-end). mr-flow TETAP frozen (hash `6f56408a…` cocok). 3-lapis utuh: JANTUNG(beku) + RANGKA(main.go+feature_registry beku) + TITIK-TUMBUH(feature_*.go/GUI editable). **SISA (nanti):** OS-sealer otomatis pas `--arm` (N3).

---

## 14. SKILL SUBSYSTEM (router :2402) — 2 sistem + registry komunitas

Ada **2 konsep "skill" terpisah** di router (jangan ketuker):

**(A) Prompt-template skills** (`store.Skill`, disimpan di config DB `kv` prefix `skill:<uuid>`):
- Template prompt reusable: `{name(slug), description, systemPrompt, userTemplate (pakai {{var}}), defaultModel, temperature, maxTokens}`. Variabel **auto-extract** dari `{{...}}` (`extractVariables`).
- CRUD: `GET/POST /api/skills` + `PUT/DELETE /api/skills/<id>` — handler THIN di `handlers_resources.go` (non-frozen), logic di `internal/store/skills.go` (**FROZEN**: ListSkills/GetSkillByName/UpsertSkill/DeleteSkill/RenderSkillTemplate).
- Invoke: `GET /v1/skills/` (list) · `POST /v1/skills/<name>` `{variables,model?,temperature?,max_tokens?,stream?}` → render template (`{{var}}`→nilai) → susun pesan (system+user) → `DispatchChatCompletion` → balikin completion (`handlers_skills_invoke.go`, **FROZEN**). Model kosong → `defaultModel` skill (boleh `flowork-brain` lokal).
- GUI: tab **Skills** (create/list/run). NON-frozen (viz/form evolve).

**(B) SKILL.md behavioral skills** (markdown + frontmatter `---`, di `DynamicSkillsDir` = `~/.flow_router/skills`):
- Skill BAWAAN = embedded (`//go:embed`); skill AUTHORED = file `.md` di dir. Di-inject ke request lewat Brain config "Inject skills" (topK) — `internal/brain/skills.go` (**FROZEN**: DynamicSkillsDir/loadDynamicSkills/Skills/SelectSkills).

**REGISTRY KOMUNITAS** (`internal/skillregistry/registry.go` + `handlers_skillregistry.go`, **FROZEN**) — share SKILL.md via GitHub `flowork-os/flowork-skills` (override env `FLOWORK_SKILL_REGISTRY`):
- **3 GERBANG kepercayaan:** (1) **publish** butuh karma-gate (`skillpack.CanPublish`: endorsed-owner ATAU proven-lokal uses≥min & positif≥min) + **sign** provenance (`mesh.SignData`). (2) **pull** = verify **signature** (`mesh.VerifyData`) + verify **content** (`skillpack.VerifyContent`: tolak dangerous/injection) + frontmatter `---` wajib + anti path-traversal, SEBELUM import (registry = untrusted). Karma+pack di `handlers_skillpack.go` + `internal/skillpack/{skillpack,karma}.go`.
- Endpoint: `GET status/browse` (publik, **tanpa token**) · `POST pull?name=` · `POST publish?skill=` (loopback-only + `FLOWORK_GITHUB_TOKEN`). Browse/pull baca **fresh** via GitHub contents API (`Accept: raw`, anti-cache CDN). Publish = PUT `skills/<n>/<n>.fwskill` + merge `registry/index.json`.
- **Bukti e2e (2026-06-22):** endorse→publish(sign+push GitHub)→browse(count:1, fresh)→pull(verify-sig+verify-content+import) **round-trip LULUS**. Repo `flowork-os/flowork-skills` (public) seed skill `ringkas-terstruktur`.
