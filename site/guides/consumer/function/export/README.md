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

[GitHub Actions]: /guides/consumer/function/export/github-actions/
[GitLab CI]: /guides/consumer/function/export/gitlab-ci/
[Jenkins]: /guides/consumer/function/export/jenkins/
[Cloud Build]: /guides/consumer/function/export/cloud-build/
[CircleCI]: /guides/consumer/function/export/circleci/
[Tekton]: /guides/consumer/function/export/tekton/
[Issue]: https://github.com/GoogleContainerTools/kpt/issues
[Pull Request]: https://github.com/GoogleContainerTools/kpt/pulls
