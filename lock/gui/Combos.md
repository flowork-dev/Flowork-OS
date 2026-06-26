# Combos

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab grouping model. User membuat "combo" — satu alias yang otomatis me-route ke beberapa model sekaligus dengan strategi pemilihan (priority, round_robin, random, cost_optimal). Saat chat memakai nama combo sebagai model, router memilih salah satu model anggota sesuai strategi. Tab ini juga menampilkan Presets Library (daftar preset provider siap pakai) sebagai referensi.

## Endpoint (router/routes.go)
- `GET|POST /api/combos` → `combosListAddHandler` (handlers_resources.go) — list / tambah combo.
- `PUT|DELETE /api/combos/{id}` → `comboCRUDHandler` (handlers_resources.go) — ubah / hapus combo per id.
- `GET /api/presets` → `presetsHandler` (handlers_resources.go) — daftar preset statis (`store.Presets`).

## Logic / Alur
- `combosListAddHandler`:
  - GET → `store.ListCombos` (urut nama ASC), balikkan `{data:[...], count:n}`.
  - POST → decode `store.Combo` (`{name, models[], strategy}`), `store.UpsertCombo`, balikkan 201.
- `comboCRUDHandler`: ambil `id` dari path `/api/combos/`.
  - PUT → set `c.ID=id`, `store.UpsertCombo`.
  - DELETE → `store.DeleteCombo` (204).
  - GET/POST tidak didukung di route `{id}` (405).
- `store.UpsertCombo`: kalau `ID` kosong → buat UUID + `CreatedAt` sekarang; kalau `Strategy` kosong → default `priority`; simpan `models` sebagai JSON ke tabel `combos` (upsert `ON CONFLICT(id)`).
- Strategi (konstanta di `internal/store/combos.go`): `priority`, `round_robin`, `random`, `cost_optimal`. Urutan model penting untuk strategi `priority`. Pemilihan aktual dilakukan dispatcher/strategy saat chat (`internal/router`).
- `presetsHandler`: balikkan `{data: store.Presets}` — slice statis (~30+ preset provider: Claude sub, OpenAI, DeepSeek, Gemini, Groq, lokal llama/ollama/lmstudio, dst.) untuk membantu user setup provider; bukan dari DB.
- Frontend (`loadCombos`): GET `/api/combos`, render kartu + badge strategi; form simpan POST/PUT ke `/api/combos[/id]`, hapus DELETE.

## File yang dilewati
- `/home/mrflow/Documents/FLowork_os/router/routes.go` — route `/api/combos`, `/api/combos/`, `/api/presets`.
- `/home/mrflow/Documents/FLowork_os/router/handlers_resources.go` — `combosListAddHandler`, `comboCRUDHandler`, `presetsHandler`.
- `/home/mrflow/Documents/FLowork_os/router/internal/store/combos.go` — struct `Combo`, konstanta strategi, `ListCombos`, `GetComboByName`, `UpsertCombo`, `DeleteCombo`.
- `/home/mrflow/Documents/FLowork_os/router/internal/store/presets.go` — struct `Preset`, slice `Presets`, `GetPreset`.
- `/home/mrflow/Documents/FLowork_os/router/internal/router/strategy.go` — pemilihan model/provider per strategi saat dispatch.
- `/home/mrflow/Documents/FLowork_os/router/web/static/index.html` — `data-tab="combos"`, section `#tab-combos`, fungsi `loadCombos` + form `combo_strategy` (options: priority/round_robin/random/cost_optimal).

## Teknologi
- Go `net/http` stdlib (handler manual + parse path).
- SQLite store `internal/store` — tabel `combos` (models disimpan sebagai JSON string).
- `github.com/google/uuid` untuk id combo baru.
- Presets = data statis Go (`store.Presets`), bukan DB.
- `internal/router` (strategy) untuk eksekusi strategi saat chat memakai alias combo.
- Frontend vanilla JS + `fetch`.

## Status freeze
- FROZEN (header `⚠️ FROZEN`): `routes.go`, `handlers_resources.go`, `internal/store/combos.go`, `internal/store/presets.go`.
- NON-FROZEN: `web/static/index.html` (GUI tidak di-freeze).
