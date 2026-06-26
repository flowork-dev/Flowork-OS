# CODEMAP ENRICH — self-map semantik + incremental by-hash (M1)

> Tab `:1987 #codemap`. Lapisan MAKNA di atas struktur codemap (file+import). Owner: Mr.Dev.

## APA INI
Tiap file source → analisa LLM kecil → `{summary, domain, role}` disimpan di tabel `codemap_semantic`
(state.db agent). Bikin codebase bisa dicari/dimengerti by-makna; dibutuhkan semantic-search +
self-evolve (`selfevolve.go` minta "reindex + enrich dulu").

## OTAK = AGENT `codemap-enricher` (bukan hardcode)
Enrich dijalanin AGENT khusus `codemap-enricher` (di `~/.flowork/agents/codemap-enricher.fwagent`,
di-seed boot, persona "CODEBASE CARTOGRAPHER", **model dari GUI** per-agent). Host kirim isi file →
agent balas JSON → parse → simpan. Fallback `routerChat` (model enricher sama) kalau agent invalid.
Endpoint: `POST /api/codemap/enrich?limit=&force=&model=` (owner-gated, incremental, cap 24KB/file).

## AUDIT 2026-06-26 (VERIFIED berfungsi)
- `codemap_files` = 411 (ter-index). `codemap_semantic` sempat 0 → **enrich belum pernah dijalanin**,
  BUKAN rusak.
- Dijalanin → row terisi, provenance `model = "claude-opus-4-8 (codemap-enricher)"` = **lewat AGENT,
  bukan fallback** → fitur + agent khusus TERBUKTI berfungsi.
- Full-populate 411 file dijalanin (background, model lokal lambat).

## M1 — incremental BY-HASH (cabut-akar staleness)
**Masalah lama:** enrich skip by-PATH (`done[path]`) → file yang BERUBAH **tak pernah** di-enrich
ulang (peta-diri basi; fatal krn Flowork self-evolve = kode berubah terus). Tabel tak simpan hash.
**Fix:** kolom `content_hash` (migrasi additif `ALTER TABLE`, idempotent) + skip by-HASH: hash sha256
konten file → sama = skip, **beda/baru = re-enrich**. File:
- `agent/internal/agentdb/codemap_semantic.go` — kolom + `CodemapSemanticHashes()` + Upsert simpan hash.
- `agent/internal/agentmgr/codemap_semantic.go` — handler baca file → hash → skip by-hash.
Verifikasi: kolom `content_hash` kebentuk; build+vet+TestKernelFreeze PASS.

## STATUS FREEZE
Codemap files **TIDAK di-freeze dulu** (sengaja): masih ada kerja lanjutan terencana (auto-trigger
enrich saat kode berubah, projeksi enrich → CGM/brain). Freeze sekarang = mateng evolusi yg belum
selesai. Tabel `codemap_semantic` = extension additive (aman dari AI lain: gak sentuh file frozen).

## SISA (opus_roadmap.md Bagian 6 M2–M4)
- M2: wire codemap → Knowledge Graph (`LinkCodemapToGraph` dormant) + bawa makna enrich ke node.
- M3: ingest enrich → brain drawers (`brain_search` bisa jawab "fitur X di file mana").
- M4: AUTO-trigger enrich saat kode berubah (hook self-evolve = butuh sesi owner-attended; +
  safety-net terjadwal). Pakai agent codemap-enricher sbg pekerja.

## UPDATE 2026-06-26 — M4 AUTO-ENRICH SELESAI (jalur aman)
`feature_autosync.go` loop `codemapAutoEnrichLoop` tiap interval panggil `CodemapEnrichHandler`
LANGSUNG (bypass middleware auth = trigger sistem). Hash-skip (M1) → MURAH pas kode stabil (0 LLM);
file BARU/BERUBAH di-enrich ulang. Switch GUI: `FLOWORK_CODEMAP_AUTOENRICH` (default ON) + `_MIN` (30).
Hook self-evolve write-path (root) tetap parked (core sensitif); scheduled-trigger ini nutup goalnya.
