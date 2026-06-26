# Chat

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Playground di dalam dashboard untuk uji setup: kirim prompt lewat router ke model mana pun yang sudah dikonfigurasi. User pilih/ketik nama model, centang opsi stream, ketik pesan, lalu router men-dispatch ke provider yang cocok dan menampilkan jawaban (streaming token demi token atau sekali jadi). Berguna untuk verifikasi cepat bahwa routing + provider sudah jalan tanpa client luar.

## Endpoint (router/routes.go)
- `POST /v1/chat/completions` → `chatCompletionsHandler` (handlers_chat.go) — jalur utama tab Chat.
- `GET /v1/models` → `modelsHandler` (handlers_chat.go) — isi datalist pilihan model di GUI (`loadChatModels`).
- (terkait, format Anthropic) `POST /v1/messages` → `messagesV1Handler` (handlers_chat_v1.go) — bukan dipakai tab Chat, tapi satu jalur dispatch yang sama.

## Logic / Alur
- Frontend `chatSend`: baca `model`, `stream`, dan akumulasi `_chatMessages`, POST `{model, stream, messages}` ke `/v1/chat/completions`. Default model di GUI mengacu ke `flowork-brain` (otak lokal); jika input model kosong, GUI ambil model pertama dari `/v1/models`.
- `chatCompletionsHandler` (POST-only, 405 kalau bukan POST):
  1. Baca body (limit 8 MiB) → unmarshal ke `router.OpenAIRequest`.
  2. `tryClaudeCliBypass` — kalau request cocok jalur bypass Claude CLI, langsung dilayani dan handler return.
  3. `InjectSystemStatus(&req)` — sisipkan status sistem ke request.
  4. Jika `req.Stream == true` → `router.DispatchChatCompletionStream` (tulis SSE langsung ke `w`); kalau error & status != 200, kirim JSON error.
  5. Jika non-stream → `router.DispatchChatCompletion`, ukur durasi, `captureMITM` (kalau capture aktif) + `captureLearningRecording`, lalu tulis JSON respons.
- Dispatch (`internal/router/dispatcher.go`): `DispatchChatCompletion` buka store, cek apakah model = otak `flowork-brain` / enrich brain (`maybeEnrichBrain`), `ListProviders`, pilih provider sesuai strategi, lalu `dispatchSingleModel` meneruskan ke upstream provider.
- Stream di GUI: baca `ReadableStream`, parse baris `data: ...` SSE, ambil `choices[0].delta.content`, append ke bubble assistant; `[DONE]` mengakhiri.

## File yang dilewati
- `/home/mrflow/Documents/FLowork_os/router/routes.go` — route `/v1/chat/completions`, `/v1/models`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_chat.go` — `chatCompletionsHandler`, `captureMITM`, `modelsHandler`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_chat_v1.go` — jalur Anthropic `messagesV1Handler` (dispatch sama).
- `/home/mrflow/Documents/FLowork_os/router/internal/router/dispatcher.go` — `DispatchChatCompletion`, `dispatchSingleModel`, pilih provider.
- `/home/mrflow/Documents/FLowork_os/router/internal/router/dispatcher_stream.go` — `DispatchChatCompletionStream`.
- `/home/mrflow/Documents/FLowork_os/router/internal/router/strategy.go` — pemilihan provider per strategi.
- `/home/mrflow/Documents/FLowork_os/router/internal/store/providers.go` — `ListProviders`.
- `/home/mrflow/Documents/FLowork_os/router/internal/safego/` — `safego.GoLabel` (capture async).
- `/home/mrflow/Documents/FLowork_os/router/web/static/index.html` — `data-tab="chat"`, section `#tab-chat`, fungsi `loadChatModels` + `chatSend` (default `flowork-brain`).

## Teknologi
- Go `net/http` stdlib; streaming via SSE (Server-Sent Events) ditulis langsung ke `http.ResponseWriter`.
- `internal/router` (dispatcher + strategy) untuk routing model→provider.
- `internal/providers` (sub-pkg embedding/image/stt/tts) untuk jenis media lain; chat teks lewat dispatcher inti.
- `internal/store` (SQLite) untuk daftar provider + settings.
- `internal/safego` untuk goroutine berlabel (MITM capture).
- Frontend vanilla JS + `fetch` + `ReadableStream` reader.

## Status freeze
- FROZEN (header `⚠️ FROZEN`): `routes.go`, `handlers_chat.go`, `handlers_chat_v1.go`, `internal/router/dispatcher.go` (dan file dispatcher/strategy seinti), `internal/store/providers.go`.
- NON-FROZEN: `web/static/index.html` (GUI tidak di-freeze).
