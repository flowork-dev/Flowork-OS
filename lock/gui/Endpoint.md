# Endpoint

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab info koneksi. Menampilkan base URL OpenAI-compatible `http://<origin>/v1` (default `http://localhost:2402/v1`) untuk dipakai client eksternal (Cursor, Codex, Claude Code, SDK apa pun). Menampilkan 3 statistik ringkas: Active Providers, Available Models, Presets Library, plus contoh perintah curl untuk uji cepat. Tab ini read-only — tidak mengubah data, hanya membaca dan menampilkan.

## Endpoint (router/routes.go)
- `GET /v1` → `v1IndexHandler` (handlers_gaps.go) — daftar dialect (openai/anthropic/gemini) + daftar endpoint yang tersedia.
- `GET /api/health` → handler inline di `registerStaticAndHealth` (routes.go) — service/status/version/uptime.
- `GET /api/version` → `versionHandler` (handlers_locale.go) — version, runtime Go, OS/arch, startedAt, uptimeSec, updateChan.
- Untuk hitung statistik, frontend (`loadEndpointStats`) memanggil:
  - `GET /api/providers` → `providersListAddHandler` (handlers_resources.go)
  - `GET /api/presets` → `presetsHandler` (handlers_resources.go)
  - `GET /v1/models` → `modelsHandler` (handlers_chat.go) — CATATAN: count "Available Models" pakai `/v1/models`, bukan `/api/models`.

## Logic / Alur
- Base URL dihitung di sisi browser dari `window.location.origin` (fungsi `renderEndpoints`), jadi selalu cocok dengan port aktual; tidak hardcode.
- `loadEndpointStats` jalankan 3 fetch paralel (`Promise.all`):
  - Active Providers = jumlah `pr.data` yang `isActive === true`.
  - Available Models = `md.data.length` dari `GET /v1/models`.
  - Presets Library = `ps.data.length` dari `GET /api/presets`.
- `modelsHandler` (GET): buka store, list provider aktif, kumpulkan model dari `Data[CfgModels]` (skip `*`, kosong, duplikat, dan model yang di-disable), lalu tambah custom model + alias, balikkan `{object:"list", data:[...]}`.
- `presetsHandler`: balikkan `{data: store.Presets}` (slice statis di kode, bukan dari DB).
- `versionHandler` (GET-only): tolak method non-GET dengan 405.
- Contoh curl ditampilkan statis di elemen `#ep-curl` (model contoh `claude-haiku-4-5`).

## File yang dilewati
- `/home/mrflow/Documents/FLowork_os/router/routes.go` — registrasi route `/v1`, `/api/health`, `/api/version`, `/api/providers`, `/api/presets`, `/v1/models`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_gaps.go` — `v1IndexHandler`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_locale.go` — `versionHandler`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_chat.go` — `modelsHandler` (sumber count model).
- `/home/mrflow/Documents/FLowork_os/router/handlers_resources.go` — `providersListAddHandler`, `presetsHandler`.
- `/home/mrflow/Documents/FLowork_os/router/internal/store/providers.go` — `ListProviders`, struct `ProviderConnection`.
- `/home/mrflow/Documents/FLowork_os/router/internal/store/presets.go` — slice `Presets`.
- `/home/mrflow/Documents/FLowork_os/router/web/static/index.html` — `data-tab="endpoint"`, section `#tab-endpoint`, fungsi `renderEndpoints` + `loadEndpointStats`.

## Teknologi
- Go `net/http` stdlib (ServeMux, HandleFunc).
- SQLite store `internal/store` (provider list, custom models, alias).
- Presets = data statis di Go (`store.Presets`), bukan DB.
- Frontend vanilla JS + `fetch` (`Promise.all`), Tailwind utility classes di index.html.

## Status freeze
- FROZEN (header `⚠️ FROZEN` di baris atas file): `routes.go`, `handlers_gaps.go`, `handlers_locale.go`, `handlers_chat.go`, `handlers_resources.go`, `internal/store/providers.go`, `internal/store/presets.go`.
- NON-FROZEN: `web/static/index.html` (GUI tidak pernah di-freeze).
