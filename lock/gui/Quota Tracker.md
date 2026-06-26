# Quota Tracker

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab `data-tab="quota-tracker"` menampilkan konsumsi token dan biaya per provider untuk jendela hari ini / 7 hari / 30 hari, plus info reset dan status health tiap provider. Selain rekap dari store, tab ini bisa fetch kuota live langsung ke API provider (mis. Claude) untuk lihat sisa kuota nyata (used/total/remaining per window).

## Endpoint (router/routes.go)
Didaftarkan di `registerManagementRoutes`:

- `GET /api/quota-tracker` -> `quotaTrackerHandler` (handlers_obs.go) — rekap kuota per provider dari store.
- `GET /api/quota-tracker/live` -> `quotaLiveHandler` (handlers_quotalive.go) — fetch kuota live dari provider, wajib query `provider`, opsi `token`.

## Logic / Alur
Keduanya hanya `GET` (selain itu 405).

- `quotaTrackerHandler` (GET): `store.Open()` -> `store.ListQuotaStatus(d)` -> JSON `{data, count}`. `ListQuotaStatus` loop tiap provider (dari `ListProviders`), lalu SUM dari tabel `usageDaily` untuk window hari ini (`day = today`), minggu (`day >= now-7d`), dan bulan (`day >= now-1mo`); kalau ada `quotaResetHours` di data provider, hitung `resetAt`.
- `quotaLiveHandler` (GET):
  1. Parse query `provider`; kalau kosong balas 400 + daftar `quotalive.List()`.
  2. `quotalive.Get(provider)` cari fetcher terdaftar; kalau nil balas 501 (not implemented) + supported list.
  3. Resolve token: pakai query `token` kalau ada; kalau tidak, `resolveLiveToken(provider)` — untuk `claude` ambil dari `creds.LoadValid().ClaudeAiOauth.AccessToken`, provider lain balas error "no auto-token loader".
  4. `context.WithTimeout` 30 detik -> `impl.Fetch(ctx, quotalive.Params{Token})`.
  5. Sukses -> JSON `Snapshot` (provider, plan, fetchedAt, windows[]); error fetch -> 502.

## File yang dilewati
- `/home/mrflow/Documents/FLowork_os/router/routes.go` (registrasi route)
- `/home/mrflow/Documents/FLowork_os/router/handlers_obs.go` (`quotaTrackerHandler`)
- `/home/mrflow/Documents/FLowork_os/router/handlers_quotalive.go` (`quotaLiveHandler`, `resolveLiveToken`)
- `/home/mrflow/Documents/FLowork_os/router/internal/store/quota.go` (`ListQuotaStatus`, `quotaResetHours`)
- `/home/mrflow/Documents/FLowork_os/router/internal/quotalive/quotalive.go` (registry: `Register`/`Get`/`List`, tipe `Snapshot`/`Window`/`Params`/`LiveFetcher`)
- Fetcher per provider: `internal/quotalive/claude.go`, `codex.go`, `copilot.go`, `gemini_cli.go`, `glm.go`, `kiro.go`, `minimax.go`, `antigravity.go`, `informational.go`
- `/home/mrflow/Documents/FLowork_os/router/internal/creds/` (`LoadValid` untuk token Claude)
- Tabel SQLite: `usageDaily` (+ `ListProviders`)
- `/home/mrflow/Documents/FLowork_os/router/web/static/index.html` (`data-tab="quota-tracker"`)

## Teknologi
- Go `net/http`.
- SQLite via `internal/store` (agregat `usageDaily`).
- `internal/quotalive`: registry fetcher (`sync.RWMutex` map), `http.Client` timeout 30 detik ke API provider.
- `internal/creds` untuk resolve OAuth access token (mis. Claude `~/.claude/.credentials.json`).
- `context` timeout untuk fetch live; output JSON via `writeJSON`.

## Status freeze
- `handlers_obs.go` — FROZEN.
- `handlers_quotalive.go` — FROZEN.
- `internal/store/quota.go` — FROZEN.
- `internal/quotalive/quotalive.go` — FROZEN.
- `routes.go` — FROZEN.
- `web/static/index.html` (GUI) — TIDAK frozen.
