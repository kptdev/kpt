# kpt

kpt is a git-native, schema-aware, extensible client-side tool for packaging, 
customizing, validating, and applying Kubernetes resources.

## Why kpt?

1. kpt allows you to share, use and update packages of Kubernetes resources
using any git repo.  No special setup is necessary.
2. kpt allows customization of packages using an editor of your choice. The 
resource merge feature of kpt will handle a lot of the scenarios of merging 
upstream changes on update.
3. Customization in kpt is done without templates, domain specific languages
and paramters. Any engineer who is familiar with Kubernetes to work on the 
infrastructure configuration.
4. kpt apply addresses some of the functional gaps in `kubectl apply` such as
pruning and reconciling status.

The best place to get started and learn about specific features of kpt is 
to visit the [kpt website](https://kpt.dev/).

### Install kpt

kpt installation instructions can be found on 
[kpt.dev/installation](https://kpt.dev/installation/)

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