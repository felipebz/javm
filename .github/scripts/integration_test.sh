#!/usr/bin/env bash
# Intended to be *sourced* after: eval "$(javm init bash|zsh)"
# Example:
#   export PATH="$HOME/.local/bin:$PATH"
#   eval "$(javm init bash)"
#   . .github/scripts/javm_integration_common.sh
#
# Optional: export JAVA_VERSION=21

set -euo pipefail

JAVA_VERSION="${JAVA_VERSION:-25}"

echo ">>> Exercising discovery/ls-remote"
javm ls-remote 21
javm ls --details

echo ">>> Installing JDK ${JAVA_VERSION} (idempotent)"
javm install "${JAVA_VERSION}"

# deactivate must restore this value. Clear JAVA_HOME_BEFORE_JAVM so a value
# left by an earlier use cannot take its place.
ORIGINAL_JAVA_HOME="/javm-ci/original-java-home"
export JAVA_HOME="${ORIGINAL_JAVA_HOME}"
unset JAVA_HOME_BEFORE_JAVM

echo ">>> Using JDK ${JAVA_VERSION}"
javm use "${JAVA_VERSION}"

echo ">>> Validations"
echo "JAVA_HOME=${JAVA_HOME:-<unset>}"
test -n "${JAVA_HOME:-}"
test -x "${JAVA_HOME}/bin/java"

# Check java resolution on PATH
if command -v java >/dev/null 2>&1; then
  JAVA_PATH="$(command -v java)"
else
  # fallback for environments where 'command -v' doesn't exist
  JAVA_PATH="$(which java)"
fi
echo "java on PATH: ${JAVA_PATH}"
java --version

# Check internal mapping
EXPECT_HOME="$(javm which "${JAVA_VERSION}" --home)"
if [ "${JAVA_HOME}" != "${EXPECT_HOME}" ]; then
  echo "ERROR: JAVA_HOME (${JAVA_HOME}) != javm which (${EXPECT_HOME})"
  exit 1
fi

# Ensure java on PATH points to current JAVA_HOME
case "${JAVA_PATH}" in
  "${JAVA_HOME}"/*) echo "OK: java on PATH matches JAVA_HOME" ;;
  *) echo "ERROR: java on PATH (${JAVA_PATH}) does not match JAVA_HOME (${JAVA_HOME})" ; exit 1 ;;
esac

if [ "${JAVA_HOME_BEFORE_JAVM:-}" != "${ORIGINAL_JAVA_HOME}" ]; then
  echo "ERROR: JAVA_HOME_BEFORE_JAVM (${JAVA_HOME_BEFORE_JAVM:-<unset>}) != ${ORIGINAL_JAVA_HOME}"
  exit 1
fi

echo ">>> Deactivating JDK"
javm deactivate
if [ "${JAVA_HOME:-}" != "${ORIGINAL_JAVA_HOME}" ]; then
  echo "ERROR: deactivate left JAVA_HOME=${JAVA_HOME:-<unset>}, want ${ORIGINAL_JAVA_HOME}"
  exit 1
fi
if [ -n "${JAVA_HOME_BEFORE_JAVM+set}" ]; then
  echo "ERROR: JAVA_HOME_BEFORE_JAVM is still set after deactivate"
  exit 1
fi

echo ">>> Done."
