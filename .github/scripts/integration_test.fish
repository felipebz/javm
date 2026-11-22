#!/usr/bin/env fish
# Intended to be *sourced* after: javm init fish | source
# Example:
#   set -gx PATH "$HOME/.local/bin" $PATH
#   eval (javm init fish)
#   source .github/scripts/javm_integration_common.fish
#
# Optional: set -gx JAVA_VERSION 21

if not set -q JAVA_VERSION
    set -gx JAVA_VERSION 25
end

echo ">>> Exercising discovery/ls-remote"
javm ls-remote 21
javm ls --details

echo ">>> Installing JDK $JAVA_VERSION (idempotent)"
javm install $JAVA_VERSION

# deactivate must restore this value. Clear JAVA_HOME_BEFORE_JAVM so a value
# left by an earlier use cannot take its place.
set -l ORIGINAL_JAVA_HOME /javm-ci/original-java-home
set -gx JAVA_HOME $ORIGINAL_JAVA_HOME
set -e JAVA_HOME_BEFORE_JAVM
set -e JAVM_ACTIVE_JAVA_HOME
# The wrapper assigns PATH as one colon-separated string; compare what child
# processes see.
set -l ORIGINAL_PATH (string join : $PATH)

echo ">>> Using JDK $JAVA_VERSION"
javm use $JAVA_VERSION

echo ">>> Validations"
echo "JAVA_HOME="$JAVA_HOME
test -n "$JAVA_HOME"; or begin; echo "ERROR: JAVA_HOME empty"; exit 1; end
test -x "$JAVA_HOME/bin/java"; or begin; echo "ERROR: java not executable at $JAVA_HOME/bin/java"; exit 1; end

# Check java resolution in PATH
set -l JAVA_PATH (command -v java)
if test -z "$JAVA_PATH"
    echo "ERROR: java not found in PATH"
    exit 1
end
echo "java on PATH: $JAVA_PATH"
java --version

# Check internal mapping
set -l EXPECT_HOME (javm which $JAVA_VERSION --home)
if test "$JAVA_HOME" != "$EXPECT_HOME"
    echo "ERROR: JAVA_HOME ($JAVA_HOME) != javm which ($EXPECT_HOME)"
    exit 1
end

# Ensure java in PATH points to current JAVA_HOME
if string match -q "$JAVA_HOME/*" -- "$JAVA_PATH"
    echo "OK: java on PATH matches JAVA_HOME"
else
    echo "ERROR: java on PATH ($JAVA_PATH) does not match JAVA_HOME ($JAVA_HOME)"
    exit 1
end

if test "$JAVA_HOME_BEFORE_JAVM" != "$ORIGINAL_JAVA_HOME"
    echo "ERROR: JAVA_HOME_BEFORE_JAVM ($JAVA_HOME_BEFORE_JAVM) != $ORIGINAL_JAVA_HOME"
    exit 1
end

echo ">>> Deactivating JDK"
javm deactivate
if test "$JAVA_HOME" != "$ORIGINAL_JAVA_HOME"
    echo "ERROR: deactivate left JAVA_HOME=$JAVA_HOME, want $ORIGINAL_JAVA_HOME"
    exit 1
end
if set -q JAVA_HOME_BEFORE_JAVM
    echo "ERROR: JAVA_HOME_BEFORE_JAVM is still set after deactivate"
    exit 1
end
if test (string join : $PATH) != "$ORIGINAL_PATH"
    echo "ERROR: deactivate left PATH="(string join : $PATH)", want $ORIGINAL_PATH"
    exit 1
end
if set -q JAVM_ACTIVE_JAVA_HOME
    echo "ERROR: JAVM_ACTIVE_JAVA_HOME is still set after deactivate"
    exit 1
end

echo ">>> Done."
