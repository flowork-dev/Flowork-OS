# Console Log

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Log request live: nampilin dispatch terakhir dengan provider, model, status, token, latency, dan cost. Dipakai buat ngintip trafik yang lewat router secara real-time. Untuk inspeksi body penuh dan replay, dipasangkan dengan endpoint recordings (capture full body + ambil ulang per id).

## Endpoint (router/routes.go)
- `/api/console-log` → `consoleLogHandler` (handlers_obs.go) — GET.
- `/api/recordings` → POST `recordingsPostHandler` / GET `recordingsListHandler` (handlers_recordings.go) — capture full body + list.
- `/api/recordings/get` → `recordingsGetHandler` (handlers_recordings.go) — GET satu recording buat replay.

## Logic / Alur
- GET `/api/console-log`: hanya GET (selain itu 405). Query `limit` (default 100), `provider`, `status`. Panggil `store.ListRecent(d, limit, provider, status)` → balikin `{data, count}`.
- POST `/api/recordings`: `ensureBrainReady` dulu; body dibatasi 128KB (`MaxBytesReader`), decode strict (`DisallowUnknownFields`) field `{model, request_body, response_text, input_tokens, output_tokens, cost_usd, build_pass, tool_calls, agent}`. Simpan via `recorder.Save` → balikin `{id, algo_version}`.
- GET `/api/recordings`: `recorder.List` dengan opts `model`, `agent`, `include_body=1`, `limit` (max 500) → `{items, count}`.
- GET `/api/recordings/get?id=`: validasi id positif, `recorder.Get` → kalau model kosong 404, selain itu balikin recording penuh (buat replay body).

## File yang dilewati
- `router/handlers_obs.go` — `consoleLogHandler`.
- `router/handlers_recordings.go` — `recordingsPostHandler`, `recordingsListHandler`, `recordingsGetHandler`.
- `router/internal/store` — `ListRecent` (sumber log dispatch).
- `router/internal/recorder` — `Save`, `List`, `Get`, `RecordOpts`, `ListOpts`, `AlgoVersion`.
- `router/web/static/index.html` — `data-tab="console-log"`.

## Teknologi
Go net/http, SQLite (recent dispatch + recordings), JSON. Body capture dibatasi 128KB. Frontend HTML/JS statis.

## Status freeze
FROZEN — `handlers_obs.go` dan `handlers_recordings.go` punya header FROZEN. GUI `web/static/index.html` TIDAK frozen.
