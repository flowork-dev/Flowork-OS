# Settings

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Konfigurasi router yang auto-save tiap ada perubahan (PATCH). Atur default model, fallback strategy, RTK token saver, local AI autostart, budget, brain, plus snapshot/inspeksi database, backup, proxy-test, dan require-login. Password tidak pernah dikirim balik ke client.

## Endpoint (router/routes.go)
- `/api/settings` → `settingsHandler` (handlers_obs.go) — GET load, PUT/PATCH `store.PatchSettings`.
- `/api/settings/database` → `settingsDatabaseHandler` (handlers_settings_sub.go) — GET snapshot.
- `/api/settings/backups` → `settingsBackupsHandler` (handlers_backup.go) — GET list, POST backup.
- `/api/settings/proxy-test` → `settingsProxyTestHandler` (handlers_settings_sub.go) — POST.
- `/api/settings/require-login` → `settingsRequireLoginHandler` (handlers_settings_sub.go) — GET/PUT.

## Logic / Alur
- GET `/api/settings`: `store.LoadSettings`, kosongin `Password`, balikin JSON Settings (defaultModel, fallbackStrategy, rtkTokenSaver, budget, brain, requireApiKey, intent/cost routing, dll).
- PUT/PATCH `/api/settings`: decode body jadi `map[string]any`, panggil `store.PatchSettings(d, patch)` (merge partial → auto-save), kosongin Password, balikin Settings terbaru. Ini yang dipakai GUI buat auto-save on change.
- GET `/api/settings/database`: hitung `COUNT(*)` tiap tabel inti (providerConnections, providerNodes, apiKeys, usageDaily, usageHistory, requestDetails, combos, proxyPools, kv, tags, pricing, modelAlias, modelAvailability, authSessions, translatorDrafts, modelsCustom, modelsDisabled) plus `store.DBPath()`.
- GET `/api/settings/backups`: `store.ListBackups`. POST: body `{label, keepN}` → `store.Backup(label, keepN)` (status 201).
- POST `/api/settings/proxy-test`: body `{url, proxyUrl, timeoutMs}`. Guard SSRF lewat `blockMetadataURL` buat url dan proxyUrl. Bikin http.Client (timeout default 10s, optional proxy), GET target, balikin `{reachable, statusCode, latencyMs}` atau error.
- GET/PUT `/api/settings/require-login`: GET balikin `{requireLogin, authMode, passwordSet, oidcConfigured}`. PUT update `requireLogin`, `authMode`, `password` (di-hash via `hashPassword`), `oidcConfig` lalu `store.SaveSettings`.

## File yang dilewati
- `router/handlers_obs.go` — `settingsHandler`.
- `router/handlers_settings_sub.go` — `settingsDatabaseHandler`, `settingsProxyTestHandler`, `settingsRequireLoginHandler`.
- `router/handlers_backup.go` — `settingsBackupsHandler`.
- `router/internal/store/settings.go` — `Settings`, `LoadSettings`, `PatchSettings`, `SaveSettings`, `Budget`, `BrainConfig`, `DBPath`.
- `router/internal/store` — `ListBackups`, `Backup`.
- `router/handlers_ssrf_guard.go` — `blockMetadataURL` (guard proxy-test).
- `router/web/static/index.html` — `data-tab="settings"`.

## Teknologi
Go net/http, SQLite (tabel settings/kv + tabel inti lain), JSON merge-patch (PatchSettings), SSRF guard buat proxy-test, password hashing. Frontend HTML/JS statis dengan auto-save.

## Status freeze
FROZEN — `handlers_obs.go`, `handlers_settings_sub.go`, `handlers_backup.go`, dan `internal/store/settings.go` punya header FROZEN. GUI `web/static/index.html` TIDAK frozen.
