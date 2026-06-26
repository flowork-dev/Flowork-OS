# MCP Servers

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab untuk mendaftarkan Model Context Protocol (MCP) server dengan transport `stdio`, `http`, atau `sse`. Bisa melihat daftar tools live dari tiap server (`tools/list` via JSON-RPC), bertindak sebagai gateway pesan ke server, dan menampilkan katalog plugin siap pakai (Exa, Tavily, Browser MCP).

## Endpoint (router/routes.go)
- `mux.HandleFunc("/api/mcp", mcpRouterHandler)` (baris 245)
- `mux.HandleFunc("/api/mcp/catalog", mcpCatalogHandler)` (baris 246)
- `mux.HandleFunc("/api/mcp/", mcpRouterHandler)` (baris 247)

Handler: `mcpRouterHandler` di `handlers_mcp.go`; `mcpCatalogHandler` di `handlers_mcp_catalog.go`.

## Logic / Alur
`mcpRouterHandler` mem-parse sisa path setelah `/api/mcp`:
- `""` → `mcpListUpsertHandler`: `GET` list server (`store.ListMCPServers`), `POST` upsert (`store.UpsertMCPServer`, wajib `id`+`name`).
- `<id>/tools` → `mcpToolsHandler`: ambil server, panggil `mcpListToolsLive` (timeout 20s). Untuk `stdio` → `mcpStdioListTools` (spawn command, kirim `initialize` + `notifications/initialized` + `tools/list`, baca stdout JSON-RPC id=2). Untuk `http`/`sse` → `mcpHTTPListTools` (POST `tools/list`, parse JSON atau SSE `data:`).
- `<id>/message` → `mcpGatewayMessageHandler`: `POST`. Untuk `http`/`sse` → proxy POST ke `srv.URL` (dilindungi SSRF guard `blockMetadataURL`). Untuk `stdio` → `mcpStdioRoundTrip` (spawn, kirim pesan, cocokkan response by id, timeout 15s).
- `<id>/sse` → `mcpGatewaySSEHandler`: stream SSE dari `srv.URL` (stdio ditolak `501`, lewati SSRF guard).
- sisanya → `mcpCRUDHandler(id)`: `GET` 1 server, `PUT` update, `DELETE` hapus.

Keamanan stdio: `mcpsecurity.IsAllowed(srv.Command)` membatasi command ke allowlist (npx/node/uvx/python/bunx/bun/deno/pnpm/yarn) + `exec.LookPath`. Konstanta `mcpInitParams` (protocolVersion `2024-11-05`), helper `jsonRPCMsg`.

`mcpCatalogHandler`: `GET` saja, kembalikan `mcpcatalog.Catalog()` — gabungan `DefaultPlugins` + plugin custom (dedup by name). Default: Exa (http), Tavily (http, oauth), Browser MCP (stdio npx).

## File yang dilewati
- `router/handlers_mcp.go` — handler utama, gateway message/SSE, list tools live (stdio+http), round-trip stdio.
- `router/handlers_mcp_catalog.go` — handler katalog.
- `router/internal/mcpcatalog/catalog.go` — `Plugin`, `DefaultPlugins`, `Catalog`, `Register`, `Set`, `Lookup`.
- `router/internal/mcpsecurity/allowlist.go` — `IsAllowed`, allowlist command.
- `router/internal/store` — `MCPServer`, `ListMCPServers`, `GetMCPServer`, `UpsertMCPServer`, `DeleteMCPServer`.
- `router/handlers_ssrf_guard.go` — `blockMetadataURL` (SSRF guard), `mediaHTTPClient`.
- `router/web/static/index.html` — frontend, `data-tab="mcp"` (baris 160).

## Teknologi
Go `net/http`, `os/exec` (spawn stdio MCP), JSON-RPC 2.0, SSE, SSRF guard, SQLite via `internal/store`.

## Status freeze
FROZEN — `handlers_mcp.go`, `handlers_mcp_catalog.go`, `internal/mcpcatalog`, `internal/mcpsecurity`, `internal/store` semua ber-header `⚠️ FROZEN`. `routes.go` dan `web/static/index.html` (GUI) NON-frozen.
