# Repository analyst

Start here. This read-only Task maps the repository, identifies concrete gaps,
and suggests one small contribution. It does not execute repository code or
create files, issues, or pull requests.

Its Task overrides the shared Agent with Codex's exact `Glob`, `Grep`, `Read`
policy and `allowBash: false`. Keep that trio intact; arbitrary partial native
tool allowlists are not supported by the Codex runtime.

Use the root [shared setup and analyst commands](../../README.md#run-the-analyst).
The manifest reads the public sample repository's default branch. For another
repository, copy `task.yaml` to `.local/`, change `workspace.gitRepo`, and update
the prompt's expected directories. Set `workspace.ref` to a commit for a repeatable
snapshot. Private sources also need a `readCredentialRef`; do not add publication
credentials to this workflow.

## Expected result

- A report under 700 words with paths and line references.
- Setup and test commands marked as not executed.
- One bounded contribution with an acceptance check.
- An unchanged workspace and `ReadValidated` delivery outcome.

The Task has a 10-minute timeout, 16 turns, and no retries. Save the report before
deleting the completed `repository-analyst` Task to rerun it. The optional
[weekly schedule](../../schedules/repository-analyst) uses the same prompt and
workspace policy without duplicating this manifest.
