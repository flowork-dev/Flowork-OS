# Plug-and-Play Audit — fitur eksternal yang BELUM/SUDAH copot-pasang

> Owner: Aola Sahidin (Mr.Dev) · floworkos.com. Prinsip: fitur EKSTERNAL wajib DATA/SEAM (tambah &
> share-mesh tanpa buka frozen). Hardcode di frozen = cacat. Arsitektur: lock/ARSITEKTUR.md.

## Status (audit 2026-06-27)
| Fitur | Status | Cara nambah sekarang | Aksi |
|---|---|---|---|
| Models / alias / custom | ✅ | DB (`/api/models/custom`,`/alias`, provider `CfgModels`) | — |
| Provider (koneksi) | ✅ | DB ProviderConnection (GUI) | — |
| Provider protokol/dialect | ✅ | sibling translator/{request,response} + `translator.Register` | — |
| Provider media (embed/img/tts/stt) | ✅ | sibling providers/<kat> + `Register` | — |
| Executors (cursor/codex/kiro/…) | ✅ | sibling internal/executors + `Register` | — |
| Combos | ✅ | DB (`/api/combos`, UpsertCombo) | — |
| MCP servers (custom) | ✅ | DB `mcpServer` + `mcpcatalog.Register` | — |
| Skills | ✅ | DB + registry pull/publish | — |
| Sensors/webhook | ✅ | ENV token + webhook generic | — |
| Auth (oidc/local/apikey) | 🔒 CORE | config `authMode` + frozen handler | biar frozen (security inti, bukan plugin) |
| MCP default catalog | ⚠️ | 3 default hardcode (custom via API OK) | minor: pindah default ke DB seed |
| Presets | ⚠️ | `store.Presets` statis di kode (frozen) | minor: combos (DB) sudah jadi versi dinamis |
| **CLI Tools** | ⚠️ | `clitools/registry.go` `All()` hardcode (frozen) | **seam: DB `cliTool` merge + sibling Register** |
| **Cloaking** | ⚠️ | decoy list + suffix/version hardcode (`internal/router/cloaking.go` frozen) | **seam: profil cloaking ke DB/switch** |
| **Proxy Pools (deploy)** | ❌ | 3 handler hardcode per-provider (`handlers_proxy_deploy.go` frozen) | **seam: RegisterProxyDeployTemplate + sibling per-target** |
| **Tunnel** | ❌ | tailscale+cloudflared hardcode (`handlers_tunnel.go` frozen) | **seam: RegisterTunnelProvider + sibling per-provider** |

## Yang perlu di-seam (roadmap, urut prioritas)
1. **Tunnel** ❌ — `RegisterTunnelProvider{Name, Detect, Enable, Disable, Status}` di file frozen baru
   (registry infra) → provider tailscale/cloudflared jadi sibling file; nambah ngrok/bore = sibling baru.
2. **Proxy deploy** ❌ — `RegisterProxyDeployTemplate{Name, Script, Steps}` → cloudflare/deno/vercel jadi
   sibling/DATA; nambah AWS/Netlify = sibling/baris DB.
3. **CLI Tools** ⚠️ — tabel DB `cliTool` (merge hardcode default + user) → `clitools.All()` append DB.
4. **Cloaking** ⚠️ — tabel/switch `cloaking_profile` (decoyTools[], suffix, version) → default frozen +
   override DATA. CATATAN: cloaking = teknik sensitif; profil di DATA biar fleksibel, mesin tetap frozen.
5. **MCP default / Presets** ⚠️ — pindah seed default ke DB (atau biarkan; custom sudah via DATA).

## Pola seam (sama untuk semua ❌/⚠️)
File frozen BARU = registry infra (`var registry []X` + `RegisterX()` + runner default no-op/builtin).
Provider/target/profil = sibling non-frozen `<fitur>_<nama>.go` + `init(){ RegisterX(...) }` ATAU baris DB.
Runner-nya FROZEN (core), tapi extension via sibling/DATA → nol buka frozen. Detail pola: lock/frozen-core.md.

> Catatan: ⚠️/❌ ini fitur EKSTERNAL ke-hardcode di frozen — melanggar prinsip (lock/ARSITEKTUR.md).
> Belum diimplementasi seam-nya; ini daftar kerja, bukan klaim selesai.
