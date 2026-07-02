# Secret scrubber rekursif (F-G) — 2026-07-02

## Kenapa
Token/API-key/password yang ke-echo di chat/error/tool-output bisa awet di DISK
(state.db agent: interactions/decisions/mistakes). Scrubber ini motong kredensial
jadi `[REDACTED:<prefix>…<4 akhir>]` sebelum persist — bocor dicegah, korelasi log
tetep bisa (4 char terakhir disisain).

## Arsitektur (chokepoint, semua colokan)
- `internal/secscrub/secscrub.go` (FROZEN 2026-07-02): mesin scrub REKURSIF
  (string/map/slice, maxDepth 12, non-destruktif = return copy). Pola token: Anthropic
  `sk-ant-`, OpenAI `sk-`, GitHub `ghp_`/`github_pat_`, AWS `AKIA`, Slack `xox`,
  Google `AIza`, JWT, `Bearer …`, assignment `password=`/`"api_key":…`, + redact
  by-NAMA key (password/token/secret/authorization/cookie/…). Dependency-free.
- `internal/agentdb/sanitize_seam.go` (FROZEN, baru): SEAM Pola B `SanitizeText` +
  `SanitizeMeta` (default no-op). **CHOKEPOINT**: dipanggil DI DALAM
  `LogInteraction`/`LogDecision`/`AddMistake` (3 file agentdb, FROZEN, re-hash) →
  SEMUA call-site sink (kernelhost, agentmgr, builtins, slashcmd, runtime host)
  ke-cover sekali. agentdb TIDAK import secscrub (colokan doang).
- `internal/kernelhost/kernelhost.go` (FROZEN, re-hash): SEAM `SanitizeLogged*` di
  choke-point invoke (lapisan kedua, defense in depth).
- `agent/secscrub_ext.go` (FROZEN, deletable-secara-arsitektur): colok
  secscrub → seam agentdb + kernelhost via init(). Hapus file → sink balik no-op
  (perilaku lama), core tetep build (delete-test PASS).

## QC 2026-07-02
Unit secscrub (token/clip-tail/rekursif map+slice/nil+depth) PASS · integration
`agentdb.TestSanitizeSeam_WiredIntoSinks` (tulis token ke 3 sink → baca balik, nol
bocor) PASS · build/vet/TestKernelFreeze/delete-test hijau · live: token di /api/chat
ga muncul mentah di interactions (0 match `ghp_...`).

## Sisa (buat penerus)
- Router-side sink (kalau ada log token di router) — piistrip udah handle PII prompt;
  cek apakah perlu secscrub juga di router trace.
- Tool-output besar (grep/bash) → udah lewat sink yang sama; kalau ada sink lain
  di luar 3 method agentdb, colok ke seam yang sama.
