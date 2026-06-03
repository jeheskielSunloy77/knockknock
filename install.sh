#!/bin/sh
set -eu

REPO="jeheskielSunloy77/knockknock"
BIN="knockknock"

die() {
  echo >&2 "error: $*"
  exit 1
}

info() {
  echo "$*"
}

# --- detect OS / arch ---
OS="$(uname -s)"
ARCH="$(uname -m)"

# normalize OS name to lowercase
case "$OS" in
  Linux|linux)   OS="linux"   ;;
  Darwin|darwin) OS="darwin"  ;;
  Windows|windows) OS="windows" ;;
esac

case "$OS" in
  linux)   ;;
  darwin)  ;;
  windows) ;;
  *)       die "unsupported OS: $OS" ;;
esac

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64"  ;;
  arm64)   ;; # darwin already reports arm64
  amd64)   ;;
  *)       die "unsupported architecture: $ARCH" ;;
esac

# --- resolve version ---
if [ -n "${VERSION:-}" ]; then
  TAG="v${VERSION#v}"
else
  info "fetching latest release version ..."
  TAG="$(curl -sSfL "https://api.github.com/repos/$REPO/releases/latest")"
  TAG="$(printf '%s' "$TAG" | sed -n 's/.*"tag_name": "\(.*\)",/\1/p')"
  [ -n "$TAG" ] || die "could not determine latest release"
fi

VERSION="${TAG#v}"
ASSET="$BIN-${VERSION}-checksums.txt"
BINARY="$BIN"

# --- download directory ---
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

info "downloading knockknock $VERSION ($OS/$ARCH) ..."

BASEURL="https://github.com/$REPO/releases/download/$TAG"
BINARY_FILE="${BIN}_${VERSION}_${OS}_${ARCH}"
if [ "$OS" = "windows" ]; then
  BINARY_FILE="${BINARY_FILE}.exe"
fi

curl -sSfL -o "$TMPDIR/$BINARY_FILE" "$BASEURL/$BINARY_FILE"
curl -sSfL -o "$TMPDIR/checksums.txt" "$BASEURL/$ASSET"

# --- verify checksum ---
info "verifying checksum ..."
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$TMPDIR" && sha256sum --ignore-missing --check "checksums.txt") || die "checksum mismatch"
elif command -v shasum >/dev/null 2>&1; then
  (cd "$TMPDIR" && shasum -a 256 --ignore-missing --check "checksums.txt") || die "checksum mismatch"
else
  die "neither sha256sum nor shasum found"
fi

chmod +x "$TMPDIR/$BINARY_FILE"

# --- install ---
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

mv "$TMPDIR/$BINARY_FILE" "$INSTALL_DIR/$BIN"
info "installed $INSTALL_DIR/$BIN"

# --- verify ---
if command -v "$BIN" >/dev/null 2>&1; then
  exec "$BIN" --version
else
  info "ensure $INSTALL_DIR is in your PATH"
fi
