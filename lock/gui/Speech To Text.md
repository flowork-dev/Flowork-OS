# Speech To Text

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab untuk transkripsi audio menjadi teks (STT) lewat provider media kategori `stt`. Audio diunggah sebagai multipart form, router memilih provider STT aktif dari store, lalu memanggil adapter in-process untuk mentranskripsi. Mendukung format respons `json`, `text`, dan `verbose_json`.

## Endpoint (router/routes.go)
- `POST /v1/audio`, `POST /v1/audio/` → `audioV1Handler` (handlers_chat_v1.go)
  - Bila path diawali `/transcriptions` atau `/translations` → diteruskan ke `transcriptionsHandler` (handlers_stt.go)
  - Selain itu → `dispatchMedia` kategori `tts` (jalur TTS)

## Logic / Alur
audioV1Handler (handlers_chat_v1.go):
1. Ambil sisa path setelah `/v1/audio`. Jika berawalan `/transcriptions` atau `/translations` → panggil `transcriptionsHandler`. Selain itu dispatch ke media TTS.

transcriptionsHandler (handlers_stt.go, POST):
1. Hanya POST. Parse multipart form (batas 32 MB).
2. Ambil field `file`, baca audio (LimitReader 32 MB).
3. Buka store, ambil provider kategori `stt` (`store.ListMediaProviders`); pilih provider aktif pertama. Bila tidak ada → 501 (saran tambah provider deepgram/assemblyai/gemini/openai).
4. Ambil adapter `stt.Get(provider)`. Bila tidak ada → 400 + daftar `stt.List()`.
5. Susun `stt.Request` (model dari form atau default provider, audio, MIME header, `language`, filename, APIKey, BaseURL). Panggil `impl.Transcribe(ctx)` dengan timeout 3 menit.
6. Format respons via `response_format`: `text` → text/plain; `verbose_json` → JSON mentah provider (atau ringkasan text/language/duration); default → `{text, language?, duration?}`.

## File yang dilewati
- `router/handlers_stt.go` — `transcriptionsHandler`, `pickFormValue` (FROZEN)
- `router/handlers_chat_v1.go` — `audioV1Handler`, `dispatchMedia` (FROZEN)
- `router/internal/store` — `ListMediaProviders`, `MediaProvider`, `MediaCategorySTT`
- `router/internal/providers/stt` — registry adapter (`Get`/`List`/`Transcribe`, `resolveAudioMIME`); provider terdaftar: openai (Whisper), deepgram, assemblyai, gemini
- `router/web/static/index.html` — sidebar `data-tab="media-stt"` (label "Speech To Text")

## Teknologi
Go `net/http` (multipart upload), SQLite store, adapter pattern per-provider STT (registry `sync.RWMutex`, HTTP client timeout 5 menit), resolusi MIME dari ekstensi file, skema respons kompatibel OpenAI transcriptions. Frontend HTML/JS statis.

## Status freeze
FROZEN — handlers_stt.go, handlers_chat_v1.go, dan paket internal/providers/stt berheader `⚠️ FROZEN`. Penambahan fitur via SEAM non-frozen + SWITCH (internal/fwswitch/registry.go). GUI `web/static/index.html` TIDAK frozen.
