# Translator

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab untuk mengonversi request/response antar format provider LLM: OpenAI, Anthropic, dan Gemini. Bisa konversi statis (translate payload tanpa kirim) atau live (`send`): payload dinormalisasi ke format kanonik OpenAI, di-dispatch ke router, lalu response dikembalikan dalam format target. Mendukung simpan/muat draft konversi.

## Endpoint (router/routes.go)
- `mux.HandleFunc("/api/translator", translatorRouterHandler)` (baris 140)
- `mux.HandleFunc("/api/translator/", translatorRouterHandler)` (baris 141)

Handler: `translatorRouterHandler` di `handlers_translator.go`.

## Logic / Alur
`translatorRouterHandler` mem-parse sisa path setelah `/api/translator` lalu mengarahkan ke sub-handler:
- `""` atau `save` → `translatorListUpsertHandler`: `GET` list draft (`store.ListTranslatorDrafts`), `POST` upsert draft (`store.UpsertTranslatorDraft`, wajib `sourceFormat`+`targetFormat`).
- `load` → `translatorLoadHandler`: `GET` ambil 1 draft by id (`store.GetTranslatorDraft`).
- `translate` → `translatorTranslateHandler`: `POST` konversi statis via `translateFormat(src, dst, payload)` (tidak kirim ke upstream). Konversi internal: `anthropicToOpenAI`/`geminiToOpenAI` → kanonik OpenAI → `openAIToAnthropic`/`openAIToGemini`.
- `send` → `translatorSendHandler`: `POST` jalur live. `normalizeToCanonical` ubah payload sumber jadi `router.OpenAIRequest`, set `MaxTokens` default (`helpers.DefaultMaxTokens`) bila kosong, lalu `router.DispatchChatCompletion` (timeout 120s). Response di-format ulang ke target via `formatResponseAs` + sertakan usage token.
- `console-logs` → `translatorConsoleLogsHandler`: `GET` mengembalikan data kosong (phase2_pending).
- `console-logs/stream` → `translatorConsoleLogsStreamHandler`: SSE snapshot 20 entri terakhir (`store.ListRecent`) + keepalive tiap 5 detik.
- sisanya → `translatorCRUDHandler(id)`: `GET` 1 draft, `DELETE` hapus draft (`store.DeleteTranslatorDraft`).

Helper konversi: `anyToText` (string/array content → teks), `normalizeToCanonical` (openai/anthropic/gemini → OpenAIRequest), `formatResponseAs` (OpenAIResponse → format target).

## File yang dilewati
- `router/handlers_translator.go` — handler utama + semua sub-handler + fungsi konversi format.
- `router/internal/store` — `TranslatorDraft`, `ListTranslatorDrafts`, `UpsertTranslatorDraft`, `GetTranslatorDraft`, `DeleteTranslatorDraft`, `ListRecent`.
- `router/internal/router` — `OpenAIRequest`, `OpenAIResponse`, `OpenAIMessage`, `DispatchChatCompletion`.
- `router/internal/translator/helpers` — `DefaultMaxTokens`.
- `router/web/static/index.html` — frontend, `data-tab="translator"` (baris 159).

## Teknologi
Go `net/http`, `encoding/json`, SSE (`http.Flusher`, `text/event-stream`), SQLite via `internal/store`, dispatch chat via `internal/router`.

## Status freeze
FROZEN — `handlers_translator.go` ber-header `⚠️ FROZEN`. Package `internal/store`, `internal/router`, `internal/translator` juga frozen. `routes.go` (route-registration) dan `web/static/index.html` (GUI) NON-frozen.
