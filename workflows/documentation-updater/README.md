# Documentation updater

The demo README has a working example but does not describe both functions or
their edge cases. This Task reads the implementation and tests, verifies examples,
and changes only `demo/README.md`.

Follow [shared setup](../../README.md#shared-setup) and
[run a writing workflow](../../README.md#run-a-writing-workflow) with
`WORKFLOW=documentation-updater`. Use your fork for source and publication.

## Expected result

A one-file PR documenting `slugify` and `unique_slugs`, including normalization,
empty input, deduplication, ordering, and output newlines. Examples should work
when copied from the repository root. The result should record a passing
`sh demo/test.sh` before and after the edit, plus executed example checks.

The Task has a 10-minute timeout, 24 turns, and no retries. After merging a complete
documentation update, another run may correctly return `NoChange`.

## Adapt it

Choose one documentation file and the code that is authoritative for it. Update
the prompt, `allowedPaths`, and validation command together. Keep implementation
and test files outside the write scope. Do not ask for repository-wide cleanup.
