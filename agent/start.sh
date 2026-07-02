#!/usr/bin/env bash
# start.sh — bangun (kalau perlu) dan jalanin flowork-gui di background.
#
# Layout:
#   bin/flowork-gui    binary hasil build (auto re-build kalau source lebih baru)
#   $RUN_DIR/flowork-gui.pid   PID proses jalan   (RUN_DIR per-user, lihat bawah)
#   $RUN_DIR/flowork-gui.log   stdout + stderr    (+ symlink kompat /tmp/flowork-gui.log)
#
# Convention:
#   exit 0   sukses (sudah jalan atau baru di-start)
#   exit 1   build gagal / start gagal
#   exit 2   sudah jalan, ngga di-restart

set -euo pipefail

# --pause = tunggu Enter di akhir (dipakai oleh .desktop biar terminal
# ngga langsung nutup setelah script kelar).
PAUSE=0
if [ "${1:-}" = "--pause" ]; then
  PAUSE=1
  shift
fi
trap '[ "$PAUSE" = "1" ] && { printf "\n[Enter untuk tutup] "; read -r _; }' EXIT

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

BIN="$ROOT/bin/flowork-gui"
# CABUT-AKAR (2026-07-02): dulu pid/log hardcode /tmp/flowork-gui.* — path FIX yang
# dipakai bareng >1 user (mis. mrflow manual vs service user `flowork`) → file punya
# user lain + sticky /tmp = "Permission denied" → service GAGAL BOOT. Akar = shared
# fixed path; fix = runtime dir PER-USER (no-hardcode, multi-OS). Override: FLOWORK_RUN_DIR.
RUN_DIR="${FLOWORK_RUN_DIR:-${XDG_RUNTIME_DIR:-/tmp}/flowork-$(id -un)}"
mkdir -p "$RUN_DIR" 2>/dev/null || RUN_DIR="$(mktemp -d)"
PID_FILE="$RUN_DIR/flowork-gui.pid"
LOG_FILE="$RUN_DIR/flowork-gui.log"
# Kompat: `tail /tmp/flowork-gui.log` (kebiasaan QC/dok lama) tetap jalan lewat symlink.
# Best-effort: kalau path lama punya user lain (ga bisa diganti), skip diam-diam.
ln -sfn "$LOG_FILE" /tmp/flowork-gui.log 2>/dev/null || true
ln -sfn "$PID_FILE" /tmp/flowork-gui.pid 2>/dev/null || true
ADDR="${FLOWORK_ADDR:-127.0.0.1:1987}"

# .desktop launchers run dengan PATH minim — Go SDK custom (mis. di
# ~/go-sdk/bin) sering ngga ke-include. Source rc files + augment PATH
# dari lokasi standar.
for rc in "$HOME/.profile" "$HOME/.bashrc"; do
  # .bashrc biasanya guard against non-interactive — set PS1 dummy biar lewat.
  [ -f "$rc" ] && PS1='${PS1-x}' . "$rc" >/dev/null 2>&1 || true
done
for d in \
  "$HOME/go-sdk/bin" \
  /usr/local/go/bin \
  "$HOME/go/bin" \
  "$HOME/.local/go/bin" \
  /snap/bin \
  /opt/go/bin; do
  [ -d "$d" ] && case ":$PATH:" in *":$d:"*) :;; *) PATH="$d:$PATH";; esac
done
export PATH

# Warna log
c_ok()   { printf '\e[32m%s\e[0m\n' "$*"; }
c_warn() { printf '\e[33m%s\e[0m\n' "$*"; }
c_err()  { printf '\e[31m%s\e[0m\n' "$*"; }
c_info() { printf '\e[36m%s\e[0m\n' "$*"; }

# Cek sudah jalan?
if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  PID="$(cat "$PID_FILE")"
  c_warn "flowork-gui sudah jalan (PID $PID, http://$ADDR)"
  c_info "Pakai ./stop.sh atau ./restart.sh"
  exit 2
fi

# Cek port masih dipake proses orphan?
if ss -tlnp 2>/dev/null | grep -q ":${ADDR##*:} "; then
  # PID-file sekarang per-user → instance yang di-start USER LAIN ga keliatan di cek
  # atas. Kalau port udah jawab HTTP = flowork-gui udah hidup → idempoten (exit 2),
  # BUKAN failure — biar systemd/start ulang ga dianggap error.
  if curl -s -m 2 -o /dev/null "http://$ADDR/" 2>/dev/null; then
    c_warn "flowork-gui sudah jalan di http://$ADDR (instance user lain / sesi lama)"
    c_info "Pakai ./stop.sh atau ./restart.sh dari user yang sama kalau mau ganti."
    exit 2
  fi
  c_err "Port ${ADDR##*:} masih dipake proses lain (bukan HTTP flowork). Cek:"
  ss -tlnp | grep ":${ADDR##*:} " || true
  c_info "Jalankan ./stop.sh dulu, atau set FLOWORK_ADDR=127.0.0.1:PORT_LAIN"
  exit 1
fi

# Build kalau binary belum ada atau source lebih baru.
NEED_BUILD=0
if [ ! -x "$BIN" ]; then
  NEED_BUILD=1
else
  # Rebuild kalau ada .go / web/ lebih baru dari binary
  if [ -n "$(find . -newer "$BIN" \( -name '*.go' -o -path './web/*' \) -not -path './bin/*' 2>/dev/null | head -1)" ]; then
    NEED_BUILD=1
  fi
fi

if [ "$NEED_BUILD" = "1" ]; then
  if ! command -v go >/dev/null 2>&1; then
    c_err "Go compiler ngga ketemu di PATH."
    c_info "Cek install: which go || ls /usr/local/go/bin/go ~/go/bin/go 2>/dev/null"
    c_info "PATH sekarang: $PATH"
    c_info "Set GOROOT/GOPATH di ~/.profile biar .desktop launcher kebawa."
    exit 1
  fi
  c_info "Build flowork-gui ($(go version 2>&1))..."
  mkdir -p "$ROOT/bin"
  if ! go build -o "$BIN" . ; then
    c_err "Build gagal"
    exit 1
  fi
  c_ok "Build OK ($(stat -c%s "$BIN") bytes)"
fi

# ── adapter binaries (repo→app: kontrak CLI + HTTP) ────────────────────────
# core_entry app hasil-adopt. Wajib sebelah flowork-gui biar adapterBinPath() nemu (lihat
# lock/apps-adopt.md). Idempotent (rebuild kalau missing / source baru), non-fatal — gagal cuma
# matiin adopt, sisanya jalan. fw-app-adapter=CLI, fw-http-adapter=server (streamlit/fastapi/dll).
if command -v go >/dev/null 2>&1; then
  for AD in fw-app-adapter:cliadapter fw-http-adapter:httpadapter; do
    name="${AD%%:*}"; pkg="${AD##*:}"; bin="$ROOT/bin/$name"
    [ -d "$ROOT/cmd/$name" ] || continue
    if [ ! -f "$bin" ] || [ -n "$(find "$ROOT/cmd/$name" "$ROOT/internal/apps/$pkg" -name '*.go' -newer "$bin" 2>/dev/null | head -1)" ]; then
      c_info "Build $name (repo→app)…"
      if ( cd "$ROOT" && CGO_ENABLED=0 go build -o "$bin" "./cmd/$name" ); then
        c_ok "$name OK ($(stat -c%s "$bin") bytes)"
      else
        c_warn "$name build gagal — fitur adopt mati sampai ini ke-build"
      fi
    fi
  done
fi

# ── SEMUA agent + template wasm (repo→agent: konsisten dev/portable/img) ────
# agent.wasm GITIGNORED → kalau ga di-build, "dev jalan, portable/img GAGAL" (wasm ilang).
# build-all-agents.sh = SATU mekanisme (Go-standar wasip1, no tinygo) buat SEMUA agent
# (mr-flow dkk) + template. Idempotent + non-fatal. WAJIB sebelum seed agent ke ~/.flowork.
if [ -x "$ROOT/scripts/build-all-agents.sh" ]; then
  c_info "Build semua agent + template wasm…"
  "$ROOT/scripts/build-all-agents.sh" "$ROOT" || c_warn "sebagian agent wasm gagal ke-build"
fi

# Sidecar TOOLS binary (tools/<name>/<name>) — GITIGNORED, SAMA KELAS wasm: WAJIB build dulu biar
# fresh-checkout ga "tool mati" (5 builtin sidecar tool ga ke-register). Isolated per-tool (GOWORK=off).
# Acuan: os/builder/bacaduluini.md.
if [ -x "$ROOT/../tools/build-tools.sh" ]; then
  c_info "Build sidecar tools…"
  bash "$ROOT/../tools/build-tools.sh" || c_warn "sebagian sidecar tool gagal ke-build"
fi

# ── GROUP template wasm ───────────────────────────────────────────────────
# (Ditangani build-all-agents.sh di atas; blok ini fallback idempotent — skip kalau udah fresh.)
GROUP_TPL="$ROOT/templates/group-template"
GROUP_WASM="$GROUP_TPL/agent.wasm"
if [ -f "$GROUP_TPL/main.go" ] && command -v go >/dev/null 2>&1; then
  if [ ! -f "$GROUP_WASM" ] || [ -n "$(find "$GROUP_TPL" -name '*.go' -newer "$GROUP_WASM" 2>/dev/null | head -1)" ]; then
    c_info "Build group template wasm (wasip1)…"
    if ( cd "$GROUP_TPL" && GOOS=wasip1 GOARCH=wasm go build -o agent.wasm . ); then
      c_ok "group template wasm OK ($(stat -c%s "$GROUP_WASM") bytes)"
    else
      c_warn "group template wasm build gagal — Group create bakal error sampai ini ke-build"
    fi
  fi
fi

# ── AGENT template wasm ───────────────────────────────────────────────────
# coderTemplate (AI Studio) clone templates/agent-template/agent.wasm jadi agent
# baru. Wasm gitignored (build dari source) → FRESH CHECKOUT mesti build di sini,
# else AI Studio bikin agent gagal. Standard wasip1 (no tinygo). Idempotent.
AGENT_TPL="$ROOT/templates/agent-template"
AGENT_WASM="$AGENT_TPL/agent.wasm"
if [ -f "$AGENT_TPL/main.go" ] && command -v go >/dev/null 2>&1; then
  if [ ! -f "$AGENT_WASM" ] || [ -n "$(find "$AGENT_TPL" -name '*.go' -newer "$AGENT_WASM" 2>/dev/null | head -1)" ]; then
    c_info "Build agent template wasm (wasip1)…"
    if ( cd "$AGENT_TPL" && GOWORK=off GOOS=wasip1 GOARCH=wasm go build -o agent.wasm . ); then
      c_ok "agent template wasm OK ($(stat -c%s "$AGENT_WASM") bytes)"
    else
      c_warn "agent template wasm build gagal — AI Studio bikin agent bakal error sampai ini ke-build"
    fi
  fi
fi

# ── POWER ARM ────────────────────────────────────────────────────────────
# system_power tool EKSEKUSI aksi daya beneran (shutdown/reboot/suspend/lock/
# logout) kalau FLOWORK_POWER_ARMED=1 (DEFAULT). Aman walau default armed:
# satu-satunya yg punya cap `exec:power` itu agent `operator-komputer`, dan
# agent itu GA di-ship (gitignored, dibikin manual via scripts/setup-operator.sh).
# Clone repo = ga ada operator = ga ada yg bisa matiin. Mau matiin kemampuan ini?
# Disable agent operator-komputer di GUI, ATAU set FLOWORK_POWER_ARMED=0
# (mis. lewat file gitignored flowork.local.env).
[ -f "$ROOT/flowork.local.env" ] && set -a && . "$ROOT/flowork.local.env" && set +a
export FLOWORK_POWER_ARMED="${FLOWORK_POWER_ARMED:-1}"

# ── SEED AGENTS (fresh-install convenience) ──────────────────────────────────
# The repo ships every agent/group DEFINITION + built agent.wasm under ./agents,
# but the runtime reads ~/.flowork/agents (or $FLOWORK_AGENTS_DIR). On a fresh
# checkout that dir is empty, so none of the bundled agents would load. Install
# any bundled *.fwagent that isn't there yet — NEVER overwrite an existing one
# (preserves a live workspace/state.db with the owner's tokens). Idempotent: on a
# machine that already has its agents this copies nothing.
AGENTS_SRC="$ROOT/agents"
AGENTS_DST="${FLOWORK_AGENTS_DIR:-$HOME/.flowork/agents}"
if [ -d "$AGENTS_SRC" ]; then
  mkdir -p "$AGENTS_DST"
  seeded=0
  for d in "$AGENTS_SRC"/*.fwagent/; do
    [ -d "$d" ] || continue
    name="$(basename "$d")"
    if [ ! -e "$AGENTS_DST/$name" ]; then
      cp -r "$d" "$AGENTS_DST/$name" || continue
      # Strip any stray runtime state so a seeded agent starts clean (it builds its
      # own workspace/state.db on first run). A real clone has none of this; this is
      # belt-and-suspenders against a dirty working tree.
      rm -rf "$AGENTS_DST/$name/workspace" "$AGENTS_DST/$name/shared"
      find "$AGENTS_DST/$name" \( -name '*.db' -o -name '*.db-shm' -o -name '*.db-wal' \) -delete 2>/dev/null || true
      seeded=$((seeded + 1))
    fi
  done
  [ "$seeded" -gt 0 ] && c_ok "Seeded $seeded bundled agent(s) → $AGENTS_DST"
fi

# ── SEED APPS (fresh-install convenience) ────────────────────────────────────
# The repo ships bundled apps (e.g. the Shared Notepad) under ./apps, but the
# runtime reads <data-home>/apps (sibling of the agents dir, i.e. ~/.flowork/apps).
# On a fresh checkout that dir is empty → "No apps yet". Copy any bundled app that
# isn't installed yet — NEVER overwrite an existing one (preserves its state).
APPS_SRC="$ROOT/apps"
APPS_DST="$(dirname "$AGENTS_DST")/apps"
if [ -d "$APPS_SRC" ]; then
  mkdir -p "$APPS_DST"
  app_seeded=0
  for d in "$APPS_SRC"/*/; do
    [ -d "$d" ] || continue
    name="$(basename "$d")"
    if [ ! -e "$APPS_DST/$name" ]; then
      cp -r "$d" "$APPS_DST/$name" || continue
      find "$APPS_DST/$name" \( -name '*.db' -o -name '*.db-shm' -o -name '*.db-wal' -o -name 'state.json' \) -delete 2>/dev/null || true
      app_seeded=$((app_seeded + 1))
    fi
  done
  [ "$app_seeded" -gt 0 ] && c_ok "Seeded $app_seeded bundled app(s) → $APPS_DST"
fi

# Start in the background
c_info "Starting flowork-gui at http://$ADDR … (power armed=$FLOWORK_POWER_ARMED)"
nohup "$BIN" -addr "$ADDR" >"$LOG_FILE" 2>&1 &
PID=$!
echo "$PID" > "$PID_FILE"

# Tunggu sebentar + cek listen
sleep 0.7
if ! kill -0 "$PID" 2>/dev/null; then
  c_err "Proses crash saat boot. Cek log:"
  tail -n 20 "$LOG_FILE"
  rm -f "$PID_FILE"
  exit 1
fi

c_ok "flowork-gui jalan — PID $PID"
c_info "URL : http://$ADDR"
c_info "Log : $LOG_FILE"
c_info "PID : $PID_FILE"

# YouTube watcher DICABUT akar 2026-06-27 (owner: basi). Integrasi kelak via integration-registry.
