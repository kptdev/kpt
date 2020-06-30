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

Auto-generating function pipelines for different workflow orchestrators.

### Examples
<!--mdtogo:Examples-->
```sh
# read functions from DIR, run them against it as one step
# write the generated GitHub Actions pipeline to main.yaml
kpt fn export github-actions DIR/ --output main.yaml
```

```sh
# discover functions in FUNCTIONS_DIR and run them against Resource in DIR.
kpt fn export github-actions DIR/ --fn-path FUNCTIONS_DIR/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt fn export ORCHESTRATOR DIR/ [--fn-path FUNCTIONS_DIR/] [--output OUTPUT_FILENAME]

ORCHESTRATOR:
  Supported orchestrators are github-actions, cloud-build, and gitlab-ci.
DIR:
  Path to a package directory. If you do not specify the --fn-path flag, this command will discover functions in DIR and run them against resources in it.
```
<!--mdtogo-->
