# CLI Tools

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab `data-tab="cli-tools"` mendeteksi AI CLI yang terinstall di mesin (Claude Code, OpenAI Codex, Cline, GitHub Copilot, Cowork, Droid, Kilo, OpenCode, dll) dengan cek binary di PATH dan file settings-nya. User bisa Configure (arahkan CLI ke base URL router :2402 + API key/model) lalu Reset (hapus key env yang dipasang router). Tab juga sediakan registry MCP untuk mode Cowork.

## Endpoint (router/routes.go)
Didaftarkan di `registerInfraRoutes`:

- `/api/cli-tools` dan `/api/cli-tools/` -> `cliToolsRouterHandler` (handlers_cli_tools_ext.go).

Sub-route di dalam router:
- `` (kosong) / `all-statuses` -> `cliToolsListHandler` (GET) — deteksi semua tool.
- `<toolID>-settings` / `antigravity-mitm` -> `cliToolSettingsHandler` (GET/POST/DELETE).
- `antigravity-mitm/alias` -> `antigravityAliasHandler` (kv di store).
- `cowork-mcp-registry` -> `coworkMCPRegistryHandler` (GET, daftar MCP statis).
- `cowork-mcp-tools` -> `coworkMCPToolsHandler` (GET, live tools dari server MCP enabled).

## Logic / Alur
- `cliToolsListHandler` (GET): `clitools.DetectAll()` -> tiap tool di-upsert ke `store.UpsertCLIToolState` (status: error / connected (HasFlowRouter) / installed / missing) -> JSON `{data, count}`.
- `cliToolSettingsHandler`:
  - GET: `clitools.Detect(toolID)` -> status satu tool (404 kalau unknown).
  - POST (Configure): decode body JSON. Kalau ada `env` pakai itu; kalau ada `baseUrl` bangun env via `clitools.BuildConnectEnv(toolID, baseUrl, apiKey, model)`; lalu `clitools.WriteEnv(toolID, env)` menulis file settings -> JSON `{success, toolId, settings}`.
  - DELETE (Reset): `clitools.ResetEnv(toolID)` menghapus key env yang ditanam router dari file settings.
- Deteksi (`clitools.Detect`): cek binary via `exec.LookPath` (nama + alias), cek `os.Stat` SettingsPath, baca/parse settings (JSON/TOML/Env/Proxy), tentukan `HasFlowRouter` kalau BaseURL mengandung `127.0.0.1:2402`/`localhost:2402`/`flow_router`/`flow-router`.
- Tulis settings: `WriteEnv` -> format-specific writer (`writeJSONSettings`, `writeTOMLSettings`, `writeEnvSettings`, atau custom writer); untuk `claude` ditulis di bawah `env` + paksa `ANTHROPIC_BASE_URL` berakhiran `/v1` dan set `hasCompletedOnboarding`. File ditulis mode `0600`, dir `0700`.

## File yang dilewati
- `/home/mrflow/Documents/FLowork_os/router/routes.go` (registrasi route)
- `/home/mrflow/Documents/FLowork_os/router/handlers_cli_tools_ext.go` (router + semua sub-handler)
- `/home/mrflow/Documents/FLowork_os/router/internal/clitools/detect.go` (`Detect`, `DetectAll`, `WriteEnv`, `ResetEnv`, writer/parser format)
- `/home/mrflow/Documents/FLowork_os/router/internal/clitools/registry.go` (daftar tool: claude/codex/cline/copilot/cowork/deepseek-tui/droid/hermes/jcode/kilo/openclaw/opencode/antigravity-mitm; `BuildConnectEnv`)
- `/home/mrflow/Documents/FLowork_os/router/internal/clitools/custom.go` (custom writer per tool)
- `/home/mrflow/Documents/FLowork_os/router/internal/store/` (`UpsertCLIToolState`, kv untuk antigravity alias, `ListMCPServers`)
- `/home/mrflow/Documents/FLowork_os/router/web/static/index.html` (`data-tab="cli-tools"`)

## Teknologi
- Go `net/http`.
- `os/exec` (`LookPath`) untuk deteksi binary di PATH.
- File I/O `os.ReadFile`/`WriteFile`/`MkdirAll`/`Stat` ke config CLI di home dir (`~/.claude`, `~/.codex`, `~/.config/...`).
- Parser/writer multi-format: JSON (`encoding/json`), TOML sederhana, dotenv, dan format Proxy.
- SQLite via `internal/store` untuk simpan state CLI tool dan kv alias.

## Status freeze
- `handlers_cli_tools_ext.go` — FROZEN.
- `internal/clitools/detect.go` — FROZEN.
- `routes.go` — FROZEN.
- `web/static/index.html` (GUI) — TIDAK frozen.
