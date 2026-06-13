#!/usr/bin/env bash
# ============================================================================
# stop.sh — ROOT: stop the WHOLE Flowork stack (agent + router).
#   ./stop.sh
# Stops the agent first (so it stops dispatching), then the router.
# Safe to run when nothing is up — each sub-stopper is a no-op then.
# ============================================================================
set -uo pipefail

# --pause keeps the terminal open at the end (used by stop.desktop).
PAUSE=0
[ "${1:-}" = "--pause" ] && { PAUSE=1; shift; }
trap '[ "$PAUSE" = "1" ] && { printf "\n[Enter to close] "; read -r _; }' EXIT

ROOT="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"

c_ok()   { printf '\e[32m%s\e[0m\n' "$*"; }
c_info() { printf '\e[36m%s\e[0m\n' "$*"; }

c_info "⚡ Flowork — stopping…"
if [ -x "$ROOT/agent/stop.sh" ]; then
  c_info "→ Agent…";  ( cd "$ROOT/agent"  && ./stop.sh ) || true
fi
if [ -x "$ROOT/router/stop.sh" ]; then
  c_info "→ Router…"; ( cd "$ROOT/router" && ./stop.sh ) || true
fi
c_ok "✅ Flowork stopped."
