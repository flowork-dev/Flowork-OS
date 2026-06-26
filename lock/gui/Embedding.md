# Embedding

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab untuk mengelola provider embedding (teks → vector): OpenAI (ada-002 dll), BGE, Cohere, Voyage, dan kompatibel-OpenAI. Provider disimpan sebagai media-provider kategori `embedding`. Endpoint `/v1/embeddings` meneruskan request ke provider aktif sebagai gateway.

## Endpoint (router/routes.go)
- `mux.HandleFunc("/api/media-providers", mediaProvidersHandler)` (baris 125) — CRUD provider (`category=embedding`).
- `mux.HandleFunc("/api/media-providers/", mediaProviderCRUDHandler)` (baris 132) — update/delete per id.
- `mux.HandleFunc("/v1/embeddings", embeddingsV1Handler)` (baris 56) — gateway inference.

Handler: `mediaProvidersHandler` / `mediaProviderCRUDHandler` di `handlers_resources.go`; `embeddingsV1Handler` di `handlers_chat_v1.go`.

## Logic / Alur
- `mediaProvidersHandler`: `GET` list by `category` query (`store.ListMediaProviders(d, cat)`), `POST` upsert (`store.UpsertMediaProvider`). Untuk tab ini `category=embedding` (`store.MediaCategoryEmbedding`).
- `mediaProviderCRUDHandler`: id diambil dari path setelah `/api/media-providers/`. `PUT` update (set `Category` dari query bila kosong), `DELETE` hapus (wajib `category` query).
- `embeddingsV1Handler`: panggil `dispatchMedia(w, r, store.MediaCategoryEmbedding, "/embeddings")`.
- `dispatchMedia`: list provider kategori embedding, pilih provider pertama yang `IsActive`. Bila tidak ada → `501` + hint. Endpoint = `TrimRight(BaseURL,"/") + "/embeddings"`. Body diteruskan (limit 32MB), header disalin kecuali Host/Authorization/Content-Length, set `Authorization: Bearer <APIKey>` bila ada, kirim via `router.OutboundClient(ctx)` (timeout 60s), response upstream disalin balik apa adanya.

## File yang dilewati
- `router/handlers_resources.go` — `mediaProvidersHandler`, `mediaProviderCRUDHandler`.
- `router/handlers_chat_v1.go` — `embeddingsV1Handler`, `dispatchMedia`.
- `router/internal/store/media.go` — `MediaProvider`, `MediaCategoryEmbedding="embedding"`, `ListMediaProviders`, `UpsertMediaProvider`, `DeleteMediaProvider`.
- `router/internal/providers/embedding/` — registry provider embedding (`Register`/`Get`/`List`); implementasi: `openai`, `openaiCompat`, `gemini`, `local` (bge-m3). Catatan: jalur `/v1/embeddings` saat ini gateway HTTP murni via `dispatchMedia` ke `BaseURL` provider; registry ini menyediakan interface `EmbeddingProvider`.
- `router/internal/router` — `OutboundClient`.
- `router/web/static/index.html` — frontend, `data-tab="media-embedding"` (baris 174).

## Teknologi
Go `net/http` (reverse-proxy/gateway), SQLite via `internal/store`, registry provider `internal/providers/embedding`, outbound client `internal/router`.

## Status freeze
FROZEN — `handlers_resources.go`, `handlers_chat_v1.go`, `internal/store`, `internal/providers/embedding` semua ber-header `⚠️ FROZEN`. `routes.go` dan `web/static/index.html` (GUI) NON-frozen.
