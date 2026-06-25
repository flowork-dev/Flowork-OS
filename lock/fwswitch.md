# FWSWITCH — Plug-and-Play Settings (switch fitur dari GUI, bukan env edit-tangan)

> Owner: Aola Sahidin (Mr.Dev). 2026-06-26. Rule 6 (plug-and-play, multi-OS, no-hardcode).
> Repo: https://github.com/flowork-os/Flowork-OS.

## AKAR yang dicabut (Rule 5)
~80 ENV `FLOWORK_*`. Switch FITUR (perilaku) kebawa di `router/flowork.local.env` — file
edit-tangan, **gitignored, INVISIBLE buat user install fresh** → ga ada yang tau cara nyalain
fitur. Cacat plug-and-play. GUI udah punya Settings (`/api/settings/keys` → `os.Setenv` live)
TAPI: (a) didesain buat secret/token, (b) `os.Setenv` cuma di proses **host :1987** → **router
:2402 ga kebagian** (beda proses) — kebukti pas live-test #11 external-scope gagal.

## SOLUSI (frozen-safe, ZERO refactor call-site)
File **lintas-proses** `~/.flowork/flowork_settings.json` (pola sama `agent_brain_config.json`):
ditulis GUI (host), **dibaca router DAN host**. Tiap proses, di STARTUP, `fwswitch.Boot()`:
1. **Apply()** — baca file → `os.Setenv` tiap key (file overwrite ENV) SEBELUM server nyala.
2. **watcher** mtime (poll 3 dtk) → re-Apply pas file berubah → **live tanpa restart**.

Efek: SEMUA `os.Getenv("FLOWORK_*")` yang udah ada (FROZEN sekalipun) otomatis baca nilai GUI.
**Ga ada satu pun call-site di-edit** → ga buka frozen (Rule 7).

## PRESEDENSI (keputusan owner 2026-06-26): **GUI menang**
`file-GUI (kalau key ada & non-kosong) > ENV > default-kode`. Key DIHAPUS dari GUI → restore ke
ENV asli (di-snapshot pas pertama di-manage). Key ga ada di file → ENV asli utuh → call-site
pakai default-nya sendiri. Cuma kelola `FLOWORK_*` (key lain spt PATH ga disentuh).

## FILE
| File | Peran | Status |
|---|---|---|
| `router/internal/fwswitch/fwswitch.go` | core Apply/Boot/watcher (router) | **FROZEN** 2026-06-26 |
| `router/fwswitch_boot.go` | `init()` → `Boot()` di package main (router/main.go ga disentuh) | **FROZEN** 2026-06-26 |
| `agent/internal/fwswitch/fwswitch.go` | core (kembaran; modul terpisah → duplikat sengaja) | **FROZEN** 2026-06-26 |
| `agent/internal/fwswitch/registry.go` | **registry kurasi switch** (metadata GUI) + `Resolve()` (sumber gui/env/default) — **EXTENSION POINT** | NON-frozen (sengaja) |
| `agent/fwswitch_ext.go` | `init()` Boot + endpoint `/api/settings/switches` (GET resolve, POST tulis file) | **FROZEN** 2026-06-26 |
| `agent/web/tabs/settings.js` | segmen GUI "🎛️ Switch Fitur" (toggle/number + badge sumber + simpan diff) | NON-frozen (GUI) |
| `~/.flowork/flowork_settings.json` | data user lintas-proses (gitignored, di luar repo) | runtime |

## NAMBAH SWITCH BARU (plug-and-play)
Tambah 1 entri di `registry.go` `Registry` (key/label/desc/type/default/category). Default WAJIB
sama dgn default di call-site. Otomatis muncul di GUI + dikelola. Ga sentuh kode lain.

## CAKUPAN (owner: "switch fitur aja")
Registry = switch perilaku: `FLOWORK_INSTINCT_SCOPED`, `FLOWORK_INSTINCT_SEMANTIC`,
`FLOWORK_INSTINCT_INJECT`, `FLOWORK_BRAIN_EXTERNAL_SCOPE`, `FLOWORK_SEARCH_MINSCORE`,
`FLOWORK_TOOLCALL_RECOVER`, `FLOWORK_DEFER_TOOLS`, `FLOWORK_EXPOSE_ALL_TOOLS`,
`FLOWORK_ROUTER_RETRY`, `FLOWORK_ORCHESTRATOR`. Hardware/path (NGL/CPU_MOE/KV_TYPE/paths) +
secret SENGAJA di luar (tetep env/auto-detect; secret lewat `/api/settings/keys`).

## VERIFIKASI
- Unit: `go test ./internal/fwswitch/` — router 4 PASS (file>env, restore, ignore non-FLOWORK),
  agent 2 PASS (WriteValues round-trip, Resolve source).
- **Live lintas-proses (Rule-9):** tulis `flowork_settings.json {FLOWORK_SEARCH_MINSCORE:"0.99"}`
  → router (:2402, proses lain) search count 5→0 dalam ≤3 dtk TANPA restart; hapus file → balik
  0.45/count 5. Host bin jalan = build dgn endpoint + segmen GUI ke-embed.

## CATATAN
- `flowork.local.env` TETAP jalan (ENV) sbg fallback ops/power-user — tapi GUI menang kalau di-set.
- Watcher pakai `os.Setenv` (goroutine-safe Go modern). Idempotent; Boot sekali per-proses.
