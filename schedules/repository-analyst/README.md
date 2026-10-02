# Weekly repository analysis

This Kustomize overlay reuses the [repository analyst](../../workflows/repository-analyst).
It adds a Monday 09:00 UTC schedule, forbids overlapping child runs, retains three
successful runs and one failed run, and skips starts missed by more than five
minutes. It starts **suspended** and has no publication credentials.

Complete the root [shared setup](../../README.md#shared-setup) first. Inspect the
rendered Task and dry-run it against your chosen cluster:

```sh
kubectl kustomize schedules/repository-analyst
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" apply \
  --dry-run=server -k schedules/repository-analyst
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" apply \
  -k schedules/repository-analyst
```

This creates `repository-analyst-weekly`. No child runs start while it is suspended.
Check the source repository, schedule, model cost, and history limits before
explicitly enabling it:

```sh
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" patch task \
  repository-analyst-weekly --type=merge -p '{"spec":{"suspend":false}}'
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" get task \
  repository-analyst-weekly -o yaml
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" get tasks \
  -l orka.ai/parent-task=repository-analyst-weekly
```

Read each child Task's result using the [root instructions](../../README.md#check-the-result).
Runs are independent reports, not a comparison against previous reports. Every
enabled run consumes model tokens. Applying this overlay again restores
`suspend: true`.

## Stop and remove

```sh
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" patch task \
  repository-analyst-weekly --type=merge -p '{"spec":{"suspend":true}}'
```

Suspension stops future runs, not already-running children. Let active children
finish and save their results before deleting the parent and its owned children:

```sh
kubectl --context "$KUBE_CONTEXT" -n "$ORKA_NAMESPACE" delete task repository-analyst-weekly
```

Keep recurring writers out of your first rollout. They can produce repeated PRs
while previous ones remain unmerged; a read-only schedule avoids that problem.
