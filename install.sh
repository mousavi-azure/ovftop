#!/usr/bin/env bash
#
# install-ovftool.sh — Installs VMware ovftool from the official Linux Zip bundle.
#
# Usage:
#   ./install-ovftool.sh /path/to/VMware-ovftool-5.1.0-25410048-lin.x86_64.zip
#   ./install-ovftool.sh                     # looks for a matching zip in the current dir
#
# Safe to re-run: replaces any previous install cleanly.

set -euo pipefail

INSTALL_DIR="/opt/ovftool"
BIN_LINK="/usr/local/bin/ovftool"

log()  { echo "[install-ovftool] $*"; }
fail() { echo "[install-ovftool] ERROR: $*" >&2; exit 1; }

# --- resolve the zip file ---------------------------------------------------
ZIP_FILE="${1:-}"
if [[ -z "$ZIP_FILE" ]]; then
    ZIP_FILE=$(find . -maxdepth 1 -iname 'VMware-ovftool-*-lin.x86_64.zip' | head -n1 || true)
fi

[[ -n "$ZIP_FILE" ]] || fail "No zip file given and none found in current directory. Usage: $0 <path-to-ovftool-zip>"
[[ -f "$ZIP_FILE" ]] || fail "File not found: $ZIP_FILE"

log "Using bundle: $ZIP_FILE"

# --- must be root ------------------------------------------------------------
if [[ $EUID -ne 0 ]]; then
    fail "This script must be run as root (use sudo)."
fi

# --- dependency check ---------------------------------------------------------
command -v unzip >/dev/null 2>&1 || fail "unzip is required but not installed. Run: apt-get install -y unzip"

# --- clean any previous install -----------------------------------------------
if [[ -d "$INSTALL_DIR" ]]; then
    log "Removing previous install at $INSTALL_DIR"
    rm -rf "$INSTALL_DIR"
fi

# --- extract -------------------------------------------------------------------
log "Extracting to $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
unzip -q "$ZIP_FILE" -d "$INSTALL_DIR"

# The zip contains a nested 'ovftool/' folder; find the actual binary
OVFTOOL_BIN=$(find "$INSTALL_DIR" -maxdepth 2 -type f -name 'ovftool' | head -n1 || true)
[[ -n "$OVFTOOL_BIN" ]] || fail "Could not locate ovftool binary after extraction — bundle layout may have changed."

chmod +x "$OVFTOOL_BIN"
[[ -f "$(dirname "$OVFTOOL_BIN")/ovftool.bin" ]] && chmod +x "$(dirname "$OVFTOOL_BIN")/ovftool.bin"

# --- symlink into PATH -----------------------------------------------------
log "Linking $BIN_LINK -> $OVFTOOL_BIN"
ln -sf "$OVFTOOL_BIN" "$BIN_LINK"

# --- verify ------------------------------------------------------------------
if ! VERSION_OUTPUT=$("$BIN_LINK" --version 2>&1); then
    fail "ovftool did not run correctly after install. Output: $VERSION_OUTPUT"
fi

log "Success: $VERSION_OUTPUT"
log "Installed at: $OVFTOOL_BIN"
log "Available on PATH as: ovftool"
