# Models

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Kelola metadata model: alias (map nama virtual → model real), custom models (model yang ditambah manual), disabled models (model yang disembunyiin dari listing/dispatch), plus catatan availability (hasil test up/down/degraded). Juga nyediain katalog model OpenAI-style (`/v1/models`) dan katalog model Kiro/Amazon Q (`/api/kiro/models`).

## Endpoint (router/routes.go)
- `GET  /api/models` → `modelsListHandler` (registerProviderRoutes) — delegasi ke `modelsHandler`
- `*    /api/models/` → `modelsRouterHandler` — sub-route dispatcher
- `GET  /v1/models` → `modelsHandler` (registerChatRoutes) — katalog gabungan
- `GET  /api/kiro/models` → `kiroModelsHandler` (registerManagementRoutes)
- `POST /api/kiro/models/invalidate` → `kiroModelsInvalidateHandler`

Sub-route lewat `modelsRouterHandler` (di `handlers_models_meta.go`):
- `/api/models/alias` (GET list, POST upsert) → `modelsAliasHandler`
- `/api/models/alias/{alias}` (DELETE) → `modelsAliasCRUDHandler`
- `/api/models/availability` (GET, POST) → `modelsAvailabilityHandler`
- `/api/models/custom` (GET, POST) → `modelsCustomHandler`
- `/api/models/custom/{id}` (DELETE) → `modelsCustomCRUDHandler`
- `/api/models/disabled` (GET list, POST disable, DELETE enable) → `modelsDisabledHandler`
- `/api/models/test` (POST) → `modelsTestHandler`

## Logic / Alur
- **modelsHandler (`/v1/models`, `/api/models`)**: iterasi semua provider aktif, kumpulin model (skip `*` dan kosong), buang yang `IsModelDisabled` (cek per `provider` dan per `ID`), lalu tambahin custom models (`owned_by=custom`) dan alias (`owned_by=alias`, `provider`=model real). Output `{object:"list", data:[...]}`.
- **alias**: GET → `ListModelAliases`; POST wajib `alias`+`model`, `UpsertModelAlias`; DELETE per alias path.
- **custom**: GET → `ListCustomModels`; POST wajib `model`, `UpsertCustomModel`; DELETE per id.
- **disabled**: GET → `ListDisabledModels`; POST wajib `provider`+`model` (+reason) `DisableModel`; DELETE via query `provider`+`model` → `EnableModel`.
- **availability**: GET → `ListModelAvailability`; POST → `RecordAvailability`.
- **test (POST)**: kirim prompt kecil (default "ping", MaxTokens 16) lewat `router.DispatchChatCompletion` dengan timeout 30s, ukur latency, klasifikasi `up/degraded/down`, dan `RecordAvailability`.
- **kiroModelsHandler (GET)**: butuh query `token` (Kiro OAuth access_token), opsional `profileArn`/`region`. Panggil `kiromodels.Fetch` (timeout 30s) yang fetch ke `q.<region>.amazonaws.com/ListAvailableModels`, cache 5 menit. `invalidate` (POST) → `InvalidateCache`.

## File yang dilewati
- Handler meta: `router/handlers_models_meta.go`
- Handler katalog `/v1/models`: `router/handlers_chat.go` (`modelsHandler`)
- Handler Kiro: `router/handlers_kiromodels.go`
- Store: `router/internal/store/modelmeta.go` (`ModelAlias`, `ModelCustom`, `ModelDisabled`, `ModelAvailability`, `List/Upsert/Delete/Disable/Enable/IsModelDisabled/RecordAvailability`)
- Provider listing: `router/internal/store` (`ListProviders`, `ListCustomModels`, `ListModelAliases`)
- Dispatch test: `router/internal/router` (`DispatchChatCompletion`, `OpenAIRequest`)
- Kiro pkg: `router/internal/kiromodels/kiromodels.go` (`Fetch`, `InvalidateCache`, `Params`)
- Frontend: `router/web/static/index.html` (`data-tab="models"`)

## Teknologi
Go `net/http`, manual path-prefix routing (`modelsRouterHandler`), SQLite (store), HTTP client ke Amazon Q endpoint dengan in-memory cache (sha256 key, TTL 5 menit), JSON encode/decode.

## Status freeze
FROZEN — `handlers_models_meta.go`, `handlers_kiromodels.go`, `handlers_chat.go`, `internal/store/modelmeta.go`, dan `internal/kiromodels` semua punya header `⚠️ FROZEN`. Penambahan fitur lewat SEAM non-frozen + SWITCH (`internal/fwswitch/registry.go`). GUI `web/static/index.html` TIDAK frozen.
