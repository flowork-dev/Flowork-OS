# Tunnel

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Expose router yang jalan di localhost ke internet lewat Cloudflare Tunnel (cloudflared) atau Tailscale. Cloudflare ngasih URL publik `*.trycloudflare.com` sementara; Tailscale nyambungin lewat IP private mesh (`http://<TailscaleIP>:2402`). Tab ini cuma kontrol on/off + cek status; binary cloudflared/tailscale dijalanin lewat shell, bukan library.

## Endpoint (router/routes.go)
Didaftarkan di `registerInfraRoutes`:
- `GET  /api/tunnel/status` → `tunnelStatusHandler`
- `POST /api/tunnel/enable` → `tunnelEnableHandler`
- `POST /api/tunnel/disable` → `tunnelDisableHandler`
- `GET  /api/tunnel/tailscale-check` → `tailscaleCheckHandler`
- `POST /api/tunnel/tailscale-install` → `tailscaleInstallHandler`
- `POST /api/tunnel/tailscale-enable` → `tailscaleEnableHandler`
- `POST /api/tunnel/tailscale-disable` → `tailscaleDisableHandler`

## Logic / Alur
- **status (GET)**: load `TunnelState` dari store, cek apakah proses cloudflared masih jalan (`isCloudflaredRunning`), lalu `exec.LookPath("tailscale")` + `tailscale status --json` buat deteksi `BackendState=Running` dan ekstrak Tailscale IP. State disimpan ulang lewat `SaveTunnelState`.
- **enable (POST)**: cek `cloudflared` ada di PATH (kalau nggak → 501 + hint install). Guard keamanan: tolak (403) kalau login belum dipaksa (`RequireLogin` false atau `AuthMode=none`) supaya admin API nggak kebuka tanpa auth ke internet. Spawn `cloudflared tunnel --no-autoupdate --url http://127.0.0.1:<port>` (default port 2402), scan stdout/stderr cari URL `*.trycloudflare.com` lewat regex, tunggu max 15 detik. Goroutine pakai `safego.GoLabel`.
- **disable (POST)**: `Process.Kill()` proses cloudflared, reset state.
- **tailscale-check (GET)**: cek terinstall + status JSON.
- **tailscale-install (POST)**: cuma balikin perintah install per-OS (`tailscale.com/install.sh` dst); router TIDAK manggil sudo sendiri.
- **tailscale-enable (POST)**: jalanin `tailscale up --accept-routes --accept-dns=true`, ekstrak auth URL `login.tailscale.com` kalau ada.
- **tailscale-disable (POST)**: jalanin `tailscale down`.
- Helper: `runShort` jalanin command dengan timeout 30 detik (`CombinedOutput`).

## File yang dilewati
- Handler: `router/handlers_tunnel.go`
- State persist: `router/internal/store/kvmisc.go` (`TunnelState`, `LoadTunnelState`, `SaveTunnelState`)
- Settings guard: `router/internal/store` (`LoadSettings` — `RequireLogin`, `AuthMode`)
- Goroutine helper: `router/internal/safego`
- Binary eksternal (shell-out): `cloudflared`, `tailscale`
- Watchdog terkait: `router/tunnel_watchdog.go`
- Frontend: `router/web/static/index.html` (`data-tab="tunnel"`)

## Teknologi
Go `net/http` + `os/exec` (shell-out ke cloudflared & tailscale), regexp parsing output, goroutine via safego, SQLite (store) untuk persist TunnelState.

## Status freeze
FROZEN — `handlers_tunnel.go` punya header `⚠️ FROZEN — jangan edit file ini`. Begitu juga `internal/store/kvmisc.go` dan `internal/kiromodels`. Penambahan fitur lewat SEAM non-frozen + SWITCH (`internal/fwswitch/registry.go`). GUI `web/static/index.html` TIDAK frozen.
