#!/usr/bin/env bash
# === LOCKED FILE === STABLE — DO NOT MODIFY without owner approval (Aola Sahidin / Mr.Dev).
# Locked 2026-06-13 after the data-seed bake + launcher first-run seeding was verified on a real stick.
# 2026-06-13 (audit→fix→test): re-seed + bake now strip stale *.db-wal/-shm — a left-over WAL made
#   SQLite REVERT a freshly-seeded DB on first open (blanked API Keys + Schedule). Verified: baked
#   seed is WAL-free (15 secrets / 6 schedules) and data persists across agent boot.
# make-portable.sh — build "Flowork Portable": run Flowork ON TOP of a running Windows / Linux /
# macOS straight from a folder (e.g. on the USB), NO reboot. Cross-compiles the Go core for every
# OS and ships TWO launch modes per OS:
#   • GUI        — opens the panel in your browser.
#   • Background — runs silently behind your work (no window/browser). Schedules + triggers keep
#                  running (they live inside the agent process, not the UI). Stop with Stop-Flowork.
# Data lives in ./flowork-data next to the launcher, so everything travels with the stick.
#
#   usage: make-portable.sh [OUT_DIR]     (default: <os>/out/flowork-portable)
set -euo pipefail
SELF="$(cd "$(dirname "$0")" && pwd)"
OS_DIR="$(cd "$SELF/.." && pwd)"
REPO="$(cd "$OS_DIR/.." && pwd)"
OUT="${1:-$OS_DIR/out/flowork-portable}"
case "$OUT" in /*) ;; *) OUT="$PWD/$OUT" ;; esac   # absolute — build() cd's around, relative breaks
rm -rf "$OUT"; mkdir -p "$OUT/bin"

build() { # $1 GOOS  $2 GOARCH  $3 subdir  $4 ext
	local goos="$1" goarch="$2" sub="$3" ext="${4:-}"
	mkdir -p "$OUT/bin/$sub"
	echo "[portable] $sub  (agent + router · $goos/$goarch)"
	( cd "$REPO/agent"  && GOWORK=off CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
		go build -ldflags '-s -w' -o "$OUT/bin/$sub/flowork-agent$ext" . )
	# fw-app-adapter — core_entry app hasil-adopt (repo→app). Wajib sebelah flowork-agent biar
	# adapterBinPath() nemu (lihat lock/apps-adopt.md). Tanpa ini, fitur adopt mati di portable.
	( cd "$REPO/agent"  && GOWORK=off CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
		go build -ldflags '-s -w' -o "$OUT/bin/$sub/fw-app-adapter$ext" ./cmd/fw-app-adapter )
	( cd "$REPO/router" && GOWORK=off CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
		go build -ldflags '-s -w' -o "$OUT/bin/$sub/flowork-router$ext" . )
}
build windows amd64 windows .exe
build linux   amd64 linux
build darwin  amd64 macos-intel
build darwin  arm64 macos-apple

# ── SIDE-CAR ASSETS ─────────────────────────────────────────────────────────
# Copy repo-local sidecar folders so the portable output can be mounted and manually
# synced into a target install without baking them into the build scratch.
# This keeps the build clean while still making `bin`, `models`, and `apps` available.
SIDECAR_ENABLED="${FLOWORK_PORTABLE_SIDECAR:-1}"
if [ "$SIDECAR_ENABLED" != "0" ]; then
    echo "[portable] copying sidecar assets (bin/models/apps)"
    mkdir -p "$OUT/sidecar"
    if [ -d "$REPO/router/bin" ]; then
        cp -a "$REPO/router/bin" "$OUT/sidecar/"
    fi
    if [ -d "$REPO/router/models" ]; then
        cp -a "$REPO/router/models" "$OUT/sidecar/"
    fi
    if [ -d "$REPO/agent/apps" ]; then
        cp -a "$REPO/agent/apps" "$OUT/sidecar/"
    fi
fi

# ── DEV/Full data-seed ───────────────────────────────────────────────────────
# A Full/dev build bakes a SLIM snapshot of the owner's live state so the portable boots
# ready-to-use (settings, tokens, agents) — the launcher seeds it into the local data dir on
# first run. PUBLIC builds leave FLOWORK_STATE_SEED unset → no data ships → clean portable.
# (Same contract as the OS image's state-seed; mirror its consistent-DB + version-stamp tricks.)
SEED_SRC="${FLOWORK_STATE_SEED:-}"
if [ -n "$SEED_SRC" ] && [ -d "$SEED_SRC" ] && [ -f "$SEED_SRC/flowork.db" ]; then
	DS="$OUT/data-seed"; rm -rf "$DS"; mkdir -p "$DS"
	# Consistent single-file DB snapshot (WAL merged) — a hot copy can ship empty/torn settings.
	if command -v sqlite3 >/dev/null 2>&1 \
		&& sqlite3 "$SEED_SRC/flowork.db" ".timeout 5000" ".backup '$DS/flowork.db'" 2>/dev/null; then :; else
		cp -f "$SEED_SRC/flowork.db" "$DS/flowork.db" 2>/dev/null || true
	fi
	# The small, essential state — skip multi-GB noise (brain, marketplace, quarantine, logs…).
	# Include `apps` (flowalpha/notepad) and `guardian` (force-disarmed just below).
	for item in agents apps connectors config.yaml skills state status guardian; do
		[ -e "$SEED_SRC/$item" ] && cp -a "$SEED_SRC/$item" "$DS/" 2>/dev/null || true
	done
	# Never ship an ARMED guardian: a fresh build's binary won't match the owner's recorded baseline,
	# which would trip SAFE-MODE (exec/install blocked) on first run — and re-seeding overwrites any
	# armed vault left in the runtime cache. The owner re-arms post-install (sudo flowork --arm).
	if [ -f "$DS/guardian/vault.json" ] && command -v python3 >/dev/null 2>&1; then
		python3 -c "import json;p='$DS/guardian/vault.json';d=json.load(open(p));d['armed']=False;json.dump(d,open(p,'w'))" 2>/dev/null || true
	fi
	# The `cp -a` above hot-copies nested agent DBs (workspace/state.db, loket.db) TOGETHER WITH
	# their live -wal/-shm. Shipping those sidecars is poison: on first run SQLite replays the stale
	# WAL and reverts the seed (the same revert that blanked Settings/Schedule). Re-snapshot every
	# nested DB as a consistent single file via .backup, then strip ALL WAL/SHM sidecars so the seed
	# is purely complete single-file DBs.
	if command -v sqlite3 >/dev/null 2>&1; then
		find "$DS" -name '*.db' 2>/dev/null | while read -r dbcopy; do
			src="$SEED_SRC/${dbcopy#$DS/}"
			[ -f "$src" ] && sqlite3 "$src" ".timeout 5000" ".backup '$dbcopy'" 2>/dev/null || true
		done
	fi
	find "$DS" \( -name '*.db-wal' -o -name '*.db-shm' \) -delete 2>/dev/null || true
	ver="$(cat "$OS_DIR/VERSION" 2>/dev/null || echo 0)"
	h="$(sha256sum "$DS/flowork.db" 2>/dev/null | cut -c1-12)"
	# Include the data-seed total size: the global flowork.db hash alone misses changes to the
	# per-agent state.db (brain/scanner/codemap), so a richer owner snapshot must still bump the
	# stamp or the launcher won't re-seed an already-seeded machine.
	sz="$(du -sb "$DS" 2>/dev/null | cut -f1)"
	echo "${ver}-${h:-nohash}-${sz:-0}" > "$DS/.seed-version"
	echo "[portable] baked owner data-seed ($(du -sh "$DS" 2>/dev/null | cut -f1), v${ver}-${h:-nohash}) — CONTAINS SECRETS, dev only"
fi

# ── UNIX launchers (Linux + macOS share one body, OS auto-detected) ──────────
# Common preamble: cd here, point data at ./flowork-data, pick the right bin dir.
unix_head() {
	cat <<'SH'
STICK="$(cd "$(dirname "$0")" && pwd)"
case "$(uname -s)" in
	Linux)  SUB=linux ;;
	Darwin) [ "$(uname -m)" = arm64 ] && SUB=macos-apple || SUB=macos-intel ;;
	*) echo "unsupported OS"; exit 1 ;;
esac
# The stick is FAT: it can't mark binaries executable (mount shows them noexec) and SQLite can't
# lock its DB there. So run from a local, exec-capable, lockable dir; the stick just delivers.
WORK="${XDG_CACHE_HOME:-$HOME/.cache}/flowork-portable"
BIND="$WORK/bin"; mkdir -p "$BIND"
cp -f "$STICK/bin/$SUB/flowork-agent" "$STICK/bin/$SUB/flowork-router" "$STICK/bin/$SUB/fw-app-adapter" "$BIND/" 2>/dev/null
chmod +x "$BIND/flowork-agent" "$BIND/flowork-router" "$BIND/fw-app-adapter" 2>/dev/null || true
export FLOWORK_HOME="$WORK/data"; export HOME="$FLOWORK_HOME"; mkdir -p "$FLOWORK_HOME"
# Guardian: detection-only auto-arm re-baselines every boot and its 5-min sentinel re-verifies
# the core files — but a freshly-built/distributed binary (and a mutating workspace) makes that a
# near-certain false-positive that drops the whole node into SAFE-MODE (exec/install blocked →
# AI Studio/Scanner/tools dead). A distributable must NOT auto-arm; the owner arms on their FINAL
# device with `sudo flowork --arm` (which OS-seals a real baseline). Off here keeps the node usable.
export FLOWORK_GUARDIAN_AUTO=0
# PID/running helpers — defined BEFORE the re-seed so we never copy a fresh DB over one the agent
# has OPEN (that corrupts it; the agent then re-creates an empty DB — exactly how Settings → API Keys
# went blank). The re-seed must only ever touch state while nothing is running.
PIDS="$FLOWORK_HOME/flowork.pids"
running() { [ -f "$PIDS" ] && kill -0 $(cat "$PIDS" 2>/dev/null) 2>/dev/null; }
# First run of a Full/dev stick: seed the baked owner state into the live data dir. The agent reads
# $HOME/.flowork (HOME=FLOWORK_HOME here), so DB+agents land in $FLOWORK_HOME/.flowork. Re-seed only
# when a NEWER stick is plugged in (stamp differs) AND nothing is running; back up the old DB first.
# Plain restarts are untouched; a public stick ships no data-seed so this is a no-op there.
if [ -d "$STICK/data-seed" ] && ! running; then
	STATE="$FLOWORK_HOME/.flowork"
	want="$(cat "$STICK/data-seed/.seed-version" 2>/dev/null || echo unknown)"
	have="$(cat "$STATE/.seed-version" 2>/dev/null || echo none)"
	if [ ! -f "$STATE/flowork.db" ]; then
		mkdir -p "$STATE"; cp -a "$STICK/data-seed/." "$STATE/" 2>/dev/null || true
		find "$STATE" \( -name '*.db-wal' -o -name '*.db-shm' \) -delete 2>/dev/null || true
		echo "Loaded your Flowork data from the stick."
	elif [ "$want" != unknown ] && [ "$want" != "$have" ]; then
		cp -f "$STATE/flowork.db" "$STATE/flowork.db.prev" 2>/dev/null || true
		cp -a "$STICK/data-seed/." "$STATE/" 2>/dev/null || true
		# SQLite footgun: the seed ships COMPLETE single-file DBs (.backup snapshots). A STALE
		# -wal/-shm left in STATE from a previous run makes SQLite REPLAY that old WAL when the agent
		# opens the freshly-seeded DB and REVERT it — exactly how Settings→API Keys + Schedule went
		# blank right after a re-seed. Drop every WAL/SHM sidecar so each DB opens as the clean seed.
		find "$STATE" \( -name '*.db-wal' -o -name '*.db-shm' \) -delete 2>/dev/null || true
		echo "Updated your Flowork data from a newer stick (old DB -> flowork.db.prev)."
	fi
fi
start_core() {
	nohup "$BIND/flowork-router" >"$FLOWORK_HOME/router.log" 2>&1 & RP=$!
	sleep 1
	nohup "$BIND/flowork-agent" -addr 127.0.0.1:1987 >"$FLOWORK_HOME/agent.log" 2>&1 & AP=$!
	echo "$RP $AP" > "$PIDS"
}
SH
}
# GUI launcher: start (if needed) + open the panel as a chromeless APP WINDOW (looks like the
# Flowork OS appliance, not a browser tab) + stay until closed.
{ echo '#!/usr/bin/env bash'; unix_head; cat <<'SH'
URL="http://127.0.0.1:1987"
# Open the panel like the OS appliance: a dedicated, chromeless app window. We pick a REAL browser
# ourselves and NEVER go through xdg-open / the system default — that default can be hijacked by a
# random app (e.g. an anti-detect browser), which is why "Start" used to wrongly open the wrong app.
# Any candidate whose path looks like an anti-detect/donut browser is skipped on purpose.
open_panel() {
	local prof="$WORK/browser"; mkdir -p "$prof"
	local b p
	# Chromium-family → real "app mode": no tabs, no address bar — it reads as the OS, not a browser.
	for b in chromium chromium-browser google-chrome google-chrome-stable brave-browser \
	         brave microsoft-edge microsoft-edge-stable vivaldi-stable vivaldi; do
		p="$(command -v "$b" 2>/dev/null)" || continue
		case "$(printf '%s' "$p" | tr 'A-Z' 'a-z')" in *donut*|*adspower*|*dolphin*|*multilogin*) continue;; esac
		"$p" --app="$URL" --class=Flowork --user-data-dir="$prof" \
		     --no-first-run --no-default-browser-check --start-maximized >/dev/null 2>&1 &
		return 0
	done
	# Firefox fallback: a fresh, dedicated window (still not the hijackable default).
	for b in firefox firefox-esr; do
		p="$(command -v "$b" 2>/dev/null)" || continue
		"$p" --new-window "$URL" >/dev/null 2>&1 & return 0
	done
	# macOS / last resort. On macOS `open -na "Google Chrome" --args --app=` gives an app window too.
	if command -v open >/dev/null 2>&1; then
		open -na "Google Chrome" --args --app="$URL" --class=Flowork >/dev/null 2>&1 && return 0
		open "$URL" >/dev/null 2>&1 && return 0
	fi
	echo "Open $URL in your browser."
}
if running; then echo "Flowork already running."; else echo "Starting Flowork..."; start_core; fi
for _ in $(seq 1 40); do curl -fsS -o /dev/null "$URL/" 2>/dev/null && break; sleep 0.4; done
open_panel
echo "Panel: $URL — it keeps running in the background even if you close this."
SH
} > "$OUT/start-flowork.sh"

# Background launcher: start silently, NO browser, return immediately. Triggers + schedules run.
{ echo '#!/usr/bin/env bash'; unix_head; cat <<'SH'
if running; then echo "Flowork already running in the background."; exit 0; fi
start_core
echo "Flowork is now running in the BACKGROUND (schedules + triggers active, no window)."
echo "Open the panel anytime: http://127.0.0.1:1987   ·   Stop: ./stop-flowork.sh"
SH
} > "$OUT/start-flowork-background.sh"

# Stop launcher.
{ echo '#!/usr/bin/env bash'; unix_head; cat <<'SH'
if running; then kill $(cat "$PIDS") 2>/dev/null; rm -f "$PIDS"; echo "Flowork stopped."; else echo "Flowork is not running."; fi
SH
} > "$OUT/stop-flowork.sh"

# Self-update (on demand): pull the latest portable bundle from the public channel + swap binaries.
{ echo '#!/usr/bin/env bash'; cat <<'SH'
cd "$(cd "$(dirname "$0")" && pwd)"
REPO="${FLOWORK_REPO:-flowork-os/Flowork-OS}"; API="${FLOWORK_GH_API:-https://api.github.com}"
url="$(curl -fsSL "$API/repos/$REPO/releases/latest" 2>/dev/null | grep -oE '"browser_download_url": *"[^"]*flowork-portable\.zip"' | sed -E 's/.*"(http[^"]*)"/\1/' | head -1)"
[ -n "$url" ] || { echo "No portable update found in the latest release."; exit 1; }
echo "Downloading the latest Flowork…"
t="$(mktemp -d)"; trap 'rm -rf "$t"' EXIT
curl -fSL "$url" -o "$t/p.zip" || { echo "download failed"; exit 1; }
( cd "$t" && unzip -oq p.zip ) || { echo "unzip failed (is 'unzip' installed?)"; exit 1; }
src="$t/flowork-portable"; [ -d "$src" ] || src="$t"
cp -rf "$src/bin/." bin/ && echo "Updated. Restart Flowork to use the new version."
SH
} > "$OUT/update-flowork.sh"

# macOS double-clickable copies (Finder runs .command).
cp "$OUT/start-flowork.sh"            "$OUT/Start-Flowork.command"
cp "$OUT/start-flowork-background.sh" "$OUT/Start-Flowork-Background.command"
cp "$OUT/stop-flowork.sh"             "$OUT/Stop-Flowork.command"
cp "$OUT/update-flowork.sh"           "$OUT/Update-Flowork.command"
chmod +x "$OUT"/*.sh "$OUT"/*.command

# Linux double-click launchers (.desktop). Self-locating via %k (the .desktop's own path) so they
# work wherever the USB mounts; run the .sh through `bash` so no +x bit is needed (FAT loses it).
gen_desktop() { # $1 file, $2 Name, $3 script, $4 icon, $5 comment
	{
		printf '[Desktop Entry]\nType=Application\nVersion=1.0\n'
		printf 'Name=%s\nComment=%s\n' "$2" "$5"
		# Exec WITHOUT nested double-quotes — GNOME's GDesktopAppInfo parser returns an empty Exec
		# ("didn't specify Exec field") if the value contains escaped \" inside. %k = this .desktop's
		# own path → cd to its folder → run the script via bash (no +x needed; FAT loses it). The
		# FLOWORK partition label has no spaces, so the unquoted %k is safe.
		printf 'Exec=bash -c "cd $(dirname %%k); exec bash %s"\n' "$3"
		printf 'Icon=%s\nTerminal=false\nCategories=Utility;\n' "$4"
	} > "$OUT/$1"
	chmod +x "$OUT/$1"
}
gen_desktop "Start-Flowork.desktop"            "Start Flowork"              start-flowork.sh            applications-internet "Run Flowork and open the panel"
gen_desktop "Start-Flowork-Background.desktop" "Start Flowork (Background)" start-flowork-background.sh system-run            "Run Flowork silently in the background"
gen_desktop "Stop-Flowork.desktop"             "Stop Flowork"               stop-flowork.sh             process-stop          "Stop the Flowork node"
gen_desktop "Update-Flowork.desktop"           "Update Flowork"             update-flowork.sh           system-software-update "Download the latest Flowork"

# Linux app-menu installer — a USB is FAT, which can't store the exec/trust bit a .desktop needs to
# be double-clicked. So on Linux you run this ONCE (`bash Flowork-Setup-Linux.sh`) to add proper
# menu entries (absolute Exec → this stick); then "Flowork — Start" appears in your app menu.
cat > "$OUT/Flowork-Setup-Linux.sh" <<'SH'
#!/usr/bin/env bash
HERE="$(cd "$(dirname "$0")" && pwd)"
APP="${XDG_DATA_HOME:-$HOME/.local/share}/applications"; mkdir -p "$APP"
mk() { cat > "$APP/$1" <<EOF
[Desktop Entry]
Type=Application
Version=1.0
Name=$2
Comment=Flowork portable (from USB)
Exec=bash "$HERE/$3"
Icon=$4
Terminal=false
Categories=Utility;
EOF
}
mk flowork-start.desktop    "Flowork — Start"               start-flowork.sh            applications-internet
mk flowork-start-bg.desktop "Flowork — Start (Background)"  start-flowork-background.sh system-run
mk flowork-stop.desktop     "Flowork — Stop"                stop-flowork.sh             process-stop
mk flowork-update.desktop   "Flowork — Update"              update-flowork.sh           system-software-update
update-desktop-database "$APP" 2>/dev/null || true
echo "Done — open your app menu (Super key) and search 'Flowork'."
SH
chmod +x "$OUT/Flowork-Setup-Linux.sh"

# ── WINDOWS launchers ────────────────────────────────────────────────────────
# GUI: starts the core (hidden) + opens the browser + waits; close window to keep running bg.
cat > "$OUT/Start-Flowork.bat" <<'BAT'
@echo off
cd /d "%~dp0"
set FLOWORK_HOME=%~dp0flowork-data
set HOME=%~dp0flowork-data
if not exist "%FLOWORK_HOME%" mkdir "%FLOWORK_HOME%"
cscript //nologo "%~dp0_flowork-bg.vbs" >nul 2>&1
echo Starting Flowork...
timeout /t 3 >nul
rem Open the panel as a chromeless APP WINDOW (looks like the OS, not a browser tab). Prefer
rem Chrome/Edge app-mode; fall back to the default only if neither is installed. We deliberately
rem do NOT just hand the URL to the default browser, which could be a hijacking app.
set "FWURL=http://127.0.0.1:1987"
set "CHROME=%ProgramFiles%\Google\Chrome\Application\chrome.exe"
if not exist "%CHROME%" set "CHROME=%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe"
set "EDGE=%ProgramFiles(x86)%\Microsoft\Edge\Application\msedge.exe"
if not exist "%EDGE%" set "EDGE=%ProgramFiles%\Microsoft\Edge\Application\msedge.exe"
if exist "%CHROME%" ( start "" "%CHROME%" --app=%FWURL% --class=Flowork --start-maximized ) ^
else if exist "%EDGE%" ( start "" "%EDGE%" --app=%FWURL% --start-maximized ) ^
else ( start "" "%FWURL%" )
echo Panel: %FWURL%  (Flowork keeps running in the background)
BAT

# Background: launch hidden via VBS, no window, no browser, return immediately.
cat > "$OUT/Start-Flowork-Background.bat" <<'BAT'
@echo off
cd /d "%~dp0"
set FLOWORK_HOME=%~dp0flowork-data
set HOME=%~dp0flowork-data
if not exist "%FLOWORK_HOME%" mkdir "%FLOWORK_HOME%"
cscript //nologo "%~dp0_flowork-bg.vbs" >nul 2>&1
echo Flowork is running in the BACKGROUND (schedules + triggers active). Panel: http://127.0.0.1:1987
BAT

# Stop.
cat > "$OUT/Stop-Flowork.bat" <<'BAT'
@echo off
taskkill /im flowork-agent.exe /f >nul 2>&1
taskkill /im flowork-router.exe /f >nul 2>&1
echo Flowork stopped.
BAT

# Self-update (Windows): download the latest portable bundle + swap binaries via PowerShell.
cat > "$OUT/Update-Flowork.bat" <<'BAT'
@echo off
cd /d "%~dp0"
echo Downloading the latest Flowork...
powershell -NoProfile -Command ^
  "$r = Invoke-RestMethod 'https://api.github.com/repos/flowork-os/Flowork-OS/releases/latest';" ^
  "$u = ($r.assets | Where-Object { $_.name -eq 'flowork-portable.zip' }).browser_download_url;" ^
  "if (-not $u) { Write-Host 'No portable update found.'; exit 1 };" ^
  "$t = Join-Path $env:TEMP 'flowork-upd'; Remove-Item $t -Recurse -Force -ErrorAction SilentlyContinue;" ^
  "Invoke-WebRequest $u -OutFile \"$env:TEMP\fwp.zip\"; Expand-Archive \"$env:TEMP\fwp.zip\" $t -Force;" ^
  "Copy-Item (Join-Path $t 'flowork-portable\bin\*') 'bin' -Recurse -Force;" ^
  "Write-Host 'Updated. Restart Flowork to use the new version.'"
pause
BAT

# Hidden launcher helper (windowStyle 0 = no window). Idempotent-ish: relies on Stop to clear.
cat > "$OUT/_flowork-bg.vbs" <<'VBS'
Set sh = CreateObject("WScript.Shell")
base = CreateObject("Scripting.FileSystemObject").GetParentFolderName(WScript.ScriptFullName)
sh.CurrentDirectory = base
sh.Run """" & base & "\bin\windows\flowork-router.exe""", 0, False
WScript.Sleep 1000
sh.Run """" & base & "\bin\windows\flowork-agent.exe"" -addr 127.0.0.1:1987", 0, False
VBS

cat > "$OUT/README.txt" <<'TXT'
Flowork Portable — run Flowork on top of your current OS, no reboot.

TWO ways to start (pick one):
  • Windows : double-click  Start-Flowork.bat   (Background: Start-Flowork-Background.bat)
  • macOS   : double-click   Start-Flowork.command   (Background: Start-Flowork-Background.command)
  • Linux   : run ONCE  ->  bash Flowork-Setup-Linux.sh
              (a FAT USB can't make a .desktop double-clickable, so this adds menu entries.)
              Then open your app menu, search "Flowork", click "Flowork — Start" / "… (Background)".

Open the panel anytime at  http://127.0.0.1:1987
Stop:    Stop-Flowork.bat   /  ./stop-flowork.sh    /  Stop-Flowork.command
Update:  Update-Flowork.bat /  ./update-flowork.sh  /  Update-Flowork.command  (gets the latest)

Your data stays in the "flowork-data" folder here, so it travels with this stick.
Default AI = Claude — paste your token in Settings. Local AI (Ollama) is optional.
TXT

echo "[portable] done -> $OUT"
du -sh "$OUT" 2>/dev/null | sed 's/^/[portable] size: /'
