# Flowork Architecture — How It Works End-to-End (Router → Agent)

> This document gives the **one whole picture** of how Flowork works: from the **Router**
> (the LLM gateway) all the way to the **Agent** (the thing that acts), with **every feature
> connected**. Derived from the user/operator tutorial (§1–§38).
>
> Two processes, two ports, one mind:
> - **Router** `:2402` — sovereign LLM gateway (use your own subscription, anti-ban, central brain).
> - **Agent** `:1987` — microkernel + control panel; the WASM agent that thinks & acts.
> - **Mesh** — P2P network between nodes (share knowledge/tools with no central server).

---

## 0. THE BIG PICTURE (one diagram)

```
        HUMANS / OUTSIDE WORLD                        LLM PROVIDERS (your subscription)
   Telegram·Discord·Slack·WhatsApp                 Anthropic · OpenAI · Gemini ·
   CLI · Voice(STT/TTS) · Web · MCP                 Cursor · Copilot · LocalAI(llama)
        │   ▲                                                 ▲   │
        │   │ reply                                    cloak  │   │ /v1 (real)
        ▼   │                                          +auth  │   ▼
 ┌───────────────────── AGENT :1987 ─────────────────┐   ┌──────────── ROUTER :2402 ───────────┐
 │ Connections (channel/MCP/native)                  │   │ /v1/* ─ apiKeyMiddleware (flr_,budget)│
 │   │ message in → BUS / loket                      │   │   │                                   │
 │   ▼                                               │   │   ▼ Dispatch:                         │
 │ AI AGENT (Mr.Flow, WASM)  ── "thinks" ─┐          │   │   Providers→Combos→Tags→cost/tier     │
 │   │ call(cap,args) / tools.run         │ llm.complete│ →fallback→Translator→CLOAK→ProxyPool   │
 │   ▼                                    └──────────┼───┼──► real provider → answer             │
 │ SandboxRunV3 (protect→approval→        │          │   │   ▲ Router BRAIN: constitution +      │
 │   interceptor→capability→ratelimit)    │ /api/brain│   │   │ knowledge + antibodies (tier-gate)│
 │   ├─ Tools (file/git/web/shell/OS/...) │ /api/mesh │   │ Usage·Quota·Pricing·ConsoleLog·MITM   │
 │   ├─ Apps (1 state, 2 drivers)         │◄─────────┼───┤ MCP servers · Media(img/tts/stt/emb)  │
 │   ├─ Agent BRAIN (FTS5, private)       │          │   │ Tunnel · API Keys(flr_) · Skills      │
 │   └─ GROUP (gateway to other agents)   │          │   └───────────────┬───────────────────────┘
 │ Schedule+Trigger · AI Studio(Coder)    │          │                   │ ed25519 gossip
 │ Threat Radar · Code Map · Guardian     │          │                   ▼
 └───────────────────────────────────────┘          │   ┌──── MESH P2P (mDNS + rendezvous) ─────┐
            │ knowledge rises (federation)           └──►│ signed packets · 9-layer filter        │
            └───────────────────────────────────────────│ peer karma · Policy (budget fence)     │
                                                         └────────────────────────────────────────┘
```

**The thread that ties it all together:** a message arrives via **Connections** → the **Agent**
thinks → it needs an LLM, so it calls the **Router** `/v1` → the Router assembles the answer
(provider + brain + cloak) → the Agent uses the result to run **Tools/Apps** → the reply leaves
through the same channel. New knowledge can rise into the **Router Brain** and then spread across
the **Mesh**.

---

## 1. THE TWO PILLARS + WHY THEY'RE SEPARATE

| | **ROUTER** `:2402` | **AGENT** `:1987` |
|---|---|---|
| Role | LLM gateway — the "mouth" to every provider | Microkernel + the acting brain — "hands + head" |
| Module | `github.com/flowork-os/flowork_Router` | `flowork-gui` (WASM host) |
| Holds | providers, combos, central brain, cloak, mesh, proxy | WASM agents, tools, apps, group, scanner, guardian |
| Data | `~/.flow_router/` (providers, OAuth, central brain) | `~/.flowork/` (agents, flowork.db, apps) |
| Standalone | YES (drop-in LiteLLM/OpenRouter) | YES (but needs a router/LLM to "think") |

**Why split:** the router owns the concern of "which AI and how to reach it" (cloak, savings,
fallback, subscription); the agent owns "what work gets done" (tools, files, automation). The
agent **does not need to know** any provider — it just does `POST /v1/chat/completions` to the
router. Swap a provider = change the router; the agent doesn't change. That's what makes it
**plug-and-play** + **portable**.

---

## 2. END-TO-END FLOW — one message from IN to OUT

```
[1] IN      Telegram voice note → Connections(channel WASM, polling) → STT(router media) → text
              → dropped on the agent's BUS/loket. Channel = "dumb pipe"; all thinking is in the agent.
                                   │
[2] PICK    loket → mr-flow (orchestrator). Heavy work → delegated to a GROUP (ant-colony).
            GROUP = GATEWAY: an agent can only be called if its group is ON (except mr-flow).
                                   │
[3] THINK   agent needs the LLM → host.llm.complete → POST :2402/v1/chat/completions {model,messages}
              │
              ▼ ROUTER:
              apiKeyMiddleware (flr_? budget cap? → 429/401) → DispatchChatCompletion
              → pick provider (priority/scope/tag/cost) → fallback chain
              → Router BRAIN: if the model is a "commander" → inject constitution+knowledge+antibodies
                              if it's a "crew/haiku" → SKIP the heavy stuff (savings), antibodies only
              → Translator (OpenAI⇄Anthropic⇄Gemini) → CLOAK (if a subscription token) → ProxyPool
              → real provider → answer → record Usage+Quota → reply to the agent
                                   │
[4] ACT     agent gets the LLM answer (e.g. "create file X, run Y") → each action goes through SandboxRunV3:
              host-protection(baseline) → approval-queue → 3 interceptors(path/sensitive/persona)
              → capability-gate(allowed?) → rate-limit → Tool.Run() / App.InvokeOp()
              Examples: file_write (workspace), shell (denylist+classifier), app_<id>_<op> (python core),
              brain_search (local brain), brain_search_shared (router brain)
                                   │
[5] LEARN   agent records the experience → Agent BRAIN (FTS5). What's worthy → federation → Router BRAIN
              → gossiped to the MESH (ed25519, 9-layer filter) → other nodes get smarter too.
                                   │
[6] OUT     reply → TTS(router media) → same channel → user.   (audit_log records everything)
```

---

## 3. ROUTER — INTERNALS (all menus §1–§27 connected)

### 3.1 Entry point + Auth (§1 Endpoint, §22 API Keys, §23/§27 Policy)
```
client (Claude Code/Cursor/SDK/curl/AGENT) → base = origin/v1
  → apiKeyMiddleware (the /v1 gate):
      store client-IP (proxy affinity) → global Budget cap? →429
      flr_ token? ─ verify ─ daily/monthly cap? →429 ─ attach scope+attribution
  → handler /v1/* → DispatchChatCompletion
```
- **flr_ keys** (§22) = client keys with scope + cap; **anonymous** allowed in open local mode.
- **Policy/Budget** (§23/§27) = a `cost_usd` fence per-scope, periodic reset, warn%.

### 3.2 The selection brain: Providers → Combos → Tags → Fallback (§4, §5, §13, §11, §12)
```
dispatchSingleModel(model):
  FindActiveByModel (exact|"claude-*"|"*") sorted by priority ASC
  → pinned? → drop inactive → drop out-of-key-scope
  → private prompt? keep tag:local (§13)   → cost-routing? filter tag:tier (§13)
  → fallback-strategy + cooldown → try in order:
        429 → backoff+retry SAME · 5xx/conn → next provider · exhausted → 502
        success → applyAuth (api_key | subscription-OAuth(auto-refresh) | none)
```
- **Combos** (§5) = a virtual model name ("smart-cheap") → strategy (priority/round_robin/random/
  cost_optimal) → picked + fallback order → each model goes through the full Provider pipeline.
- **Models/Pricing** (§11/§12) = metadata + tariffs (input×output token) for cost-routing & estimates.

### 3.3 Translator (§14)
Converts **request, response, streaming, tool-calls** between OpenAI ⇄ Anthropic ⇄ Gemini ⇄
Cursor/Codex formats. Any client speaks any format → any provider. (Drop-in for every IDE.)

### 3.4 ANTI-BAN — four layers (§9 OAuth Imports, §26 Proxy Pools)
```
1. Auth modes (§4.2): api_key (Bearer) · subscription (OAuth subscription token) · none(local)
2. CLOAK (§9.3): a subscription token is used → cloaking.go mimics Claude Code:
     rename each tool to <name>_cc + 20 decoy tools + billing-header + fake user_id
3. Handshake identity: login-exchange & auto-refresh send User-Agent "claude-cli/…"
     → the login handshake is indistinguishable from an official client
4. Auto-refresh (§9.4): token expired → refresh_token grant (carries the anti-ban identity) → SaveClaude
5. Proxy Pools (§26): rotate outbound IP → one IP never gets burned
```
Result: your **per-device login** token stays alive 24/7 on Android/USB without Claude Code, hard to flag.

### 3.5 ROUTER BRAIN — the knowledge injector (§20, §19 Skills)
```
DB: ~/.flow_router/brain/flowork-brain.sqlite  (SEPARATE from the agent brain!)
before the LLM (in the dispatcher):
  isCrewLightModel(model)?
    YES (haiku/worker, high volume) → SKIP the heavy stuff → antibodies only (saves quota)
    NO (commander/sonnet) → maybeInjectConstitution (top-20×600)
                          + maybeEnrichBrain → Retrieve(query, top5×600) + SelectSkills(top3)
  + Antibodies (anti-hallucination, karma-ranked mistakes) for ALL tiers (small)
  empty brain → Available()=false → 0 injection (safe on fresh install)
```
**Doctrine** (constitution, sacred amplitude 999999, 5W1H gate, always_inject) is injected every
turn — this is the "conscience that won't lie". **Antibody loop**: a mistake → ranked by
karma×relevance×recency → injected BEFORE the model speaks → a hallucination is caught → that
antibody is reinforced. Deterministic, no GPU.

### 3.6 Tool & media bridges (§15 MCP, §16 Media, §21 MITM, §10 Tunnel, §8 CLI)
- **MCP Servers** (§15): paste `mcpServers` JSON → the router spawns the server → its tools become usable by ALL agents.
- **Media** (§16): embedding · text→image · TTS · STT · web — media providers for the agent/voice.
- **MITM Proxy** (§21): intercept AI-coding IDEs (Cursor/Claude Code) → route through the router (savings+cloak).
- **Tunnel** (§10): reach the panel remotely. **CLI Tools** (§8): integrate `flowork-router` into CLI tools.

### 3.7 Observability (§6 Usage, §7 Quota, §24 Console Log, §25 Document)
Every request is recorded → **Usage** (token/cost analytics), **Quota** (remaining provider quota),
**Console Log** (live feed), **Document** (handbook index). **Settings** (§27) = core config + security.

### 3.8 Mesh & Policy (§23) — see section §5 of this document (MESH).

---

## 4. AGENT — INTERNALS (all menus §28–§38 connected)

### 4.1 Microkernel + the security pipe (§38)
```
a module (agent/tool/app) can do nothing on its own → it asks the kernel for a capability: call(cap,args)
each action goes through SandboxRunV3:
  1 host-protection (immune baseline)  2 approval-queue  3 interceptors(path/sensitive/persona)
  4 capability-gate (manifest allows?)  5 rate-limit  →  execute
WASM (wazero): each agent in its own folder + its own state.db → a fault in A = contained to A.
```

### 4.2 AI AGENT (Mr.Flow) — THE CORE (§38): 6 capabilities
1. **Use Apps** — each op with `tool:true` becomes a tool `app_<id>_<op>`, cap `app:<id>`.
2. **Tools + OS control** — `system_power` (shutdown/reboot, multi-OS: linux/RasPi/STB·mac·win;
   Android is differentiated-needs-root), `shell` (denylist+classifier), cap `exec:power`/`exec:shell`
   + ARM switch + dry-run. **The default chat agent does NOT have these caps** (safe).
3. **Skill evolution + TWO BRAINS + mesh** — see §5.
4. **EDUCATIONAL errors** — a block is written as "hug + hint + this is a rule" (never scolding); the
   mistakes_recall loop (learning from mistakes).
5. **Private & shared workspace** — `/workspace` (always, isolated) + `/shared/<id>` (cap
   `fs:shared`); 4-layer confinement.

### 4.3 GROUP — the GATEWAY to agents (§35)
```
ALL agent execution MUST go through a GROUP (except mr-flow).
group OFF/DELETED → Mr.Flow/Schedule/Trigger CANNOT call the agents inside it:
  ToggleAgent → SetDisabled+Reload → kernelhost Runtime.Unload → Runtime.Get()=nil → "not loaded"
mr-flow = the orchestrator (ant-colony): split work to tiny specialists → a synthesizer merges them.
```

### 4.4 Schedule + Trigger (§36) — automatic triggers
One engine (`internal/triggers`): **Schedule** = cron/time; **Trigger** = webhook + file-watch.
The action → `InvokeAgentMessage(target)` → **through the GROUP gateway** (§35). Webhook secret is
constant-time. Seeding: a fresh clone gets the owner's 6 schedules (social.seed.json) — tokens excluded.

### 4.5 App — the application platform (§37): 1 state, 2 drivers
A human (GUI iframe) + an Agent (tool) call the **SAME operation** (`InvokeOp`). The core is
cross-language (`runtime:process`, stdio JSON). Install needs `approve_exec` + a zip-slip guard.
(Android: needs an interpreter → awaiting the wasm runtime.)

### 4.6 Connections (§29) — the connector hub
Channels (Telegram/Discord/Slack/WhatsApp/CLI + voice STT/TTS) · MCP (client+server) · native.
A channel is a dumb pipe; its token lives in the connector's own folder (one leak = one folder).

### 4.7 AI Studio — Coder (§30): an agent builds an agent
The LLM fills a *spec* → the engine deterministically assembles a `.fwpack` → **Verifier** (adversarial
dry-run: red-flag syscall scan, capability-safety) → **Reaper** (apoptosis: failing agents pruned) →
**Death Letter** (knowledge handover across generations).

### 4.8 Agent BRAIN + Skills + Educational Errors (§38.3/§38.4)
`brain_drawers` (FTS5/BM25, no embedding, offline) in each agent's workspace — the "personal notebook".
skill_author (immune+verifier gate) · skill_suggest · mistakes_recall. Self-improving.

### 4.9 Body security (§28 Threat Radar, §31 Code Map, §32 Code Progress)
- **Threat Radar** — 116 scanners: 100+ static auditors (secret/SQLi/SSRF/path-traversal/nil-map) +
  trivy/nuclei (~16K checks, targets on an owner-allowlist). Every file edit → auto-scan.
- **Guardian** (§27) — verifies the binary+kernel at every boot; tamper → SAFE-MODE; root → OS-immutable.
- **Code Map** (§31) — a dependency map of the whole monorepo (d3 "node" graph: file A calls/is-called-by).
- **Code Progress** (§32) — an audit log of every action. **Document** (§33) · **Settings** (§34, owner-level).

---

## 5. TWO BRAINS + FEDERATION + MESH (the link between Router ⇄ Agent ⇄ Network)

```
┌── AGENT BRAIN (private, per-agent) ─┐        ┌── ROUTER BRAIN (central, server) ┐
│ ~/.flowork/agents/<id>/state.db     │        │ ~/.flow_router/brain/*.sqlite    │
│ brain_drawers FTS5/BM25, OFFLINE    │        │ ~5M-drawer + ~1M-vector corpus   │
│ tool: brain_add/search/get          │        │ injected into the LLM (tier-gate)│
│ the agent's "personal notebook"     │        │ anti-hallucination antibody loop │
└──────────────┬──────────────────────┘        └──────────────┬───────────────────┘
               │  FEDERATION (GATED promotion):                 │  call("brain.shared.search")
               │  only mem_type experience/eureka/fact,         │  (agent → router /api/brain)
               │  confidence≥0.7, SECRETS (constitution/        │
               │  secret/kill-switch) IN SEPARATE TABLES →      │
               │  NEVER promoted                                │
               └───────────────────────►  Router Brain  ───────┘
                                              │ ed25519 gossip (every ~10s, fanout 3)
                                              ▼
        ┌──────────────── MESH P2P (mDNS LAN + rendezvous WAN) ────────────────┐
        │ ed25519 identity (pubkey=address) · signed packets                   │
        │ RECEIVE: rate-limit → ≤1MB → Verify sig → HopCount → dedup           │
        │ 9-LAYER FILTER (knowledge): L1 sig·L2 freshness(anti-replay)·L3 karma │
        │   ≥0.2·L4 quarantine·L5 PII·L6 prompt-injection·L7 cosine·L8         │
        │   consensus·L9 promote                                               │
        │ peer KARMA (starts at 0.5, daily decay→0.5, <0.2 rejected)           │
        │ POLICY: a cost_usd budget fence per-scope (cron sweep)               │
        └──────────────────────────────────────────────────────────────────────┘
```
**Analogy:** the agent brain = each person's private notebook; the router brain = the office library;
the mesh = offices photocopying each other's good notes — **but secrets never leave the drawer**.

---

## 6. LAYERED SECURITY (cross-cutting)

```
1. Capability deny-by-default  — a module asks for each permission by name; not granted = can't (WASM cage)
2. SandboxRunV3                — host-protection → approval → interceptor → cap-gate → rate-limit
3. Guardian + Kernel Freeze    — SHA256 manifest of the core; tamper → SAFE-MODE; root → OS-immutable
4. Threat Radar                — 116 scanners guard the code the agent writes/runs
5. Anti-ban (router)           — cloak + identity + auto-refresh + proxy pools
6. Mesh 9-layer filter + karma — anti-poisoning of incoming knowledge
7. Secrets kept separate       — kill-switch/heir/DMS in constitution/secrets tables, NEVER
                                 indexed into the brain / pushed to git / injected into the LLM
8. Educational errors          — the agent is guided, not punished (anti-hallucination by design)
```

---

## 7. PORTABILITY (one codebase → many bodies)

```
DESKTOP/USB-OS (Linux·Mac·Windows)  : start.sh / start.bat / Start-Flowork.command → :1987 + :2402
                                      data in $HOME (~/.flowork, ~/.flow_router) — clone-able
USB APPLIANCE                       : bare-metal boot (Alpine+kiosk), dm-verity→A/B→LUKS
ANDROID (⏳)                         : a 24/7 node in your pocket; OS power-control is differentiated
                                      (needs root), process-runtime apps await the wasm runtime
AUTO-UPDATE                         : start.sh/start.bat → git fast-forward pull + rebuild
                                      (or the signed-binary Release path — dormant, ready to wire)
```
Pure-Go, zero cgo → clean cross-compile. An agent is a folder: brain+skills+doctrine travel with it.

---

## 8. ONE REAL EXAMPLE (touching almost every feature)

> *"The owner sends a voice note to Telegram: 'review the trending repo, write a promo article, share to X'."*

```
1. Connections(§29)  Telegram channel receives the voice → router Media STT(§16) → text
2. loket → mr-flow(§38) orchestrator → delegates to the GROUP "repo-reviewer + promo-x"(§35)
3. repo-reviewer agent thinks → llm.complete → ROUTER:
     apiKeyMiddleware(§1) → combo "smart-cheap"(§5) → subscription Claude provider(§4)
     → Router BRAIN injects doctrine+knowledge(§20) → CLOAK(§9) → ProxyPool(§26) → answer
4. agent runs tools(§38) via SandboxRunV3: web_research → brain_search(local brain) →
     file_write(workspace) → promo app (1-state-2-driver §37)
5. promo-x agent posts to X (token from Settings→API Keys §34/§22, anti-ban §9)
6. on success → Agent BRAIN records(§38) → federation → Router BRAIN → MESH gossip(§23)
7. reply → router Media TTS(§16) → Telegram → owner. Everything logged in Code Progress(§32) +
     Usage/Quota(§6/§7); Threat Radar(§28) guards the code; Guardian(§27) guards the kernel.
```

One message, one loop — **Router + Agent + Brain + Mesh working as a single organism.**

---

## 9. SELF-EVOLUTION & AUTONOMY (R6–R9, 2026-06 — "an organism that lives & evolves")

The endgame: Flowork is a self-evolving organism. These layers (built on the spine above) make it
understand itself, improve itself, and stay alive — gated so it never breaks itself.

- **Orchestrator note (R3):** the single orchestrator is **`mr-flow-next`** (Telegram via
  `telegram-channel` → mr-flow-next; HTTP/CLI `/api/chat` → mr-flow-next). Legacy `mr-flow` is
  retired AS orchestrator but still alive as a primary worker (hosts scanner/diagnostics/codescan).

- **R6 Self-map (Codemap + meaning):** deterministic structure (`/api/codemap/reindex` → file
  nodes + import edges) + a SEMANTIC layer (`/api/codemap/enrich` → LLM summary/domain/role per
  file, stored in `codemap_semantic`). The organism reads its own map before changing itself.
  ⚠️ "Zombie Files" = a WEAK heuristic (Go imports are package-level) — advisory only, NEVER auto-delete.

- **Self-awareness feed:** the `InvokeAgentMessage` choke-point records interaction + decision +
  karma (+ mistakes on real failure, workspace catalog) for EVERY agent → fuels the karma gate +
  makes the Diagnostics menus meaningful across all agents.

- **R7 Self-evolution (the autonomy loop):** reflect (`/api/evolve/reflect` → read self-map → LLM
  proposes additive improvements → backlog `evolve_proposal` + karma). Execution (phase-2b) =
  sandbox (git worktree) → test-gate (build+vet+test + guard: no LOCKED, no delete) → commit/STAGE/
  auto-rollback. **Multi-layer gate** (`EvolveAutoCommitAllowed`): GUI switch (OFF/STAGE/AUTO,
  default OFF) AND karma mature (≥20 success, ≥90% ratio) AND model passes the capability FLOOR
  (compile+run eval — guards against weak/local LLM) AND test-gate passes. Token runs out → local
  LLM → auto-commit auto-blocked. Capability bar is by RESULTS (karma + test-gate), not model name.

- **2 EDITIONS (`FLOWORK_EDITION`, default `public`):** resolves auto-update vs self-evolution.
  - **DEV** (owner): full evolution (core Go/JS + behavior). Owner curates → pushes upstream.
  - **PUBLIC** (default, safe): self-evolution only on the BEHAVIOR layer (agents/skills/apps in
    `~/.flowork`, outside git → no auto-update collision); core auto-updates from the canonical repo.
  Same engine for everyone (auto-update); each instance grows its own colony (behavior).

- **R9 Self-heal:** `os/selfheal/watchdog.sh` (systemd `flowork-docktor`, Restart=always) watches
  router :2402 + agent :1987, restarts on crash (anti-flap). Layer: systemd → watchdog → stack.
  + provider failover (R5) + Guardian (kernel integrity → SAFE-MODE on tamper).

- **Local LLM = opt-in (#10):** NOT auto-started at boot (heavy; most users use cloud/API). Start
  via the router GUI (Models tab) or `FLOWORK_LOCALAI_AUTOSTART=1`. Provider seed is cloud-first.

> Reference: the AGI→ASI map (DeepMind) names three recursive-self-improvement modes — genotypic
> (code), memetic (data/skills), sociogenic (specialization/colony). Flowork implements all three:
> dev-edition core-evolution, public-edition behavior-evolution, and the ant-colony Groups.

---

*Source: the user/operator tutorial §1–§38 (release audit 2026-06).*
