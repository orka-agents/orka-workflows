# Test improver

The demo has five passing checks with deliberate coverage gaps. This Task adds
three to five deterministic cases without changing implementation behavior or
weakening existing assertions. Only `demo/test.sh` can be published.

Follow [shared setup](../../README.md#shared-setup) and
[run a writing workflow](../../README.md#run-a-writing-workflow) with
`WORKFLOW=test-improver`. Use your fork for source and publication.

## Expected result

A one-file PR covering previously untested behavior, such as all-separator input,
embedded newlines, or no titles. Existing assertions remain, and the reported
test count matches the checks. The result should record a passing baseline,
`sh -n demo/test.sh`, and a passing `sh demo/test.sh` after the edit.

The Task has a 10-minute timeout, 24 turns, and no retries. A discovered
implementation bug should be reported, not fixed outside the allowed scope or
committed as a known-failing test.

## Adapt it

Select one test file and a bounded behavior to cover. Replace the fixture-specific
prompt and test command, then set `allowedPaths` to that test file. Preserve the
rule against weakening assertions and check that your runtime has the test tools.
