#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/javm-installer-test-XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

FIXTURE_DIR="$TEST_ROOT/fixture"
mkdir -p "$FIXTURE_DIR"

OS="$(uname | tr '[:upper:]' '[:lower:]')"
case "$(uname -m)" in
  x86_64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported test architecture" >&2; exit 1 ;;
esac

FILENAME="javm-${OS}-${ARCH}.tar.gz"
mkdir -p "$FIXTURE_DIR/archive"
printf '#!/usr/bin/env sh\necho test\n' > "$FIXTURE_DIR/archive/javm"
chmod +x "$FIXTURE_DIR/archive/javm"
tar -czf "$FIXTURE_DIR/$FILENAME" -C "$FIXTURE_DIR/archive" javm

if command -v sha256sum >/dev/null 2>&1; then
  HASH="$(sha256sum "$FIXTURE_DIR/$FILENAME" | awk '{print $1}')"
else
  HASH="$(shasum -a 256 "$FIXTURE_DIR/$FILENAME" | awk '{print $1}')"
fi
printf '%s  %s\n' "$HASH" "$FILENAME" > "$FIXTURE_DIR/javm_1.2.3_checksums.txt"

curl() {
  local output=""
  local url=""
  while (($#)); do
    case "$1" in
      -o) output="$2"; shift 2 ;;
      http*) url="$1"; shift ;;
      *) shift ;;
    esac
  done
  if [[ -z "$output" ]]; then
    printf '  "tag_name": "v1.2.3"\n'
  elif [[ "$url" == *_checksums.txt ]]; then
    cp "$MOCK_FIXTURE_DIR/javm_1.2.3_checksums.txt" "$output"
  else
    cp "$MOCK_FIXTURE_DIR/$MOCK_FILENAME" "$output"
  fi
}

gh() {
  printf '%s\n' "$*" >> "$MOCK_GH_LOG"
  return "${MOCK_GH_EXIT:-0}"
}
export -f curl gh

run_installer() {
  local home_dir="$1"
  shift
  mkdir -p "$home_dir"
  env \
    HOME="$home_dir" \
    XDG_CACHE_HOME="$home_dir/cache" \
    XDG_DATA_HOME="$home_dir/data" \
    MOCK_FIXTURE_DIR="$FIXTURE_DIR" \
    MOCK_FILENAME="$FILENAME" \
    MOCK_GH_LOG="$TEST_ROOT/gh.log" \
    "$@" \
    bash "$ROOT_DIR/install.sh" latest
}

run_installer "$TEST_ROOT/default" JAVM_VERIFY_ATTESTATION=0 >/dev/null
if [[ -e "$TEST_ROOT/gh.log" ]]; then
  echo "default installation invoked gh" >&2
  exit 1
fi

run_installer "$TEST_ROOT/verified" JAVM_VERIFY_ATTESTATION=1 >/dev/null
grep -Fq "attestation verify --repo felipebz/javm" "$TEST_ROOT/gh.log"
if grep -Fq ".github/workflows/" "$TEST_ROOT/gh.log"; then
  echo "installer exposed an internal workflow path" >&2
  exit 1
fi

if run_installer "$TEST_ROOT/rejected" JAVM_VERIFY_ATTESTATION=1 MOCK_GH_EXIT=1 >/dev/null 2>&1; then
  echo "installation succeeded after attestation verification failure" >&2
  exit 1
fi
if [[ -e "$TEST_ROOT/rejected/.local/bin/javm" ]]; then
  echo "binary was installed after attestation verification failure" >&2
  exit 1
fi

if run_installer "$TEST_ROOT/invalid" JAVM_VERIFY_ATTESTATION=yes >/dev/null 2>&1; then
  echo "installation accepted an invalid JAVM_VERIFY_ATTESTATION value" >&2
  exit 1
fi

echo "Installer policy tests passed."
