#!/usr/bin/env bash
# flow_router launcher. Builds the binary on first run, then serves on
# 127.0.0.1:2402 (override with FLOW_ROUTER_PORT). Auto-detects Go from
# common install dirs because .desktop launchers run with a minimal PATH.

set -e

ROUTER_ROOT="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"
PORT="${FLOW_ROUTER_PORT:-2402}"
BIN="$ROUTER_ROOT/flow-router-bin"

cd "$ROUTER_ROOT"

# Owner-local env (gitignored) — per-machine tuning (GPU offload, MoE/KV-cache) that must
# NOT ship in the portable repo. Sourced if present, no-op otherwise (zero default change).
# E.g. FLOWORK_NGL / FLOWORK_CPU_MOE / FLOWORK_KV_TYPE for local-model speed.
[ -f "$ROUTER_ROOT/flowork.local.env" ] && set -a && . "$ROUTER_ROOT/flowork.local.env" && set +a

# .desktop launchers run with a minimal PATH (no shell rc files loaded), so
# `go` from a non-standard install dir won't be found even though it works in
# the terminal. Probe common Go install locations and prepend the first one
# that contains a working `go` binary.
ensure_go_in_path() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi
  local candidates=(
    "/usr/local/go/bin"
    "/usr/lib/go-1.25/bin"
    "/usr/lib/go-1.24/bin"
    "/usr/lib/go-1.23/bin"
    "/usr/lib/go/bin"
    "/opt/go/bin"
    "$HOME/go-sdk/bin"
    "$HOME/.local/go/bin"
    "$HOME/sdk/go1.25.0/bin"
    "$HOME/.asdf/shims"
    "$HOME/go/bin"
  )
  local d
  for d in "${candidates[@]}"; do
    if [ -x "$d/go" ]; then
      export PATH="$d:$PATH"
      return 0
    fi
  done
  # Final attempt: find /home/* /usr -maxdepth 5 (cheap brute force)
  local found
  found=$(command -v -- find >/dev/null 2>&1 && find "$HOME" -maxdepth 5 -type f -name go -executable 2>/dev/null | head -1)
  if [ -n "$found" ]; then
    export PATH="$(dirname "$found"):$PATH"
    return 0
  fi
  return 1
}

# Auto-build the binary on first run (or when any .go is newer than the bin).
# Mirrors start.bat's behavior so the .desktop launcher works out of the box.
need_build=false
if [ ! -x "$BIN" ]; then
  need_build=true
elif [ -n "$(find . -name '*.go' -newer "$BIN" -print -quit 2>/dev/null)" ]; then
  need_build=true
fi
if [ "$need_build" = true ]; then
  if ! ensure_go_in_path; then
    echo "ERROR: 'go' tidak ditemukan."
    echo
    echo "Lokasi yang sudah dicek:"
    echo "  /usr/local/go/bin · /usr/lib/go-1.25/bin · /opt/go/bin"
    echo "  \$HOME/go-sdk/bin · \$HOME/.local/go/bin · \$HOME/sdk/go1.25.0/bin"
    echo "  \$HOME/.asdf/shims · \$HOME/go/bin"
    echo
    echo "Jika Go di lokasi lain, jalankan via terminal: bash $0"
    echo "(.desktop launcher tidak baca .bashrc/.zshrc — PATH terbatas)"
    echo
    echo "Atau install Go 1.25+ dari https://go.dev/dl"
    read -r -p "Press Enter to close…"
    exit 1
  fi
  echo "flow_router: building binary using $(command -v go)…"
  if ! go build -o "$BIN" . ; then
    echo "ERROR: go build gagal. Cek output di atas."
    read -r -p "Press Enter to close…"
    exit 1
  fi
fi

# Credentials are OPTIONAL — the router runs without them; subscription auth
# only kicks in if the file exists.
CREDS="${HOME}/.claude/.credentials.json"
if [ -f "$CREDS" ]; then
  echo "Credentials: $CREDS (mode $(stat -c '%a' "$CREDS"))"
fi

# Heavy brain assets (flowork-brain.sqlite, *.gguf) live inside the project
# root under brain/ and models/ — both git-ignored. Override ONLY the brain
# path so the main data.sqlite stays at the canonical ~/.flow_router/db/
# location (preserves provider config, OAuth tokens, etc.).
if [ -f "$ROUTER_ROOT/brain/flowork-brain.sqlite" ] && [ -z "$FLOW_ROUTER_BRAIN_DB" ]; then
  export FLOW_ROUTER_BRAIN_DB="$ROUTER_ROOT/brain/flowork-brain.sqlite"
  echo "Brain: $FLOW_ROUTER_BRAIN_DB ($(du -h "$FLOW_ROUTER_BRAIN_DB" 2>/dev/null | cut -f1))"
fi

echo "flow_router starting on http://127.0.0.1:$PORT"
exec "$BIN" --addr "127.0.0.1:$PORT"
