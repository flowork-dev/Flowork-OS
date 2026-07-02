# Model deprecation auto-remap + warning (F-G) — 2026-07-02

## Kenapa
Config/alias yang nunjuk model id PENSIUN (mis. `claude-2.1`, `claude-3-opus`,
`gpt-3.5-turbo`) diteruskan mentah ke upstream → 404 `not_found_error` → boros
percobaan + gagal (dulu keselamatan cuma dari fallback chain ke lokal). Remap ini
motong di depan: id pensiun → pengganti hidup + warning.

## Arsitektur (colokan, nol unlock)
- `router/internal/router/model_deprecat_ext.go` (FROZEN 2026-07-02): wrap seam
  `applyInjectShaper` (jalan PERSIS sebelum `resolveModel` di dispatcher +
  dispatcher_stream) → cek `req.Model` vs `deprecatedModelMap`; kalau pensiun,
  remap + `log.Printf ⚠️` + seam `modelRemapWarn` (surface GUI, default no-op).
  Chain composable (pola inject_budget_ext) — file beku dispatcher/modelresolve
  NOL disentuh.
  - KONSERVATIF: cuma id yang JELAS pensiun dipetakan; model hidup Flowork
    (claude-haiku-4-5, claude-opus-4-8, claude-sonnet-5) TIDAK disentuh.
  - Prefix-insensitive (cabut cc//anthropic//claude//openai/ dulu).
  - Switch `FLOWORK_MODEL_REMAP` (default ON) — OFF = teruskan mentah (buat
    sengaja tes provider lama).
- Switch GUI didaftarin di `agent/internal/fwswitch/registry.go` (Router / Resilience).

## Cara nambah entri
1 baris di `deprecatedModelMap` (`"id-pensiun": "pengganti-hidup"`). Kalau
pengganti pun kelak pensiun, update nilainya.

## QC 2026-07-02
Unit (remap tabel + via shaper + escape-hatch OFF + model-hidup-utuh) PASS ·
build/vet/TestKernelFreeze/delete-test hijau · LIVE: POST claude-2.1 → log
"⚠️ model DEPRECATED claude-2.1 → auto-remap claude-opus-4-8", dapet jawaban asli
(msg_01…) bukan 404.

## Sisa
- `modelRemapWarn` masih no-op → badge/notif GUI (gabung MCP status monitor item).
