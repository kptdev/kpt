---
title: "`auto-ignore-non-krm`"
linkTitle: "auto-ignore-non-krm"

description: |
  Automatically skip non-KRM files in a package instead of failing.
---

The `kpt.dev/auto-ignore-non-krm` annotation tells kpt to skip any
non-KRM files in a package automatically, instead of failing with an
error.

A non-KRM file is a `.yaml`/`.yml`/`.json` file (or `Kptfile`-named
file) whose content is not a valid KRM resource: it is missing
`apiVersion`, `kind`, or `metadata.name` (list-kind resources such as
`ConfigMapList` are exempt from the `metadata.name` requirement, and
`kustomization.yaml` files are exempt entirely).

[`.krmignore`]({{% relref "/book/03-packages#excluding-files-with-krmignore" %}})
already lets you exclude specific files or patterns by hand. This
annotation is for when you'd rather not maintain that list manually:
any non-KRM file found is skipped the same way, without needing to
know its name or path ahead of time.

### Malformed YAML is different, and isn't covered

This annotation only applies to files that parse as valid YAML but
aren't a valid KRM resource -- the definition above. A file that
isn't valid YAML at all (invalid syntax) is a different failure
neither this annotation nor `.krmignore` can excuse: parsing fails
before kpt ever checks whether the content looks like a KRM resource,
so there's no non-KRM classification to skip.

In practice: the error looks different (a raw parse error, not
`resource must have` apiVersion/kind/name), it doesn't mention this
annotation as a fix, and if a package has multiple malformed files,
only the first one encountered is reported, not the consolidated
"every non-KRM file across the tree" report described below. The
only way to work around a malformed file is to fix its syntax or
remove it.

### Schema

The annotation value accepts string values of `true` and `false`.

Make sure to surround the value in quotes, otherwise it will be
considered a YAML boolean (invalid annotation), not a string.

### Behavior

Set on a package's `Kptfile`, in `metadata.annotations`:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
  annotations:
    kpt.dev/auto-ignore-non-krm: "true"
```

With the annotation absent (or set to `false`) and a non-KRM file
present, the command fails before doing any work, listing every
non-KRM file found across the whole package tree in one error:

```
Error: input resource list must contain only KRM resources (1 file(s)):
  values.yaml: resource must have `apiVersion`
To resolve: either exclude these files via .krmignore (manually, or with a
pattern rule), or add `kpt.dev/auto-ignore-non-krm: "true"` to this
package's Kptfile to skip any non-KRM files automatically.
```

With the annotation set to `"true"`, the same command instead skips
the file and proceeds, printing which files were skipped:

```
[info] ignoring 1 non-KRM file(s) in "my-package" due to kpt.dev/auto-ignore-non-krm annotation: values.yaml
```

The non-KRM file is skipped in memory for that invocation only.
Nothing is written to disk because of this annotation: the file
itself is left completely untouched, and no `.krmignore` file is ever
created or modified.

#### Scoping: per package, like `.krmignore`

The annotation only covers the package whose own `Kptfile` carries
it. It is **not** inherited by subpackages, and a subpackage's
annotation does not cover its parent, the same way a `.krmignore`
file only applies within the package directory it's placed in.

For example, given a package `parent` with a subpackage
`parent/sub`, where `sub` contains a non-KRM file:

```
parent/
├── Kptfile
├── deployment.yaml
└── sub/
    ├── Kptfile
    ├── deployment.yaml
    └── values.yaml      # non-KRM
```

Adding the annotation to `parent/Kptfile` does **not** excuse
`sub/values.yaml`; rendering still fails, naming `values.yaml`. The
annotation has to be added to `sub/Kptfile` itself, the package that
actually contains the non-KRM file, for that file to be skipped.

#### Supported commands

| Command | Supported |
| ------- | --------- |
| `kpt fn render` | Yes |
| `kpt fn eval` | Yes |
| `kpt fn source` | Yes |
| `kpt live init` | Yes |
| `kpt live apply` | Yes |
| `kpt live status` | Yes |
| `kpt live destroy` | Yes |
| `kpt alpha live plan` | Yes |
| `kpt live migrate` | No. The legacy ConfigMap-based inventory reader this command uses to support migrating from older `kpt` versions has no way to skip a file in memory, so a non-KRM file always blocks `migrate`, with or without this annotation. Use `.krmignore` instead. |

`kpt fn render` additionally records which files were skipped in the
root `Kptfile`'s `status.conditions` after a successful render; see
the [`fn render` Render Status Conditions]({{% relref "/reference/cli/fn/render#render-status-conditions" %}})
section.

### Example

```shell
kpt pkg init my-pkg
cd my-pkg
```

```shell
cat > values.yaml << EOF
replicaCount: 3
EOF
```

Rendering now fails, since `values.yaml` is not a valid KRM resource:

```shell
kpt fn render
```

```
Error: input resource list must contain only KRM resources (1 file(s)):
  values.yaml: resource must have `apiVersion`
...
```

Open the Kptfile and add the annotation under `metadata.annotations` (`kpt pkg init` already added an
`annotations` key for `config.kubernetes.io/local-config`, so add the new line alongside it):

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-pkg
  annotations:
    kpt.dev/auto-ignore-non-krm: "true"
    config.kubernetes.io/local-config: "true"
info:
  description: sample description
```

Rendering now succeeds, skipping `values.yaml`:

```shell
kpt fn render
```

```
[info] ignoring 1 non-KRM file(s) in "my-pkg" due to kpt.dev/auto-ignore-non-krm annotation: values.yaml
Successfully executed 0 function(s) in 1 package(s).
```
