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
javm discover list --details

echo ">>> Installing JDK ${JAVA_VERSION} (idempotent)"
javm install "${JAVA_VERSION}"

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

echo ">>> Done."
