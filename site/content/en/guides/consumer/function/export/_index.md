---
title: "Exporting Workflow Config Files"
linkTitle: "Exporting a Workflow"
weight: 8
type: docs
no_list: true
description: >
    Export config files for different workflow orchestrators that run kpt functions
---

`kpt fn export` reduces the work to run kpt functions in workflow orchestrators. It allows to export a workflow pipeline that runs kpt functions alongside necessary configurations. The generated pipeline files can be easily integrated into the existing one manually.

## Examples

These quickstarts cover how to exporting workflow config files for different orchestrators:
 
- [CircleCI](./circleci)
- [Cloud Build](./cloud-build)
- [GitHub Actions](./github-actions)
- [GitLab CI](./gitlab-ci)
- [Jenkins](./jenkins)
- [Tekton](./tekton)

*Unable to find support for your orchestrator? Please file an [Issue](https://github.com/GoogleContainerTools/kpt/issues)/[Pull Request](https://github.com/GoogleContainerTools/kpt/pulls).*
