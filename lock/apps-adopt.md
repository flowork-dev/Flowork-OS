# 🌉 APPS-ADOPT — Jembatan "Repo → App" (Sidecar)

Arsitektur fitur **adopt**: repo mentah (git-URL / folder) → app yang dijalanin **MANUSIA (GUI) & AI (tool)**,
tanpa build manual. Reuse penuh substrat apps (`runtime:process`, manifest, reloadOne, app_grants).

## Inti
Repo mentah ga ngerti protokol core Flowork. Jembatannya = **CLI-Adapter Core generik** (`fw-app-adapter`):
1 binary, ngomong protokol stdio (`proc.go`: `{op,args}` ↔ `{result,state_version}`), nerjemahin tiap `op`
→ command repo yg dipetakan di `adapter.json`. Engine `runtime:process` jalanin adapter sbg core → **nol ubah engine**.

```
clone/copy repo → deteksi runtime → install dep KE FOLDER → tulis manifest.json + adapter.json → reloadOne → LIVE
  app/<id>/repo/        (kode + venv/node_modules/target = dep lokal; hapus folder = bersih)
  app/<id>/adapter.json (workdir "repo" + ops: run→RunCmd, arg_style args_list)
  app/<id>/manifest.json(runtime:process, core_entry=<fw-app-adapter>, op run tool:true → tool agent app_<id>_run)
```

## File (semua SEAM — nol file frozen lama disentuh)
| File | Peran | Status |
|---|---|---|
| `internal/apps/cliadapter/adapter.go` | **CORE** adapter CLI: loop stdio + exec argv (no shell) + placeholder/flags/args_list/json_stdin + resolve program relatif ke workdir + timeout | **LOCKED** (hash+chattr) |
| `cmd/fw-app-adapter/main.go` | binary core_entry CLI (cwd=folder app) | **LOCKED** (hash+chattr) |
| `internal/apps/adopt/detect.go` | **CORE** deteksi runtime (python/node/go/rust) + **registry switch** `RegisterDetector` (POLA A: runtime baru via sibling, NOL unfreeze) | **LOCKED** (hash+chattr) |
| `internal/apps/httpadapter/adapter.go` | **CORE** adapter HTTP (F5): spawn server repo + tunggu port + op→HTTP proxy + `_url`/`_alive` | **LOCKED** (hash+chattr) |
| `cmd/fw-http-adapter/main.go` | binary core_entry HTTP (web app/API) | **LOCKED** (hash+chattr) |
| `internal/apps/adopt/scan.go` | **CORE** pre-flight scanner (F6): pola berbahaya (rm-rf/pipe-shell/reverse-shell/SSRF) → `ScanRepo` | **LOCKED** (hash+chattr) |
| `internal/apps/adopt_ext.go` | orchestration `AdoptRepo`/`AdoptHTTPRepo`/`DetectSource`/`prepareAdopt` (sibling; panggil reloadOne) | non-frozen (growth) |
| `internal/apps/adopt_fsutil_ext.go` | util fs/json (copyTree, writeJSON) | non-frozen |
| `feature_app_adopt_ext.go` | SEAM route `/api/apps/adopt` + `/api/apps/detect` (init→RegisterFeature) | non-frozen (deletable) |

## Kontrak (cara repo dijembatani)
| Kontrak | Buat | Adapter | Alur op |
|---|---|---|---|
| **CLI** | script/CLI (yt-dlp dll) | `fw-app-adapter` | op "run" → exec command repo → stdout |
| **HTTP** (F5) | server (streamlit/fastapi/express) | `fw-http-adapter` | spawn server → tunggu port → op→HTTP; `_url` buat GUI iframe |
| MCP (F5, belum) | MCP server | — | register ke MCP client router (reuse) |

`AdoptHTTPRepo` → `httpadapter.json` {workdir, start_cmd, port, ready_path, url_path, ops} + manifest op
`_url`(gui)+ops(tool). Kontrak dipilih owner di adopt (`contract:"http"`+`http:{...}`) = bagian "setting dikit".

## Switch / evolusi (Rule #7)
- **Runtime baru** (ruby/php/deno…) → sibling `init(){ adopt.RegisterDetector(...) }`, ga sentuh `detect.go` (beku).
- **Pola scan baru** (ransomware/miner/dll) → sibling `init(){ adopt.RegisterScanRule(label,sev,regex) }`, ga sentuh `scan.go` (beku).
- **Kontrak baru** (MCP dll) → adapter/binary BARU (cliadapter & httpadapter beku) — bukan edit yang ada.
- Hapus `feature_app_adopt_ext.go` → fitur adopt mati mulus, core utuh (self-sufficient).
- **Self-sufficient terbukti** (delete-test 2026-06-27): hapus adopt_ext/fsutil/feature → `go build ./...` exit 0.

## Keamanan
- **Consent exec WAJIB** (`?approve_exec=1`) — clone+install = perintah OS, owner buka gerbang (bukan AI).
- **Pre-flight scan (F6)** `adopt.ScanRepo`: scan kode repo (sebelum install/run) buat pola berbahaya
  (rm-rf destruktif, pipe-ke-shell, reverse-shell, cloud-metadata SSRF, fork-bomb, dll). **Critical → adopt
  DIBLOK** (rollback) kecuali `accept_risk=1` (consent sadar). Warn → catat di notes. Findings tampil di `detect` preview.
- Dep di folder (isolasi: hapus folder = bersih). Path adapter di-resolve runtime (no-hardcode). White-label.

## Tier isolasi (JUJUR — beda per-OS, jangan janji rata)
- **OS-appliance (Linux):** bubblewrap (`os/rootfs-overlay/.../flowork-app-run`) → isolasi proses KUAT (no-net default, ga bisa baca `~/.flowork`).
- **Portable Win/Mac/Linux:** **NOL sandbox proses** — cuma isolasi DEP (folder). Untrusted = akses home user.
  Mitigasi: pre-flight scan + consent. Web app port = relax (server denger port).
- Implikasi ditulis terang biar owner sadar saat approve repo untrusted.

## Verifikasi (litmus LULUS 2026-06-27)
CLI: `detect`→`adopt` app LIVE · op run (manusia) `LIVE-ADOPT-OK` · mr-flow (bahasa-manusia) "Outputnya: `LIVE-ADOPT-OK`".
HTTP: E2E spawn `python http.server` → ready → GET proxy 200. Scan: malicious repo (rm-rf) DIBLOK; clean PASS. `TestKernelFreeze` PASS.
**LITMUS DUNIA-NYATA (2026-06-27):** MoneyPrinterTurbo (github) → adopt contract=http → install 100+ dep (py3.12) →
`_alive` start Streamlit → UI :8501 HTTP 200. Scanner nangkep Dockerfile `rm-rf` (false-positive standar apt) → owner accept_risk.

## Build wiring adapter (wajib sebelah flowork-agent, resolve via binPath)
fw-app-adapter (CLI) + fw-http-adapter (HTTP), KEDUANYA di 3 jalur:
- ✅ **dev** (`agent/start.sh`): loop build dua adapter (idempotent, non-fatal).
- ✅ **portable** (`os/portable/make-portable.sh`): build per-OS + copy+chmod ke install bin.
- ✅ **appliance** (`os/build/build-flowork-os.sh`): build static + install `/usr/local/bin/`.

## Belum (roadmap)
- F5-MCP (deteksi repo MCP → register MCP client router) — deprioritas (HTTP udah cukup buat web app).
- Auto-deteksi kontrak (saran http kalau ada streamlit/fastapi/docker-compose port) — sekarang owner pilih manual.
- Scanner refine: kurangi false-positive Dockerfile (rm-rf apt-cleanup) tanpa lengah ke yg jahat.
