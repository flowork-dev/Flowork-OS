# Text to Image

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab untuk mengelola provider image-generation (teks → gambar): DALL-E/OpenAI, Stable Diffusion (sd_webui/ComfyUI), Flux (Black Forest Labs), Replicate/fal.ai, Stability AI, dll. Provider disimpan sebagai media-provider kategori `text-to-image`. Endpoint `/v1/images` meneruskan request ke provider aktif sebagai gateway.

## Endpoint (router/routes.go)
- `mux.HandleFunc("/api/media-providers", mediaProvidersHandler)` (baris 125) — CRUD provider (`category=text-to-image`).
- `mux.HandleFunc("/api/media-providers/", mediaProviderCRUDHandler)` (baris 132) — update/delete per id.
- `mux.HandleFunc("/v1/images", imagesV1Handler)` (baris 57) dan `mux.HandleFunc("/v1/images/", imagesV1Handler)` (baris 58) — gateway inference.

Handler: `mediaProvidersHandler` / `mediaProviderCRUDHandler` di `handlers_resources.go`; `imagesV1Handler` di `handlers_chat_v1.go`.

## Logic / Alur
- `mediaProvidersHandler`: `GET` list by `category` query, `POST` upsert. Untuk tab ini `category=text-to-image` (`store.MediaCategoryTextToImage`).
- `mediaProviderCRUDHandler`: `PUT` update, `DELETE` hapus (wajib `category` query).
- `imagesV1Handler`: ambil sisa path setelah `/v1/images` (mis. `/generations`), lalu `dispatchMedia(w, r, store.MediaCategoryTextToImage, "/images"+rest)`.
- `dispatchMedia`: list provider kategori text-to-image, pilih provider pertama yang `IsActive`. Bila kosong → `501` + hint. Endpoint = `TrimRight(BaseURL,"/") + "/images"+rest`. Body diteruskan (limit 32MB), header disalin kecuali Host/Authorization/Content-Length, set `Authorization: Bearer <APIKey>` bila ada, kirim via `router.OutboundClient(ctx)` (timeout 60s), response upstream disalin balik apa adanya.

## File yang dilewati
- `router/handlers_resources.go` — `mediaProvidersHandler`, `mediaProviderCRUDHandler`.
- `router/handlers_chat_v1.go` — `imagesV1Handler`, `dispatchMedia`.
- `router/internal/store/media.go` — `MediaProvider`, `MediaCategoryTextToImage="text-to-image"`, `ListMediaProviders`, `UpsertMediaProvider`, `DeleteMediaProvider`.
- `router/internal/providers/image/` — registry provider image; implementasi: `openai`, `stabilityAi`, `blackForestLabs`, `falAi`, `runwayml`, `comfyui`, `sdwebui`, `cloudflareAi`, `huggingface`, `gemini`, `nanobanana`, `codex`. Catatan: jalur `/v1/images` saat ini gateway HTTP murni via `dispatchMedia` ke `BaseURL` provider.
- `router/internal/router` — `OutboundClient`.
- `router/web/static/index.html` — frontend, `data-tab="media-text2img"` (baris 175).

## Teknologi
Go `net/http` (reverse-proxy/gateway), SQLite via `internal/store`, registry provider `internal/providers/image`, outbound client `internal/router`.

## Status freeze
FROZEN — `handlers_resources.go`, `handlers_chat_v1.go`, `internal/store`, `internal/providers/image` semua ber-header `⚠️ FROZEN`. `routes.go` dan `web/static/index.html` (GUI) NON-frozen.
