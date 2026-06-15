#!/usr/bin/env bash
# ============================================================================
# start.sh — ROOT launcher: boot the WHOLE Flowork stack with ONE command.
#
#   ./start.sh
#
# What it does:
#   1. Router  (:2402) — your sovereign LLM gateway (build-on-first-run).
#   2. Agent   (:1987) — the AI-agent engine + web control panel.
#
# The agent boots its TRIGGER + SCHEDULE engine automatically on startup, so
# once it's up, all your schedules & triggers (and the YouTube watcher, if
# connected) run on their own — nothing else to start.
#
# Both sub-launchers build their Go binary on first run (or when source is
# newer) and daemonize (PID + log files), so this script returns once the
# stack is live. Re-running is safe: an already-running service is left alone.
#
# Open the panel:  http://127.0.0.1:1987
# Stop everything: ./stop.sh
# ============================================================================
set -uo pipefail

# --pause keeps the terminal open at the end (used by start.desktop so the
# window doesn't vanish before you can read build output / errors).
PAUSE=0
[ "${1:-}" = "--pause" ] && { PAUSE=1; shift; }
trap '[ "$PAUSE" = "1" ] && { printf "\n[Enter to close] "; read -r _; }' EXIT

ROOT="$(cd "$(dirname "$0")" && pwd)"   # portable (no readlink -f → works on macOS too)

c_ok()   { printf '\e[32m%s\e[0m\n' "$*"; }
c_info() { printf '\e[36m%s\e[0m\n' "$*"; }
c_warn() { printf '\e[33m%s\e[0m\n' "$*"; }
c_err()  { printf '\e[31m%s\e[0m\n' "$*"; }

c_info "⚡ Flowork — starting the full stack…"
echo

# ── AUTO-UPDATE ──────────────────────────────────────────────────────────────
# Pull the latest from the repo so a downloaded clone keeps itself current.
# Safe: only when the working tree is CLEAN and the update is a fast-forward
# (no local commits clobbered, no merge conflicts). The sub-launchers rebuild
# automatically when the pulled source is newer than their binary.
# Opt out with FLOWORK_NO_UPDATE=1 (or work offline — fetch just fails quietly).
if [ "${FLOWORK_NO_UPDATE:-0}" != "1" ] && [ -d "$ROOT/.git" ] && command -v git >/dev/null 2>&1 \
   && git -C "$ROOT" remote get-url origin >/dev/null 2>&1; then
  c_info "→ Checking for updates…"
  if [ -n "$(git -C "$ROOT" status --porcelain 2>/dev/null)" ]; then
    c_warn "  local changes present — skipping auto-update (commit or stash to enable)"
  elif git -C "$ROOT" fetch -q origin 2>/dev/null; then
    LOCAL="$(git -C "$ROOT" rev-parse @ 2>/dev/null)"
    REMOTE="$(git -C "$ROOT" rev-parse '@{u}' 2>/dev/null)"
    if [ -n "$REMOTE" ] && [ "$LOCAL" != "$REMOTE" ] \
       && git -C "$ROOT" merge-base --is-ancestor "$LOCAL" "$REMOTE" 2>/dev/null; then
      git -C "$ROOT" pull --ff-only -q origin && c_ok "  updated → $(git -C "$ROOT" rev-parse --short @) (will rebuild)"
    else
      c_info "  already up to date"
    fi
  else
    c_info "  offline — skipping update check"
  fi
  echo
fi

# port_up PORT — true if something is already listening on 127.0.0.1:PORT.
# Makes this launcher idempotent: a service already up is left alone (no
# rebuild, no duplicate, no port-bind error). Falls back to /dev/tcp if no ss.
port_up() {
  if command -v ss >/dev/null 2>&1; then
    ss -ltn 2>/dev/null | grep -qE "[:.]$1[[:space:]]"
  else
    (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null && { exec 3>&-; return 0; } || return 1
  fi
}

# 1) Router first — the agent routes its LLM calls through it (:2402).
# router/start.sh exec's the server in the FOREGROUND (it blocks), so we launch
# it DETACHED (setsid + log file) and move on — otherwise the agent never starts.
if port_up 2402; then
  c_warn "→ Router  (:2402) already running — skip"
elif [ -x "$ROOT/router/start.sh" ]; then
  RLOG="$ROOT/router/.flowork-router.log"
  c_info "→ Router  (:2402) starting in background (builds on first run)…"
  c_info "   log → $RLOG"
  ( cd "$ROOT/router" && setsid ./start.sh >"$RLOG" 2>&1 </dev/null & )
  # brief wait so a fast start / immediate error surfaces; first-run build keeps
  # going in the background while the agent (below) builds too.
  for _ in 1 2 3 4 5 6 7 8; do port_up 2402 && break; sleep 1; done
  if port_up 2402; then c_ok "  router up"; else c_info "  router still building — tail $RLOG"; fi
else
  c_warn "→ router/start.sh not found — skipping router"
fi
echo

# 2) Agent — boots the trigger + schedule engine automatically (:1987).
if port_up 1987; then
  c_warn "→ Agent   (:1987) already running — skip"
elif [ -x "$ROOT/agent/start.sh" ]; then
  c_info "→ Agent   (:1987) — trigger & schedule engine auto-start…"
  ( cd "$ROOT/agent" && ./start.sh ) ; rc=$?
  [ "$rc" = "1" ] && { c_err "  agent start failed — check agent/ log"; exit 1; }
else
  c_err "→ agent/start.sh not found — cannot start Flowork"; exit 1
fi
echo

# 3) Self-heal — pasang/start watchdog (systemd Restart=always → jaga router+agent,
# restart yang mati). R9 autonomi. Idempotent; opt-out FLOWORK_NO_WATCHDOG=1.
if [ "${FLOWORK_NO_WATCHDOG:-0}" != "1" ] && [ -x "$ROOT/os/selfheal/install-watchdog.sh" ]; then
  if "$ROOT/os/selfheal/install-watchdog.sh" >/dev/null 2>&1; then
    c_ok "→ Self-heal watchdog armed (docktor) — stack auto-restart on crash"
  else
    c_info "→ Self-heal watchdog skipped (no systemd user session?)"
  fi
fi

echo
c_ok  "✅ Flowork is live:"
c_ok  "   Control panel →  http://127.0.0.1:1987"
c_ok  "   LLM router    →  http://127.0.0.1:2402/v1"
c_info "   Schedules & triggers run automatically inside the agent."
c_info "   Stop everything:  ./stop.sh"
