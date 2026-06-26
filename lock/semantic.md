# SEMANTIC ("Simatic") — Recall By-Makna + Brain Self-Maintaining

> Dokumen kanonik (white-label). Cara Flowork ngerti & narik ingatan **by-MAKNA** (bukan
> cuma kata), + gimana doktrin/insting/knowledge baru **otomatis** jadi semantic.
> Owner: Aola Sahidin. Repo: https://github.com/flowork-os/Flowork-OS.

---

## 0. APA ITU SEMANTIC

**Semantic = recall by-MAKNA.** Tiap potong ingatan (drawer/insting/doktrin) diubah jadi
**vektor** (sidik-jari makna, 1024-dim). Pas ada query, query juga di-vektor → dibandingin
pakai **cosine similarity** → narik yang PALING MIRIP MAKNANYA, walau ga ada kata yang sama.

Contoh: query "audit kontrak" nemu insting "smart-contract checklist" — beda kata, sama makna.
Itu yang keyword (FTS) ga bisa, semantic bisa.

⚠️ **Penting (batas semantic):** vektor cuma buat **NEMU** drawer yang relevan; yang DIPAKAI/
di-inject ke model = **TEKS** drawer-nya (dari tabel `drawers`), BUKAN vektornya. Vektor ga bisa
di-decode balik ke teks. Jadi semantic ≠ tempat nyimpen rahasia — dia mesin pencari, bukan brankas.

---

## 1. KOMPONEN

| Komponen | Peran |
|---|---|
| **Embedder `bge-m3`** (lokal, via Ollama/llama.cpp) | ubah teks → vektor 1024-dim. ENGINE SAMA buat index & query (biar vektor align — beda engine = recall turun diam-diam). |
| **`brain.vindex`** | index vektor utama, **8-bit quantized** (kecil + cepat). Hasil `cmd/brain-buildindex`. |
| **`drawers` (sqlite)** | sumber TEKS + metadata (room/wing/mem_type). Yang di-inject = ini. |
| **`memory_fts` (FTS5)** | index KEYWORD (BM25). Instan tiap nambah drawer. Fallback + komplemen semantic. |
| **vec store `_rag/flowork-brain-vec-v2.sqlite`** | embedding mentah hasil `cmd/brain-reembed`, bahan buildindex. |

---

## 2. ALUR RECALL — `SemanticRetrieve`

```
query → embed (bge-m3) → cari di brain.vindex (cosine top-k) → ambil TEKS dari drawers
                                      └─ MERGE dgn fresh-index (lihat §3) → urut skor → top-k
   (kalau vindex belum ada / kosong → FALLBACK FTS keyword sementara, fail-safe)
```

Murni VECTOR (bukan hybrid FTS+vector yg bikin bingung). FTS cuma fallback transisi. Tombstoned
(`deleted_at`) selalu di-exclude. Endpoint: `GET /api/brain/search-drawers?query=…&k=…`.

### 2b. LANTAI RELEVANSI ABSOLUT (#3 fix 2026-06-26) — "search valid"

**Masalah:** `SemanticRetrieve` (FROZEN) menormalisasi skor ke hit teratas (`sn.Score =
h.Score/top`) → hasil #1 **selalu 1.0** walau query-nya sampah, dan ga ada cara nyaring hit
loosely-related. bge-m3 punya baseline cosine tinggi (~0.40 buat apa pun se-bahasa) → tanpa lantai,
search selalu balikin top-k → **berasa "ngak valid"** (semua keliatan match sama rata).

**Fix (NON-frozen, semantic.go tetep beku):** handler `/api/brain/search-drawers` pakai
`brain.SemanticRetrieveScored` (file `semantic_threshold_ext.go`) yg:
- skor = **cosine ABSOLUT** (`vecindex.(*Index).Cosine`, file `cosine_ext.go`) — bukan rasio ke top;
- buang hit di bawah **lantai** `FLOWORK_SEARCH_MINSCORE` (default **0.45**; `0` = matiin lantai).

Hasil terkalibrasi (data asli): query relevan 0.50–0.56 (topik tepat: crypto→instinct_crypto,
tool→instinct_tool), query di luar korpus (mis. "hukum waris islam") → **kosong** (jujur "ga ada
knowledge", bukan noise). Rumus konversi int8-dot→cosine + bukti order-equivalent: lihat header
`cosine_ext.go` + test `cosine_ext_test.go` (`TestCosineApproxAbsolute`).

---

## 3. ⭐ TIGA LAPIS INDEX (kunci "nambah apapun → otomatis semantic")

| Lapis | Auto pas nambah drawer? | Cakupan |
|---|---|---|
| **FTS keyword** (`memory_fts`) | ✅ **INSTAN** | semua drawer; ke-recall by-kata + token-overlap injection seketika |
| **Fresh-index** (vektor kecil in-memory) | ✅ **AUTO ≤2 menit** | drawer paling-baru (cap `freshMaxDrawers`), mem_type di `freshMemTypes` |
| **Main `brain.vindex`** | ❌ **MANUAL** (`reembed`+`buildindex`) | seluruh korpus; rebuild "nyerap" fresh → main, reset fresh-index |

**Fresh-index** = jawaban "brain self-maintaining": index vektor KEDUA, kecil, in-memory,
di-rebuild **tiap 2 menit** (change-detect → skip kalau set ga berubah → murah), di-MERGE ke
`SemanticRetrieve`. Aman by-construction: index utama (jutaan) GAK disentuh; fresh kosong/error →
recall persis perilaku lama (0 regresi).

**Cakupan fresh-index = `freshMemTypes`:** `project` + `doctrine` + `basic_instinct` + `reference`
+ federation (`recovery_instinct`/`collective_knowledge`). Artinya **nambah doktrin/insting/
knowledge apapun → otomatis ke-recall by-makna dalam ≤2 menit, TANPA rebuild manual.**

---

## 4. SEMANTIC DIPAKAI DI MANA

1. **GUI search** (`/api/brain/search-drawers`) + halaman Brain (instinct/knowledge).
2. **Seleksi insting injeksi** — router maksa-inject insting relevan tiap request; pemilihan
   pakai semantic (cosine via vindex) lewat `RegisterInstinctSelector`. Lihat `lock/FLoworkInstincts.md`.
3. **Brain-as-service** (enrich request external) — knowledge relevan disuntik server-side.
4. **Tool recall** (`instinct_recall`/`brain_search` agent-side).

---

## 5. SWITCH (ENV — evolusi tanpa buka freeze)

| ENV | Guna |
|---|---|
| `FLOWORK_BRAIN_VINDEX` | path index utama (default `<exe>/brain/brain.vindex` > cwd `brain/`) |
| `FLOWORK_FRESH_MEMTYPES` | override TOTAL daftar mem_type fresh-index (comma) — atur cakupan auto-semantic |
| `FLOWORK_INSTINCT_SEMANTIC` | `0` = seleksi insting balik ke token-overlap (matiin semantic-select) |
| `FLOWORK_SEARCH_MINSCORE` | lantai relevansi absolut search-drawers (default `0.45`; `0` = matiin; naikin = lebih ketat) |

---

## 6. NAMBAH DATA + MAINTENANCE

- **Nambah drawer** (doktrin/insting/knowledge): `POST /api/brain/drawer {content, wing, room, memType}`.
  → langsung kena FTS + token-overlap; semantic nyusul ≤2 menit (fresh-index). **Cukup ini buat harian.**
- **Full re-index** (perlu kalau korpus GEDE berubah / mau "nyerap" fresh ke main vindex):
  ```
  cd router
  go run ./cmd/brain-reembed     -brain brain/flowork-brain.sqlite -out brain/_rag/flowork-brain-vec-v2.sqlite
  go run ./cmd/brain-buildindex  -vec brain/_rag/flowork-brain-vec-v2.sqlite -out brain/brain.vindex
  # restart router
  ```
  `reembed` resumable + non-destruktif (change-detect, embed cuma yg baru). `buildindex -scale 0` = auto.
- ⛔ **JANGAN** hapus `brain.vindex` (itu BUKAN sampah — ngancurin recall semantic) · **JANGAN** embed
  pakai engine beda dari runtime (vektor ga align → recall turun diam-diam).

---

## 7. FILE KUNCI

| File | Peran | Status |
|---|---|---|
| `router/internal/brain/semantic.go` | `SemanticRetrieve` (vector + FTS-fallback + merge fresh) | LOCKED |
| `router/internal/brain/fresh_index.go` | fresh-index core (`RebuildFreshIndex`/`freshRetrieve`) | LOCKED soft |
| `router/internal/brain/fresh_index_ext.go` | lebarin `freshMemTypes` (auto-semantic semua tipe) + ENV switch | **FROZEN** |
| `router/internal/brain/vecindex/ann.go` | index vektor (`Build`/`Search`) + `Quantize` 8-bit | — |
| `router/internal/brain/vecindex/cosine_ext.go` | int8-dot mentah → cosine ABSOLUT (`(*Index).Cosine`) | **FROZEN** 2026-06-26 |
| `router/internal/brain/semantic_threshold_ext.go` | `SemanticRetrieveScored` (cosine absolut + lantai relevansi) | **FROZEN** 2026-06-26 |
| `router/handlers_brain_views.go` | handler search pakai scored+lantai (`searchMinCosine`) | NON-frozen |
| `router/cmd/brain-reembed` · `cmd/brain-buildindex` | re-embed + build vindex | soft-lock |
| Data: `router/brain/flowork-brain.sqlite` · `brain.vindex` · `_rag/…vec-v2.sqlite` | korpus + index + vec | gitignored |

---

## 8. BINARY VECTOR RECALL (#5) — ✅ SELESAI 2026-06-26 (lever SKALA, auto >=1jt)

**Binary-vector** (sign-LSH): sign-bit tiap dim (int8>0→1) → query XNOR vektor → **popcount** =
agreement (≈cosine). 2-tahap: (1) COARSE biner ambil top-M kandidat (cepet, 128 byte/vektor); (2)
RERANK int8 exact (`SearchSubset`) di M kandidat → top-k final akurat. **Recall@10 = 1.00** (test
`TestBinaryVectorRecall`, rerank exact).

- File: `vecindex/binary_ext.go` (NON-frozen) + route 1-baris di `vecindex.go Search`.
- **AUTO-DETECT** (owner): `FLOWORK_BINARY_VECTOR=auto` (default) → AKTIF cuma kalau drawer **>= 1 juta**
  (`FLOWORK_BINARY_VECTOR_MIN`). Di bawah itu = int8 full-scan biasa → **ZERO perubahan** (sekarang 860k
  → dormant). `on`=paksa, `off`=mati. GUI Switch Fitur "Brain / Search".
- Sig di-cache per-Index (lazy, ~128MB @ 1jt). Cuma jalur VECTOR (FTS/graph/konstitusi ga kena —
  **konstitusi default tetep selalu nempel**, verified inject 13 rules post-change).
- Belum di-freeze: teruji recall tapi BELUM proven di skala 1jt nyata → freeze pas udah jalan di 1jt.
