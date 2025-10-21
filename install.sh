#!/usr/bin/env bash
set -euo pipefail

REPO="felipebz/javm"
MODE="${1:-latest}"

OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="x86_64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

FILENAME="javm-${OS}-${ARCH}.tar.gz"

if [[ "$OS" == "linux" || "$OS" == "darwin" ]]; then
  INSTALL_DIR="$HOME/.local/bin"
else
  echo "Unsupported OS: $OS" >&2
  exit 1
fi
mkdir -p "$INSTALL_DIR"

SAFE_TMP="$HOME/.cache/javm-tmp"
mkdir -p "$SAFE_TMP"
TMPDIR="$(mktemp -d "$SAFE_TMP/install-XXXXXX")"

cleanup() {
  rm -rf "$TMPDIR"
}
trap cleanup EXIT

echo "Installing javm [$MODE] for $OS/$ARCH → $INSTALL_DIR"

download_artifact() {
  if ! command -v gh >/dev/null; then
    echo "GitHub CLI (gh) is required for nightly install"
    exit 1
  fi
  echo "Downloading latest nightly artifact..."
  if ! gh run download \
      --repo "$REPO" \
      --name "javm-${OS}-${ARCH}" \
      --dir "$TMPDIR"; then
    echo "Failed to download nightly artifact javm-${OS}-${ARCH}" >&2
    exit 1
  fi
}

download_release() {
  local tag="$1"
  local url="https://github.com/$REPO/releases/download/$tag/$FILENAME"
  echo "Downloading $FILENAME from release $tag..."
  if ! curl -fsSL -o "$TMPDIR/$FILENAME" "$url"; then
    echo "Failed to download release $tag" >&2
    exit 1
  fi
}

case "$MODE" in
  nightly)
    download_artifact
    ;;
  latest)
    # Get the latest release tag
    TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
      | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)
    if [[ -z "$TAG" || "$TAG" == "null" ]]; then
      echo "Could not determine latest release tag" >&2
      exit 1
    fi
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

echo "Extracting..."
tar -xzf "$TMPDIR/$FILENAME" -C "$INSTALL_DIR"

# Check PATH
if ! command -v javm >/dev/null; then
  echo "   $INSTALL_DIR is not in your PATH."
  echo "   Add this line to your shell config:"
  echo "   export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "✅ javm installed successfully."
