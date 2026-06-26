# MR-FLOW — Agent Owner (Jantung Flowork): Telegram I/O, Media, Format, Routing & Switch

> Dokumen referensi (white-label). mr-flow = agent owner, orchestrator utama, yang ngobrol sama
> owner di Telegram. Dok ini: arsitektur, I/O Telegram (format + baca dokumen/foto/voice), routing,
> switch, build, cabang, freeze. Owner: Aola Sahidin (Mr.Dev).
> Repo: https://github.com/flowork-os/Flowork-OS. Update: 2026-06-25.
> ⚠️ KE-TRACK repo → NOL data personal owner.

---

## ⛔ WAJIB BACA DULU

`agents/mr-flow/main.go` = **FROZEN brain-core** (chattr +i + hash KERNEL_FREEZE). **JANGAN buka.**
Filtur Telegram baru (format, tipe media, vision, dst) → **CABANG NON-frozen `agents/mr-flow/telegram_media.go`**
(+ switch env). main.go cuma manggil fungsi cabang. Routing/persona → §5. Arsitektur otak → `lock/brain.md`.

---

## 0. APA INI

mr-flow = **agent owner** (tier primary), jalan sbg **WASM** (wasip1, **standard Go** — bukan tinygo →
full stdlib). Tugas: long-poll Telegram → proses (LLM via router :2402 / route ke squad) → balas.
Persona di kv `prompt` (Settings GUI). Model per-agent kv `router_model` (default `flowork-brain`).

---

## 1. ARSITEKTUR I/O

```
Owner ⇄ Telegram Bot ⇄ mr-flow (WASM)
   getUpdates (long-poll) → Message{text | document | photo | voice | caption}
       │ media? → enrichMedia() [telegram_media.go] → teks
       ▼
   proses: slash? deterministic-route? → LLM (router) / task_run squad
       ▼
   sendMessage() → formatTelegram() [telegram_media.go] → Telegram HTML (rapi)
```

---

## 2. BACA MEDIA (dokumen/foto/voice) — `telegram_media.go`

Dulu mr-flow **cuma handle text** (`Text==""` → drop). 2026-06-23: media di-baca. Loop manggil
`enrichMedia(msg, token)` → ubah media jadi teks yg diproses LLM. Semua **GRACEFUL** (gagal = tetap
balas acknowledge, ga crash). Switch `FLOWORK_TG_MEDIA` (default ON; "off" = text-only).

| Tipe | Cara | Dependensi |
|---|---|---|
| **Dokumen** | getFile→download→`mediaDocument`: teks (txt/md/code/json/csv) dibaca isinya (cap 12k char); binary (pdf/docx) → note minta kirim teks | — |
| **Voice/Audio** | download→`sttTranscribe`: multipart POST `router /v1/audio/transcriptions` → transkrip | **STT provider AKTIF** (Settings → Media Providers: deepgram/assemblyai/gemini/openai). Ga ada → graceful note |
| **Foto** | download→base64→`visionDescribe`: POST chat endpoint `image_url` → deskripsi ("yang gw LIHAT…") | model **vision-capable** (Claude) + router pass image content. Ga support → acknowledge + caption |

Catatan: download via `fetch` (host bridge, base64 round-trip, **cap 4MB**). File > 4MB ke-truncate.
Foto pakai resolusi terbesar (`Photo[last]`).

---

## 3. FORMAT PESAN (rapi di Telegram) — `telegram_media.go`

**Akar:** LLM output markdown (`**bold**`, `` `code` ``, `# header`) tapi sendMessage dulu **tanpa
parse_mode** → muncul mentah = "ngak rapi". **Fix:** `formatTelegram()` convert markdown → **Telegram
HTML** (`<b>`/`<i>`/`<code>`/`<pre>`/`<a>`), sendMessage kirim dgn `parse_mode=HTML`. Code block/inline
di-"parkir" dulu biar isinya literal. **FALLBACK:** kalau HTML ditolak Telegram (400) → kirim ulang
POLOS (`stripMarkdown`) — pesan ga pernah ilang. Switch `FLOWORK_TG_FORMAT`: `html` (default) | `plain`/`off`.

**CHUNKING (2026-06-26):** pesan PANJANG dulu kepotong limit Telegram (4096). `splitForTelegram()`
(telegram_media.go, non-frozen) potong jadi chunk ≤3500 rune di batas BAGUS (paragraf > baris > spasi,
ga motong tengah kata); `sendMessage` (main.go) rekursi kirim tiap chunk. Switch `FLOWORK_TG_CHUNK=0`
= matiin (1 pesan). Verified: 8930c → 3 chunk bersih; pendek → 1. **INPUT** >4096 = limit Telegram
client-side (ga bisa di-fix server) → workaround: kirim teks panjang sbg **file .txt** (mr-flow baca via enrichMedia).

---

## 4. SWITCH (env, NON-frozen — jalan evolusi)

| Switch | Default | Guna |
|---|---|---|
| `FLOWORK_TG_FORMAT` | html | format pesan keluar (html rapi / plain polos) |
| `FLOWORK_TG_MEDIA` | on | baca media masuk (off = text-only) |
| `FLOWORK_GROUP_SLASH` | off | slash group Telegram (lihat lock/group.md) |
| `TELEGRAM_BOT_TOKEN` / `TELEGRAM_ALLOWED_CHATS` | — | secret (Settings) — token bot + chat owner |

---

## 5. ROUTING & OTAK (ringkas — detail di doc lain)

- **Route ke squad**: `task_list`→`task_run(category)` (anti-nyasar, lihat `lock/group.md`). Pre-classifier (`deterministicRoute` keyword + `classifyRoute` LLM) jalan SEBELUM `callLLM` di runDaemon + doHandle.
- **SELF-HANDLE GATE** (akar fix "disuruh sendiri malah nyalain crew"): kalau owner eksplisit nolak delegasi ("jangan pake agent/crew", "lakuin/kerjain sendiri") → `wantsSelfHandle()` true → **4 kondisi route di-SKIP** (`&& !wantsSelfHandle`) → jatuh ke `callLLM`, mr-flow kerjain SENDIRI (browser/web/file). Frasa di `self_handle_ext.go` (NON-frozen) + ENV `FLOWORK_SELF_HANDLE_PHRASES` (nambah tanpa buka freeze).
- **TOOL-CALL LEAK GUARD** (router, harness angkat beban): model lokal kadang emit `<tool_call>{...}</tool_call>`
  sebagai TEKS di content (key "parameters" / JSON invalid unquoted) → --jinja ga parse → BOCOR ke user + tool ga
  jalan. AKAR di doktrin: AOLA-001 + insting dulu nyuruh "GUNAKAN format `<tool_call>`" (udah DIBETULIN di
  `doctrine_seed.json`/`instinct_seed.json` → "panggil tool LANGSUNG, jangan tulis sintaks sbg teks"). HARNESS:
  `router/internal/router/toolcall_recover_ext.go` (`recoverTextToolCalls`, dipanggil dispatcher) parse-LENIENT teks
  tool_call → native tool_calls + strip content. Switch `FLOWORK_TOOLCALL_RECOVER=0`. Verified Rule-9 + 4 unit test.
- **GHOST-GUARD v2** (`mr-flow/main.go` + di-port ke `agentkit.go`): model lokal kadang cuma NARASI niat
  ("Scanning workspace... 🔍") tanpa pernah manggil tool → user nungguin jawaban yg ga balik. AKAR: `tool_choice:"auto"`
  bikin model boleh milih "ngomong doang". FIX (stabil 2026-06-26): kalau `looksLikeGhostPromise(content)` →
  nudge **DAN set `forceToolChoice=true`** → req berikutnya `reqMap["tool_choice"]="required"` (paksa model manggil
  tool, ga bisa ngeles narasi). Bounded `maxGhostNudges=6`; abis itu **honest-fallback** ("tool ga kepanggil, model
  lokal ngeyel — coba ulang / lebih spesifik"), BUKAN return ghost-promise. Verified Rule-9: prompt yg dulu ghosting →
  log `ghost-guard: nudge 1 (narasi tanpa tool → force-tool)` → `tool_call: capabilities_list` BENERAN eksekusi →
  reply berisi hasil. Frozen (re-hash 2026-06-26). Template/worker auto-warisan via `agentkit.Main()`.
- **Anti-halu + akses internet**: web_search/webfetch + **browser asli** (akses penuh) + cek tahun, ga ngarang. Lihat persona block "ANTI-HALU".
- **Kontradiksi data**: `cognitive_tensions`/`cognitive_resolve` + tanya owner 3x/hari. Lihat `lock/CognitiveGraph.md`.
- **Persona** (kv `prompt`): identitas + ROUTER TEAM + ANTI-HALU + browser + kontradiksi. Edit AMAN: GET config UTUH → ubah `prompt` → POST (Save full-replace, secret ke-reconcile).

---

## 6. BUILD & DEPLOY (PENTING)

WASM = **tinygo** (wasi, `-scheduler=none -opt=z`) via `scripts/build-agent.sh` — toolchain deployed+proven (~499KB):
```
cd agent && GOWORK=off GOTOOLCHAIN=go1.23.4 bash scripts/build-agent.sh mr-flow
```
*(standard-go `GOOS=wasip1 GOARCH=wasm go build` JUGA compile (~4.8MB), tapi yang di-deploy = tinygo.)*
Deploy: copy `agent.wasm` → `~/.flowork/agents/mr-flow.fwagent/agent.wasm` (= **wasm** yg kernel EKSEKUSI;
start.sh NEVER overwrite yg udah ada) → restart host (kill :1987, docktor rebuild) → kernel load wasm baru.
Edit main.go (frozen) butuh: chattr -i → edit → rebuild wasm → deploy → **re-hash KERNEL_FREEZE** → chattr +i.
⚠️ **state.db ≠ ada di samping wasm itu** — buat agent builtin (mr-flow dkk) DB di-redirect ke SOURCE TREE. Lihat §6b.

---

## 6b. RUNTIME DB & ORCHESTRATOR — GROUND TRUTH (anti-misdiagnosis)

**Di mana state.db HIDUP.** Kernel scan agent dari `~/.flowork/agents` (gui log: `kernel: agents dir`), TAPI
workspace tiap agent di-resolve `agentdb.SourceWorkspace(id, stagedPath)` (`agentdb.go`):
- `<ProjectRoot>/agents/<id>/` ADA (= agent **builtin**: mr-flow, fb-*, browse-*) → state.db = **`<ProjectRoot>/agents/<id>/workspace/state.db`** (SOURCE TREE).
- ngga ada (agent **terinstall murni**: scan-distiller, dll) → `<staged>/workspace/state.db` (`~/.flowork/...`).
`ProjectRoot()` = env `FLOWORK_PROJECT_ROOT` > cwd (non-hardcode, multi-OS). Workspace gitignored.
→ jadi: **wasm** dieksekusi dari `~/.flowork/.../agent.wasm`, **state.db** mr-flow ditulis di source tree. Beda lokasi.

**Akar "interaction-logging stale" (roadmap d) = MISDIAGNOSIS.** Logging SEHAT — diverifikasi Rule-9 (pesan
masuk real-time). DB AKTIF mr-flow = `agent/agents/mr-flow/workspace/state.db` (telegram+rpc current).
`~/.flowork/agents/mr-flow.fwagent/workspace/state.db` **TIDAK PERNAH dipakai** sejak redirect → beku 2026-06-09
= **STALE, jangan dipercaya buat debug**. Cek DB aktif: `grep "kernel: loaded <id>" /tmp/flowork-gui.log` → `ws=`.

**Data nyangkut → ✅ MERGED (2026-06-25, owner-authorized):** 264 interaksi lama (Mei–9 Jun) dari staged DB
di-gabung ADDITIVE ke DB aktif (dedup 4-tuple, integrity-check ok → 367 rows total) → kontinuitas episodic mr-flow
PULIH. Reversible: `DELETE FROM interactions WHERE occurred_at<'2026-06-24'`. Staged DB dibiarkan (arsip; JANGAN delete tanpa izin owner).

**ORCHESTRATOR — default = mr-flow (RESOLVED 2026-06-25, via SWITCH).** Dulu semua channel (chat.go `/api/chat`,
native.go CLI/MCP, flowork-mcp, connector-template, groupsapi `OrchestratorID`) nge-default ke `mr-flow-next` yg
BELUM ke-deploy (`~/.flowork/agents/mr-flow-next.fwagent` ngga ada manifest/wasm → kernel reject) → `/api/chat`
default mati. Owner revert ke akar: **SATU sumber kebenaran + ENV switch `FLOWORK_ORCHESTRATOR` (default `mr-flow`
= LIVE)**. File: `orchestrator_default.go` → `defaultOrchestratorID()` (host, NON-frozen) + `groupsapi_ext.go` →
`effectiveOrchestratorID()` (orchestrator.go frozen init `var OrchestratorID = effectiveOrchestratorID()`).
Verified Rule-9: default-route → mr-flow koheren + ter-log. **Migrasi orchestrator nanti (deploy mr-flow-next):
cukup set ENV `FLOWORK_ORCHESTRATOR`, NOL buka freeze (Rule 7).**

**Telegram poll 409.** gui log: `getUpdates 409 conflict — instance lain ngambil-alih poll`. Cuma 1 flowork-gui
lokal → poller LAIN (mesin/sesi lain, bot token sama) rebut long-poll → inbound Telegram bisa ke-grab di tempat lain. Cek: matiin instance/token ganda.
> STATUS 2026-06-26: **GA aktif** (ga ada "Conflict/409" di log terkini, poller tunggal). ENVIRONMENTAL —
> kalau muncul lagi = ada instance/mesin LAIN pakai bot token sama → owner matiin yg ganda / rotate token.
> Akar di luar kode (ga bisa di-fix server-side; token kepake 2 tempat = Telegram kasih 1 poller doang).

---

## 7. PETA FILE & FREEZE

| File | Peran | Freeze |
|---|---|---|
| `agents/mr-flow/main.go` | core: long-poll, loop, LLM, sendMessage, struct Message, **seam #2C deferred-tools** (re-fetch specs abis `tool_lookup` → tool deferred masuk array, lihat `lock/tools.md §7.5`) | **FROZEN** brain-core (hash `26769416…`) |
| `agents/mr-flow/telegram_media.go` | CABANG: format + media handler + switch | NON-frozen |
| `agents/mr-flow/recall_gate.go`, `working_set.go`, `recovery_capture.go` | recall/context | (lihat status masing2) |
| `agent.wasm` | artifact build (gitignored) | — |
| Persona (kv `prompt`), tool_specs.go | enabler routing | non-frozen (sesekali tune) |

---

## 8. CARA NAMBAH FILTUR TELEGRAM (tanpa buka frozen)

- **Format / tipe media baru / vision / STT tweak** → `telegram_media.go` (cabang) + switch env.
- **Field Telegram baru** (mis. `sticker`, `location`) → tambah field di struct `Message` (main.go) BUTUH
  unfreeze (minim: cuma field), logic-nya di cabang. Minta izin owner buat unfreeze main.go.
- **Routing/persona** → kv `prompt` (data) / tool_specs (non-frozen).

---

## 8b. CATATAN PENDING / TODO (per 2026-06-23 — belum dikerjain, dicatat biar ga lupa)

- **Foto VISION belum penuh.** `visionDescribe` udah kirim format OpenAI `image_url` (data URI base64), TAPI
  router (`chatCompletionsHandler`) **belum pass content-array (image block) clean** ke model →
  foto sekarang kemungkinan cuma **acknowledge + caption**, belum bener-bener "dilihat". **TODO:** edit
  router biar preserve content array (image_url / Anthropic image block) + forward ke model vision-capable
  (Claude). Setelah itu "foto owner bisa dilihat" beneran jalan. Jalur Anthropic `/v1/messages`
  (`Content json.RawMessage`) lebih gampang preserve image — bisa dipakai sbg endpoint vision khusus.
- **Voice STT butuh provider AKTIF.** Jalan kalau owner set STT provider di Settings → Media Providers
  (deepgram/assemblyai/gemini/openai). Belum diset = mr-flow balas graceful note. **TODO:** cek/aktifin provider.
- **Dokumen binary (PDF/docx) belum dibaca isinya** — cuma teks (txt/md/code/json/csv). **TODO (opsional):**
  extractor PDF (di host-side tool, bukan WASM) kalau perlu.
- Semua di atas **GRACEFUL** sekarang (ga crash, selalu balas) — aman dipakai walau belum penuh.

### ARAH BESAR — buang subscription-gating (PROVEN mr-flow 2026-06-25; GLOBAL nunggu agentkit)
Owner usul: **buang gating subscription tool** (footgun "lupa centang GUI → agent lumpuh") → **SEMUA tool ke-expose nama-nya** (murah, lewat #2C deferred-katalog) + pilihan tool dikemudiin **DOKTRIN+INSTING+KONSTITUSI** (bukan allowlist statik). AMAN sebab **exposure ≠ permission**: tool bahaya tetep ke-gate cap pas RUN (`filterPrivilegedCaps` + `SandboxRun` Gate-1, INDEPENDEN subscription) — **divalidasi live + regression test 4/4**.

**STATUS mr-flow (SUDAH):**
- **Cap migrasi ke manifest** (langkah-3): 4 cap (`exec:shell`/`fs:read:/shared/*`/`fs:write:/shared/*`/`net:fetch:telegram`) ditambah ke `capabilities_required` (20→24) → mr-flow gak gantung subscription buat cap. Manifest re-frozen. (`net:fetch:telegram` KRITIS = I/O Telegram.)
- **All-tools ON** (langkah-4, switch `FLOWORK_DEFER_TOOLS`+`FLOWORK_EXPOSE_ALL_TOOLS`, scoped primary): mr-flow liat **202 tool** (22 schema + 180 katalog), tool non-sub bisa lookup+run, **Rule-9 LLM koheren NOL flail**. Agen lain gak kena.

**SISA (GLOBAL):** (1) ~~**agentkit**~~ ✅ **SELESAI 2026-06-25** — loop+guard+seam #2C ke-ekstrak ke modul SHARED `agent/agentkit/`; 5 worker + template warisan (verified Rule-9). Semua worker punya seam → all-tools GLOBAL ke-unblock. mr-flow tetep loop sendiri (referensi). Kanonik: `lock/agentkit.md`. (2) perkuat insting/konstitusi (roadmap #2/#2B) sbg kemudi pilih dari 200 tool; (3) GUI tool-catalog → repurpose ke kurasi doktrin/insting + toggle per-agent (ganti ENV). Detail tool: `lock/tools.md §7.5–7.6`.

---

## 9. PANTANGAN

- ❌ Jangan kirim pesan Telegram tanpa fallback polos (pesan ga boleh ilang gara2 HTML rusak).
- ❌ Jangan anggap STT/vision pasti ada — selalu graceful (provider bisa belum diset).
- ❌ Jangan print/log isi dokumen/foto owner ke tempat publik (privasi).
- ❌ Jangan buka main.go (frozen) buat filtur Telegram — pakai `telegram_media.go`.
- ❌ Jangan turunin cap fetch atau hapus fallback — bot owner = ga boleh diem/crash.
