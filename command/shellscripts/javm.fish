function javm
    set -l javm_executable "::JAVM::"
    set -l fd3 (mktemp)
    $javm_executable --fd3 $fd3 $argv

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
end
