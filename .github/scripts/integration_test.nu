#!/usr/bin/env nu

# Example:
#   javm init nu | save -f javm.nu
#   source javm.nu
#   nu .github/scripts/integration_test.nu

def main [java_version: string = "25"] {
    echo ">>> Exercising discovery/ls-remote"
    javm ls-remote 21
    javm ls --details

    echo (">>> Installing JDK " + $java_version + " (idempotent)")
    javm install $java_version

    # deactivate must restore this value. Clear JAVA_HOME_BEFORE_JAVM so a value
    # left by an earlier use cannot take its place.
    let original_java_home = "/javm-ci/original-java-home"
    $env.JAVA_HOME = $original_java_home
    hide-env --ignore-errors JAVA_HOME_BEFORE_JAVM
    hide-env --ignore-errors JAVM_ACTIVE_JAVA_HOME
    let original_path = (path_string)

    echo (">>> Using JDK " + $java_version)
    javm use $java_version

    echo ">>> Validations"
    let java_home = ($env.JAVA_HOME? | default "<unset>")
    echo (">>> JAVA_HOME=" + $java_home)

    if ($java_home == "<unset>") {
        error make {msg: "JAVA_HOME is unset"}
    }

    let java_bin = ([$java_home, "bin", "java"] | path join)
    if not ($java_bin | path exists) {
        error make {msg: (">>> java binary not found at " + $java_bin)}
    }

    # Check java resolution on PATH
    let java_path = (which java | get path | get 0)
    echo (">>> java on PATH: " + $java_path)
    ^java --version

    # Check internal mapping
    let expect_home = (javm which $java_version --home | str trim)
    if ($java_home != $expect_home) {
        error make {msg: (">>> ERROR: JAVA_HOME " + $java_home + " != javm which " + $expect_home)}
    }

    # Ensure java on PATH points to current JAVA_HOME
    if not ($java_path | str starts-with $java_home) {
        error make {msg: (">>> ERROR: java on PATH " + $java_path + " does not match JAVA_HOME " + $java_home)}
    }

    let before = ($env.JAVA_HOME_BEFORE_JAVM? | default "<unset>")
    if ($before != $original_java_home) {
        error make {msg: (">>> ERROR: JAVA_HOME_BEFORE_JAVM " + $before + " != " + $original_java_home)}
    }

    echo ">>> Deactivating JDK"
    javm deactivate
    let restored = ($env.JAVA_HOME? | default "<unset>")
    if ($restored != $original_java_home) {
        error make {msg: (">>> ERROR: deactivate left JAVA_HOME " + $restored + ", want " + $original_java_home)}
    }
    if ("JAVA_HOME_BEFORE_JAVM" in $env) {
        error make {msg: ">>> ERROR: JAVA_HOME_BEFORE_JAVM is still set after deactivate"}
    }
    let restored_path = (path_string)
    if ($restored_path != $original_path) {
        error make {msg: (">>> ERROR: deactivate left PATH " + $restored_path + ", want " + $original_path)}
    }
    if ("JAVM_ACTIVE_JAVA_HOME" in $env) {
        error make {msg: ">>> ERROR: JAVM_ACTIVE_JAVA_HOME is still set after deactivate"}
    }

    echo ">>> Done."
}

# PATH is a list after startup but a string once the wrapper sets it; compare
# the value child processes see.
def --env path_string [] {
    let value = $env.PATH
    if ($value | describe | str starts-with "list") {
        $value | str join (char esep)
    } else {
        $value
    }
}
