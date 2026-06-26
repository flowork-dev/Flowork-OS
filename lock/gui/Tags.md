# Tags

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Label berwarna (tag) buat ditempel ke provider biar gampang dikategorikan/disaring. Tiap tag punya `id` (UUID), `name`, `color` (default `#8b5cf6`), dan `kind` (default `generic`). Tab ini CRUD murni: list, create, update, delete tag.

## Endpoint (router/routes.go)
Didaftarkan di `registerProviderRoutes`:
- `GET/POST /api/tags` → `tagsHandler`
- `PUT/DELETE /api/tags/{id}` → `tagCRUDHandler`

## Logic / Alur
- **tagsHandler**:
  - GET: `ListTags` (urut by name ASC) → `{data, count}`.
  - POST: decode `store.Tag`, `UpsertTag` (kalau `ID` kosong dibikin UUID baru + `createdAt` now; default color/kind diisi). Balik 201.
- **tagCRUDHandler**: ambil `id` dari path suffix `/api/tags/`.
  - PUT: decode body, set `t.ID = id`, `UpsertTag` → 200.
  - DELETE: `DeleteTag(id)` → 204.

## File yang dilewati
- Handler: `router/handlers_tags.go`
- Store: `router/internal/store/tags.go` (`Tag` struct, `ListTags`, `UpsertTag`, `DeleteTag`)
- Frontend: `router/web/static/index.html` (`data-tab="tags"`)

## Teknologi
Go `net/http`, SQLite (tabel `tags`), UUID via `github.com/google/uuid`, upsert pakai `ON CONFLICT(id) DO UPDATE`, timestamp RFC3339.

## Status freeze
FROZEN — `handlers_tags.go` dan `internal/store/tags.go` punya header `⚠️ FROZEN — jangan edit file ini`. Penambahan fitur lewat SEAM non-frozen + SWITCH (`internal/fwswitch/registry.go`). GUI `web/static/index.html` TIDAK frozen.
