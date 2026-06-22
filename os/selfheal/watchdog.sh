#!/usr/bin/env bash
# watchdog.sh — R9 SELF-HEAL (owner-approved 2026-06-15, FASE 2 autonomi).
# Pantau service INTI Flowork (router :2402, agent :1987) → restart yang MATI via
# start.sh komponen (idempotent). Anti-flap (cooldown per-service). Native di repo →
# ikut auto-update. Gantiin flowork-docktor lama (binary external Flowork_agent ilang).
#
# Layer self-heal: systemd (Restart=always) → JAGA watchdog ini → watchdog JAGA stack.
#   - watchdog mati  → systemd hidupin lagi.
#   - router/agent mati → watchdog hidupin lagi.
#   - llama (:8088) di-manage ROUTER sendiri (localai autostart) → cukup jaga router.
#
# ENV: FLOWORK_NO_WATCHDOG=1 (matiin) · FLOWORK_WATCHDOG_INTERVAL (def 30s) ·
#      FLOWORK_WATCHDOG_COOLDOWN (def 120s) · FLOWORK_WATCHDOG_LOG (def /tmp/flowork-watchdog.log)

set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"   # os/selfheal/ → FLowork_os root
LOG="${FLOWORK_WATCHDOG_LOG:-/tmp/flowork-watchdog.log}"
INTERVAL="${FLOWORK_WATCHDOG_INTERVAL:-30}"
COOLDOWN="${FLOWORK_WATCHDOG_COOLDOWN:-120}"

if [ "${FLOWORK_NO_WATCHDOG:-0}" = "1" ]; then
  echo "watchdog disabled (FLOWORK_NO_WATCHDOG=1)"; exit 0
fi

log() { echo "$(date '+%F %T') $*" >> "$LOG"; }

port_up() {
  if command -v ss >/dev/null 2>&1; then
    ss -ltn 2>/dev/null | grep -q "127.0.0.1:$1 "
  else
    (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null && { exec 3>&-; return 0; } || return 1
  fi
}

declare -A LAST_RESTART
heal() { # $1=name $2=port $3=dir(start.sh)
  local name="$1" port="$2" dir="$3" now last
  port_up "$port" && return 0
  now=$(date +%s); last="${LAST_RESTART[$name]:-0}"
  if [ $((now - last)) -lt "$COOLDOWN" ]; then
    log "[$name] :$port DOWN tapi masih cooldown ($((COOLDOWN-(now-last)))s) — skip"
    return 0
  fi
  if [ ! -x "$dir/start.sh" ]; then
    log "[$name] start.sh gak ada di $dir — skip"; return 0
  fi
  log "[$name] :$port DOWN → restart via $dir/start.sh"
  LAST_RESTART[$name]=$now
  ( cd "$dir" && FLOWORK_NO_UPDATE=1 setsid ./start.sh >>"$LOG" 2>&1 </dev/null & )
}

# B3 IMUN self-evolution: organ independen ini (survive walau Flowork mati) jaga juga dari
# commit LETAL. Source helper rollback (kalau ada). Otak = ROUTER (:2402) wajib ikut sehat.
ROLLBACK_HELPER="$ROOT/os/selfheal/evolve-rollback.sh"
# shellcheck source=/dev/null
[ -f "$ROLLBACK_HELPER" ] && . "$ROLLBACK_HELPER"
ROLLBACK_THRESHOLD="${FLOWORK_ROLLBACK_THRESHOLD:-4}"

log "watchdog START (root=$ROOT interval=${INTERVAL}s cooldown=${COOLDOWN}s pid=$$)"
trap 'log "watchdog STOP (pid=$$)"; exit 0' TERM INT

UNHEALTHY=0
while true; do
  heal router 2402 "$ROOT/router"
  heal agent  1987 "$ROOT/agent"
  # Sehat = OTAK (router) + agent dua-duanya up → catat last-known-good + reset streak.
  if port_up 2402 && port_up 1987; then
    UNHEALTHY=0
    command -v evolve_record_good >/dev/null 2>&1 && evolve_record_good "$ROOT"
  else
    UNHEALTHY=$((UNHEALTHY+1))
    # Gagal sehat BERULANG + build router/agent BENERAN rusak → commit letal → revert last-good.
    if command -v evolve_rollback_if_needed >/dev/null 2>&1 \
       && evolve_rollback_if_needed "$ROOT" "$UNHEALTHY" "$ROLLBACK_THRESHOLD"; then
      log "[evolve] 🧬 ROLLBACK ke last-known-good — commit baru ngerusak build (router/agent). Penyebab → $(evolve_rollback_log)"
      UNHEALTHY=0
      LAST_RESTART[agent]=0; LAST_RESTART[router]=0   # cabut cooldown → heal rebuild source yg udah di-revert
    fi
  fi
  # Keamanan browser (owner 2026-06-23): bersihin chromium Flowork YATIM — match profil
  # `flowork-browser-profile` (BUKAN Chrome owner; beda profil) + parent mati (PPID=1 =
  # agent crash mid-tugas). Anti zombie numpuk. Idle-30mnt ditangani reaper IN-AGENT
  # (dia tau last-used + cuma nutup instance go-rod-nya). Ini backstop kalau agent crash.
  for bpid in $(pgrep -f 'user-data-dir=[^ ]*flowork-browser-profile' 2>/dev/null); do
    bppid=$(ps -o ppid= -p "$bpid" 2>/dev/null | tr -d ' ')
    [ "$bppid" = "1" ] && kill "$bpid" 2>/dev/null && log "reaped orphan flowork-browser pid=$bpid"
  done
  sleep "$INTERVAL"
done
