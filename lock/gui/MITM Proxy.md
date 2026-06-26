# MITM Proxy

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Interceptor HTTPS lokal buat IDE AI-coding (Antigravity, Copilot, Cursor, Kiro). Generate root CA per-mesin, auto-sign leaf cert per SNI, lalu arahkan trafik IDE (lewat hijack DNS ke 127.0.0.1) ke listener MITM yang rewrite request dan teruskan ke dispatcher router. Bisa toggle capture full body buat inspeksi, plus install/uninstall root CA ke trust store OS.

## Endpoint (router/routes.go)
- `/api/mitm/status` → `mitmStatusHandler` (handlers_mitm_proxy.go) — GET.
- `/api/mitm/start` → `mitmStartHandler` (handlers_mitm_control.go) — POST.
- `/api/mitm/stop` → `mitmStopHandler` (handlers_mitm_control.go) — POST.
- `/api/mitm/capture-toggle` → `mitmCaptureToggleHandler` (handlers_mitm_ext.go) — POST.
- `/api/mitm/root-ca` → `mitmRootCADownloadHandler` (handlers_mitm_proxy.go) — GET.
- `/api/mitm/install-ca` → `mitmInstallCAHandler` (handlers_mitm_proxy.go) — POST.
- `/api/mitm/uninstall-ca` → `mitmUninstallCAHandler` (handlers_mitm_proxy.go) — POST.
- `/api/mitm/dns/add` → `mitmDNSAddHandler` (handlers_mitm_proxy.go) — POST.
- `/api/mitm/dns/remove` → `mitmDNSRemoveHandler` (handlers_mitm_proxy.go) — POST.
- `/api/mitm/full/` → `mitmFullDetailHandler` (handlers_mitm_ext.go) — GET detail per id.
- `/api/mitm/recent-full` → `mitmRecentFullHandler` (handlers_mitm_ext.go) — GET list ringkas.

## Logic / Alur
- GET status: balikin `dataDir`, `mitmDir`, `certPath` (rootCA.pem), `isRunning` (`mitm.IsRunning`), `pid` (`mitm.ReadPidFile`), `isAdmin`, `targetHosts` (`mitm.TargetHosts`), `toolMap` (host→tool dari `mitm.GetToolForHost`), `certExists`/`certBytes`, `dnsHijacked` (`mitm.CheckDNSStatus`), `hostsPath`.
- POST start: body `{addr, hijackDNS}`. Buat `mitm.NewCertManager`, kalau `hijackDNS` true pakai `mitm.TargetHosts`, `mitm.NewManager(addr, cm, hosts).Start`. Default addr `127.0.0.1:443` (override env `FLOW_ROUTER_MITM_ADDR`). Manager disimpan di var global ber-mutex; kalau sudah jalan balikin `already:true`.
- POST stop: `mitmMgr.Stop()` lalu reset var global. `stopMITMOnShutdown` dipanggil saat shutdown.
- POST capture-toggle: body `{enabled}`. Set flag in-memory (`mitmCaptureEnabled`) dan persist ke tabel `kv` key `mitm:capture`. `loadMITMCaptureState` baca ulang saat boot. `recordMITMRequest` simpan request/response (truncate 256KB) ke `requestDetails` kalau capture aktif.
- GET root-ca: baca `rootCA.pem`, kalau hilang coba bikin via `mitm.NewCertManager`, balikin sebagai `application/x-pem-file` (attachment).
- POST install-ca / uninstall-ca: `mitm.InstallRootCA` / `mitm.UninstallRootCA` ke trust store OS; gagal balikin `manualCommand` hint.
- POST dns/add: body `{hosts}` (default `mitm.TargetHosts`) → `mitm.AddDNSEntries` (tulis hosts file ke 127.0.0.1). dns/remove → `mitm.RemoveAllDNSEntries`.
- GET full/{id}: set query `id` lalu delegasi ke `usageRequestDetailsHandler`. recent-full: query `requestDetails` (id, ts, provider, model, status, error, durasi, panjang body) order desc limit, plus `captureEnabled`.

## File yang dilewati
- `router/handlers_mitm_proxy.go` — status, root-ca, install/uninstall-ca, dns add/remove.
- `router/handlers_mitm_control.go` — start, stop, manager global.
- `router/handlers_mitm_ext.go` — capture-toggle, full detail, recent-full, `recordMITMRequest`.
- `router/internal/mitm/` — `config.go` (TargetHosts, URLPatterns, ModelPatterns, GetToolForHost), `cert.go`/`cert_install.go` (CertManager, install/uninstall), `manager.go`, `listener.go`, `dns_config.go`, `paths.go`, `dbreader.go`.
- `router/internal/store` — tabel `kv` dan `requestDetails`.
- `router/web/static/index.html` — `data-tab="mitm-proxy"`.

## Teknologi
Go net/http + crypto/tls (root CA + leaf cert per-SNI), SQLite (`kv`, `requestDetails`), hosts-file DNS hijack ke 127.0.0.1, listener default `127.0.0.1:443`. Frontend HTML/JS statis.

## Status freeze
FROZEN — `handlers_mitm_proxy.go`, `handlers_mitm_control.go`, `handlers_mitm_ext.go`, dan file `internal/mitm/*` punya header FROZEN. GUI `web/static/index.html` TIDAK frozen.
