---
draft: true
title: "Export"
linkTitle: "export"
type: docs
description: >
   Auto-generating function pipelines for different workflow orchestrators
---
<!--mdtogo:Short
   Auto-generating function pipelines for different workflow orchestrators
-->

Auto-generating function pipelines for different workflow orchestrators.

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
```
kpt fn export DIR/ [--fn-path FUNCTIONS_DIR/] --workflow ORCHESTRATOR [--output OUTPUT_FILENAME]

DIR:
  Path to a package directory. If you do not specify the --fn-path flag, this command will discover functions in DIR and run them against resources in it.
```
<!--mdtogo-->
