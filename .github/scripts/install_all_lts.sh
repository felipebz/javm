#!/usr/bin/env bash

for version in 8 11 17 21 25; do
    javm ls-remote ${version} --distribution all | tail -n +2 | while IFS= read -r line
    do
        set -- $line
        id="$1"
        [ -n "$id" ] && javm install "$id" --quiet
    done
done
