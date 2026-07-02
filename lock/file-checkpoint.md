# File checkpointing + undo (F-G #1) — 2026-07-02

## Kenapa
Jaring pengaman evolusi mandiri: agent boleh salah/typo/error — tiap `file_write`/`edit`
otomatis nyimpen isi LAMA dulu, jadi selalu bisa DIBALIKIN. Melengkapi default
approval `bypass` (kebebasan penuh + rollback murah).

## Arsitektur (semua dicolok, core frozen NOL disentuh)
- `agent/internal/tools/builtins/file_checkpoint.go` (FROZEN 2026-07-02, perintah owner):
  - Interceptor `file-checkpoint` (colok `tools.RegisterInterceptor`): sebelum
    `file_write`/`edit`, snapshot isi lama ke `<workspace-agent>/.checkpoints/`
    (nama `<unixnano-19digit>__<base64url(rel)>.snap`; file belum ada → marker `.snap.absent`).
    **Best-effort: gagal snapshot TIDAK pernah ngeblok tulisan.** Cap 100 (tertua dibuang).
  - Tool `file_checkpoints` (read-only) — list snapshot per file (id/waktu/ukuran/absen).
  - Tool `undo_file` — restore (default terbaru; pilih via `checkpoint:<id>`);
    kondisi sekarang di-snapshot dulu → redo bisa; marker absen → file DIHAPUS (undo create).
  - Resolusi path pakai `resolveFileArgs` yang SAMA dengan tool file → nol drift.
- `file_checkpoints` didaftarin read-only di `permission_policy.go`.

## QC 2026-07-02
Unit 5/5 PASS (snapshot/undo/redo, undo-create-hapus, pilih id, prune cap,
interceptor-never-blocks). E2E via /api/agents/tools/run: write v1 → write v2 →
undo → isi balik "versi 1" di disk. Delete-test: build sukses tanpa file ini.
Catatan: workspace file-tool = `agent/workspace/<id>/` (bukan ~/.flowork).
