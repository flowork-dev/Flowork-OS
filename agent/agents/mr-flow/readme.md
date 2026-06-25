# `agents/mr-flow/` — SOURCE KERJA mr-flow (EDIT DI SINI)

> ⚠️ JANGAN KETUKER sama `agents/mr-flow.fwagent/` (SEED fresh-install — wasm current) atau
> `~/.flowork/agents/mr-flow.fwagent/` (yang beneran ke-load). Baca ini dulu.

## Dir ini APA
**Source code KANONIK + CURRENT** dari agent **mr-flow**. `main.go` di sini = versi
terbaru (±1789 baris, full orchestrator). **Semua edit mr-flow dilakuin DI SINI.**
- `main.go` — kode agent (LOCKED, edit owner-approved).
- `manifest.json` — id=`mr-flow`, caps, entry=`agent.wasm`.
- `workspace/state.db` — **STATE LIVE** mr-flow (interactions, constitution, app_grants,
  tool_subscriptions, cognitive graph). Resolver milih workspace REPO ini kalau ada
  (SourceWorkspace override, §4.6 roadmap_agent) → jadi DB yang lo lihat pas debug
  ADA DI SINI, bukan di data-home.

## Siapa mr-flow + TUGASNYA
mr-flow = **SATU agent owner-facing** (DB-based, kenal Mr.Dev via brain). Tugas:
- **Daemon Telegram sendiri** (poll getUpdates, balas owner).
- **Orchestrator 3-jalur** (sejak 2026-06-20): (1) jawab sendiri pake tool, (2) DIRECT
  ke 1 agent via `agent_command`, (3) ke GROUP via `task_run`. Alur lama
  `mr-flow→group→agent` UDAH ga wajib.
- Persona **DB-based** (`config.prompt`+`self_prompt`+constitution sacred), genome pipe
  (brain_search_shared/instinct_recall/graph_recall), ghost-guard + autonomy-mode.
- = **TEMPLATE REFERENSI** standar agent Flowork (lihat `agents/readme.md`).

## ALUR BUILD + DEPLOY (PENTING — sumber kebingungan)
Repo ga nyimpen `agent.wasm` di sini (gitignore). Loader **scan dir `*.fwagent` di
`~/.flowork/agents/`** (data-home), BUKAN dir ini. Jadi edit di sini ga otomatis live:

```sh
# 1. edit agents/mr-flow/main.go
# 2. build wasm (standard wasip1, BUKAN tinygo):
cd agents/mr-flow && GOWORK=off GOOS=wasip1 GOARCH=wasm go build -o agent.wasm .
# 3. DEPLOY ke yang beneran ke-load:
cp agent.wasm ~/.flowork/agents/mr-flow.fwagent/agent.wasm
rm agent.wasm                      # repo ga simpan wasm
# 4. restart ./restart.sh
```

## Peta 3 lokasi mr-flow (biar ga ketuker)
| Lokasi | Apa | Edit? | Ke-load? |
|---|---|---|---|
| `agents/mr-flow/` (DIR INI) | **source current** + state.db live | ✅ YA | ❌ (build dulu) |
| `agents/mr-flow.fwagent/` | **SEED fresh-install** (wasm current = deployed; start.sh seed ke ~/.flowork) | ❌ (refresh dari live) | ❌ (di-seed only-if-absent) |
| `~/.flowork/agents/mr-flow.fwagent/` | wasm DEPLOYED | ❌ (hasil build) | ✅ YA |

> Beda dari **`mr-flow-next`** = rencana orchestrator R3, TAPI BELUM ke-deploy → default
> channel di-revert ke mr-flow via ENV `FLOWORK_ORCHESTRATOR` (lihat `lock/mrflow.md §6b`).
