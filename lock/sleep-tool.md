# Tool `sleep` sejati (F-C) — 2026-07-02

## Kenapa
Kernel WASM SINKRON: agent "tidur" dengan nge-block = 1 semut nyandera engine +
turn ke-kill turn-timeout. `sleep` sejati = jadwalin bangun lalu AKHIRI turn;
bangun otomatis pas jatuh tempo. Hemat resource pas idle / nunggu lama.

## Arsitektur (reuse, nol mesin baru)
- `agent/internal/tools/builtins/sleep_tool.go` (FROZEN 2026-07-02): tool `sleep`
  nulis 1 baris ke tabel `wakeups` yang SAMA dengan `ScheduleWakeup` → engine
  `RunDueWakeups` (wakeup_engine.go, frozen) yang UDAH ada nge-fire-nya. NOL engine baru.
  - Return `{sleeping:true, end_turn:true, wake_at, wakeup_id, seconds}` — `end_turn`
    = sinyal ke model/loop biar berhenti bersih (jangan lanjut iterasi).
  - Default prompt bangun = **tick cari-kerjaan**: "cek kerjaan pending / tugas
    nyangkut / pesan owner SEBELUM tidur lagi" → tidur ga nutupin kerjaan.
  - Beda dari sibling: `Monitor` (nunggu kondisi singkat, sinkron ≤60s) ·
    `ScheduleWakeup` (lanjutan tugas spesifik). `sleep` = idle/jeda hemat.
- NON-FROZEN-secara-arsitektur (deletable): hapus → tool ilang, engine wakeup +
  agent tetep jalan.

## QC 2026-07-02
Unit (schedule row + validasi) PASS · build/vet/TestKernelFreeze/delete-test hijau ·
live: /api/agents/tools/run sleep → row wakeups kejadwal, return end_turn:true.
