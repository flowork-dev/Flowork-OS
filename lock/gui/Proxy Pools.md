# Proxy Pools

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Tab untuk mengelola pool proxy outbound (HTTP/SOCKS5) beserta strategi rotasinya (round-robin/random/sticky/single) — dipakai untuk privasi, bypass geo, dan multi-akun. CRUD pool disimpan di tabel `proxyPools`. Tab juga menghasilkan skrip + langkah deploy proxy edge ke Cloudflare Workers, Deno Deploy, dan Vercel Edge.

## Endpoint (router/routes.go)
- `GET|POST /api/proxy-pools` → `proxyPoolsListAddHandler` (handlers_resources.go)
- `PUT|DELETE /api/proxy-pools/{id}`, dan `/{id}/test` → `proxyPoolCRUDHandler` (handlers_resources.go) → `proxyPoolTestHandler` (handlers_gaps.go)
- `POST /api/proxy-pools/cloudflare-deploy` → `cloudflareDeployHandler` (handlers_proxy_deploy.go)
- `POST /api/proxy-pools/deno-deploy` → `denoDeployHandler` (handlers_proxy_deploy.go)
- `POST /api/proxy-pools/vercel-deploy` → `vercelDeployHandler` (handlers_proxy_deploy.go)

## Logic / Alur
proxyPoolsListAddHandler:
- GET → `store.ListProxyPools`, balas `{data, count}`.
- POST → decode `ProxyPool`, `store.UpsertProxyPool` (generate UUID + createdAt bila baru, default rotation `round-robin`), balas 201.

proxyPoolCRUDHandler:
- Ambil sisa path setelah `/api/proxy-pools/`. Bila kosong → 400.
- Bila ada `/`: pisahkan `id`/`action`; `action == "test"` → `proxyPoolTestHandler`; selain itu → 404.
- PUT → decode `ProxyPool`, set `ID`, `UpsertProxyPool`, balas pool.
- DELETE → `store.DeleteProxyPool`, balas 204.

proxyPoolTestHandler (handlers_gaps.go): cari pool by id; bila ada balas `{reachable:true, note:"config present; live egress test Phase 3"}`; bila tidak → 404. (Belum ada uji egress live.)

Deploy handler (cloudflare/deno/vercel): POST, decode `{name, targetUrl, apiKeyEnv, project}` (default targetUrl `https://your-tunnel.trycloudflare.com`). Bangun skrip proxy edge (Worker/Deno.serve/Vercel edge handler) yang meneruskan request ke target. Cek ketersediaan CLI (`wrangler`/`deployctl`/`vercel`) via `exec.LookPath`. Buat entri `ProxyPool` (rotation `single`) lewat `UpsertProxyPool`. Balas skrip + `deployCommand` + `setupSteps` + `cliAvailable` + `poolId`.

## File yang dilewati
- `router/handlers_resources.go` — `proxyPoolsListAddHandler`, `proxyPoolCRUDHandler` (FROZEN)
- `router/handlers_proxy_deploy.go` — `cloudflareDeployHandler`, `denoDeployHandler`, `vercelDeployHandler`, `jsString` (FROZEN)
- `router/handlers_gaps.go` — `proxyPoolTestHandler` (FROZEN)
- `router/internal/store/proxypools.go` — `ProxyPool`, `ListProxyPools`, `UpsertProxyPool`, `DeleteProxyPool`, konstanta rotation
- `router/web/static/index.html` — sidebar `data-tab="proxy-pools"` (label "Proxy Pools")

## Teknologi
Go `net/http`, SQLite (tabel `proxyPools`, proxies disimpan JSON, UUID via `google/uuid`), `exec.LookPath` untuk deteksi CLI deploy, generator skrip edge (Cloudflare Workers / Deno Deploy / Vercel Edge). Frontend HTML/JS statis.

## Status freeze
FROZEN — handlers_resources.go, handlers_proxy_deploy.go, handlers_gaps.go, dan internal/store/proxypools.go berheader `⚠️ FROZEN`. Penambahan fitur via SEAM non-frozen + SWITCH (internal/fwswitch/registry.go). GUI `web/static/index.html` TIDAK frozen.
