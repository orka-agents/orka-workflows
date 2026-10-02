# Orka workflows

Small, native [Orka](https://github.com/orka-agents/orka) workflows to copy and
adapt. Each workflow is a Task that uses one shared Codex ACP Agent. The writing
samples make a bounded change to the included shell library and ask Orka's
trusted publisher to open a pull request in **your fork**.

| Workflow | Result | Publishable files |
| --- | --- | --- |
| [Repository analyst](workflows/repository-analyst) | Repository map and a suggested first contribution | None |
| [Documentation updater](workflows/documentation-updater) | Document the demo's public API | `demo/README.md` |
| [Test improver](workflows/test-improver) | Cover missing edge cases | `demo/test.sh` |
| [Code simplifier](workflows/code-simplifier) | Remove duplicated normalization | `demo/textkit.sh` |

Start with the analyst. Then run one writing sample on your fork. The optional
[weekly schedule](schedules/repository-analyst) reuses the analyst and starts
suspended. There is no bundle that launches all workflows.

## Prerequisites

- A working Orka installation with the `core.orka.ai/v1alpha1` Agent and Task CRDs.
  The local tests check schemas from v0.2.0 and the pinned October 1, 2026 source.
  They do not certify runtime compatibility or a live deployment.
- The built-in Codex `orka.harness.v2` runtime, its central authenticated provider
  proxy, and the trusted workspace/publisher services configured by your operator.
  The proxy must allow the Agent's model, `gpt-5.4`, or your chosen replacement.
- `kubectl`, access to Orka's watched namespace, and permission to create Agents
  and Tasks there. Use the Orka dashboard or an authenticated `orka` CLI for results.
- For writing samples, a fork containing these files, an existing base branch,
  and repository-scoped credential Secrets described below.

The demo uses POSIX shell and `tr`, available in the stock Codex runtime. It needs
no package installation or network access during model execution. Go is needed
only to run this repository's maintainer tests, not the workflows.

See Orka's [installation guide](https://github.com/orka-agents/orka/blob/main/website/docs/operations/installation.md)
and [Codex Agent example](https://github.com/orka-agents/orka/blob/1cd16c88b43410e0ea46463f8a39b473d8e5250e/config/samples/core_v1alpha1_agent_codex.yaml)
for platform setup. These samples do not install or reconfigure the cluster.

## Shared setup

Clone this repository and run commands from its root. Set the exact context and
namespace for your installation; none of these commands changes your kubeconfig.

```sh
export KUBE_CONTEXT='YOUR-CONTEXT'
export ORKA_NAMESPACE='YOUR-ORKA-NAMESPACE'
mkdir -p .local
cp agents/codex.yaml .local/codex.yaml
```

Edit `.local/codex.yaml` if your proxy requires a different model. Keep the runtime
contract and tool configuration. Do not add provider API keys to the Agent.

```sh
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" apply \
  --dry-run=server -f .local/codex.yaml
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" apply -f .local/codex.yaml
```

This creates only the shared Agent, not a Task. Use a dedicated namespace or check
that an existing Agent named `workflows-codex` is yours before applying it.

## Run the analyst

The analyst reads this public repository's default branch without a Git credential.
Copy and edit the manifest first if you want a different repository or a pinned ref.

```sh
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" create \
  --dry-run=server -f workflows/repository-analyst/task.yaml
TASK=$(kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" create \
  -f workflows/repository-analyst/task.yaml -o jsonpath='{.metadata.name}')
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" get task "$TASK" -w
```

Stop watching with Ctrl-C; that does not cancel the Task. The analyst uses the fixed
name `repository-analyst` so its manifest can also serve as the schedule's Kustomize
base. Save its result and delete the completed Task before running it again.

## Run a writing workflow

Fork this repository and choose an existing branch in your fork that contains the
demo. Do not assume its name is `main`. Each run publishes a new branch and a
non-draft PR in that fork. It never merges. A run that makes no changes produces
`NoChange`, not a PR.

Have your operator provision these Secrets in the watched namespace. Each uses a
`token` key; keep token values out of manifests, URLs, shell history, and Git.

| Secret name | Credential role |
| --- | --- |
| `workflows-read` | Read the source and publication repositories, both your fork here |
| `workflows-publish` | Write the new publication branch in your fork |
| `workflows-forge` | Create and reconcile pull requests in your fork |

Use separate, least-privilege tokens restricted to the fork. There are four
references in each Task because source-read and publication-read are separate
roles. This same-repository setup uses one read token for both. The trusted
services resolve these Secrets; the model process does not receive them. See
Orka's [credential example](https://github.com/orka-agents/orka/blob/1cd16c88b43410e0ea46463f8a39b473d8e5250e/examples/github-cicd/secret.yaml).

Choose one workflow and make a local copy:

```sh
WORKFLOW=documentation-updater  # or test-improver or code-simplifier
cp "workflows/$WORKFLOW/task.yaml" ".local/$WORKFLOW.yaml"
```

Edit that copy before submitting it:

1. Replace `YOUR-OWNER` in **both** repository URLs with your fork's owner.
2. Set `branch` and `prBaseBranch` to the same existing base branch in your fork.
3. Set all four credential references to your provisioned Secret names.
4. Read the prompt and the allowed file. Keep `allowedPaths`, `maxChangedFiles`,
   and the publication protections in place. Leave `pushBranch` unset.

The checked-in placeholders are intentionally not runnable. A dry run checks
admission, not remote repositories, credentials, or model access.

```sh
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" create \
  --dry-run=server -f ".local/$WORKFLOW.yaml"
TASK=$(kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" create \
  -f ".local/$WORKFLOW.yaml" -o jsonpath='{.metadata.name}')
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" get task "$TASK" -w
```

Use `create`, not `apply`, for these generated-name Tasks. Orka derives a unique
publication branch for each run. Run samples separately so you can review each
result before spending more model tokens or creating another PR.

## Check the result

Open the Task in the Orka dashboard, or use the CLI with your configured API
authentication and the server corresponding to the cluster above:

```sh
export ORKA_SERVER='YOUR-ORKA-API-URL'
orka --server "$ORKA_SERVER" -n "$ORKA_NAMESPACE" task result "$TASK"
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" get task "$TASK" -o yaml
```

Task output is stored by Orka, not in a `status.result` field. The API equivalent
is `GET /api/v1/tasks/<name>/result?namespace=<namespace>`.

For the analyst, expect a short report with file evidence and
`status.delivery.outcome: ReadValidated`. For a changed writing run, require
`VerifiedExact`, inspect `status.delivery.verifiedRemoteSHA` and
`status.delivery.prReceipt`, and review the actual PR diff and checks. `Finalizing`
is not terminal. `NoChange` is valid when there is nothing useful left to do.
Investigate any other delivery outcome; a model saying "done" is not proof of
successful delivery or green CI.

If a Task remains pending or fails, inspect its status and events with
`kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" describe task "$TASK"`.
Check runtime availability and model policy with your operator. For credential
failures, check Secret names and repository scopes without printing token values.

## Safety and adaptation

The analyst uses Codex's supported `Glob`, `Grep`, `Read` policy with Bash disabled
and requests an unchanged workspace. Orka verifies that workspace; this is not a
claim of an OS-level read-only mount.

Codex writing sessions require unrestricted native tools, so the shared Agent
intentionally omits `defaultAllowedTools`. A partial allowlist causes runtime
planning to fail. The prompt tells writers not to use the network, but that is
not a network-enforcement policy. The writing Tasks restrict publication to one
exact path and reject control paths, binary files, and secret-like content. These
limits do not make arbitrary repository scripts safe. Use a repository and branch
you trust before running code from it. See Orka's
[native policy validation](https://github.com/orka-agents/orka/blob/1cd16c88b43410e0ea46463f8a39b473d8e5250e/internal/controller/acp_agent_configuration.go#L197-L249).

Baseline and post-edit checks are prompt instructions, **not a controller-enforced
test gate**. The agent is told to stop or undo its own edits on failure. Review the
result and keep required CI checks on PRs before merging.

To adapt a sample, change the prompt, validation commands, source/base branches,
and allowed paths together. Keep the edit scope small. If the target needs Go or
another toolchain, arrange an approved runtime with that toolchain first rather
than asking the agent to install it. These samples use no GitHub MCP server,
cross-run memory, event bridge, or GitHub Actions workflow translation.

## Cleanup

Save any result you need before deleting a completed Task:

```sh
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" delete task "$TASK"
```

Deleting a Task does not close its GitHub PR or remove the published branch.
Clean those up in your fork separately. Stop the optional schedule using its
[instructions](schedules/repository-analyst) before deleting the shared Agent.
When no sample Tasks still use it:

```sh
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" delete agent workflows-codex
```

## Validate this repository

Maintainers need Go 1.25 or newer, `kubectl` with Kustomize, and POSIX shell:

```sh
sh demo/test.sh
go test ./...
go vet ./...
```

Tests validate the manifests against CRDs fetched from two immutable Orka commits,
check publication policy, render the suspended schedule, and run the shell demo.
The schema test needs HTTPS access to `raw.githubusercontent.com`; it fails rather
than silently skipping when CRDs cannot be fetched. No tests contact a Kubernetes
cluster or run a model. JSON Schema checks do not evaluate Kubernetes CEL or
admission policy, so use the server-side dry runs above against your installation.

## Inspiration

Inspired by [GitHub Next's Agentics workflows](https://github.com/githubnext/agentics/tree/5d11aa2a05ce2c943c085acb7b12b583f83ed375/workflows).
These are independently written Orka-native samples, not ports of the gh-aw runtime
or a compatibility layer.
