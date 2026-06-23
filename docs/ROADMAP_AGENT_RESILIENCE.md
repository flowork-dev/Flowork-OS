# ROADMAP — RESILIENT AGENT LOOP (tiru Claude: recover · retry · persist · loop tanpa muter)

> Owner: Aola Sahidin (Mr.Dev) · Repo: github.com/flowork-os/Flowork-OS + flowork-base · 2026-06-23.
> Patuh [[ruleemas]]: cabut-akar (bukan tambal), JANGAN edit file freeze (kasih cabang/switch), nano-modular
> + no-hardcode, default LOCK (freeze hanya seijin DEV), push 2 repo, koreksi keamanan+bug sebelum lock.
> Sumber prinsip: studi source Claude Code (`~/Downloads/claude-code-main`) — control-flow + PROMPT.

---

## 0. MASALAH (grounded, bukan nebak)

Agent loop Flowork **mati + muter** pas error transient. Bukti live (log fbspecial 22 Jun 20:20):
tugas "posting status ke Facebook" → **"router error" 2×** → **auto-continue nyangkut sampai #8** (loop sia-sia).

**Akar** (`agent/templates/agent-template/main.go:210-212`, identik di fb-writer/fbspecial/fb-repofinder):
```go
resp, err := fetch("POST", routerURL, ..., 240000)
if err != nil || resp == nil { return "router error" }   // NOL retry, NOL klasifikasi, NOL backoff
```
1 kedip router → seluruh turn mati → task ga kelar → auto-continue buta → kedip lagi → #8.
Akar fatal: Flowork **nyamain "error transient" = "task belum kelar, lanjut bikin progres"**. Mr-flow doang
punya retry kecil (`main.go:809`, attempt<3, 5xx) — squad agents (incl fbspecial si poster) NOL.

## PRINSIP CLAUDE yang ditiru (verbatim dari source)

**Control-flow:**
- Transient error (429/5xx/timeout/conn-reset) di-RETRY di level panggilan API, **exp backoff + jitter**
  (500ms×2ⁿ, cap 32s, hormati `Retry-After`), **TRANSPARAN ke loop** — loop ga pernah lihat retry.
- Fatal (400/auth/404) → STOP, ga di-retry.
- Loop lanjut HANYA pas ada PROGRES (tool dipanggil) / recovery ber-batas; balik teks tanpa tool → STOP.
- Anti-runaway: counter recovery ber-batas (bukan ∞).

**PROMPT (prompts.ts:233 — "special prompt" yg dicari owner):**
> *"If an approach fails, diagnose why before switching tactics—read the error, check your assumptions,
> try a focused fix. Don't retry the identical action blindly, but don't abandon a viable approach after a
> single failure either."*
- Honest report (prompts.ts:240): *"never characterize incomplete or broken work as done."*
- *"Do not retry failing commands in a sleep loop — diagnose the root cause."*
- *"If waiting for a background task… you will be notified — do not poll."*
- Verify-before-done + "rationalizations trap" (verificationAgent.ts).

## KENDALA FREEZE (rule emas #1/#7)
- `agent/templates/agent-template/main.go` + `agents/mr-flow/main.go` = **chattr-FROZEN** → HARAM edit tanpa
  izin DEV. `agents/fbspecial/main.go` + `agents/fb-writer/main.go` + `fb-repofinder/main.go` = **editable**.
- Tiap agent = MODUL Go standalone (go.mod sendiri) → **ga ada shared package** → retry ke-duplikasi/agent.
- Akar arsitektur (rule #7): loop baked-in tiap main frozen, ga ada lapis switchable. Fix abadi = **shared
  loop module** yg tiap agent import → improve sekali, ga buka frozen lagi. Tapi itu refactor besar (ITEM 4).
- **PROMPT side nyampe SEMUA agent TANPA sentuh frozen** → lewat cabang non-frozen yg UDAH ada
  (`edu_errors_ext.go` + `SelfPromptRenderHandler`, pola sama injeksi WIB). Ini menang termurah & paling patuh.

---

## ITEM 1 — PROMPT recovery-doctrine ke SEMUA agent (cabang, NOL frozen) ⭐ MULAI SINI
**Kenapa pertama:** termurah, paling patuh rule emas (nol frozen-edit), nyampe semua agent, langsung ngangkat
kualitas "sadar error + cari jalan keluar" bahkan sebelum control-flow dibenerin.
**Apa:** suntik doktrin recovery (adaptasi prompts.ts:233/240) ke system-prompt tiap agent — lewat
`internal/agentdb/edu_errors_ext.go` (ExtraEduErrors, on-recall) + 1 blok ringkas di `SelfPromptRenderHandler`
(non-frozen, pola WIB) ATAU edu-error baru `ERR_TRANSIENT_RETRY`/`ERR_STUCK_DIAGNOSE`.
**Isi doktrin (ringkas, hemat token, Indonesia):**
- Approach gagal → **diagnosa dulu** (baca error, cek asumsi, fix terfokus). JANGAN ulang aksi sama buta.
  JANGAN nyerah setelah 1 gagal. Error transient (router/jaringan) → tunggu sebentar lalu coba lagi,
  BUKAN nganggap tugas kelar/nyerah.
- JANGAN ngaku kelar kalau belum kebukti. Lapor jujur kalau gagal.
- Nunggu kerjaan background → bakal di-notify, JANGAN polling ketat.
**Test:** agent dapat error transient → respon "diagnosa + retry", bukan "nyerah/ngaku-kelar" (live via
dbgchat). **Koreksi keamanan+bug → LOCK → push 2 repo.**

## ITEM 2 — Control-flow retry+backoff+klasifikasi di squad agents (editable) 
**Kenapa:** fix AKAR "router error→#8" buat agent yg editable (fbspecial = poster FB → buktiin FB kelar).
**Apa:** ganti pola `return "router error"` jadi **retry-with-backoff transparan**:
- helper `callRouterWithRetry()`: transient (err net/timeout/`status>=500`/`429`) → retry exp-backoff+jitter
  (base 500ms, cap ~30s, hormati Retry-After), maks ~5×. Fatal (400/401/403/404) → balik error beneran.
  Transient ABIS retry → balik error jelas (BUKAN "router error" telanjang).
- Terapin ke `agents/fbspecial/main.go`, `fb-writer/main.go`, `fb-repofinder/main.go` (semua editable).
- No-hardcode: angka backoff via const + boleh ENV override.
**Test:** simulasi router 503/timeout → agent retry sampai sukses (atau STOP fatal), **TANPA** loop #8.
Hitung: berapa retry, berapa yg recover. **Koreksi → LOCK → push.**

## ITEM 3 — Auto-continue PROGRESS-AWARE + anti-runaway
**Kenapa:** stop loop-#8: auto-continue cuma boleh kalau ADA PROGRES, bukan gara-gara error.
**Apa:** di loop agent (squad/editable): bedain "turn errored" vs "turn progres". Track **consecutive-error
continuation**; cap rendah (2-3) → STOP + lapor jujur ("router ga stabil, gw stop biar ga muter"), BUKAN
lanjut ke maxAutoContinue(50). Counter recovery ber-batas (pola Claude: max-recovery N).
**Test:** router mati terus → agent STOP di ≤3 continuation dgn pesan jelas, bukan #8/#50. **LOCK → push.**

## ITEM 4 — (BESAR, NUNGGU KEPUTUSAN DEV) Shared loop module + frozen mains
**Kenapa:** fix ABADI (rule #7): loop+retry jadi 1 modul yg di-import semua agent → improve sekali, agent baru
otomatis dapat, ga pernah buka frozen main lagi. Ini cabut-akar arsitektur "loop baked-in per frozen main".
**Apa (opsi, DEV pilih):**
- (a) Ekstrak `agentloop` jadi modul Go shared → tiap agent main (termasuk frozen) import. Butuh **izin DEV
  buka freeze** `agent-template/main.go` + `mr-flow/main.go` (rule #1) buat ganti loop inline → call shared.
- (b) Atau retry di lapis host-fetch (tapi `host.go` = kernel frozen → juga butuh izin DEV + cabang).
**STATUS: BLOKIR — butuh keputusan + izin DEV** (buka freeze = rule #1). Sebelum ini: ITEM 1-3 udah nutup
mayoritas masalah (squad agents + prompt semua agent). mr-flow udah punya retry dasar.

---

## DISIPLIN EKSEKUSI (rule emas)
1 ITEM dulu, jangan lompat. Tiap item: todolist → eksekusi → **koreksi keamanan+bug** → **test (kasih ANGKA)**
→ **LOCK** (bukan freeze; freeze hanya seijin DEV) → tandai selesai → **push 2 repo** → item berikut.
Build agent = rebuild WASM (lihat [[flowork-mrflow-build-toolchain]]). GUI test pakai login (GITHUB_ACCOUNT.MD).

## URUTAN: ITEM 1 (prompt, nol-frozen) → ITEM 2 (control-flow squad) → ITEM 3 (progress-aware) → ITEM 4 (DEV decision).
