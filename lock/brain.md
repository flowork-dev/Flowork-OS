# BRAIN — Arsitektur Memori Flowork: Subsistem & Cara Terhubung
> Dokumen referensi (white-label). Menjelaskan SEMUA subsistem memori, file penghubung,
> keputusan teknologi, dan cara mereka tersambung. Owner: Mr.Dev. Update terakhir: 2026-06-22.
> ⚠️ File ini KE-TRACK repo → NOL data personal owner (mekanisme generic doang).

---

---

## 📑 INDEX — doc memori Flowork (SPLIT per-topik, 2026-06-25)

brain.md dulu monolitik 49KB → dipecah biar bersih + gampang maintain. Mapping § lama:
- **`lock/brain.md`** (INI) — §0 filosofi · §1 lapis penyimpanan · §2 subsistem · §12 ringkas + index ini.
- **`lock/brain-recall.md`** — §4 embedding (bge-m3) · §5 recall 3-lapis · §6 ingestion ke graph · §7 auto-recall + router-inject · §8 GUI · §11 alur end-to-end.
- **`lock/brain-files.md`** — §9 peta file lengkap · §10 keputusan teknologi.
- **`lock/brain-skill.md`** — §14 skill subsystem (router :2402).
- **`lock/CognitiveGraph.md`** — §3 Cognitive Graph (CGM) detail (udah ada, jangan dobel).
- **`KERNEL_FREEZE.md`** — §13 daftar file FROZEN + SHA256 (manifest kanonik, di-enforce TestKernelFreeze).

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

→ **Detail PINDAH ke `lock/CognitiveGraph.md`** (model node/edge · orphan · kontradiksi/tension + loop klarifikasi · tools CGM · switch `cognitive_ext.go`). brain.md cuma index sekarang.

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

## 13. BRAIN-CORE — file inti FROZEN

→ **Daftar file frozen + SHA256 + history PINDAH ke `KERNEL_FREEZE.md`** (manifest kanonik, di-enforce `TestKernelFreeze`). Pola nano-modular: file brain-pathway terpisah di-FREEZE; orkestrator (`main.go`) EDITABLE.
