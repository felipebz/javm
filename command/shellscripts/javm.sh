javm() {
    local javm_executable="::JAVM::"
    local fd3
    fd3=$(mktemp)

    "$javm_executable" "$@" --fd3 "$fd3"

    if [ -s "$fd3" ]; then
        . "$fd3"
    fi

    rm -f "$fd3"
}
