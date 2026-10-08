---
title: Understanding 3-Way Merge in kpt
linkTitle: 3-Way Merge
description: |
  Learn how kpt uses 3-way merge to intelligently combine local customizations 
  with upstream package updates.
toc_hide: false
menu:
  main:
    parent: "Guides"
---

## Overview

When you run `kpt pkg update`, kpt needs to intelligently combine three versions of your package:

1. **Original** (origin): The upstream package version you initially fetched
2. **Updated** (upstream): The new upstream package version you're updating to
3. **Local** (destination): Your local package with your customizations

kpt uses a **3-way merge** algorithm to combine these versions, similar to how `git merge` works. This allows you to receive upstream improvements while preserving your local customizations.

## How 3-Way Merge Works

Instead of just replacing your local package with the upstream version, kpt analyzes all three versions:

```
    Original (origin)          Updated (upstream)
           │                           │
           └───────────┬───────────────┘
                       │
                       ▼
                  3-Way Merge
                       │
                       ▼
        Local Package Merged with Updates
```

**The Three Sources**:
- **Original**: The package you fetched (common ancestor)
- **Updated**: The new upstream version
- **Local**: Your modified package

By comparing all three, kpt can determine:
- What you changed locally
- What the upstream changed
- How to combine them intelligently

## Merge Strategies

kpt supports four merge strategies via the `--strategy` flag. Choose the one that fits your workflow:

### 1. resource-merge (Default)

Uses structural comparison of Kubernetes resources to intelligently merge changes.

**How it works**:
- Compares resources field-by-field using Kubernetes schema information
- Matches resources by their identity (group, kind, name, namespace)
- Intelligently handles lists using merge keys (e.g., containers matched by name)
- Preserves your customizations while applying upstream improvements
- (Experimental) By default, a null (`null`, `~`, or an empty value) in local
  or upstream is treated as a field deletion. Pass `--preserve-nulls`
  (persisted as `upstream.preserveNulls` in the Kptfile) to keep those nulls
  instead.

**When to use**:
- **Default choice** - recommended for most situations
- When you have significant local customizations
- When you want to receive upstream improvements
- Works with schema/CRD-aware changes

**Example result**:
```yaml
# Your local changes: replicas: 3
# Upstream adds: affinity, new env vars
# Result: Your replicas preserved + upstream changes added
```

For detailed technical information on how resource-merge works, see the [update command reference]({{% relref "/reference/cli/pkg/update#resource-merge-strategy" %}}).

### 2. copy-merge

Replaces local files with the upstream version at the file level, rather than
merging field-by-field.

**How it works**:
- For most files present in both local and upstream, the upstream version wins
  (in-file local edits to upstream-owned files are lost)
- The root Kptfile is an exception: it is 3-way merged rather than overwritten,
  preserving local-only, non-conflicting customizations; conflicting values
  follow the Kptfile merge rules and upstream metadata is refreshed
- Files that were added purely locally (never in upstream) are kept
- A file that originated upstream but was edited locally is still upstream-owned:
  if upstream deletes it, it is removed and the local edits are lost

**When to use**:
- When you trust the upstream content over local in-file edits
- When kpt cannot structurally parse the files (e.g. non-KRM files)
- When you still want to keep files you added alongside the package

**Example**:
```bash
$ kpt pkg update my-pkg --strategy copy-merge
# Shared files replaced by upstream; purely-local files preserved
```

### 3. fast-forward

Ensures your package hasn't changed since you fetched it.

**How it works**:
- Checks if your local package is identical to the original
- If yes: updates cleanly to the upstream version
- If no: fails and requires you to resolve changes first

**When to use**:
- When you want guaranteed clean updates
- For packages you don't customize
- When you want the update to stop rather than silently overwrite local edits
- For "pin to upstream" workflows

**Example**:
```bash
$ kpt pkg update my-pkg --strategy fast-forward
# Works if you haven't modified anything
# Fails if any local changes exist
```

### 4. force-delete-replace

Replaces your entire local package with upstream, discarding all local changes.

**When to use**: Only when you intentionally want to discard all customizations

{{% alert title="Warning" color="warning" %}}
This strategy will **lose all your local modifications**. Use with caution.
{{% /alert %}}

For complete strategy details and examples, see the [update command reference]({{% relref "/reference/cli/pkg/update" %}}).

## A Complete Example

Here's a realistic scenario showing how 3-way merge works:

**Scenario**: You've customized a WordPress package with your own resource limits and replica count. The upstream releases a new version with better security settings and a new environment variable.

```yaml
# ORIGINAL (nginx:v1.0) - what you initially fetched
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wordpress
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.14
          resources:
            limits:
              memory: 256Mi
```

```yaml
# UPDATED (nginx:v2.0) - what upstream released
# Changes: image updated, affinity added, new env var
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wordpress
spec:
  replicas: 1
  affinity:  # NEW
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchExpressions:
              - key: app
                operator: In
                values: [wordpress]
          topologyKey: kubernetes.io/hostname
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:2.0  # CHANGED
          resources:
            limits:
              memory: 256Mi
          env:  # NEW
            - name: LOG_LEVEL
              value: info
```

```yaml
# LOCAL (your changes) - replicas and memory customized
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wordpress
spec:
  replicas: 3  # YOUR customization
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.14
          resources:
            limits:
              memory: 512Mi  # YOUR customization
```

```yaml
# RESULT after kpt pkg update
# Your customizations preserved + upstream changes applied
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wordpress
spec:
  replicas: 3  # ✓ YOUR value kept
  affinity:  # ✓ UPSTREAM value added
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchExpressions:
              - key: app
                operator: In
                values: [wordpress]
          topologyKey: kubernetes.io/hostname
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:2.0  # ✓ UPSTREAM value (you didn't customize)
          resources:
            limits:
              memory: 512Mi  # ✓ YOUR value kept
          env:  # ✓ UPSTREAM value added
            - name: LOG_LEVEL
              value: info
```

**Key insight**: Your customizations (replicas, memory) are preserved, while upstream improvements (affinity, env vars, image update) are applied.

## Handling Conflicts

A conflict occurs when both you and upstream modified the same field **differently**:

```yaml
# Original
image: nginx:1.14

# Upstream changed to
image: nginx:2.0

# You also changed to
image: nginx:1.20

# Conflict: which value wins?
```

### How resource-merge resolves conflicts

Unlike `git merge`, resource-merge does **not** stop or write conflict markers
when upstream and local assign different non-null values to the same scalar or
non-associative list. It chooses the new upstream value and the update
**succeeds**. Local field removals and nulls follow separate deletion semantics
and may remain deleted even when upstream changes the field (unless
`--preserve-nulls` is set).

In the example above, the result is `image: nginx:2.0` (the upstream value); your
local `nginx:1.20` is overwritten. This is deterministic and silent, so the
important thing to understand is that **a local edit can be replaced by an
upstream change to the same field without any warning**.

### Keeping a local value instead

Because the merge never fails, there is nothing to "resolve" interactively. If
you need to retain a local value that upstream also changed, choose one of these:

**Option A — re-apply after update**
- Run the update (upstream wins the conflicting field)
- Re-apply your local value and commit

**Option B — avoid the clash**
- Pin to an upstream ref that doesn't change that field, or
- Move your customization to a field upstream doesn't touch

**Option C — take everything from upstream**
- Use `--strategy force-delete-replace` to reset to upstream, then re-apply
  customizations deliberately

> Note: the other strategies behave differently on the same clash — `copy-merge`
> and `force-delete-replace` also take the upstream value (at the file level),
> while `fast-forward` refuses to run at all if the local package was modified.

### Tips for Avoiding Conflicts

- Keep customizations minimal and well-documented
- Update frequently to catch upstream changes to fields you also edit
- Communicate with upstream about your customizations
- Use the Kptfile `upstream` field to control update targets

## Common Use Cases

### Use Case 1: Security Update

You need to update a package to get a security patch.

```bash
$ git add . && git commit -m "Current state before update"
$ kpt pkg update my-app
# Security fixes applied, customizations preserved
$ git diff  # Review changes
$ kpt eval . # Validate
$ git push
```

### Use Case 2: Version Migration

You need to update to match a new upstream API version.

```bash
$ kpt pkg update my-app --strategy resource-merge
# Schema changes applied
# Your values preserved
# Test in dev environment first
```

### Use Case 3: Keeping in Sync

You want to stay synchronized with upstream but maintain customizations.

```bash
# Regular update cycle
$ kpt pkg update packages/*/  # Update all packages
$ kpt eval .  # Validate
$ git commit -am "Updated packages to latest"
```

## Best Practices

1. **Commit before updating**
   ```bash
   git add . && git commit -m "State before update"
   kpt pkg update
   ```
   This gives you a git history of what changed.

2. **Use resource-merge by default**
   - Most flexible strategy
   - Preserves your customizations
   - Applies upstream improvements
   - Only use other strategies when you have a specific reason

3. **Review merge results**
   ```bash
   git diff         # See what changed
   kpt eval .       # Validate resources
   git diff HEAD~1  # Compare to before
   ```

4. **Test after merging**
   - Deploy to dev environment
   - Run your validation checks
   - Ensure customizations still work

5. **Review fields that upstream may also change**
   - Remember resource-merge silently takes the upstream value on a clash
   - Re-apply any local value that got overwritten, and document why you need it
   - Consider whether you still need the customization

6. **Update frequently**
   - Smaller, more frequent updates
   - Fewer surprising overwrites of locally-edited fields
   - Easier to track changes
   - Better security posture

## Understanding Your Package

### Resource Identity

Resources are matched across versions using their identity:
- **Group** (e.g., `apps`, `v1`)
- **Kind** (e.g., `Deployment`)
- **Name** (metadata.name)
- **Namespace** (metadata.namespace)

When you rename or move a resource, kpt adds a `# kpt-merge` comment to help track identity across updates.

### Merge Keys

For lists like containers, kpt uses **merge keys** to intelligently match elements:

```yaml
spec:
  containers:  # Matched by name field
    - name: nginx      # This is the merge key
      image: nginx:1.0
    - name: sidecar
      image: helper:1.0
```

If you add a new container, it's added to the merged result. If you remove one, it stays removed.

For technical details on merge keys, resource identity, and the complete merge algorithm, see the [update command reference]({{% relref "/reference/cli/pkg/update#merge-rules" %}}).

## Understanding Git Integration

The `kpt pkg update` command works with your package files, similar to how you'd use `git merge`. The key difference:

- **Git merge**: Combines text-based files (any format) using line-based diff/merge algorithms
- **kpt 3-way merge**: Combines Kubernetes resources using field-level, schema-aware merge logic

After running `kpt pkg update`, use standard Git commands to review and manage changes:

```bash
kpt pkg update my-pkg        # Perform 3-way merge on resources
git diff                      # Review resource changes
git add . && git commit       # Track changes in version control
```

kpt's 3-way merge is **independent** of Git's merge algorithm—it applies intelligently to YAML resources even if you're not using Git, though Git is commonly used for version tracking.

## Related Resources

- [kpt pkg update command reference]({{% relref "/reference/cli/pkg/update" %}}) - Complete technical specification and all merge strategies
- [Kptfile specification]({{% relref "/reference/schema/kptfile" %}}) - Package configuration schema

## See Also

**Related Technologies Using 3-Way Merge:**

- **[Porch](https://porch.kpt.dev/docs/7_cli_api/porchctl/#rpkg-upgrade)** - Declarative package orchestration platform that uses 3-way merge for the `rpkg upgrade` operation (similar but at the package revision level)
- **[Git merge](https://git-scm.com/docs/git-merge)** - Uses 3-way merge for combining branches, but at the line level rather than field level
- **[Kubernetes Strategic Merge Patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/#how-apply-calculates-differences)** - Kubernetes native 3-way merge for applying declarative configurations
