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

    echo (">>> Using JDK " + $java_version)
    javm use $java_version

    echo ">>> Validations"
    let java_home = ($env | get -i JAVA_HOME | default "<unset>")
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

    echo ">>> Done."
}
