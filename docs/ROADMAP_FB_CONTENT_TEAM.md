# ROADMAP — Facebook Content Team (branding mr-flow: ahli AI + IT)

> Status: SPEC (owner Aola, 2026-06-23). Blueprint multi-agent buat auto-konten Facebook.
> Self-contained buat handoff. Patuhi [Rule Emas](../../.claude/.../rule-emas) — cabut-akar, no-duplikat.
> Fondasi udah ADA: browser-control Go-native + cookie-inject + plug-and-play (lihat
> `ROADMAP_MULTI_OS_TOOLS.md` §13 + memory `flowork-browser-control-go-native`).

## 0. Tujuan
Branding **mr-flow sebagai ahli AI + IT** lewat postingan Facebook otomatis. Konten berkualitas,
**ga boleh duplikat**, ramah-viral (pakai gambar).

## 1. Tim agent (hemat: model sesuai beban)
| Agent | Model | Tugas |
|---|---|---|
| **fbspecial** | **OPUS** (mahal → cuma eksekutor) | POSTING ke FB via browser (bagian susah: reasoning adaptif atas DOM FB yg berubah-ubah). Cuma post. |
| **writer** | **HAIKU** (hemat) | Nulis status/artikel: Bahasa **Inggris**, jelasin repo singkat, bahas **kelebihan & kekurangan**. |
| **repo-finder** | Haiku/tool | Cari repo GitHub + ambil **cover/screenshot** (gambar = lebih gampang viral). |

mr-flow = **ORCHESTRATOR** (bukan eksekutor): kasih tugas → group "facebook" → repo-finder →
writer(Haiku) → fbspecial(Opus) post. mr-flow ga ngotorin tangan FB.

## 2. Aturan konten
- **Bahasa Inggris**, ringkas, bahas **pros & cons** repo.
- **TANPA link clickable** di body (FB nge-derank link) → sumber jadi TEKS:
  `source : ~github.com/alamat/repodisini` (tilde/format biar bukan auto-link).
- **Gambar wajib** (cover repo / screenshot) → viral.

## 3. Topik, sumber, & rasio
- Sumber: (a) **repo GitHub**, (b) **data hacking** yg kita punya (knowledge base) jadi topik,
  (c) **promo Flowork** dari **changelogs** (`router/Changelog`, `agent/CHANGELOG.md`).
- **Rasio 2:1** → 2× bahas umum (hacking/repo) : 1× promo Flowork.
- Branding: mr-flow = ahli AI + ahli IT.

## 4. Anti-duplikat (KERAS)
- Repo/topik yg udah pernah di-post **JANGAN di-post lagi**. Simpan di **database**, **cek dulu**
  sebelum post. Loop (boleh ulang) **CUMA kalau bahan habis**.
- Store dedup: tabel/kv/brain-drawer "posted_topics" (key = repo url / topic hash + tanggal).

## 5. Hygiene browser (DONE 2026-06-23)
- `browser_close` tiap tugas kelar (skill wajib) + idle-reaper 30mnt in-agent + docktor orphan-backstop.

## 6. Mekanisme Flowork (hasil investigasi 2026-06-23 — buat build)
- **Model per-agent** = kv `router_model` (lihat `codemap_enricher_seed.go` / `codemap_semantic.go`
  "kv router_model"). Set fbspecial→opus, writer→haiku via kv.
- **Bikin agent** = pola `agent/scripts/setup-*.sh`: `SRC="agents/$ID"` (clone agent-template) +
  set kv (prompt/persona/model) + register. Contoh: `setup-operator.sh`, `setup-saham-crew.sh`,
  `setup-thinking-group.sh`.
- **Group/crew** = roster di loket kv: `members=...`, `synthesizer=...` (lihat `setup-operator-group.sh`)
  + taskflow categories. mr-flow delegasi via task tools (`taskListTool`/`taskRunTool`) / categories.
- **Per-agent tool akses** = subscribe (cap auto-grant — fix plug-and-play udah ADA). fbspecial
  subscribe `browser_*` (cap browser:control). writer ga butuh browser.
- **Skill** = POLA (FB sering ganti div → agent Opus re-snapshot + cocokin by-makna, JANGAN
  hardcode selector). Lihat browser-flow di memory `flowork-browser-control-go-native`.

## 7. Rencana build (fase, incremental + verified)
- **F1 — fbspecial (Opus) + skill post.** Bikin agent fbspecial (kv model=opus), subscribe browser_*,
  skill "facebook-post" (pola: cookie→navigate→cari-komposer→paste→cari-tombol-post→close). Bukti:
  fbspecial di-kasih {teks, gambar} → post ke FB (privat dulu) → close.
- **F2 — writer (Haiku).** Agent writer (kv model=haiku): input repo → output status Inggris
  (pros/cons + `source : ~...` + saran caption gambar). Bukti: 1 repo → status bagus.
- **F3 — repo-finder + image.** Cari repo GitHub (tool web/search) + ambil cover/screenshot
  (browser_screenshot halaman repo ATAU og:image). Bukti: repo → {meta, gambar}.
- **F4 — dedup DB + sumber + rasio 2:1.** Store posted_topics + cek-sebelum-post. Sumber:
  hacking-data + changelog (promo). Scheduler 2:1. Loop hanya kalau habis.
- **F5 — orchestration.** Group "facebook" (roster: repo-finder, writer, fbspecial). mr-flow
  delegasi: trigger → group → pipeline → post. Jadwal otomatis.

## 8. Catatan
- Cookie FB di profil browser Flowork (persisten) — bisa expire, re-import dari profil Chrome
  "FLOWORK" (lihat memory). Akun FB = "Sundan".
- Mulai POSTING audience PRIVAT (Hanya saya) pas test; publik pas owner OK.
- Opus cuma di fbspecial (post). Sisa Haiku/tool = hemat. (Owner: "opus mahal, haiku hemat wkwk".)
