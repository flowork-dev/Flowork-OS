# Dok GUI Tab — Flowork Router (:2402)

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com

Satu file `.md` per tab GUI: **fungsi · logic · file yang dilewati · teknologi · status freeze**.
GUI = 1 file `router/web/static/index.html` (go:embed, **TIDAK frozen** biar bisa evolusi).
Logic = handler Go di `router/` (mayoritas **FROZEN + chattr +i**; manifest privat
`flowork-secrets/KERNEL_FREEZE.md`, enforce `TestKernelFreeze`). Standar freeze: `lock/frozen-core.md`.

## Peta handler frozen → dok tab
Header tiap file frozen idealnya nunjuk ke dok tab-nya. Sebagian handler **dipakai banyak tab**
(shared) — di situ pakai peta ini sebagai rujukan, bukan 1 baris header.

| Handler (router/) | Tab dok (lock/gui/) |
|---|---|
| handlers_resources.go | Providers · Combos · Embedding · Text to Image · Proxy Pools · API Keys |
| handlers_obs.go | Usage Analytics · Quota Tracker · OAuth Imports · Console Log · Settings |
| handlers_chat.go | Chat · Models |
| handlers_chat_v1.go | Chat · Embedding · Text to Image · Text To Speech · Speech To Text · Web Fetch & Search |
| handlers_providers_ext.go, handlers_provider_nodes.go | Providers |
| handlers_usage_breakdown.go | Usage Analytics |
| handlers_quotalive.go | Quota Tracker |
| handlers_cli_tools_ext.go | CLI Tools |
| handlers_oauth.go, handlers_oauth_device.go, handlers_claude_login.go | OAuth Imports |
| handlers_tunnel.go | Tunnel |
| handlers_models_meta.go, handlers_kiromodels.go | Models |
| handlers_pricing.go, handlers_llm_policy.go | Pricing |
| handlers_tags.go | Tags |
| handlers_translator.go | Translator |
| handlers_mcp.go, handlers_mcp_catalog.go | MCP Servers |
| handlers_media_ext.go, handlers_media_tts_voices.go | Text To Speech |
| handlers_stt.go | Speech To Text |
| handlers_fetch.go | Web Fetch & Search |
| handlers_proxy_deploy.go | Proxy Pools |
| handlers_apikey_auth.go | API Keys |
| handlers_mitm_proxy.go, handlers_mitm_control.go, handlers_mitm_ext.go | MITM Proxy |
| handlers_settings_sub.go, handlers_backup.go | Settings |
| handlers_recordings.go | Console Log |
| handlers_gaps.go | Endpoint |
| routes.go | (registrasi semua route) |

## Catatan QC (2026-06-26)
24 tab lolos QC GUI beneran (render + uji aksi round-trip, headless Chrome) — 0 bug, 0 data palsu.
Penanda jujur yang ditemukan (bukan bug): `proxyPoolTestHandler` masih stub ("live egress test
Phase 3"). Rekam QC penuh: `lock/qc-gui-24tab.md`.
