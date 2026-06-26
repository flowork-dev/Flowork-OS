# Providers

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab CRUD koneksi provider LLM. User menambah/mengubah/menghapus provider (base URL, auth type, format, daftar model), menguji koneksi (single atau batch), mengambil daftar model yang disarankan dari upstream, dan mengelola provider-node ringan. Secret API key di-mask saat ditampilkan. Tab ini sumber data utama untuk routing chat.

## Endpoint (router/routes.go)
- `GET|POST /api/providers` → `providersListAddHandler` (handlers_resources.go) — list (masked) / tambah provider.
- `GET|PUT|DELETE /api/providers/{id}` → `providerCRUDHandler` (handlers_resources.go) — baca/ubah/hapus per id; sub-aksi `{id}/models`, `{id}/test`, `{id}/test-models` → `providerSubActionHandler` (handlers_gaps.go).
- `POST /api/providers/validate` → `providerValidateHandler` (handlers_providers_ext.go) — probe baseURL+apiKey.
- `POST /api/providers/test-batch` → `providerTestBatchHandler` (handlers_providers_ext.go) — test banyak provider sekaligus.
- `POST /api/providers/suggested-models` → `providerSuggestedModelsHandler` (handlers_providers_ext.go) — ambil model dari upstream + filter preset.
- `GET /api/providers/client` → `providersClientHandler` (handlers_gaps.go) — info base URL & endpoint untuk client.
- `GET /api/providers/kilo/free-models` → `providersKiloFreeModelsHandler` (handlers_gaps.go) — daftar model gratis (provider no-auth, aktif).
- `GET|POST /api/provider-nodes` + `/api/provider-nodes/{id|validate}` → `providerNodesRouterHandler` (handlers_provider_nodes.go) — node provider ringan disimpan di tabel `kv`.

## Logic / Alur
- `providersListAddHandler`: GET → `store.ListProviders`, tiap item dilewatkan `maskProviderSecret` (CfgAPIKey jadi `abcd••••wxyz`), balikkan `{data:[...]}`. POST → decode `store.ProviderConnection`, `store.UpsertProvider`, balikkan 201.
- `providerCRUDHandler`: parse path setelah `/api/providers/`. Kalau ada `/` → sub-aksi (`models`/`test`/`test-models`). GET → `GetProvider` (masked); PUT → set `p.ID=id` lalu `UpsertProvider`; DELETE → `DeleteProvider` (204).
- `providerSubActionHandler`: ambil provider, lalu `models` = `fetchProviderModels` (GET `<baseURL>/models` + auth), `test` = `probeProviderConn`, `test-models` = probe + list model lokal dengan flag reachable.
- `providerValidateHandler` (POST): wajib `baseUrl`; blok URL metadata (`blockMetadataURL`, anti-SSRF), `probeProvider` → GET `<baseURL>/models`; 401/403 = auth rejected, <500 = reachable.
- `providerTestBatchHandler` (POST): kalau `providerIds` kosong → test semua; tiap id `GetProvider` + `probeProviderConn`, kumpulkan hasil.
- `providerSuggestedModelsHandler` (POST): GET upstream `/models`, parse `data`/`models`/array bare, lalu `applyModelPreset` (mis. `openrouter-free`, `opencode-free`, atau default id+name).
- `probeProviderConn`: pilih auth dari `p.AuthType` — None / APIKey (`applyProbeAuth`: Bearer atau `x-api-key`+`anthropic-version` untuk format anthropic) / Subscription (tokenSource `claude_credentials` via `creds.Load`, cek expired).
- `providerNodesRouterHandler`: simpan node sebagai JSON di tabel `kv` prefix `provider-node:`; GET list, POST upsert (default format `openai`), `{id}` GET/PUT/DELETE, `validate` = `probeProvider`.

## File yang dilewati
- `/home/mrflow/Documents/FLowork_os/router/routes.go` — semua route di atas (`registerProviderRoutes`).
- `/home/mrflow/Documents/FLowork_os/router/handlers_resources.go` — `providersListAddHandler`, `providerCRUDHandler`, `maskProviderSecret`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_providers_ext.go` — `providerValidateHandler`, `providerTestBatchHandler`, `providerSuggestedModelsHandler`, `probeProvider`, `probeProviderConn`, `applyProbeAuth`, `applyModelPreset`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_gaps.go` — `providerSubActionHandler`, `fetchProviderModels`, `providersClientHandler`, `providersKiloFreeModelsHandler`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_provider_nodes.go` — `providerNodesRouterHandler`, `providerNodesListUpsert`, `providerNodeCRUD`, `providerNodeValidate`, `saveProviderNode`.
- `/home/mrflow/Documents/FLowork_os/router/internal/store/providers.go` — `ProviderConnection`, `ListProviders`, `GetProvider`, `UpsertProvider`, `DeleteProvider`, konstanta `Cfg*`/`AuthType*`.
- `/home/mrflow/Documents/FLowork_os/router/internal/creds/` — `creds.Load` (subscription Claude).
- `/home/mrflow/Documents/FLowork_os/router/handlers_ssrf_guard.go` — `blockMetadataURL` (anti-SSRF).
- `/home/mrflow/Documents/FLowork_os/router/web/static/index.html` — `data-tab="providers"`, section `#tab-providers`, fungsi `loadProviders`, save/test/delete provider.

## Teknologi
- Go `net/http` stdlib (ServeMux + handler manual, parse path string).
- SQLite store `internal/store` untuk provider; provider-node disimpan di tabel `kv` (key-value JSON).
- `http.Client` (`providerProbeClient`, timeout 10s) untuk probe upstream `/models`.
- Anti-SSRF `blockMetadataURL` sebelum tiap fetch keluar.
- `internal/creds` untuk auth subscription (Claude credentials).
- Frontend vanilla JS + `fetch`.

## Status freeze
- FROZEN (header `⚠️ FROZEN`): `routes.go`, `handlers_resources.go`, `handlers_providers_ext.go`, `handlers_gaps.go`, `handlers_provider_nodes.go`, `internal/store/providers.go`.
- NON-FROZEN: `web/static/index.html` (GUI tidak di-freeze).
