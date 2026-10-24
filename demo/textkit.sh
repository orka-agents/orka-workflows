#!/bin/sh

# Normalization is deliberately ASCII-only, independent of the caller's locale.
# shellcheck disable=SC2018,SC2019

slugify() (
    LC_ALL=C
    export LC_ALL

    slug=$(
        printf '%s' "${1-}" |
            tr 'A-Z' 'a-z' |
            tr -cs 'a-z0-9' '-'
    )
    slug=${slug#-}
    slug=${slug%-}
    printf '%s' "$slug"
)

unique_slugs() (
    LC_ALL=C
    export LC_ALL

    seen='|'
    for title do
        slug=$(
            printf '%s' "$title" |
                tr 'A-Z' 'a-z' |
                tr -cs 'a-z0-9' '-'
        )
        slug=${slug#-}
        slug=${slug%-}
        [ -n "$slug" ] || continue

        case "$seen" in
            *"|$slug|"*) continue ;;
        esac
        seen="${seen}${slug}|"
        printf '%s\n' "$slug"
    done
)
