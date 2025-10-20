javm() {
    local javm_executable="::JAVM::"
    local fd3
    fd3=$(mktemp)

    "$javm_executable" --fd3 "$fd3" "$@"
    local rc=$?

    if [ -s "$fd3" ]; then
        while IFS=$'\t' read -r op key val; do
            [ "$op" = "SET" ] && export "$key=$val"
            [ "$op" = "UNSET" ] && unset "$key"
        done < "$fd3"
    fi

    rm -f "$fd3"
    return $rc
}
