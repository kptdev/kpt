---
title: "Exporting Workflow Config Files"
linkTitle: "Exporting a Workflow"
weight: 8
type: docs
no_list: true
description: >
    Export config files for different workflow orchestrators that run kpt functions
---

`kpt fn export` reduces the work to run kpt functions in workflow orchestrators. It exports a workflow pipeline that runs kpt functions alongside necessary configurations. The generated pipeline files can be easily integrated into an existing one manually.

## Examples

These quickstarts cover how to export workflow config files for different
orchestrators:

- [GitHub Actions]
- [GitLab CI]
- [Jenkins]
- [Cloud Build]
- [CircleCI]
- [Tekton]

*Unable to find support for your orchestrator? Please file an [Issue]/[Pull Request].*

[GitHub Actions]: guides/consumer/function/github-actions/
[GitLab CI]: guides/consumer/function/gitlab-ci/
[Jenkins]: guides/consumer/function/jenkins/
[Cloud Build]: guides/consumer/function/cloud-build/
[CircleCI]: guides/consumer/function/circleci/
[Tekton]: guides/consumer/function/tekton/
[Issue]: https://github.com/GoogleContainerTools/kpt/issues
[Pull Request]: https://github.com/GoogleContainerTools/kpt/pulls
