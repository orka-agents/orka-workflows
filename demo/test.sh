#!/bin/sh
set -eu

. ./demo/textkit.sh

assert_output() {
    expected=$1
    shift
    actual=$("$@")
    if [ "$actual" != "$expected" ]; then
        printf 'FAIL: %s\nExpected: <%s>\nActual: <%s>\n' \
            "$*" "$expected" "$actual" >&2
        exit 1
    fi
}

assert_output 'hello-world' slugify 'Hello World'
assert_output 'orka-123' slugify 'Orka 123'
assert_output 'ship-it-now' slugify '  ship__it!!now  '
assert_output 'beta-notes
alpha
release-2' unique_slugs 'Beta Notes' 'alpha' 'BETA_notes' 'Release 2' 'Alpha'
assert_output 'hello
world' unique_slugs '' 'Hello' '---' 'World'

printf '5 tests passed\n'
