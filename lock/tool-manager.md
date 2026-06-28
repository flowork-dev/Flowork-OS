# 🔌 TOOL MANAGER — plug-and-play tools (manager beku, tool bebas)

Arsitektur tool Flowork = **POLA A (registry)**. Owner 2026-06-28: "tools buatin kernel/manager,
managernya di-lock, tool plug-and-play — edit/nambah tool TANPA buka freeze."

## Pembagian (apa yang BEKU vs BEBAS)
- **MANAGER = BEKU** (papan colokan + primitive, jangan disentuh tanpa unfreeze sadar):
  - `internal/tools/registry.go` — `Register(t Tool)` (panic kalau nama dobel) + `Lookup`.
  - `internal/tools/types.go` — `Tool` interface, `Schema`, `Param`.
  - `internal/tools/sandbox.go`, `sandbox_v3.go` — eksekusi + gate caps/approval.
  - `internal/tools/interceptors.go` — chain (`RegisterInterceptor`) + built-in path/sensitive guards.
  - `internal/tools/builtins/builtins.go` — ANCHOR package + `Init()` (tinggal primitive in-file:
    echo/now/mem). Tetep beku biar package non-kosong (import blank di main.go masih resolve) + delete-test.
- **TOOL = BEBAS (NON-frozen plug-in):** tiap file tool yang SELF-REGISTER via `init()`-nya sendiri.
  Edit/hapus/tambah bebas tanpa buka freeze. Preseden lama: `web_research.go`. Hasil migrasi: `system_power.go`.

## Cara nambah / ubah tool (TANPA buka freeze)
- **Nambah tool baru:** bikin file `internal/tools/builtins/<nama>.go` (NON-frozen), isi:
  ```go
  func init() { tools.Register(&myTool{}) }   // daftar ke papan-colokan
  type myTool struct{}
  func (myTool) Name() string { return "my_tool" }
  func (myTool) Schema() tools.Schema { ... }
  func (myTool) Capability() string { return "..." }
  func (myTool) Run(ctx, args) (tools.Result, error) { ... }
  ```
  JANGAN daftar di `builtins.go Init()` (itu beku + bakal dobel-register → panic).
- **Ubah tool:** edit file plug-in-nya langsung. Ga ada freeze. Build + restart agent.

## Prosedur MIGRASI tool lama (dari builtins.go beku → plug-in bebas)
Per tool (file yang tipenya udah di file sendiri):
1. `unlock.sh builtins.go <file>` + `chattr -i KERNEL_FREEZE.md` (FD colok).
2. Tambah `func init() { tools.Register(&xTool{}) }` di file tool-nya.
3. HAPUS `tools.Register(&xTool{})` dari `builtins.go Init()` (anti dobel → panic).
4. `go build ./...` + `go test ./internal/tools/builtins/` (cek ga dobel-register).
5. Hapus baris `internal/tools/builtins/<file>.go` dari `KERNEL_FREEZE.md`.
6. `lock.sh builtins.go` (re-hash + re-freeze + re-freeze manifest). File tool **TETAP non-beku**.
7. QC: `TestKernelFreeze` PASS + delete-test (hapus file tool → `go build` tetep OK).

## Status migrasi (SELESAI 2026-06-28 — full QC ijo)
- ✅ PLUG-IN (NON-frozen, edit/hapus/tambah tool TANPA buka freeze):
  `system_power.go`, `git.go`, `app_open.go`, `file_advanced.go`(edit/glob/grep), `skill.go`(skill/skillSearch),
  `skill_suggest.go`, `skill_author.go`, `taskflow_tools.go`(taskList/Run), `orchestration.go`(plan/todo/goal),
  `web_research.go`(scraper). → daftar via `init()` masing2.
- 🔒 TETAP BEKU (define helper/seam dipakai frozen-core → delete-test gagal kalau dihapus; daftar di builtins.go Init()):
  `shell.go`(bashTool — capWriter/scrubEnv/shellDenyPatterns dipakai claude_tools.go),
  `file.go`(fileRead/Write/List — validateCategoryAndName dipakai file_path_resolver.go),
  `web.go`(webFetch — isBlockedIP), `telegram.go`(telegramSend — telegramAPIBase),
  `agent_command.go`(InvokeAgentFunc dipakai main.go), + `builtins.go`(anchor: echo/now/mem primitive).
  → buat jadiin plug-in: SEAM-SPLIT dulu (pindah helper ke file seam beku, tool ke file plug-in).

## ⚠️ CATATAN REBUILD (penting, flaw laten lockbox)
Sebagian file `.go` mode `600` (mrflow-only) → user `flowork` GA BISA baca buat rebuild. Jadi:
- Abis EDIT tool plug-in: **rebuild sbg mrflow** dulu (`cd agent && GOWORK=off go build -o bin/flowork-gui .`),
  BARU `sudo systemctl restart flowork.service`. (flowork cuma JALANIN binary, ga rebuild — sesuai model non-sudo.)
- Kalau lupa rebuild → service auto-rebuild (start.sh) bakal GAGAL (`permission denied` baca file 600).
  Recovery: build sbg mrflow + restart. (Fix tuntas: kasih flowork read-access ke source / chmod 644 — TODO.)
- Service `GOFLAGS=-buildvcs=false` (BUKAN `-mod=mod` — bentrok workspace-mode).

## Kenapa aman (delete-test invariant)
Tiap tool SELF-REGISTER + ga direferensiin frozen-core. Hapus file tool → `tools.Register`-nya ilang →
registry kurang 1 tool, TAPI manager (papan colokan default-aman) + inti tetep build. Rumah tetep berdiri.
