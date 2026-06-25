# BRAIN — RECALL & INGESTION (pecahan `lock/brain.md`)

> Bagian arsitektur memori Flowork. Overview + index: **`lock/brain.md`**. Topik file ini: embedding · recall 3-lapis · ingestion ke graph · auto-recall + router-inject · GUI · alur e2e.
> ⚠️ KE-TRACK repo → NOL data personal owner.

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

### 7.1 ROUTER-SIDE PROACTIVE INJECTION (gateway `:2402` — server-side, SEMUA agent incl eksternal)

Selain auto-recall agent-side (§7), **router MAKSA-inject di gateway** tiap request (`dispatcher.go`/`dispatcher_stream.go`), mode **"augment"** (nempel, ga dominasi persona), **fails-open** (brain mati → request tetep jalan). Prinsip owner: *"jangan ngarep model manggil sendiri — PAKSA injeksi"* (model lemah pun patuh, deterministik). 3 lapis:
- **Doktrin** — `maybeInjectConstitution` (`brain_constitution.go`, FROZEN): 12 sacred rule, always-on.
- **Antibodi** — `maybeInjectAntibodies` (`mistakeenrich.go`, FROZEN): mistake `karma × relevansi × decay`, MAX 3.
- **⭐ Insting (2026-06-25, FROZEN)** — `maybeInjectInstinct` (`internal/router/instinctenrich.go`, **sibling antibodi**): drawer `room=instinct_*` di shared-brain → rank **token-overlap × importance** (DETERMINISTIK, **NO vindex** → jalan walau index belum di-rebuild) → inject MAX 3. **AKAR:** insting dulu **PULL-ONLY** (`instinct_recall`, agent harus manggil sendiri = telur-ayam) → agent **"ga sadar kapan manggil tool/fitur"** (owner: *"mobil mewah tapi ga tau naiknya"*). Sekarang di-PAKSA spt doktrin/antibodi. Sumber: `internal/brain/instincts.go` (FROZEN, `ListInstinctDrawers`). **SWITCH (extend TANPA unfreeze):** `RegisterInstinctSelector` (ganti seleksi → semantic pas vindex idup / scoping #6) + `instinctenrich_ext.go` (NON-frozen growth) + ENV `FLOWORK_INSTINCT_INJECT[_MAX]` + tumbuh-via-drawer (room `instinct_*`, NOL kode). Hook 1-baris di dispatcher = soft-lock (NON-chattr). **Detail penuh: `lock/FLoworkInstincts.md` §0.5.**
- **Fondasi #6 brain-as-service:** karena 3 lapis ini SERVER-SIDE, agent LUAR (OpenClaw/Cursor/Claude Code) yg nembak `:2402` **ikut ber-jiwa-AOLA** tanpa client ngerti Flowork. (#6: insting `room=instinct_tool` nanti di-SKIP buat agent luar via selector-hook — mereka punya tool sendiri.)

---

## 8. GUI — Cognitive Graph tab
- Front-end: `agent/web/tabs/cognitive.js` (D3 **force-directed graph**, "balls connected"). `TYPE_COLOR` map warna per-type + legend + truncate label (anti-berantakan) + klik node → detail.
- Fetch: `GET /api/agents/cognitive/graph?id=<agent>&limit=2000`.
- Back-end handler: `agentmgr/cognitive_handlers.go` `CognitiveGraphHandler` → `ListCogNodes` + edges.
- web di-EMBED ke binary (`//go:embed web` di `main.go`) → ubah GUI = rebuild host.

---

## 11. ALUR END-TO-END (contoh: 1 fakta dari chat → recall)
1. Owner ngomong fakta di chat → `interactions` tersimpan.
2. Ticker digest (`cognitive_digest_cron`) → agent `dream-digester` ekstrak → `cognitive_extract` → gerbang `cognitive_gate` (anti-halu) → dedup `ResolveByEmbedding` → `UpsertNode` (label di-`EmbedText`→`Quantize`→embedding).
3. Lain kali owner tanya (kata beda) → `fetchAutoRecall` (mr-flow main.go) → `graph_recall` embed query → `SearchNodesByEmbedding` cosine → fact-sheet → inject Tier-3 → LLM jawab pakai fakta.
4. GUI: node muncul di tab Cognitive Graph (D3), warna per-type.

**Untuk subsistem (skills/constitution/edu/drawer):** langkah-2 diganti **projeksi** (`graphsync`: baca tabel sumber → EmbedText → Quantize → UpsertNode type sesuai). Recall + GUI sama.

---
