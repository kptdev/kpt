<img src="https://raw.githubusercontent.com/kptdev/.github/main/kpt_stacked_color-100x123.png" width="220" alt="kpt logo">


[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/10656/badge)](https://www.bestpractices.dev/projects/10656)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkptdev%2Fkpt.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkptdev%2Fkpt?ref=badge_shield)
[![Release](https://img.shields.io/github/v/release/kptdev/kpt)](https://github.com/kptdev/kpt/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/kptdev/kpt)](https://goreportcard.com/report/github.com/kptdev/kpt)

# kpt: Automate Kubernetes Configuration Editing

> **Version 1.0.0 Released!**  
> kpt v1.0.0 is now stable with guaranteed API compatibility. See [VERSIONING.md](docs/VERSIONING.md) for details.

kpt is a package-centric toolchain that enables a WYSIWYG configuration authoring, automation, and delivery experience,
which simplifies managing Kubernetes platforms and KRM-driven infrastructure (e.g.,
[Config Connector](https://github.com/GoogleCloudPlatform/k8s-config-connector), [Crossplane](https://crossplane.io)) at
scale by manipulating declarative [Configuration as Data](docs/design-docs/06-config-as-data.md).

*Configuration as Data* is an approach to management of configuration which:

* makes configuration data the source of truth, stored separately from the live
  state
* uses a uniform, serializable data model to represent configuration
* separates code that acts on the configuration from the data and from packages
  / bundles of the data
* abstracts configuration file structure and storage from operations that act
  upon the configuration data; clients manipulating configuration data don’t
  need to directly interact with storage (git, container images).

See [the FAQ](https://kpt.dev/faq/) for more details about how kpt is different from alternatives.

Use our [public Dosu space](https://github.dosu.com/kptdev/kpt) to ask anything about kpt.

## Why kpt?

kpt enables WYSIWYG editing and interoperable automation applied to declarative configuration data, similar to how the
live state can be modified with imperative tools. 

See [the rationale](https://kpt.dev/guides/rationale) for more background.

The best place to get started and learn about specific features of kpt is to visit the [kpt website](https://kpt.dev/).

## Install kpt

kpt installation instructions can be found on [kpt.dev/installation/kpt-cli](https://kpt.dev/installation/kpt-cli/)

**Quick Install**:
```bash
# macOS (Homebrew)
brew install kpt

# Linux
curl -L https://github.com/kptdev/kpt/releases/latest/download/kpt_linux_amd64 -o kpt
chmod +x kpt
sudo mv kpt /usr/local/bin/

# Verify installation
kpt version
```

**Version Information**: kpt follows [semantic versioning](https://semver.org/). See [VERSIONING.md](docs/VERSIONING.md) for our versioning policy and compatibility guarantees.

## kpt components

The kpt toolchain includes the following components:

- **kpt CLI**: The [kpt CLI](https://kpt.dev/reference/cli/) supports package and function operations, and also
  deployment, via either direct apply or GitOps. By keeping an inventory of deployed resources, kpt enables resource
  pruning, aggregated status and observability, and an improved preview experience.

- [**Function SDK**](https://github.com/kptdev/krm-functions-sdk): Any general-purpose or domain-specific language can
  be used to create functions to transform and/or validate the YAML KRM input/output format, but we provide SDKs to
  simplify the function authoring process in [Go](https://kpt.dev/book/05-developing-functions/#developing-in-Go).

- [**Function catalog**](https://github.com/kptdev/krm-functions-catalog): A [catalog](https://catalog.kpt.dev/function-catalog) of
  off-the-shelf, tested functions. kpt makes configuration easy to create and transform, via reusable functions. Because
  they are expected to be used for in-place transformation, the functions need to be idempotent.

## Roadmap

You can read about the big upcoming features in the [roadmap doc](/docs/ROADMAP.md).

## Documentation

- **[Versioning Policy](docs/VERSIONING.md)** - Semantic versioning and compatibility guarantees
- **[Migration Guide](docs/MIGRATION_V1.md)** - Migrating to kpt v1.0.0
- **[Backward Compatibility](docs/BACKWARD_COMPATIBILITY.md)** - Compatibility policy and testing
- **[Design Docs](docs/design-docs/)** - Technical design documents
- **[Style Guides](docs/style-guides/)** - Documentation and error message guidelines

## Contributing

If you are interested in contributing please start with [contribution guidelines](CONTRIBUTING.md).

## Contact

We would love to keep in touch:

1. Join our [Slack channel](https://kubernetes.slack.com/channels/kpt). You'll
   need to join [Kubernetes on Slack](https://slack.k8s.io/) first.
1. Join our [Discussions](https://github.com/kptdev/kpt/discussions)
1. Join our [community meetings](https://zoom-lfx.platform.linuxfoundation.org/meeting/98980817322?password=c09cdcc7-59c0-49c4-9802-ad4d50faafcd&invite=true)

## License

Code is under the [Apache License 2.0](LICENSE), documentation is [CC BY 4.0](LICENSE-documentation).

### License scanning status

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkptdev%2Fkpt.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkptdev%2Fkpt?ref=badge_large)

## Governance

The governance of the kpt project and KRM Functiona Catalog are described in the
[governance repo](https://github.com/kptdev/governance).

## Code of Conduct

The kpt project and the KRM Functions Catalog are following the
[CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).
More information and links about the CNCF Code of Conduct are [here](code-of-conduct.md).

## CNCF

The kpt project including the KRM Functions Catalog is a [CNCF Sandbox](https://www.cncf.io/sandbox-projects/) project.

