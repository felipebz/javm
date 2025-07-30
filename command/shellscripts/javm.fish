function javm
    set -l javm_executable "::JAVM::"
    set -l fd3 (mktemp)
    $javm_executable $argv --fd3 $fd3

    if test -s $fd3
        source $fd3
    end

    rm -f $fd3
end
