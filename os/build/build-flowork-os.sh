#!/usr/bin/env bash
# === LOCKED FILE === STABLE — DO NOT MODIFY without owner approval (Aola Sahidin / Mr.Dev).
# Locked 2026-06-13 after the consistent sqlite .backup seed + version stamp was verified (boot rich).
# 2026-06-13 (audit→fix): the state-seed bake now re-snapshots NESTED agent DBs + strips all
#   *.db-wal/-shm — the tar only excluded the MAIN flowork.db's WAL, so a nested stale WAL could make
#   SQLite revert the seed on first boot. Mirrors the portable bake fix.
# build-flowork-os.sh — assemble the Flowork OS appliance image (P0 PoC).
#
#   Output (out/):
#     flowork-os-<ver>.vmlinuz        kernel (Alpine linux-lts)
#     flowork-os-<ver>.initramfs.gz   FULL ephemeral rootfs (primary boot artifact)
#     flowork-os-<ver>.rootfs.squashfs read-only rootfs (USB-shaped artifact, P1 base)
#     flowork-os-<ver>.iso            grub + squashfs bootable ISO (best-effort)
#     flowork-os-<ver>.manifest.txt   sizes + sha256 + build inputs
#
# Host needs: docker, go, mksquashfs (squashfs-tools), gzip, cpio. ISO step also
# wants grub-mkrescue + xorriso (optional; skipped with a warning if absent).
# No root/apk needed: the Alpine userland is built inside docker.
set -euo pipefail

# --- locations ---------------------------------------------------------------
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SELF_DIR/.." && pwd)"
OUT="$REPO_DIR/out"
WORK="$REPO_DIR/.work"
CACHE="$REPO_DIR/.cache"
OVERLAY="$REPO_DIR/rootfs-overlay"
VERSION="$(cat "$REPO_DIR/VERSION" 2>/dev/null || echo 0.1.0-p0)"
TAG="flowork-os-$VERSION"

log()  { printf '\033[1;36m[build]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m[fail]\033[0m %s\n' "$*" >&2; exit 1; }

# --- source resolution (env > sibling > clone) -------------------------------
GH_MONOREPO="https://github.com/flowork-os/Flowork-OS"
resolve_src() {
	# $1 = override var value, $2.. = candidate local paths, last token = monorepo subdir (agent|router)
	local override="$1"; shift
	local sub="${!#}"; set -- "${@:1:$(($#-1))}"
	if [ -n "$override" ]; then echo "$override"; return; fi
	# Require a buildable Go module (go.mod) so a bare/profile dir won't match.
	local c
	for c in "$@"; do [ -f "$c/go.mod" ] && { echo "$c"; return; }; done
	mkdir -p "$CACHE"
	if [ ! -d "$CACHE/Flowork-OS/.git" ]; then
		log "cloning $GH_MONOREPO (no local source found)"
		git clone --depth 1 "$GH_MONOREPO" "$CACHE/Flowork-OS" >/dev/null 2>&1 || die "clone Flowork-OS failed"
	fi
	echo "$CACHE/Flowork-OS/$sub"
}

# One repo: Flowork-OS. Use the local monorepo sibling dir if present, else clone the monorepo.
AGENT_SRC="$(resolve_src "${AGENT_SRC:-}"  "$REPO_DIR/../agent"  agent)"
ROUTER_SRC="$(resolve_src "${ROUTER_SRC:-}" "$REPO_DIR/../router" router)"
log "agent  source : $AGENT_SRC"
log "router source : $ROUTER_SRC"

rm -rf "$WORK"; mkdir -p "$WORK" "$OUT"
ROOTFS="$WORK/rootfs"

# --- STEP 1: static binaries (native linux/amd64) ----------------------------
log "STEP 1/5  building static binaries (CGO disabled)"
build_static() { # $1 src dir, $2 out name
	# GOWORK=off: build each module standalone via its own go.mod. The router is built from a
	# COPY (seed-swapped) that lives under the monorepo but isn't listed in the dev go.work,
	# which would otherwise fail with "module not one of the workspace modules".
	( cd "$1" && GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -tags netgo -ldflags '-s -w' -o "$OUT/$2" . ) || die "go build $2 failed"
	file "$OUT/$2" | grep -q 'statically linked' || die "$2 is not static"
	log "  built $2 ($(du -h "$OUT/$2" | cut -f1), static)"
}
build_static "$AGENT_SRC"  flowork-agent
# fw-app-adapter — core_entry app hasil-adopt (repo→app). Sub-package './cmd/fw-app-adapter', jadi
# build langsung (build_static cuma build '.'). Wajib sebelah flowork-agent di rootfs (adapterBinPath).
( cd "$AGENT_SRC" && GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -tags netgo -ldflags '-s -w' -o "$OUT/fw-app-adapter" ./cmd/fw-app-adapter ) || die "go build fw-app-adapter failed"
file "$OUT/fw-app-adapter" | grep -q 'statically linked' || die "fw-app-adapter is not static"

# Router: embed the SOVEREIGN seed (local Ollama default) if available, so a fresh appliance
# routes LLM calls to the local Qwen model (//go:embed is compile-time → swap the seed in a
# source COPY, keeping the canonical repo untouched).
SOV_SEED="${FLOWORK_SOVEREIGN_SEED:-}"
[ -n "$SOV_SEED" ] || for c in \
	"$REPO_DIR/../seed-sovereign/appliance/router-config.sovereign.seed.json"; do
	[ -f "$c" ] && { SOV_SEED="$c"; break; }
done
if [ -f "$SOV_SEED" ] && [ -f "$ROUTER_SRC/seed/router-config.seed.json" ]; then
	RSRC="$WORK/router-src"; rm -rf "$RSRC"; cp -a "$ROUTER_SRC" "$RSRC"
	# drop giant runtime DATA dirs — NOT compile inputs (router only embeds seed/web/i18n).
	# Copying brain (~33G) + models (~14G) into scratch was bloating os/.work ~47G PER BUILD.
	rm -rf "$RSRC/.git" "$RSRC/brain" "$RSRC/models" "$RSRC/out" "$RSRC/bin"; rm -f "$RSRC"/*.db 2>/dev/null || true
	cp "$SOV_SEED" "$RSRC/seed/router-config.seed.json"
	build_static "$RSRC" flowork-router
	log "  router built with SOVEREIGN seed (first run routes to local Ollama)"
else
	build_static "$ROUTER_SRC" flowork-router
	warn "  sovereign seed not found — router uses its canonical seed (may prefer cloud)"
fi

# --- STEP 2: rootfs via docker ----------------------------------------------
log "STEP 2/5  building Alpine rootfs (docker)"
docker build -q -t "$TAG-rootfs" -f "$SELF_DIR/Dockerfile.rootfs" "$SELF_DIR" >/dev/null
cid="$(docker create "$TAG-rootfs")"
mkdir -p "$ROOTFS"
docker export "$cid" | tar -x --no-same-owner -C "$ROOTFS"
docker rm "$cid" >/dev/null
# docker export strips /dev nodes and a couple of pseudo mounts; recreate minimally.
mkdir -p "$ROOTFS/dev" "$ROOTFS/proc" "$ROOTFS/sys" "$ROOTFS/run" "$ROOTFS/tmp" "$ROOTFS/root"

# --- STEP 3: overlay + binaries + agent seed ---------------------------------
log "STEP 3/5  applying overlay, binaries, agent definitions"
cp -a "$OVERLAY/." "$ROOTFS/"
install -Dm0755 "$OUT/flowork-agent"  "$ROOTFS/usr/local/bin/flowork-agent"
install -Dm0755 "$OUT/fw-app-adapter" "$ROOTFS/usr/local/bin/fw-app-adapter"
install -Dm0755 "$OUT/flowork-router" "$ROOTFS/usr/local/bin/flowork-router"
chmod 0755 "$ROOTFS"/etc/init.d/flowork-* "$ROOTFS"/usr/local/bin/flowork-* 2>/dev/null || true

# Seed agent definitions as a read-only TEMPLATE. The flowork-data service copies these
# into the live state dir (DATA volume or tmpfs) at boot — the live /root/.flowork is a
# mount point, so seeding it directly would be masked.
if [ -d "$AGENT_SRC/agents" ]; then
	mkdir -p "$ROOTFS/usr/share/flowork/agents-seed"
	cp -a "$AGENT_SRC/agents/." "$ROOTFS/usr/share/flowork/agents-seed/"
	log "  seeded $(find "$ROOTFS/usr/share/flowork/agents-seed" -maxdepth 1 -name '*.fwagent' | wc -l) agent definitions (template)"
fi

# DEV-builder only: bake the owner's live state (flowork.db + config) so a DEV stick boots
# ready-to-use. Set FLOWORK_STATE_SEED=/path/to/.flowork (e.g. $HOME/.flowork). PUBLIC builds
# leave this unset → no DB ships → clean stick. flowork-data-setup seeds it on first boot.
if [ -n "${FLOWORK_STATE_SEED:-}" ] && [ -d "$FLOWORK_STATE_SEED" ] && [ -f "$FLOWORK_STATE_SEED/flowork.db" ]; then
	SS="$ROOTFS/usr/share/flowork/state-seed"; mkdir -p "$SS"
	# FULL profile: bake the owner's ENTIRE live state (flowork.db = settings+tokens, agents,
	# connectors, config) so the stick boots ready-to-use with no re-setup. Skip log/temp/cache
	# noise. flowork-data-setup seeds this into /root/.flowork on first boot only.
	tar -C "$FLOWORK_STATE_SEED" -cf - \
		--exclude='*.log' --exclude='*.tmp' --exclude='dropbox/installed' \
		--exclude='dropbox/failed' --exclude='*.sock' \
		--exclude='./flowork.db' --exclude='./flowork.db-wal' --exclude='./flowork.db-shm' \
		. 2>/dev/null | tar -C "$SS" -xf - 2>/dev/null \
		|| cp -a "$FLOWORK_STATE_SEED/." "$SS/"
	# Consistent single-file DB snapshot: a hot cp/tar of a LIVE SQLite DB can miss the WAL
	# (uncommitted recent changes) or produce a torn copy — that is how a "Full" build shipped with
	# EMPTY settings/agents. `.backup` reads through the WAL and writes one clean, complete file.
	if command -v sqlite3 >/dev/null 2>&1 \
		&& sqlite3 "$FLOWORK_STATE_SEED/flowork.db" ".timeout 5000" ".backup '$SS/flowork.db'" 2>/dev/null; then
		log "  state DB snapshotted via sqlite3 .backup (WAL merged, consistent)"
	else
		cp -a "$FLOWORK_STATE_SEED/flowork.db" "$SS/flowork.db" 2>/dev/null || true
		cp -a "$FLOWORK_STATE_SEED/flowork.db-wal" "$SS/" 2>/dev/null || true
		cp -a "$FLOWORK_STATE_SEED/flowork.db-shm" "$SS/" 2>/dev/null || true
		warn "  sqlite3 absent — copied DB+WAL raw (recovers on first open)"
	fi
	# The tar above copied NESTED agent DBs (workspace/state.db, loket.db) together with their live
	# -wal/-shm (only the MAIN flowork.db wal/shm were excluded). Shipping those sidecars reverts the
	# seed on first open — SQLite replays the stale WAL. With sqlite3 present, re-snapshot every DB as
	# a consistent single file, then strip ALL WAL/SHM sidecars so the seed is complete single files.
	if command -v sqlite3 >/dev/null 2>&1; then
		find "$SS" -name '*.db' 2>/dev/null | while read -r dbcopy; do
			src="$FLOWORK_STATE_SEED/${dbcopy#$SS/}"
			[ -f "$src" ] && sqlite3 "$src" ".timeout 5000" ".backup '$dbcopy'" 2>/dev/null || true
		done
		find "$SS" \( -name '*.db-wal' -o -name '*.db-shm' \) -delete 2>/dev/null || true
	fi
	# Seed-version stamp (VERSION + hash of the snapshotted DB): changes whenever the owner's data
	# changes, so first-boot seeding can distinguish "a NEW owner image flashed over an old DATA
	# volume" from "a plain reboot" and restore owner state exactly once per flashed image.
	# Never ship an ARMED guardian: a fresh build's binary won't match the owner's recorded baseline,
	# which would trip SAFE-MODE (exec/install blocked) on first boot. Owner re-arms post-install.
	if [ -f "$SS/guardian/vault.json" ] && command -v python3 >/dev/null 2>&1; then
		python3 -c "import json;p='$SS/guardian/vault.json';d=json.load(open(p));d['armed']=False;json.dump(d,open(p,'w'))" 2>/dev/null || true
	fi
	SEED_HASH="$(sha256sum "$SS/flowork.db" 2>/dev/null | cut -c1-12)"
	# Total size too: the flowork.db hash alone misses per-agent state.db (brain/scanner/codemap)
	# changes, so a richer owner snapshot must still bump the stamp to trigger a first-boot re-seed.
	SEED_SZ="$(du -sb "$SS" 2>/dev/null | cut -f1)"
	echo "${VERSION}-${SEED_HASH:-nohash}-${SEED_SZ:-0}" > "$SS/.seed-version"
	warn "  FULL build: baked owner state (v${VERSION}-${SEED_HASH:-nohash}, $(du -sh "$SS" 2>/dev/null | cut -f1)) — CONTAINS SECRETS, do not distribute"
fi

# Build the Go sample app (static) for the app-sandbox demo (P3a). Python sample ships
# as source via the overlay; Go is compiled here.
if [ -d "$REPO_DIR/sample-apps/go-hello" ]; then
	mkdir -p "$ROOTFS/usr/share/flowork/sample-apps/go-hello"
	( cd "$REPO_DIR/sample-apps/go-hello" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags '-s -w' -o "$ROOTFS/usr/share/flowork/sample-apps/go-hello/go-hello" . ) \
		&& log "  built go-hello sample app (sandbox demo)"
fi

# --- P2: local AI (Ollama) — glibc sidecar + ollama (CPU runtime) ---------------------
# Ollama is a glibc/cgo binary; Alpine is musl. We lay a glibc userland at the standard
# multiarch paths (no conflict with musl's loader) so ollama + its llama.cpp runner run.
# The model itself rides on a writable disk (make-model-disk.sh), not in the squashfs.
OLLAMA_BIN="${OLLAMA_BIN:-/usr/local/bin/ollama}"
OLLAMA_LIB="${OLLAMA_LIB:-/usr/local/lib/ollama}"
if [ -x "$OLLAMA_BIN" ] && [ -d "$OLLAMA_LIB" ]; then
	log "  bundling ollama (glibc sidecar + CPU runtime)"
	docker run --rm debian:stable-slim sh -c \
		'tar ch /lib64/ld-linux-x86-64.so.2 /lib/x86_64-linux-gnu /usr/lib/x86_64-linux-gnu/libstdc++.so.6* /usr/lib/x86_64-linux-gnu/libgcc_s.so.1 2>/dev/null' \
		| tar x -C "$ROOTFS" 2>/dev/null || warn "  glibc sidecar extract failed"
	install -Dm0755 "$OLLAMA_BIN" "$ROOTFS/usr/local/bin/ollama"
	mkdir -p "$ROOTFS/usr/local/lib/ollama"
	( cd "$OLLAMA_LIB" && tar c --exclude=cuda_v12 --exclude=cuda_v13 --exclude=vulkan . ) \
		| tar x -C "$ROOTFS/usr/local/lib/ollama"
	log "  ollama bundled ($(du -shL "$ROOTFS/usr/local/lib/ollama" 2>/dev/null | cut -f1) runtime, GPU libs excluded)"
	FLOWORK_HAS_OLLAMA=1
else
	warn "  ollama not found at $OLLAMA_BIN (+lib) — building WITHOUT local AI (set OLLAMA_BIN/OLLAMA_LIB)"
	FLOWORK_HAS_OLLAMA=0
fi

# Per-build LUKS key for the DATA volume (PoC custody; NEVER committed — lives only in the
# built image). Real custody (passphrase/TPM) is P3/P4.
install -d -m 0700 "$ROOTFS/etc/flowork"
head -c 4096 /dev/urandom > "$ROOTFS/etc/flowork/data.key"
chmod 0400 "$ROOTFS/etc/flowork/data.key"

# Bake the update-signing PUBLIC key + the running version so the appliance can verify
# signed OS updates (flowork-update; design Section 8). Private signing key stays offline.
if [ -f "$REPO_DIR/config/update-pub.pem" ]; then
	install -Dm0444 "$REPO_DIR/config/update-pub.pem" "$ROOTFS/etc/flowork/update-pub.pem"
	log "  baked update-signing public key"
fi
echo "$VERSION" > "$ROOTFS/etc/flowork/version"

# Enable our services (rc-update == symlink; do it here since files arrived via overlay).
ln -sf "/etc/init.d/flowork-data" "$ROOTFS/etc/runlevels/boot/flowork-data"
SERVICES="flowork-router flowork-agent flowork-kiosk flowork-health flowork-sandbox-check flowork-autoupdate"
[ "${FLOWORK_HAS_OLLAMA:-0}" = 1 ] && SERVICES="$SERVICES flowork-ollama flowork-ollama-check"
for svc in $SERVICES; do
	ln -sf "/etc/init.d/$svc" "$ROOTFS/etc/runlevels/default/$svc"
done
log "  enabled flowork services (flowork-data in boot; agent/router/kiosk/health in default)"

# Make every file readable by the (unprivileged) build user. After docker export +
# `tar --no-same-owner` all files are owned by us, but suid binaries (e.g. util-linux
# mount, mode 4111) have no owner-read bit, which would make cpio/mksquashfs fail.
# cpio still records uid/gid 0, so this does not weaken the booted image.
chmod -R u+rwX "$ROOTFS" 2>/dev/null || true

# --- STEP 4: kernel + initramfs ----------------------------------------------
log "STEP 4/5  extracting kernel + packing initramfs"
KVER="$(ls "$ROOTFS/lib/modules" | head -1)"
[ -n "$KVER" ] || die "no kernel modules found in rootfs"
cp "$ROOTFS/boot/vmlinuz-lts" "$OUT/$TAG.vmlinuz"
log "  kernel $KVER -> $TAG.vmlinuz"
# Ephemeral boot: the whole rootfs IS the initramfs. Kernel unpacks it into a
# tmpfs root and runs /sbin/init (busybox -> OpenRC). Record owners as root (0:0)
# even though we build unprivileged.
( cd "$ROOTFS" && find . -print0 \
	| cpio --null --create --format=newc --owner=+0:+0 --quiet ) \
	| gzip -9 > "$OUT/$TAG.initramfs.gz"
log "  initramfs -> $TAG.initramfs.gz ($(du -h "$OUT/$TAG.initramfs.gz" | cut -f1))"

# --- STEP 4b: squashfs + ISO (USB-shaped, best-effort) -----------------------
log "STEP 5/5  squashfs + ISO (best-effort, USB-shaped artifact)"
if command -v mksquashfs >/dev/null; then
	mksquashfs "$ROOTFS" "$OUT/$TAG.rootfs.squashfs" \
		-noappend -comp zstd -quiet -e boot 2>/dev/null \
		&& log "  squashfs -> $TAG.rootfs.squashfs ($(du -h "$OUT/$TAG.rootfs.squashfs" | cut -f1))" \
		|| warn "  mksquashfs failed (non-fatal)"
else
	warn "  mksquashfs missing; skipping squashfs"
fi
"$SELF_DIR/make-iso.sh" "$OUT" "$TAG" "$ROOTFS" "$KVER" 2>&1 | sed 's/^/  /' \
	|| warn "  ISO step skipped/failed (non-fatal; initramfs is the gate artifact)"

# --- squashfs-root boot (product/USB shape): small stage-1 initramfs + squashroot ISO --
if [ -f "$OUT/$TAG.rootfs.squashfs" ]; then
	# dm-verity sidecars: hash tree + root hash over the squashfs (integrity-at-boot).
	# Sign the root hash too if an update key exists (chain: signed root hash -> verified root).
	if command -v veritysetup >/dev/null; then
		if [ -f "$REPO_DIR/config/update.key" ]; then
			"$SELF_DIR/make-verity.sh" "$OUT/$TAG.rootfs.squashfs" --sign 2>&1 | sed 's/^/  /' || warn "  verity failed"
		else
			"$SELF_DIR/make-verity.sh" "$OUT/$TAG.rootfs.squashfs" 2>&1 | sed 's/^/  /' || warn "  verity failed"
		fi
	else
		warn "  veritysetup missing on host; squashroot ISO will boot without integrity enforcement"
	fi
	"$SELF_DIR/make-initramfs-small.sh" "$ROOTFS" "$OUT/$TAG.initramfs-small.gz" \
		"$REPO_DIR/initramfs-src/init" 2>&1 | sed 's/^/  /' || warn "  small initramfs failed"
	"$SELF_DIR/make-iso-squash.sh" "$OUT" "$TAG" 2>&1 | sed 's/^/  /' \
		|| warn "  squashroot ISO failed (non-fatal)"
	# A/B writable disk image (two slots + active pointer) — the auto-update USB shape.
	"$SELF_DIR/make-ab-disk.sh" "$OUT" "$TAG" 2G 2>&1 | sed 's/^/  /' \
		|| warn "  A/B disk failed (non-fatal)"
	# Local-AI model disk (Ollama store) — only if ollama was bundled + the model is on the host.
	if [ "${FLOWORK_HAS_OLLAMA:-0}" = 1 ] && command -v mkfs.ext4 >/dev/null; then
		"$SELF_DIR/make-model-disk.sh" "$OUT" "$TAG" "${FLOWORK_MODEL:-qwen2.5:0.5b}" 2>&1 | sed 's/^/  /' \
			|| warn "  model disk skipped (model not in host Ollama store?)"
	fi
fi

# --- manifest ----------------------------------------------------------------
{
	echo "Flowork OS build manifest"
	echo "version    : $VERSION"
	echo "kernel     : $KVER"
	echo "agent src  : $AGENT_SRC"
	echo "router src : $ROUTER_SRC"
	echo
	( cd "$OUT" && for f in "$TAG".*; do
		[ -f "$f" ] && printf '%-34s %10s  %s\n' "$f" "$(du -h "$f" | cut -f1)" "$(sha256sum "$f" | cut -c1-16)"
	done )
} > "$OUT/$TAG.manifest.txt"

log "DONE. Artifacts in $OUT:"
cat "$OUT/$TAG.manifest.txt"
log "Boot it: build/run-qemu.sh"
