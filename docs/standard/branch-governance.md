# Unattended Branch Governance Runbook

Unattended branch governance is the controlled cleanup workflow for reducing a repository back to a single healthy `main` branch. It is allowed only when the operator can prove every non-`main` branch has been audited, any potentially valuable unsafe branch has a backup ref, safe commits have been integrated, and the final `main` state matches `origin/main` with a clean worktree.

This runbook is intentionally conservative: deletion is the last step, not the classification mechanism.

## Required invariants

- `main` is the only release branch and must remain aligned with `origin/main` before completion.
- Local write work must not happen directly on `main`; use a worker worktree or temporary review branch until the final fast-forward/merge point.
- Branches with unknown, unmerged, or conflicting value must be backed up before any destructive action.
- Destructive branch actions are leader-owned: workers may audit, classify, and prepare backup evidence, but deletion, force-reset, or remote branch removal must be performed by the leader only after the backup refs and audit ledger are recorded.
- Release verification must run with `XLIB_CONTEXT=release_verify` and `GOWORK=off` so local workspace state cannot mask release drift.
- Completion evidence must include the branch inventory, disposition for each non-`main` branch, backup refs created, merges performed, deletions performed, and final verification commands.

## Branch classification

Classify every branch other than `main` into exactly one bucket:

| Bucket | Meaning | Required action |
| --- | --- | --- |
| `safe-merge` | Contains useful commits that apply cleanly and pass the relevant gates. | Merge or cherry-pick into `main`, then verify. |
| `backup-only` | May contain value but cannot be safely merged unattended because it conflicts, is incomplete, or lacks clear ownership. | Create a backup ref before cleanup and record why it was not merged. |
| `duplicate` | Already reachable from `main` or another kept backup ref. | Delete after recording the duplicate base. |
| `stale-worthless` | Obsolete experiment, generated noise, or branch with no useful diff. | Delete after audit evidence proves no unique value. |
| `blocked` | Requires owner, credential, external CI, or destructive decision outside the current scope. | Stop the unattended cleanup and report the blocker. |

Do not use branch age alone as a deletion reason. Age can support a `stale-worthless` classification only when the diff and commit history also show no retained value.

## Safe local procedure

Run read-only inventory first:

```bash
git fetch --prune origin
git branch --format='%(refname:short) %(objectname:short) %(committerdate:iso8601) %(upstream:short)'
git branch -r --format='%(refname:short) %(objectname:short) %(committerdate:iso8601)'
```

For each candidate branch, inspect reachability and unique content:

```bash
git log --oneline --decorate --left-right main...<branch>
git diff --stat main...<branch>
git diff --name-status main...<branch>
```

For any `backup-only` branch, create a durable backup ref before deletion or reset:

```bash
git branch backup/<yyyy-mm-dd>/<branch-name> <branch>
git push origin refs/heads/backup/<yyyy-mm-dd>/<branch-name>
```

Only merge `safe-merge` branches after confirming the patch is intentional and testable:

```bash
git switch main
git pull --ff-only origin main
git merge --no-ff <branch>
GOWORK=off make docs-check
GOWORK=off go test ./...
```

If a merge fails, abort it and reclassify the branch as `backup-only` or `blocked`:

```bash
git merge --abort
```

After all safe integrations are complete, the leader may delete only branches that have an audit disposition and, when needed, a backup ref:

```bash
git branch -d <duplicate-or-merged-branch>
git branch -D <stale-worthless-branch>   # only with recorded audit evidence
git push origin --delete <remote-branch>  # only after local evidence is complete
```

Workers must stop before running these destructive commands and hand the recorded classification, backup refs, and verification evidence to the leader. If the leader cannot verify the backup ref or audit entry for a branch, leave the branch intact and classify it as `blocked`.

## Final verification

The final state must prove that release automation would see the same clean `main` as the remote:

```bash
git switch main
git fetch --quiet origin main --tags
test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)"
test -z "$(git status --porcelain)"
test "$(git branch --format='%(refname:short)' | grep -vc '^main$')" = "0"
XLIB_CONTEXT=release_verify GOWORK=off make release-preflight VERSION=<next-version>
```

`release-preflight` enforces the mechanical release guard: valid version, current branch is `main`, clean worktree, `HEAD == origin/main`, absent local and remote tag, matching `CHANGELOG.md` heading, required lint tooling, and vulnerability-scan tooling when the vulnerability window requires it.

## Evidence template

Record the following in the task result, PR, or release evidence:

```text
Branch governance evidence:
- Inventory command/date:
- Branch classifications:
  - <branch>: <safe-merge|backup-only|duplicate|stale-worthless|blocked> — <reason>
- Backup refs created:
- Merges/cherry-picks performed:
- Branches deleted locally:
- Branches deleted remotely:
- Final HEAD:
- origin/main:
- Final branch list:
- Verification:
  - git status --porcelain: <empty>
  - main == origin/main: <pass/fail>
  - release-preflight: <pass/fail + VERSION>
```

If any branch is `blocked`, do not claim unattended cleanup complete. Report the blocker and leave the branch or its backup ref intact.
