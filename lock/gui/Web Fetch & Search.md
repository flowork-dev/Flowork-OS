# Web Fetch & Search

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab untuk pencarian web dan pengambilan/scrape halaman lewat provider media kategori `webfetch`. Pencarian (`/v1/search`) dan operasi web umum di-proxy ke provider aktif (BaseURL upstream). Pengambilan satu URL (`/v1/web/fetch`) memakai adapter fetch in-process dengan pengaman SSRF (validasi URL hanya alamat publik).

## Endpoint (router/routes.go)
- `POST /v1/search` → `searchV1Handler` (handlers_chat_v1.go) — `dispatchMedia` kategori `webfetch` ke `/search`
- `POST /v1/web`, `POST /v1/web/` → `webV1Handler` (handlers_chat_v1.go)
  - Bila path diawali `/v1/web/fetch` → diteruskan ke `webFetchHandler` (handlers_fetch.go)
  - Selain itu → `dispatchMedia` kategori `webfetch` ke `/web`

## Logic / Alur
searchV1Handler: dispatch langsung ke media kategori `webfetch`, suffix `/search` (proxy ke BaseURL provider aktif).

webV1Handler: jika path `/v1/web/fetch` → `webFetchHandler`; selain itu dispatch ke `webfetch` suffix `/web`.

webFetchHandler (handlers_fetch.go, POST):
1. Hanya POST. Decode JSON: `url` (wajib), opsional `provider`, `mode`, `apiKey`, `baseUrl`.
2. Validasi URL via `safeurl.Validate` (timeout 5s). Bila `ErrBlocked` (alamat non-publik) → 403; URL invalid → 400.
3. Pilih provider via `pickFetchProvider`: bila `provider` eksplisit dipakai; selain itu cari provider `webfetch` aktif yang adapternya terdaftar; fallback ke `raw`.
4. Ambil adapter `fetch.Get(name)`. Bila tidak ada → 400 + daftar `fetch.List()`.
5. Panggil `impl.Fetch(ctx)` (timeout 90s) dengan URL/Mode/APIKey/BaseURL. HTTP client fetch punya guard SSRF di redirect dan `safeDialContext` (tolak IP non-publik).
6. Balas JSON: `url`, `title`, `contentType`, `status`, `body`, `provider`.

dispatchMedia (handlers_chat_v1.go): pilih provider aktif kategori; bila tidak ada → 501. Bangun request ke `BaseURL + suffix`, salin header (kecuali Host/Authorization/Content-Length), set Bearer APIKey, kirim via `router.OutboundClient` (timeout 60s), salin respons.

## File yang dilewati
- `router/handlers_fetch.go` — `webFetchHandler`, `pickFetchProvider` (FROZEN)
- `router/handlers_chat_v1.go` — `searchV1Handler`, `webV1Handler`, `dispatchMedia` (FROZEN)
- `router/internal/fetch` — registry adapter (`Get`/`List`/`Fetch`), HTTP client dengan guard SSRF; provider terdaftar: firecrawl, jina, raw
- `router/internal/safeurl` — `Validate`, `IsPublic`, `ErrBlocked`
- `router/internal/store` — `ListMediaProviders`, `MediaCategoryWebFetch`
- `router/internal/router` — `OutboundClient`
- `router/web/static/index.html` — sidebar `data-tab="media-webfetch"` (label "Web Fetch & Search")

## Teknologi
Go `net/http`, SQLite store, adapter pattern fetch (registry `sync.RWMutex`), pengaman SSRF (`safeurl`: validasi pra-request + dial guard + redirect guard), reverse-proxy ke API search/scrape vendor. Frontend HTML/JS statis.

## Status freeze
FROZEN — handlers_fetch.go, handlers_chat_v1.go, dan paket internal/fetch berheader `⚠️ FROZEN`. Penambahan fitur via SEAM non-frozen + SWITCH (internal/fwswitch/registry.go). GUI `web/static/index.html` TIDAK frozen.
