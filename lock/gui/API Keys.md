# API Keys

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab buat bikin dan cabut (revoke) API key flow_router berformat `flr_xxxx`. Key ini dipakai client (Cursor, Codex, dll) buat auth ke endpoint `/v1/...` dan `/v1beta/...` lewat header `Authorization: Bearer flr_...` atau `x-api-key`. Tiap key bisa dibatasi allowed providers plus daily cap dan monthly cap dalam USD.

## Endpoint (router/routes.go)
- `/api/keys` → `apiKeysListAddHandler` (handlers_resources.go) — GET list, POST generate.
- `/api/keys/` → `apiKeyCRUDHandler` (handlers_resources.go) — DELETE revoke per id.
- Enforcement saat request masuk: `apiKeyMiddleware` (handlers_apikey_auth.go), aktif hanya buat path v1 (`isV1Path`).

## Logic / Alur
- GET `/api/keys`: `store.ListAPIKeys(d)` baca tabel `apiKeys`, balikin list (id, name, keyPrefix, allowedProviders, cap, isActive, createdAt, lastUsedAt). KeyHash & plaintext tidak diekspos.
- POST `/api/keys`: body `{name, allowedProviders, dailyCapUsd, monthlyCapUsd}`. Kalau name kosong dikasih default `key-<pid>`. Panggil `store.GenerateAPIKey` → buat 32 byte random, plaintext `flr_` + hex, disimpan sebagai SHA-256 hash; `keyPrefix` = 14 char pertama + `...`. Plaintext hanya dibalikin sekali di response POST (status 201). `allowedProviders` kosong jadi `*`.
- DELETE `/api/keys/{id}`: hanya method DELETE diizinkan; `store.DeleteAPIKey` hapus row dari `apiKeys`. Method selain DELETE → 405.
- Enforcement (`apiKeyMiddleware`): ambil token via `extractAPIKey` (header `x-api-key` atau `Authorization: Bearer`). Kalau settings `requireApiKey` true dan token kosong/bukan prefix `flr_`/invalid → 401. `store.VerifyAPIKey` cek hash di tabel `apiKeys` (isActive=1) dan update `lastUsedAt` async. Cek cap per-key (`capExceeded` via `store.SpendSince`) dan budget global (`globalBudgetExceeded` via `store.TotalSpendSince`) → 429 kalau lewat.

## File yang dilewati
- `router/handlers_resources.go` — `apiKeysListAddHandler`, `apiKeyCRUDHandler`.
- `router/handlers_apikey_auth.go` — `apiKeyMiddleware`, `extractAPIKey`, `capExceeded`, `globalBudgetExceeded`, `writeAPIKeyError`.
- `router/internal/store/apikeys.go` — `APIKey`, `GenerateAPIKey`, `ListAPIKeys`, `DeleteAPIKey`, `VerifyAPIKey`, `SpendSince`, `TotalSpendSince`.
- `router/internal/router` — `WithAPIKey`, `WithClientIP`, `WithAgentID` (context propagation).
- `router/web/static/index.html` — `data-tab="api-keys"`.

## Teknologi
Go net/http, SQLite (tabel `apiKeys`, `usageDaily`), crypto/rand + SHA-256, google/uuid. Frontend HTML/JS statis.

## Status freeze
FROZEN — `handlers_resources.go`, `handlers_apikey_auth.go`, dan `internal/store/apikeys.go` punya header FROZEN. GUI `web/static/index.html` TIDAK frozen.
