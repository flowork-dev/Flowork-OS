<div align="center">

# Flowork OS ⚡

**The self-evolving, sovereign AI operating system. It's the only AI home that writes its own tools, runs fully on your own hardware, and keeps growing — even after you're gone.**

[![Docs](https://img.shields.io/badge/docs-flowork-blue)](https://github.com/flowork-dev/Flowork-OS)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPLv3-green.svg)](LICENSE)
[![Multi-OS](https://img.shields.io/badge/runs%20on-Linux%20·%20macOS%20·%20Windows%20·%20USB-8A2BE2)](#quick-install)
[![Local-first](https://img.shields.io/badge/100%25-local%20·%20white--label-black)]()

[🇮🇩 Bahasa Indonesia](README.id.md)

</div>

---

Flowork is a **local-first AI operating system** built around one idea: an AI that **improves itself**. It creates its own tools when it hits a gap, freezes what's stable so it never breaks, adopts new model/tunnel/voice providers as plug-ins you can pull out anytime, and carries a persistent brain that grows with every task. No cloud lock-in. No vendor you can't unplug. It runs on a `$5` box, a laptop, or a bootable USB stick — and it's yours.

> **Heads up:** Flowork ships with **empty credential slots**. Bring your own keys (or log in via OAuth) from the GUI — nothing is baked in. `GUI is the only source of truth.`

## Quick Install

**Linux / macOS / WSL2**
```bash
git clone https://github.com/flowork-dev/Flowork-OS && cd Flowork-OS
./start.sh          # builds, then serves the GUI on http://127.0.0.1:1987
```

**Portable (no install — USB / any machine)**
Grab `flowork-portable-<version>.zip` from [Releases](https://github.com/flowork-dev/Flowork-OS/releases), unzip, and run `start-flowork.sh` (or `Start-Flowork.command` on macOS, `Start-Flowork.bat` on Windows). A sanitized starter-brain is already inside — it works out of the box.

**Bootable appliance (VM / dedicated USB)**
Flash `flowork-os-<version>.iso` (or `.usb.img.gz`) to a stick and boot. A whole AI OS, no host install.

## Getting Started

| Command | What it does |
|---|---|
| `./start.sh` | Build + launch the full stack (router + agent + local model) |
| open `http://127.0.0.1:1987` | The GUI — the single source of truth for every setting |
| open `http://127.0.0.1:2402` | Router console — providers, models, usage, mesh, console log |
| chat **mr-flow** | Your flagship agent — talk to it in plain language, from the GUI or Telegram |
| `./stop.sh` | Stop everything cleanly |

## Why Flowork

- **A real self-evolution loop.** When mr-flow lacks a tool, it writes one, tests it, and promotes it — no human in the loop. Stable code gets **frozen** (hash + immutability) so future changes can never silently break the core.
- **Plug-and-play, vendor-proof.** Models, tunnels, TTS/STT, image, embedding, web-fetch — every third party is a registry plug-in. If a company disappears, you pull the file out and the core still builds. `Register / Get / List`, nothing hardcoded.
- **Sovereign & local.** Runs your own local model by default; any LLM provider (Claude, Antigravity/Gemini, OpenAI-compatible, Ollama) slots in. Your data, your brain, your machine.
- **An agent colony.** mr-flow orchestrates a team — scanners, coders, researchers, self-evolution judges — each isolated, each with its own memory.
- **White-label, multi-OS, no hardcoded paths.** Zero company branding in the code; rebrand freely. Linux, macOS, Windows, or a flash drive.

## CLI vs Messaging — quick reference

| | GUI (`:1987`) | Telegram | CLI tools |
|---|---|---|---|
| Chat mr-flow | ✅ | ✅ | ✅ |
| Configure everything | ✅ (source of truth) | — | — |
| Alerts / notifications | ✅ | ✅ | — |
| Run agents / triggers / schedules | ✅ | ✅ | ✅ |

## Documentation

| Topic | Where |
|---|---|
| Architecture & subsystems | [`docs/`](docs/) |
| Providers & models | Router console → Providers / Models |
| Brain & memory | Router console → Brain |
| Agents & self-evolution | GUI → Agents / AI Studio / Self-Evolution |
| Security (Threat Radar, MITM, scanner) | GUI → Threat Radar · Router → MITM |
| Auto-update & release | `docs/AUTO-UPDATE.md` |

## Contributing

Flowork grows by **plug-ins, not by breaking the core.** Add a feature through a sibling extension / registry seam — never edit a frozen file. See the architecture docs before opening a PR.

## Community

Issues and PRs welcome. This is open infrastructure for a sovereign, self-improving AI — build on it, fork it, make it yours.

## License

[AGPL-3.0](LICENSE) — built to stay free and to keep every derivative free.

<!-- Topics: ai, ai-agent, ai-agents, llm, claude, anthropic, openai, chatgpt, codex, claude-code, ollama, gemini, self-improving-ai, self-evolving, sovereign-ai, local-ai, ai-os, white-label, plug-and-play, multi-agent, flowork -->
