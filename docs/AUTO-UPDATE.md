# Auto-update model — new code & content in, user data untouched

> Owner's ask: *"update repo → mereka otomatis update; tambah agen → otomatis muncul di menu
> group; TAPI database user jangan ke-rusak, dan database baru bisa masuk tanpa merusak yang lama."*

The split: **CODE + bundled CONTENT** are replaced/added by an update; **user DATA is sacred** and
never overwritten. Three independent layers, each idempotent and additive.

## 1. Schema (DB structure) — additive migrations
Forward-only, idempotent `ALTER ADD COLUMN`/`CREATE TABLE IF NOT EXISTS`. Old DB + new code → new
columns/tables appear with defaults, every existing row intact. Proven by
`agent/internal/floworkdb/migration_zeroloss_test.go`. See [MIGRATIONS.md](MIGRATIONS.md).

## 2. Bundled content (schedules, agents, group config) — seed-new-by-id (P1.5)
A later release can ship a NEW item (e.g. a new schedule rule, a new agent) that must appear on an
already-running machine **without** clobbering the user's edits or resurrecting what they deleted.

- **Schedules / group config** (`seed/social.seed.json`): `seedSocialDefaults()` (agent `main.go`)
  installs each rule whose id has **never been seeded before**, tracked in the kv ledger
  `seeded_schedule_ids`. Fresh install → all installed. Update → only genuinely-new ids installed.
  User edited a rule → left as-is (never upserted over). User deleted a rule → ledger remembers it,
  so it is **not** re-added. Proven by `agent/seed_update_test.go` (`TestSeedNewByIdAdditive`).
- **Agents** (`agents/*.fwagent`): the kernel loader scans `AgentsDir` at boot, so a new `.fwagent`
  delivered by an update auto-appears in the panel + group menu — no extra wiring. Per-agent runtime
  DBs (`workspace/*.db`, `loket.db`) are DATA and are never replaced.
- Seeds carry **no secrets** — tokens stay in Settings → API Keys (`flowork.db`, never shipped).

## 3. Delivery (getting the new bits onto the device) — per target
- **Flowork OS (USB)**: `flowork-update {install|switch|status}` — A/B slot swap, signed root hash
  (EC-P256/SHA-256) verified before activate, rollback-safe; the **DATA partition is never touched**.
  Already shipped (see [os/](../os/)).
- **Desktop**: replace the binary + `agents/` from a GitHub Release; on next boot layers 1 & 2 run
  automatically. (Self-updater UX lands with the P4 launcher.)
- **Android**: store / sideload APK update; the bundled Go binaries + `agents/` ride inside the APK,
  layers 1 & 2 run on next launch. (P2.)

**Invariant:** an update may ADD schema, ADD content, and SWAP code — it must never DROP a column,
overwrite a user-edited row, resurrect a user-deleted row, or replace the DATA store.

## 4. Auto-update vs self-evolution — the 2-edition model (2026-06)

Auto-update (pull canonical code) and self-evolution (the organism rewrites its own code) conflict on
the same git substrate: a local self-commit makes `git pull --ff-only` diverge. Since the repo is
open-source (every instance would evolve differently), we split into **2 editions** (`FLOWORK_EDITION`,
default `public`):

- **DEV** (owner instance): full self-evolution — core (Go/JS) + behavior. The owner is the R&D lab;
  curated improvements are pushed upstream. Auto-update is inert (the dev tree is always ahead).
- **PUBLIC** (default, safe): self-evolution touches ONLY the **behavior layer** (agents / skills / apps
  in `~/.flowork` — outside git, never overwritten by `git pull`; seed is if-absent). The **core
  auto-updates** from the canonical repo. Core-change proposals are logged, never auto-applied → no git
  divergence → auto-update always fast-forwards.

Result: one shared engine (auto-updated for everyone) + N unique behavior-colonies (each instance grows
its own). Same DNA, different individuals. See `docs/ARCHITECTURE.md` §9 for the full self-evolution gate.
