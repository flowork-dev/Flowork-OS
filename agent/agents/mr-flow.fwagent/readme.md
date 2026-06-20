# `agents/mr-flow.fwagent/` — SNAPSHOT LAMA (JANGAN EDIT, BUKAN SUMBER)

> ⚠️ Ini SERING KETUKER sama source kerja. **BUKAN tempat lo ngedit mr-flow.**
> Source kerja = `agents/mr-flow/` (lihat readme di sana).

## Dir ini APA
**Snapshot/paket LAMA** agent mr-flow yang ke-commit di repo. id-nya tetap `mr-flow`
(agent yang SAMA — BUKAN AI beda), tapi `main.go`-nya versi **lama** (±436 baris,
locked 2026-06-12) — udah ketinggalan jauh dari source kerja `agents/mr-flow/`
(±1789 baris, current). Ada `agent.wasm` ke-commit (Jun 15) yang juga **stale**.

## Kenapa ADA + kenapa membingungkan
- Suffix **`.fwagent`** = format paket agent yang di-scan loader. Loader nyari dir
  `*.fwagent`, TAPI di **`~/.flowork/agents/`** (data-home) — **BUKAN** dir repo ini.
- Jadi dir ini **TIDAK ke-load** + **TIDAK di-referensiin kode/script** mana pun (cek:
  `grep -r mr-flow.fwagent` = 0 hit di kode). Murni snapshot historis yang nyangkut.
- Yang BENERAN jalan = `~/.flowork/agents/mr-flow.fwagent/agent.wasm` (hasil build dari
  source `agents/mr-flow/`, di-deploy manual).

## Aturan (biar ga ketuker lagi)
1. **Mau ubah mr-flow?** → edit `agents/mr-flow/main.go` → build wasip1 → `cp` ke
   `~/.flowork/agents/mr-flow.fwagent/agent.wasm` → restart. (Langkah lengkap di
   `agents/mr-flow/readme.md`.)
2. **JANGAN** edit `main.go`/`agent.wasm` di dir ini — ga ngefek (ga ke-load) + bikin
   makin bingung (dua source beda).
3. State live (state.db) **ga di sini** — ada di `agents/mr-flow/workspace/`.

## Peta 3 lokasi mr-flow
| Lokasi | Apa | Edit? | Ke-load? |
|---|---|---|---|
| `agents/mr-flow/` | source current + state.db | ✅ | ❌ (build dulu) |
| `agents/mr-flow.fwagent/` (DIR INI) | **snapshot lama, stale** | ❌ | ❌ |
| `~/.flowork/agents/mr-flow.fwagent/` | wasm deployed (hasil build) | ❌ | ✅ |

> Saran kebersihan (kalau owner setuju nanti): dir repo ini bisa dihapus/diganti jadi
> readme doang biar ga ada dua source mr-flow yang bikin ketuker.
