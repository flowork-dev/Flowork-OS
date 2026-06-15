# Flowork — Canonical Taxonomy (1 term, 1 meaning)

> Source of truth for terminology. AI Studio (architect), the GUI, and self-evolution MUST use
> these terms consistently — so Flowork can understand itself (a prerequisite for autonomy).
> Part of the consolidation refactor (R2). The old "category/crew" = DEPRECATED → use **Group**.

## Core concepts

| Term | Definition | Created via | Stored / runtime |
|---|---|---|---|
| **Agent** | ONE unit of work (1 wasm) — focused on a single task (ant principle). Has a persona (state.db) + skills. | architect (part of a Group/App) | `~/.flowork/agents/<id>.fwagent` |
| **Group** (Team) | A COLONY: several Agents (workers) + 1 synthesizer (lead). Coordinator fans out → synthesizer merges. | `build_team` (AI Studio) | group module + `group.json` (kv group=1) |
| **App** | A PROGRAM (UI/HTML or process) that runs in the App menu — dual-use (human + AI via InvokeOp). | `build_app` (AI Studio) | `~/.flowork/apps/<id>/` (manifest+ui) |
| **Skill** | A focused PLAYBOOK (SKILL.md) — knowledge INJECTED into the LLM by relevance (progressive disclosure). | `authorSkill` / brain | embedded `skilldata/` + `~/.flow_router/skills/` |
| **Schedule** | TIME automation (cron) → run an Agent/Group → deliver the result. | `schedule_team` (AI Studio) | trigger engine, `type=time` |
| **Trigger** | EVENT automation (webhook/file-watch) → run an Agent/Group → deliver the result. | `create_trigger` (AI Studio) | trigger engine, `type=webhook/file-watch` |
| **Orchestrator** | The message-routing brain → delegates to Group/App/AI Studio. THE ONE: **`mr-flow-next`**. | (system) | wasm daemon |
| **AI Studio** | THE single CREATION door — a conversational chat (architect) that builds everything above. | (system) | Coder tab / `/api/chat` |

## Hard rules
- **"category" / "crew"** (old taskflow) = **DEPRECATED**. A team of agents = **Group**. Don't invent new overlapping concepts.
- **App ≠ Group.** App = a program (clock, calculator, UI). Group = a team of agents that think/answer. The architect MUST
  distinguish: an AI-that-answers (poems/translation/fortune-telling) = **Group**; a UI program = **App**.
- **Skill = injected knowledge**, not a callable template. (The `/v1/skills/` DB = a separate template system, a different concept.)
- **1 Orchestrator** (`mr-flow-next`) — ONE routing brain across ALL channels (Telegram via `telegram-channel`
  → target=mr-flow-next; HTTP/CLI `/api/chat` → mr-flow-next). The legacy `mr-flow` is **retired AS ORCHESTRATOR**
  (its getUpdates daemon is dormant, exits cleanly at boot) BUT **stays alive as a primary worker** — it hosts the
  scanner / diagnostics / codescan / 40-tools. DO NOT DELETE it (R3 verified 2026-06-15): scanapi/diagnostics/the
  invariant auditor still call `openAgent("mr-flow")`; deleting it kills GUI tabs + trips the auditor. Retire its ROLE,
  don't kill the agent.
- **Schedule vs Trigger** = 1 engine (trigger), different `type` (time vs event). Separate menus = just views.

## Why this matters (autonomy)
A self-evolving organism must have a **single vocabulary** to reason about itself. Ambiguous terms
("app" = crew OR program) make the architect choose wrong + confuse the GUI/owner (real example: "the digital-clock
app doesn't show up" because it was built as a crew, not an App). A strict taxonomy is part of the backbone of
self-understanding.
