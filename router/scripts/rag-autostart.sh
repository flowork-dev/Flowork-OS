#!/usr/bin/env bash
# ============================================================================
# rag-autostart.sh — bring the semantic-RAG index up automatically on launch.
#
# Wired into start.sh so "click start" => RAG + embed come up on their own.
#
# SAFE & IDEMPOTENT — the whole point:
#   * Index already built (.vindex + .DONE present) => does NOTHING and exits.
#     A full re-embed is HOURS, so a normal boot must never trigger one.
#   * Build already running => does nothing (no double-run).
#   * No brain db yet => nothing to embed, exits cleanly.
#   * Otherwise (fresh install / interrupted) => builds/resumes in this process.
#     start.sh launches it DETACHED, so the build runs in the background while
#     the router lazy-loads brain.vindex the moment it appears (no restart).
#
# Pipeline (resumable — brain-reembed keeps last_rowid, picks up where it left):
#   1) brain-reembed   : live drawer -> bge-m3 vector (via Ollama) -> v2 sqlite
#   2) brain-buildindex : v2 -> brain.vindex
#   3) drop v2 intermediate, touch .DONE
#
# Binaries are BUILT from router/cmd/* on first run if missing (plug-and-play —
# the brain dir is per-user data and ships no binaries). Needs Go for that path.
#
# Paths are derived from this script's own location (portable, no hardcode).
# Env knobs (all optional):
#   FLOWORK_NO_RAG=1        -> skip entirely
#   FLOWORK_OLLAMA_URL=...  -> embedding backend (default http://127.0.0.1:11434)
#   FLOWORK_EMBED_MODEL=... -> embedding model  (default bge-m3)
# Flags:
#   --check  -> print the decision (what it WOULD do) and exit; never embeds.
# ============================================================================
set -uo pipefail

SELF="$(cd "$(dirname "$0")" && pwd)"   # router/scripts
ROUTER="$(cd "$SELF/.." && pwd)"        # router
BRAIN="$ROUTER/brain"
RAG="$BRAIN/_rag"
BIN="$RAG/bin"
SRC="$BRAIN/flowork-brain.sqlite"       # embedding source (per-user data)
V2="$RAG/flowork-brain-vec-v2.sqlite"   # vector intermediate (dropped when done)
IDX="$BRAIN/brain.vindex"               # final index the router lazy-loads
DONE="$RAG/rag-pipeline.DONE"
OLLAMA="${FLOWORK_OLLAMA_URL:-http://127.0.0.1:11434}"
EMBED_MODEL="${FLOWORK_EMBED_MODEL:-bge-m3}"

CHECK=0; [ "${1:-}" = "--check" ] && CHECK=1

ts()  { date '+%Y-%m-%d %H:%M:%S'; }
say() { printf '[%s] [rag-autostart] %s\n' "$(ts)" "$*"; }

# ── decision gates (cheap stat checks — order matters) ──────────────────────
if [ "${FLOWORK_NO_RAG:-0}" = "1" ]; then
  say "FLOWORK_NO_RAG=1 -> skip"; exit 0
fi
if [ -s "$IDX" ] && [ -f "$DONE" ]; then
  say "index ready ($IDX) -> semantic search live, nothing to do"; exit 0
fi
if pgrep -f "$BIN/reembed" >/dev/null 2>&1 || pgrep -f "$BIN/buildindex" >/dev/null 2>&1; then
  say "build already in progress -> skip"; exit 0
fi
if [ ! -s "$SRC" ]; then
  say "no brain db at $SRC -> nothing to embed yet, skip"; exit 0
fi
if [ "$CHECK" = "1" ]; then
  say "WOULD build index: $SRC -> v2 -> $IDX (reembed: $( [ -x "$BIN/reembed" ] && echo present || echo will-build ), buildindex: $( [ -x "$BIN/buildindex" ] && echo present || echo will-build ))"
  exit 0
fi

mkdir -p "$BIN"

# ── ensure binaries (build from tracked sources on first run) ───────────────
ensure_bin() {  # <output-name> <cmd-dir-relative-to-router>
  local name="$1" cmddir="$2"
  [ -x "$BIN/$name" ] && return 0
  if ! command -v go >/dev/null 2>&1; then
    say "ERROR: $name missing and Go not installed -> cannot build RAG tooling"; return 1
  fi
  say "building $name from router/$cmddir (first run) ..."
  ( cd "$ROUTER" && go build -o "$BIN/$name" "./$cmddir" ) \
    || { say "ERROR: build $name failed"; return 1; }
}
ensure_bin reembed    cmd/brain-reembed    || exit 1
ensure_bin buildindex cmd/brain-buildindex || exit 1

# ── 1) re-embed (resumable retry loop) ──────────────────────────────────────
say "=== PIPELINE START (out -> $BRAIN) ==="
tries=0
until "$BIN/reembed" -brain "$SRC" -out "$V2" -ollama "$OLLAMA" -model "$EMBED_MODEL" -batch 128 -conc 6; do
  tries=$((tries+1))
  say "re-embed interrupted (#$tries) -> retry in 30s (resumable from last_rowid)"
  sleep 30
  [ "$tries" -gt 200 ] && { say "FATAL: re-embed gave up after $tries tries"; exit 1; }
done
say "re-embed done"

# ── 2) build index ──────────────────────────────────────────────────────────
"$BIN/buildindex" -vec "$V2" -out "$IDX" || { say "FATAL: build index failed"; exit 1; }
say "index built: $IDX"

# ── 3) cleanup intermediate + mark done ─────────────────────────────────────
rm -f "$V2" "$V2"-shm "$V2"-wal
rm -rf /tmp/tvproto
touch "$DONE"
say "=== COMPLETE: index ready at $IDX ==="
