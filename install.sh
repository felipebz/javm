#!/usr/bin/env bash
set -euo pipefail

REPO="felipebz/javm"
INSTALL_DIR="$HOME/.javm"
MODE="${1:-latest}"  # default to "latest"
VERSION="${2:-}"     # optional second arg

OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="x86_64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *) echo "Unsupported arch: $ARCH" && exit 1 ;;
esac

FILENAME="javm-${OS}-${ARCH}.zip"
TARGET_PATH="$INSTALL_DIR/$FILENAME"

mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

echo "Installing javm [$MODE] for $OS/$ARCH..."

download_artifact() {
  if ! command -v gh >/dev/null; then
    echo "GitHub CLI (gh) is required for nightly install"
    exit 1
  fi
  echo "Downloading latest nightly artifact..."
  gh run download --repo "$REPO" --name "javm-${OS}-${ARCH}" --dir .
}

download_release() {
  local tag="$1"
  local url="https://github.com/$REPO/releases/download/$tag/$FILENAME"
  echo "Downloading $FILENAME from release $tag..."
  curl -sSL -o "$FILENAME" "$url"
}

extract() {
  echo "Extracting..."
  unzip -o "$FILENAME"
  echo "✅ Installed to $INSTALL_DIR"
}

case "$MODE" in
  nightly)
    download_artifact
    ;;
  latest)
    # Get the latest release tag
    TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | jq -r .tag_name)
    download_release "$TAG"
    ;;
  v* | [0-9]*)
    # Specific version passed
    download_release "$MODE"
    ;;
  *)
    echo "Usage: $0 [nightly|latest|<version>]" >&2
    exit 1
    ;;
esac

extract
