# Flowork — Canonical Taxonomy (1 term, 1 meaning)

> Sumber kebenaran istilah. AI Studio (architect), GUI, dan self-evolution WAJIB pakai
> istilah ini konsisten — biar Flowork bisa paham dirinya sendiri (prasyarat autonomi).
> Bagian dari refactor konsolidasi (R2). "category/crew" lama = DEPRECATED → pakai **Group**.

## Konsep inti

| Istilah | Definisi | Dibikin via | Disimpan / runtime |
|---|---|---|---|
| **Agent** | SATU unit kerja (1 wasm) — fokus 1 tugas (prinsip semut). Punya persona (state.db) + skill. | architect (bagian dari Group/App) | `~/.flowork/agents/<id>.fwagent` |
| **Group** (Team) | KOLONI: beberapa Agent (worker) + 1 synthesizer (lead). Coordinator fan-out → synth gabung. | `build_team` (AI Studio) | group module + `group.json` (kv group=1) |
| **App** | PROGRAM (UI/HTML atau process) yg jalan di menu App — dual-use (human + AI lewat InvokeOp). | `build_app` (AI Studio) | `~/.flowork/apps/<id>/` (manifest+ui) |
| **Skill** | PLAYBOOK fokus (SKILL.md) — knowledge yg di-INJECT ke LLM by relevance (progressive disclosure). | `authorSkill` / brain | embedded `skilldata/` + `~/.flow_router/skills/` |
| **Schedule** | Otomasi WAKTU (cron) → jalanin Agent/Group → kirim hasil. | `schedule_team` (AI Studio) | trigger engine, `type=time` |
| **Trigger** | Otomasi EVENT (webhook/file-watch) → jalanin Agent/Group → kirim hasil. | `create_trigger` (AI Studio) | trigger engine, `type=webhook/file-watch` |
| **Orchestrator** | Otak router pesan → delegasi ke Group/App/AI Studio. SATU: **`mr-flow-next`**. | (sistem) | daemon wasm |
| **AI Studio** | SATU pintu CIPTA — chat conversational (architect) yg bikin semua di atas. | (sistem) | tab Coder / `/api/chat` |

## Aturan tegas
- **"category" / "crew"** (taskflow lama) = **DEPRECATED**. Tim agent = **Group**. Jangan bikin konsep baru yg overlap.
- **App ≠ Group.** App = program (jam, kalkulator, UI). Group = tim agent yg mikir/jawab. Architect HARUS bedain:
  AI-yang-jawab (pantun/terjemah/ramalan) = **Group**; program UI = **App**.
- **Skill = inject** (knowledge), bukan template callable. (DB `/v1/skills/` = sistem template terpisah, beda konsep.)
- **1 Orchestrator** (`mr-flow-next`). `mr-flow` legacy → dipensiunin (R3).
- **Schedule vs Trigger** = 1 engine (trigger), beda `type` (time vs event). Menu beda = view doang.

## Kenapa ini penting (autonomi)
Makhluk yg evolusi-sendiri harus punya **vocabulary tunggal** buat reasoning tentang dirinya. Istilah ambigu
("app" = crew ATAU program) bikin architect salah pilih + bikin GUI/owner bingung (cth nyata: "app jam digital
ngak muncul" karena dibikin sbg crew, bukan App). Taksonomi tegas = bagian dari tulang punggung self-understanding.
