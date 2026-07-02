# PROMPT-DIET — enrichment selektif + budget agregat + sticky-union tools (router)

> Owner: Aola Sahidin (Mr.Dev) · 2026-07-02. Roadmap F-A1/F-A3 + akar "sering kena limit".
> Seam dipasang atas mandat owner ("file lock yang ngak dibuatin switch buat evolusi → buatin").

## AKAR yang dicabut (3 biji, Rule 5)
1. **Enrichment selalu nyuntik** — `maybeEnrichBrain` pakai `SemanticRetrieve` yang normalisasi
   skor ke top-hit (hit #1 = 1.0 walau query sampah) → top-K snippet disuntik TIAP call.
2. **Ga ada budget agregat** — tiap injector (knowledge/skill/insting/antibodi) punya cap sendiri,
   total gabungannya ga dijaga → worst-case prompt bengkak.
3. **Intent-gated pruning nyabotase prompt-cache** — `maybeFilterTools` mangkas tool per-QUERY
   (isi+urutan beda tiap turn). Cache Anthropic hash prefix `tools → system → messages` →
   tools berubah = SEMUA breakpoint miss = persona+history dibayar ulang tarif cache-write
   tiap call. Ini biang boros limit walau prompt-cache ON.

## SEAM (di file FROZEN, POLA B — default = perilaku lama, delete-test PASS)
| Seam | File frozen | Default |
|---|---|---|
| `enrichRetrieve(ctx,db,query,opts)` | `router/internal/router/brainenrich.go` | `brain.SemanticRetrieve` (lama) |
| `applyInjectShaper(ctx,req,settings)` | `router/internal/router/dispatcher.go` (+ dipanggil di `dispatcher_stream.go`) | no-op |

`applyInjectShaper` = titik tunggal pembentuk request PASCA semua injeksi+filter — ekstensi
masa depan (reorder cache-aware, dedup, dll) tinggal wrap di sibling, JANGAN buka frozen lagi.

## EXTENSION (sibling NON-frozen — bisa dihapus, inti tetap jalan)
| File | Isi | Switch (GUI fwswitch) |
|---|---|---|
| `router/internal/router/enrich_selective_ext.go` | retrieve pakai `SemanticRetrieveScored` (cosine ABSOLUT + lantai); 0 hit relevan → SKIP suntik. Index belum siap / error → fallback lama (fail-open) | `FLOWORK_ENRICH_MINSCORE` (float, 0=off, saran 0.30–0.45) |
| `router/internal/router/inject_budget_ext.go` | total char suntikan dikenal > budget → buang PESAN UTUH per-prioritas: knowledge(1) → insting(2) → antibodi(3). Doktrin SACRED + persona caller TIDAK PERNAH disentuh | `FLOWORK_INJECT_BUDGET` (char, 0=off, saran 6000–12000) |
| `router/internal/router/tools_sticky_ext.go` | union AKUMULATIF per-agent atas hasil pruning; urutan FIRST-SEEN append-only → prefix tools stabil → cache idup. Cuma aktif saat `FLOWORK_DYNAMIC_TOOLS` on | `FLOWORK_TOOLS_STICKY` (bool, default ON) |
| `router/internal/brain/vindex_ready_ext.go` | `VectorIndexReady()` — expose kesiapan index vektor buat fail-open | — |

Header suntikan yang dikenal budget (HARUS sinkron sama builder frozen):
`## Relevant knowledge` / `You are operating with a shared knowledge brain` / `## Applicable skills`
(brainenrich) · `## Insting —` (instinctenrich) · `## Antibodi —` (mistakeenrich) ·
`## Project doctrine` = SACRED (ga disentuh).

## BONUS FIX di file yang sama (2026-07-02)
- **Parity stream**: `dispatcher_stream.go` dulu GA nyuntik konstitusi (doktrin SACRED) di jalur
  utama (cuma di fallback) → chat streaming jalan tanpa doktrin. Sekarang gate-nya sama persis
  non-stream (`!isCrewLightModel` → `maybeInjectConstitution`).
- **BUG switch retry**: `FLOWORK_ROUTER_RETRY` terdaftar GUI sebagai bool padahal pembaca
  (mr-flow `main.go:866` + `agentkit.go:197`) baca INT jumlah-attempt (default 5) → toggle ON
  ("1") malah MATIIN retry. Registry dibetulin jadi int default 5 (+ nerve seed). Nilai live
  yang salah ("1") dibetulin ke "5".
- **BUG boot service**: `agent/start.sh` pid/log hardcode `/tmp/flowork-gui.*` → bentrok
  kepemilikan antar-user (mrflow manual vs service `flowork`) = service GAGAL boot. Fix:
  `RUN_DIR` per-user (`$XDG_RUNTIME_DIR|/tmp`/flowork-`$(id -un)`, override `FLOWORK_RUN_DIR`) +
  symlink kompat `/tmp/flowork-gui.{log,pid}` + port-in-use yang jawab HTTP = exit 2 idempoten
  (bukan failure). `stop.sh` ikut + fallback path legacy. Windows `.bat` ga kena (ga pake pid file).

## STATUS FREEZE
`brainenrich.go` / `dispatcher.go` / `dispatcher_stream.go` di-unlock (FD LOCKBOX) → seam →
re-hash `KERNEL_FREEZE.md` → `chattr +i` lagi. `TestKernelFreeze` PASS · gembok verified
("Operation not permitted") · delete-test 4 sibling → build exit 0 · unit test
`inject_budget_ext_test.go` PASS · full `go test ./...` router PASS.

## BUKTI LIVE (Rule 9, bahasa manusia)
mr-flow "coba liatin isi folder utama proyek" → tool jalan, jawab jujur, no muntah/loop.
Log router: `tools-sticky: union 1→14→15 tool (baru 0)` = urutan stabil lintas-iterasi.
