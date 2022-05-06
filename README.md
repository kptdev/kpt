<img src="logo/KptLogoLarge.png" width="220">


# kpt

kpt is a package-centric toolchain that enables a WYSIWYG configuration 
authoring, automation, and delivery experience, which simplifies managing
Kubernetes platforms and KRM-driven infrastructure at scale by manipulating
declarative Configuration as Data, separated from the code that transforms it.

## Why kpt?

kpt enables WYSIWYG editing and interoperable automation applied to declarative
configuration data, similar to how the live state can be modified with imperative
tools. 

See [the rationale](https://kpt.dev/guides/rationale) for more background.

The best place to get started and learn about specific features of kpt is 
to visit the [kpt website](https://kpt.dev/).

## Install kpt

kpt installation instructions can be found on 
[kpt.dev/installation](https://kpt.dev/installation/)

## kpt components

The kpt toolchain includes the following components:

- **kpt CLI**: The [kpt CLI](https://kpt.dev/reference/cli/) supports package and function operations, and also
  deployment, via either direct apply or GitOps. By keeping an inventory of deployed resources, kpt enables resource pruning,
  aggregated status and observability, and an improved preview experience.

- **Function SDKs**: Any general-purpose or domain-specific language can be used to create functions to transform and/or validate
  the YAML KRM input/output format, but we provide SDKs to simplify the function authoring process, in 
  [Go](https://kpt.dev/book/05-developing-functions/02-developing-in-Go), 
  [Typescript](https://kpt.dev/book/05-developing-functions/03-developing-in-Typescript), and 
  [Starlark](https://catalog.kpt.dev/starlark/v0.2/), a Python-like embedded language.

- **Function catalog**: A [catalog](https://catalog.kpt.dev/) of off-the-shelf, tested functions. kpt makes configuration
  easy to create and transform, via reusable functions. Because they are expected to be used for in-place transformation,
  the functions need to be idempotent.

- **Package orchestrator**: 
  The [package orchestrator](https://github.com/GoogleContainerTools/kpt/blob/main/docs/design-docs/07-package-orchestration.md)
  enables the magic behind the unique WYSIWYG experience. It provides a control plane for creating,
  modifying, updating, and deleting packages, and evaluating functions on package data. This enables operations on packaged resources
  similar to operations directly on the live state through the Kubernetes API.

- **Config Sync**: While the package orchestrator
  can be used with any GitOps tool, Config Sync provides a reference GitOps implementation to complete the WYSIWYG management
  experience and enable end-to-end development of new features, such as 
  [OCI-based packages](https://github.com/GoogleContainerTools/kpt/issues/2300). Config Sync is also helping to drive improvements
  in upstream Kubernetes. For instance, Config Sync is built on top of [git-sync](https://github.com/kubernetes/git-sync) and
  leverages [Kustomize](https://kustomize.io) to automatically render manifests on the fly when needed. It uses the same apply
  logic as the kpt CLI.

- **Backage UI plugin**: We've created a proof-of-concept UI to demonstrate the WYSIWYG experience that's possible on top of the
  package orchestrator. More scenarios can be supported by implementing form-based editors for additional Kubernetes resource types.

## Roadmap

You can read about the big upcoming features in the 
[roadmap doc](/docs/ROADMAP.md).

## Contributing

If you are interested in contributing please start with 
[contribution guidelines](CONTRIBUTING.md).

## Contact

We would love to keep in touch:

1. Join our [Slack channel](https://kubernetes.slack.com/channels/kpt)
1. Join our [email list](https://groups.google.com/forum/?oldui=1#!forum/kpt-users)
