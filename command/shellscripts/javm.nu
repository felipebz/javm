def --env --wrapped javm [...args] {
    let javm_executable = "::JAVM::"

    if ($args | length) > 0 and ($args.0 == "use" or $args.0 == "deactivate") {
        let fd3 = (mktemp)
        ^$javm_executable --fd3 $fd3 ...$args
        let exit_code = $env.LAST_EXIT_CODE

        if ($fd3 | path exists) {
            let env_changes = (open $fd3 | lines | split column "\t" op key val)
            for change in $env_changes {
                if $change.op == "SET" {
                    load-env { ($change.key): $change.val }
                } else if $change.op == "UNSET" {
                    hide-env $change.key
                }
            }
            rm -f $fd3
        }
        if $exit_code != 0 {
            error make --unspanned { msg: "javm error", code: $exit_code }
        }
    } else {
        ^$javm_executable ...$args
    }
}
