---
title: "Export"
linkTitle: "export"
type: docs
description: >
   Auto-generating function pipelines for different workflow orchestrators
---

<!--mdtogo:Short
   Auto-generating function pipelines for different workflow orchestrators
-->

Exports a workflow pipeline that runs kpt functions alongside necessary
configurations.

### Examples

<!--mdtogo:Examples-->

```sh
# read functions from DIR, run them against it as one step.
# write the generated GitHub Actions pipeline to main.yaml.
kpt fn export DIR/ --output main.yaml --workflow github-actions
```

```sh
# discover functions in FUNCTIONS_DIR and run them against resource in DIR.
# write the generated Cloud Build pipeline to stdout.
kpt fn export DIR/ --fn-path FUNCTIONS_DIR/ --workflow cloud-build
```

<!--mdtogo-->

### Synopsis

<!--mdtogo:Long-->

```sh
kpt fn export DIR/ [--fn-path FUNCTIONS_DIR/] --workflow ORCHESTRATOR [--output OUTPUT_FILENAME]

DIR:
  Path to a package directory.
FUNCTIONS_DIR:
  Read functions from the directory instead of the DIR/.
ORCHESTRATOR:
  Supported orchestrators are:
    - github-actions
    - cloud-build
    - gitlab-ci
    - jenkins
    - tekton
    - circleci
OUTPUT_FILENAME:
  Specifies the filename of the generated pipeline. If omitted, the default
  output is stdout
```

<!--mdtogo-->

## Next Steps

- Get detailed tutorials on how to use `kpt fn export` from the
  [Export a Workflow] guide.

[Export a Workflow]: ../../../guides/consumer/function/export/
