# Text To Speech

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab untuk sintesis teks menjadi audio (TTS) lewat provider media kategori `tts`. Router memilih provider aktif dari store, lalu mengubah teks jadi audio: bila provider punya adapter in-process dipakai langsung, bila punya BaseURL maka di-proxy ke endpoint upstream `/audio/speech` (kompatibel OpenAI). Tab juga menyediakan daftar suara (voices) per vendor untuk dipilih sebelum sintesis.

## Endpoint (router/routes.go)
- `POST /api/media-providers/tts` → `mediaTTSHandler` (handlers_media_ext.go)
- `GET /api/media-providers/tts/voices` → `ttsVoicesHandler` (handlers_gaps.go)
- `GET /api/media-providers/tts/deepgram/voices` → `deepgramVoicesHandler` (handlers_media_tts_voices.go)
- `GET /api/media-providers/tts/elevenlabs/voices` → `elevenlabsVoicesHandler` (handlers_media_tts_voices.go)
- `GET /api/media-providers/tts/inworld/voices` → `inworldVoicesHandler` (handlers_media_tts_voices.go)
- `GET /api/media-providers/tts/minimax/voices` → `minimaxVoicesHandler` (handlers_media_tts_voices.go)
- `POST /v1/audio`, `POST /v1/audio/` → `audioV1Handler` (handlers_chat_v1.go) — dispatch kategori `tts` ke `/audio<rest>`
- `GET /v1/audio/voices` → `audioVoicesHandler` (handlers_gaps.go) → memanggil `ttsVoicesHandler`

## Logic / Alur
mediaTTSHandler (POST):
1. Hanya menerima POST. Buka store, ambil daftar provider kategori `tts` (`store.ListMediaProviders`).
2. Decode JSON body: `text`, `voice`, `model`, `providerId`, `format`. `text` wajib.
3. Pilih provider aktif pertama (atau yang `providerId`-nya cocok). Bila tidak ada → 501 NotImplemented.
4. Jika `BaseURL` kosong: pakai adapter in-process `tts.Get(provider)`. Bila adapter tidak ada → 400 dengan daftar `tts.List()`. Bila ada → `impl.Speak(...)` (timeout 60s), tulis audio + Content-Type (default `audio/mpeg`).
5. Jika `BaseURL` ada: bangun body upstream (`model`/`input`/`voice`/`response_format`, default voice `alloy`, default model `tts-1`), POST ke `BaseURL + /audio/speech` via `router.OutboundClient` (timeout 60s, Bearer APIKey), salin header + body balasan.

Voices handler (GET): ambil API key vendor aktif (`firstActiveAPIKey`), GET ke endpoint vendor (Deepgram `/v1/models`, ElevenLabs `/v1/voices`, Inworld `/tts/v1/voices`, MiniMax POST `/v1/get_voice`), normalisasi jadi `voiceEnvelope` (grup per bahasa), dukung filter `?lang=`. `ttsVoicesHandler` mencoba proxy `/audio/voices` upstream; bila gagal kembalikan daftar suara OpenAI bawaan.

## File yang dilewati
- `router/handlers_media_ext.go` — `mediaTTSHandler` (FROZEN)
- `router/handlers_media_tts_voices.go` — voices per vendor: deepgram/elevenlabs/inworld/minimax (FROZEN)
- `router/handlers_chat_v1.go` — `audioV1Handler`, `dispatchMedia` (FROZEN)
- `router/handlers_gaps.go` — `ttsVoicesHandler`, `audioVoicesHandler` (FROZEN)
- `router/internal/store` — `ListMediaProviders`, `MediaProvider`, `MediaCategoryTTS`
- `router/internal/providers/tts` — registry adapter (`Get`/`List`/`Speak`); provider terdaftar: openai, elevenlabs, deepgram, edgeTts, inworld, minimax, gemini, googleTts, openrouter, localDevice
- `router/internal/router` — `OutboundClient`
- `router/web/static/index.html` — sidebar `data-tab="media-tts"` (label "Text To Speech")

## Teknologi
Go `net/http` (mux `HandleFunc`), SQLite store, adapter pattern per-provider (registry `sync.RWMutex`), reverse-proxy ke API vendor, kompatibilitas skema OpenAI `/audio/speech`. Frontend HTML/JS statis.

## Status freeze
FROZEN — handlers_media_ext.go, handlers_media_tts_voices.go, handlers_chat_v1.go, handlers_gaps.go, dan paket internal/providers/tts semuanya berheader `⚠️ FROZEN`. Penambahan fitur via SEAM non-frozen + SWITCH (internal/fwswitch/registry.go). GUI `web/static/index.html` TIDAK frozen.
