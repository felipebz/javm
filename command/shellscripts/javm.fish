function javm
    set -l javm_executable "::JAVM::"

    if test (count $argv) -gt 0
        set -l cmd $argv[1]
        if test "$cmd" = use -o "$cmd" = deactivate
            set -l fd3 (mktemp)
            $javm_executable --fd3 $fd3 $argv
            set -l exit_code $status

            if test -s $fd3
                while read -l line
                    set -l parts (string split -m 2 \t -- $line)
                    set -l op $parts[1]
                    if test "$op" = SET
                        if test (count $parts) -ge 3
                            set -gx $parts[2] $parts[3]
                        end
                    else if test "$op" = UNSET
                        if test (count $parts) -ge 2
                            set -e $parts[2]
                        end
                    end
                end < $fd3
            end

            rm -f $fd3
            return $exit_code
        end
    end

    $javm_executable $argv
    return $status
end
