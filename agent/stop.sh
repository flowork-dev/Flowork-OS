#!/usr/bin/env bash
# stop.sh — matikan flowork-gui yang lagi jalan (graceful, fallback ke KILL).
#
# Cleanup:
#   - kill PID dari PID file (TERM dulu, KILL kalau ngga merespon)
#   - sweep proses orphan dengan nama flowork-gui / flowork-agent
#     (flowork-agent legacy dari arsitektur kernel-terpisah; kita matiin juga)

set -euo pipefail

PAUSE=0
if [ "${1:-}" = "--pause" ]; then
  PAUSE=1
  shift
fi
trap '[ "$PAUSE" = "1" ] && { printf "\n[Enter untuk tutup] "; read -r _; }' EXIT

ROOT="$(cd "$(dirname "$0")" && pwd)"
# Samain sama start.sh: runtime dir per-user (cabut-akar bentrok /tmp antar-user).
RUN_DIR="${FLOWORK_RUN_DIR:-${XDG_RUNTIME_DIR:-/tmp}/flowork-$(id -un)}"
PID_FILE="$RUN_DIR/flowork-gui.pid"
# Fallback path legacy (sesi lama sebelum fix) biar stop tetep nemu prosesnya.
[ -f "$PID_FILE" ] || { [ -f /tmp/flowork-gui.pid ] && PID_FILE=/tmp/flowork-gui.pid; }

c_ok()   { printf '\e[32m%s\e[0m\n' "$*"; }
c_warn() { printf '\e[33m%s\e[0m\n' "$*"; }
c_info() { printf '\e[36m%s\e[0m\n' "$*"; }

stopped=0

# 1. PID file path (PID known)
if [ -f "$PID_FILE" ]; then
  PID="$(cat "$PID_FILE")"
  if kill -0 "$PID" 2>/dev/null; then
    c_info "Send SIGTERM ke PID $PID..."
    kill -TERM "$PID" 2>/dev/null || true
    for _ in 1 2 3 4 5; do
      sleep 0.4
      kill -0 "$PID" 2>/dev/null || break
    done
    if kill -0 "$PID" 2>/dev/null; then
      c_warn "Masih hidup, force KILL"
      kill -KILL "$PID" 2>/dev/null || true
      sleep 0.2
    fi
    stopped=1
  fi
  rm -f "$PID_FILE"
fi

# 2. Sweep orphan — proses dengan nama flowork-* yang belum ke-track.
#    pkill -u $USER biar cuma proses milik user sendiri (aman di shared box).
for name in flowork-gui flowork-agent; do
  if pgrep -u "$USER" -x "$name" >/dev/null 2>&1; then
    c_info "Sweep orphan: $name"
    pkill -u "$USER" -TERM -x "$name" 2>/dev/null || true
    sleep 0.4
    pkill -u "$USER" -KILL -x "$name" 2>/dev/null || true
    stopped=1
  fi
done

if [ "$stopped" = "1" ]; then
  c_ok "flowork-gui berhenti"
else
  c_warn "Ngga ada flowork-gui yang jalan"
fi
