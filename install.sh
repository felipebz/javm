#!/usr/bin/env bash
set -euo pipefail

REPO="felipebz/javm"
MODE="${1:-latest}"

OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

FILENAME="javm-${OS}-${ARCH}.tar.gz"

if [[ "$OS" == "linux" || "$OS" == "darwin" ]]; then
  INSTALL_DIR="$HOME/.local/bin"
else
  echo "Unsupported OS: $OS" >&2
  exit 1
fi
mkdir -p "$INSTALL_DIR"

SAFE_TMP="${XDG_CACHE_HOME:-$HOME/.cache}/javm-tmp"
mkdir -p "$SAFE_TMP"
TMPROOT="$(mktemp -d "$SAFE_TMP/install-XXXXXX")"
DOWNLOAD_DIR="$TMPROOT/download"
EXTRACT_DIR="$TMPROOT/extract"

mkdir -p "$DOWNLOAD_DIR" "$EXTRACT_DIR"

TARBALL_PATH="$DOWNLOAD_DIR/$FILENAME"

cleanup() {
  rm -rf "$TMPROOT"
}
trap cleanup EXIT

echo "Installing javm [$MODE] for $OS/$ARCH → $INSTALL_DIR"

download_nightly() {
  if ! command -v gh >/dev/null 2>&1; then
    echo "GitHub CLI (gh) is required for nightly install" >&2
    exit 1
  fi

  echo "Downloading latest nightly artifact..."
  if ! gh run download \
        --repo "$REPO" \
        --name "javm-${OS}-${ARCH}" \
        --dir "$DOWNLOAD_DIR" >/dev/null 2>&1; then
    echo "Failed to download nightly artifact javm-${OS}-${ARCH}" >&2
    exit 1
  fi

  # garantir que o arquivo esperado existe
  if [[ ! -f "$TARBALL_PATH" ]]; then
    echo "Nightly artifact not found at $TARBALL_PATH" >&2
    exit 1
  fi

  RELEASE_TAG="nightly"
  CHECKSUM_PATH="" # nightly ainda não obriga checksum
}

get_latest_tag() {
  curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n1
}

download_release() {
  local tag="$1"
  local version_no_v="${tag#v}"

  local tar_url="https://github.com/$REPO/releases/download/$tag/$FILENAME"
  local checksum_filename="javm_${version_no_v}_checksums.txt"
  local checksum_url="https://github.com/$REPO/releases/download/$tag/$checksum_filename"

  echo "Downloading $FILENAME from release $tag..."
  if ! curl -fsSL -o "$TARBALL_PATH" "$tar_url"; then
    echo "Failed to download release $tag" >&2
    exit 1
  fi

  echo "Downloading checksum file..."
  if ! curl -fsSL -o "$DOWNLOAD_DIR/$checksum_filename" "$checksum_url"; then
    echo "Failed to download checksum file for $tag" >&2
    exit 1
  fi

  RELEASE_TAG="$tag"
  CHECKSUM_PATH="$DOWNLOAD_DIR/$checksum_filename"
}

calc_sha256() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    echo "No sha256sum or shasum found on this system." >&2
    exit 1
  fi
}

verify_checksum() {
  local tarball="$1"
  local checksum_file="$2"
  local expected_name="$3"

  if [[ -z "$checksum_file" || ! -f "$checksum_file" ]]; then
    echo "Checksum file not found: $checksum_file" >&2
    exit 1
  fi

  local line
  line="$(grep -E "[[:xdigit:]]+[[:space:]]+$expected_name\$" "$checksum_file" || true)"
  if [[ -z "$line" ]]; then
    echo "Could not find checksum entry for $expected_name in $(basename "$checksum_file")" >&2
    exit 1
  fi

  local expected_hash
  expected_hash="$(echo "$line" | awk '{print $1}' | tr 'A-F' 'a-f')"

  local actual_hash
  actual_hash="$(calc_sha256 "$tarball" | tr 'A-F' 'a-f')"

  if [[ "$actual_hash" != "$expected_hash" ]]; then
    echo "Checksum mismatch! expected $expected_hash got $actual_hash" >&2
    exit 1
  fi

  echo "Checksum OK."
}

extract_to_temp() {
  tar -xzf "$TARBALL_PATH" -C "$EXTRACT_DIR"
}

verify_attestation() {
  if ! command -v gh >/dev/null 2>&1; then
    echo "Skipping attestation verification (GitHub CLI not found)."
    return
  fi

  local exe_path
  if [[ -f "$EXTRACT_DIR/javm" ]]; then
    exe_path="$EXTRACT_DIR/javm"
  else
    exe_path="$(find "$EXTRACT_DIR" -type f -name javm | head -n1 || true)"
  fi

  if [[ -z "$exe_path" || ! -f "$exe_path" ]]; then
    echo "javm binary not found after extract; cannot verify attestation. Archive layout may have changed." >&2
    exit 1
  fi

  echo "Verifying attestation and provenance..."
  if ! gh attestation verify --repo $REPO "$exe_path" >/dev/null; then
    echo "Attestation verification failed." >&2
    exit 1
  fi

  echo "Attestation OK."
}

install_to_final() {
  echo "Installing to $INSTALL_DIR ..."
  cp -R "$EXTRACT_DIR"/. "$INSTALL_DIR"/
}

# --- fluxo principal ---

case "$MODE" in
  nightly)
    download_nightly
    extract_to_temp
    ;;
  latest)
    TAG="$(get_latest_tag)"
    if [[ -z "$TAG" || "$TAG" == "null" ]]; then
      echo "Could not determine latest release tag" >&2
      exit 1
    fi
    download_release "$TAG"
    verify_checksum "$TARBALL_PATH" "$CHECKSUM_PATH" "$FILENAME"
    extract_to_temp
    verify_attestation
    ;;
  v*|[0-9]*)
    TAG="$MODE"
    download_release "$TAG"
    verify_checksum "$TARBALL_PATH" "$CHECKSUM_PATH" "$FILENAME"
    extract_to_temp
    verify_attestation
    ;;
  *)
    echo "Usage: $0 [nightly|latest|<version>]" >&2
    exit 1
    ;;
esac

install_to_final

if ! command -v javm >/dev/null 2>&1; then
  echo ""
  echo "⚠ $INSTALL_DIR is not in your PATH."
  echo "   Add this line to your shell config (e.g. ~/.bashrc, ~/.zshrc, ~/.config/fish/config.fish):"
  echo "   export PATH=\"\$PATH:$INSTALL_DIR\""
  echo ""
fi

echo "✅ javm installed successfully."
