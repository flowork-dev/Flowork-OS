# Usage Analytics

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab `data-tab="usage"` menampilkan konsumsi token dan biaya (cost USD) hasil rekap request yang lewat router. User bisa lihat agregat per hari, per provider, history per request, statistik total, sampai detail body request/response satu transaksi. Sumber datanya tabel `usageDaily`, `usageHistory`, dan `requestDetails` di SQLite store.

## Endpoint (router/routes.go)
Didaftarkan di `registerManagementRoutes`:

- `GET /api/usage` -> `usageHandler` (handlers_obs.go) — agregat usage, opsi query `from`/`to`.
- `/api/usage/` -> `usageBreakdownRouter` (handlers_usage_breakdown.go) — sub-router yang membagi ke:
  - `today` -> `usageTodayHandler` (handlers_obs.go)
  - `chart` -> `usageChartHandler`
  - `history` -> `usageHistoryHandler`
  - `providers` -> `usageProvidersHandler`
  - `request-details` -> `usageRequestDetailsHandler`
  - `request-logs` -> `usageRequestLogsHandler` (delegasi ke `consoleLogHandler`)
  - `stats` -> `usageStatsHandler`
  - `stream` -> `usageStreamHandler` (SSE)
  - sisanya (path non-kosong) -> `usageByConnectionHandler` per provider/connection.

## Logic / Alur
Semua sub-handler hanya menerima method `GET` (selain itu balas 405).

- `usageHandler` (GET): `store.Open()` -> baca query `from`/`to` -> `store.AggregateUsage(d, from, to)` -> JSON `{data, count}`.
- `usageTodayHandler` (GET): `store.Open()` -> `store.TodaySummary(d)` -> JSON ringkasan hari ini.
- `usageChartHandler` (GET): parse `days` (default 7, clamp 1..366) -> SUM dari `usageDaily` di-`GROUP BY day` -> JSON `{series, days}`.
- `usageHistoryHandler` (GET): parse `limit`/`offset`/`provider` -> SELECT dari `usageHistory` urut `id DESC` -> JSON `{data, count, limit, offset}`.
- `usageProvidersHandler` (GET): GROUP BY provider dari `usageHistory` (count, token, cost, avg latency, last seen) -> JSON `{data, count}`.
- `usageRequestDetailsHandler` (GET): wajib query `id`; SELECT 1 baris dari `requestDetails` (termasuk requestBody/responseBody/statusCode/error) -> 404 kalau `sql.ErrNoRows`.
- `usageStatsHandler` (GET): hitung total request, token, cost, error count, jumlah provider distinct -> JSON.
- `usageStreamHandler` (GET): set header `text/event-stream`, kirim `event: ready` lalu polling tiap 1 detik baris baru `id > lastID` dan emit `event: usage`; berhenti saat context selesai.
- `usageByConnectionHandler` (GET, connID): agregat satu provider/connection dari `usageHistory`.

## File yang dilewati
- `/home/mrflow/Documents/FLowork_os/router/routes.go` (registrasi route)
- `/home/mrflow/Documents/FLowork_os/router/handlers_obs.go` (`usageHandler`, `usageTodayHandler`, `consoleLogHandler`)
- `/home/mrflow/Documents/FLowork_os/router/handlers_usage_breakdown.go` (router + semua sub-handler)
- `/home/mrflow/Documents/FLowork_os/router/handlers_util.go` (`writeJSON`)
- `/home/mrflow/Documents/FLowork_os/router/internal/store/requestlog.go` (`AggregateUsage`, `TodaySummary`, `ListRecent`)
- Tabel SQLite: `usageDaily`, `usageHistory`, `requestDetails`
- `/home/mrflow/Documents/FLowork_os/router/web/static/index.html` (`data-tab="usage"`)

## Teknologi
- Go `net/http` (ServeMux, handler GET).
- SQLite via `internal/store` (`database/sql`, query langsung `d.Query`/`d.QueryRow`).
- SSE (Server-Sent Events) untuk `stream` pakai `http.Flusher` + `time.Ticker`.
- Output JSON via `encoding/json` dan helper `writeJSON`.

## Status freeze
- `handlers_obs.go` — FROZEN (header "⚠️ FROZEN").
- `handlers_usage_breakdown.go` — FROZEN.
- `internal/store/requestlog.go` — FROZEN.
- `routes.go` — FROZEN.
- `web/static/index.html` (GUI) — TIDAK frozen.
