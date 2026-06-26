# OAuth Imports

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab `data-tab="oauth-imports"` mendeteksi otomatis file credential dari CLI yang sudah login (Claude Code, Codex/OpenAI, Cursor, GitLab Duo) lalu import token-nya ke store router. Tab juga menyediakan OAuth penuh per provider (PKCE authorize/callback, device-code flow, paste token/PAT/cookie) dan login Claude per-device (start authorize URL -> paste code -> exchange jadi token).

## Endpoint (router/routes.go)
Didaftarkan di `registerAuthRoutes`:

- `GET /api/oauth/imports` -> `oauthImportsHandler` (handlers_obs.go) — list hasil `creds.DetectAll()`.
- `/api/oauth` dan `/api/oauth/` -> `oauthRouterHandler` (handlers_oauth.go) — sub-router OAuth.
- `POST /api/claude-login/start` -> `claudeLoginStartHandler` (handlers_claude_login.go).
- `POST /api/claude-login/complete` -> `claudeLoginCompleteHandler` (handlers_claude_login.go).

Sub-route `oauthRouterHandler`:
- `` -> `oauthListHandler` (GET, token tersimpan + template provider).
- `imports` -> `oauthImportsHandler`.
- `<provider>` -> `oauthProviderHandler` (GET/POST paste-token/DELETE revoke).
- `<provider>/init` (atau `social-authorize`) -> `oauthInitHandler` (PKCE authorize URL).
- `<provider>/callback` (atau `social-exchange`) -> `oauthCallbackHandler` (tukar code->token).
- `<provider>/device-code` -> `oauthDeviceStartHandler` (handlers_oauth_device.go).
- `<provider>/poll` -> `oauthDevicePollHandler` (handlers_oauth_device.go).
- `<provider>/import-token|import|pat|cookie` -> `oauthImportActionHandler`.
- `<provider>/auto-import` -> `oauthAutoImportHandler`.

## Logic / Alur
- `oauthImportsHandler` (GET): `creds.DetectAll()` cek file `~/.claude/.credentials.json`, `~/.codex|.openai/auth.json`, Cursor (`state.vscdb`/`auth.json`), `~/.config/gitlab-duo/auth.json`; balas `{data, count}` dengan masked key + status expired.
- `oauthAutoImportHandler` (GET): cari status `Found` cocok provider -> `loadDetectedToken` parse token lokal -> `store.UpsertOAuthToken`; kalau tak ke-parse balas hint paste manual.
- `oauthImportActionHandler` (POST): ambil credential pertama non-kosong (accessToken/token/apiKey/pat/cookie) -> upsert ke store dengan tokenType sesuai kind.
- `oauthProviderHandler` POST: simpan paste token; untuk `claude`/`anthropic` juga `creds.SaveClaude(...)` tulis credential file.
- `oauthInitHandler` (POST): bikin PKCE (state + verifier, SHA256 challenge), simpan record `<provider>:pending`, balas `authUrl`.
- `oauthCallbackHandler` (GET): cek `<provider>:pending`, cocokkan state (`subtle.ConstantTimeCompare`); kalau clientID asli, POST ke TokenURL tukar code->token, simpan token, hapus pending.
- Device flow: `oauthDeviceStartHandler` (POST) panggil device_authorization endpoint -> simpan `<provider>:device-pending` -> balas userCode/verificationUri. `oauthDevicePollHandler` (POST) loop poll token endpoint -> status `pending`/`slow_down`/`complete`/`error`.
- Claude per-device: `claudeLoginStartHandler` (POST) bikin PKCE pair + state (`creds.PKCEPair`/`RandomState`), simpan `claude:login-pending`, balas `creds.ClaudeAuthorizeURL`. `claudeLoginCompleteHandler` (POST) parse `code#state`, cek state, `creds.ExchangeClaudeCode` -> `creds.SaveClaude` -> upsert token `claude`, hapus pending.

## File yang dilewati
- `/home/mrflow/Documents/FLowork_os/router/routes.go` (registrasi route)
- `/home/mrflow/Documents/FLowork_os/router/handlers_obs.go` (`oauthImportsHandler`)
- `/home/mrflow/Documents/FLowork_os/router/handlers_oauth.go` (router, init/callback, list, provider, import action, auto-import, template provider)
- `/home/mrflow/Documents/FLowork_os/router/handlers_oauth_device.go` (`oauthDeviceStartHandler`, `oauthDevicePollHandler`)
- `/home/mrflow/Documents/FLowork_os/router/handlers_claude_login.go` (`claudeLoginStartHandler`, `claudeLoginCompleteHandler`)
- `/home/mrflow/Documents/FLowork_os/router/internal/creds/imports.go` (`DetectAll`, `LoadCodexToken`, `LoadCursorToken`)
- `/home/mrflow/Documents/FLowork_os/router/internal/creds/` (`Load`/`LoadValid`, `SaveClaude`, `PKCEPair`, `RandomState`, `ClaudeAuthorizeURL`, `ExchangeClaudeCode` — login.go/save.go/credentials.go)
- `/home/mrflow/Documents/FLowork_os/router/internal/store/` (`OAuthTokenRecord`, `UpsertOAuthToken`, `GetOAuthToken`, `ListOAuthTokens`, `DeleteOAuthToken`)
- `/home/mrflow/Documents/FLowork_os/router/web/static/index.html` (`data-tab="oauth-imports"`)

## Teknologi
- Go `net/http` (server + outbound client untuk token/device endpoint).
- OAuth 2.0 PKCE: `crypto/rand`, `crypto/sha256`, `encoding/base64`, `crypto/subtle` (state compare konstan-waktu).
- Device-code flow (RFC 8628) dengan polling.
- `internal/creds` baca/tulis credential file CLI (`~/.claude/.credentials.json`, dll) + exchange token Claude.
- SQLite via `internal/store` untuk simpan token & record pending; token di-mask saat ditampilkan.

## Status freeze
- `handlers_obs.go` — FROZEN.
- `handlers_oauth.go` — FROZEN.
- `handlers_oauth_device.go` — FROZEN.
- `handlers_claude_login.go` — FROZEN.
- `internal/creds/imports.go` — FROZEN.
- `routes.go` — FROZEN.
- `web/static/index.html` (GUI) — TIDAK frozen.
