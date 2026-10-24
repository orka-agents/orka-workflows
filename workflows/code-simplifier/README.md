# Code simplifier

The demo deliberately duplicates its short normalization pipeline in `slugify`
and `unique_slugs`. This Task removes that duplication while preserving behavior.
Only `demo/textkit.sh` can be published; tests and docs are outside its write scope.

Follow [shared setup](../../README.md#shared-setup) and
[run a writing workflow](../../README.md#run-a-writing-workflow) with
`WORKFLOW=code-simplifier`. Use your fork for source and publication.

## Expected result

A small one-file refactor, usually making `unique_slugs` call `slugify`. It must
preserve ASCII normalization, filtering, order, output newlines, and isolation of
variables and locale settings from callers. Sourcing the library must still only
define functions.

The result should record a passing baseline, `sh -n demo/textkit.sh`, a passing
`sh demo/test.sh` after the edit, and focused shell checks for edge cases. The
Task has a 10-minute timeout, 24 turns, and no retries. After the duplication is
removed, a later run should leave the file unchanged if no useful simplification
remains.

## Adapt it

Pick one implementation file and one concrete simplification. Keep public behavior
and tests unchanged. Replace the prompt and checks, then set `allowedPaths` to
that file. Do not combine refactoring with feature work or dependency changes.
